package dao

import (
	"app/cfgparse"
	"app/config"
	"app/entity"
	"app/esindex"
	"context"
	"fmt"
	"micro_service/services"
	"os"
	"reflect"
	"web-api/event"

	"strconv"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	yaml "sigs.k8s.io/yaml/goyaml.v2"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

var redisDao *RedisDao = nil

type RedisDao struct {
	Client redis.UniversalClient
}

func RedisIns() *RedisDao {
	return redisDao
}

func NewRedisDao(hosts []string, user, pwd string) {
	if redisDao == nil {
		cli := redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs:    hosts,
			Password: pwd,
			Username: user,
			DB:       0,
		})
		redisDao = &RedisDao{Client: cli}
		redisDao.Subscribe("message", func() *event.EventMgr {
			e := &event.EventMgr{
				Events: make(map[string]*event.Event),
			}
			e.Register("config", reflect.TypeOf(entity.ConfigMsg{}), ConfigHandler)
			return e
		})
	}
}

// 配置同步
func ConfigHandler(data interface{}) {
	msg := data.(*entity.ConfigMsg)
	ParseConfig(msg.Key, msg.Data)
}

// // 添加单控
// func AddPlayerSingleCtrlHandler(data interface{}) {
// 	addCtrl := data.(entity.AddPlayerSingleCtrl)
// 	SCIns().addSingleCtrl(addCtrl.Uid, addCtrl.Rate, addCtrl.CtrlScore, SYSTEM_CTRL)
// }

// // 移除单控
// func DelPlayerSingleCtrlHandler(data interface{}) {
// 	addCtrl := data.(entity.DelPlayerSingleCtrl)
// 	SCIns().delSingleCtrl(addCtrl.Uid)
// }

func (rd *RedisDao) GetPlayer(playerId, factory uint32) (*services.HumanPlayer, error) {
	res, err := rd.Client.HGetAll(context.Background(), "player_"+strconv.FormatUint(uint64(playerId), 10)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}
	var p services.HumanPlayer
	zap.L().Info("Redis:GetPlayer", zap.Uint32("player_id", playerId), zap.Any("player", res))
	for key, value := range res {
		switch key {
		case "id":
			id, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return nil, err
			}
			p.Id = uint32(id)
		case "nickname":
			p.Nickname = value
		case "currency":
			strCurrency, err := decimal.NewFromString(value)
			if err != nil {
				return nil, err
			}
			// redis中存储的单位为分，这里需要转换一下
			p.GameCurrency = strCurrency.Div(decimal.New(1, 2)).StringFixed(2)
		case "avatar":
			p.Avatar = value
		case "gender":
			gender, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return nil, err
			}
			p.Gender = uint32(gender)
		case "exp":
		case "agent_id":
			agentId, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return nil, err
			}
			p.AgentId = uint32(agentId)
		case "login_ip":
			p.LoginIP = value
		case "login_time":
			loginTimestamp, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, err
			}
			p.LoginTimeStamp = loginTimestamp
		case "currency_limit":
			p.CurrencyLimit = value
		case "account":
			p.Account = value
		case "currency_type":
			p.CurrencyType = value
		case "website_id":
			websiteId, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return nil, err
			}
			p.WebSiteId = uint32(websiteId)
		case "all_times":
			allTimes, _ := strconv.Atoi(value)
			p.AllTimes = int32(allTimes)
		case "ex_data":
			p.ExData = value
		}
	}
	return &p, nil
}

func (rd *RedisDao) SetPlayer(p *services.HumanPlayer) error {
	pID := "player_" + strconv.FormatUint(uint64(p.Id), 10)
	err := rd.Client.HSet(context.Background(), pID, map[string]interface{}{
		"id":             p.Id,
		"nickname":       p.Nickname,
		"currency":       decimal.RequireFromString(p.GameCurrency).Mul(decimal.New(1, 2)).IntPart(), // 方便incr，转换为分
		"avatar":         p.Avatar,
		"gender":         p.Gender,
		"exp":            p.Exp,
		"agent_id":       p.AgentId,
		"login_ip":       p.LoginIP,
		"login_time":     p.LoginTimeStamp,
		"currency_limit": p.CurrencyLimit,
		"website_id":     p.WebSiteId,
		"account":        p.Account,
		"currency_type":  p.CurrencyType,
		"all_times":      p.AllTimes,
		"ex_data":        p.ExData,
	}).Err()
	if err == nil {
		rd.Client.Expire(context.Background(), pID, time.Minute*20)
		rd.Client.SAdd(context.Background(), "dirty_list", p.Id)
	}
	return err
}

func (rd *RedisDao) UpdatePlayerCurrency(playerId uint32, currencyDelta int64, exp, factory uint32, source int) (newCurrency int64, err error) {
	pID := "player_" + strconv.FormatUint(uint64(playerId), 10)
	rd.Client.Expire(context.Background(), pID, time.Minute*20)
	exist, err := rd.Client.HExists(context.Background(), pID, "id").Result() // 完整的player信息是否存在
	if err == redis.Nil || (err == nil && !exist) {
		return 0, redis.Nil
	} else if err != nil {
		return 0, err
	}
	currentCurrency, err := rd.Client.HGet(context.Background(), pID, "currency").Int64()
	if err != nil {
		return 0, err
	}
	if currentCurrency+currencyDelta < 0 {
		newCurrency = currentCurrency
		return 0, fmt.Errorf("玩家分数更新后不能小于0。玩家ID:%s,玩家在redis中的游戏币数量:%d,更新量：%d", pID, currentCurrency, currencyDelta)
	}
	newCurrency, err = rd.Client.HIncrBy(context.Background(), pID, "currency", currencyDelta).Result()
	if err != nil {
		return 0, err
	}
	rd.Client.HIncrBy(context.Background(), pID, "exp", int64(exp))
	rd.Client.SAdd(context.Background(), "dirty_list_imp", playerId)
	return newCurrency, nil
}

func (rd *RedisDao) GetUserStatData(id uint32) (decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	itemKey := fmt.Sprintf("%d", id)
	piple := rd.Client.Pipeline()
	piple.ZIncrBy(context.Background(), "userTotalEffBet", 0, itemKey)
	piple.ZIncrBy(context.Background(), "userTotalProfLoss", 0, itemKey)
	piple.ZIncrBy(context.Background(), "userBetCount", 0, itemKey)
	result, err := piple.Exec(context.Background())
	if err == nil {
		eff, effErr := result[0].(*redis.FloatCmd).Result()
		if effErr != nil {
			zap.L().Error("获取有效投注失败", zap.Error(effErr))
			return decimal.Zero, decimal.Zero, decimal.Zero
		}
		pro, proErr := result[1].(*redis.FloatCmd).Result()
		if proErr != nil {
			zap.L().Error("获取盈亏信息失败", zap.Error(proErr))
			return decimal.Zero, decimal.Zero, decimal.Zero
		}
		count, cError := result[2].(*redis.FloatCmd).Result()
		if cError != nil {
			zap.L().Error("获取下注次数失败", zap.Error(proErr))
			return decimal.Zero, decimal.Zero, decimal.Zero
		}
		return decimal.NewFromFloat(eff), decimal.NewFromFloat(pro), decimal.NewFromFloat(count)
	}
	return decimal.Zero, decimal.Zero, decimal.Zero
}

// 批量更新赊账信息
func (rd *RedisDao) BitchUpdateOnCredit(data map[string]decimal.Decimal) map[string]decimal.Decimal {
	n, r, keys := 0, make(map[string]decimal.Decimal), make([]string, 0, 100)
	piple := rd.Client.Pipeline()
	for k, v := range data {
		n++
		keys = append(keys, k)
		piple.ZIncrBy(context.Background(), "on_credit", v.InexactFloat64()*-1, k)
		if n >= 100 {
			result, err := piple.Exec(context.Background())
			if err != nil {
				zap.L().Error("批量更新赊账失败", zap.Any("err", err), zap.Any("data", data))
			} else {
				for k, v := range result {
					c := v.(*redis.FloatCmd)
					if c.Err() == nil {
						r[keys[k]] = decimal.NewFromFloat(c.Val())
					}
				}
			}
			piple = rd.Client.Pipeline()
			keys = make([]string, 0, 100)
		}
	}
	if len(keys) > 0 {
		result, err := piple.Exec(context.Background())
		if err != nil {
			zap.L().Error("批量更新赊账失败", zap.Any("err", err), zap.Any("data", data))
		} else {
			for k, v := range result {
				c := v.(*redis.FloatCmd)
				if c.Err() == nil {
					r[keys[k]] = decimal.NewFromFloat(c.Val())
				}
			}
		}
	}
	return r
}

// 更新赊账信息
func (rd *RedisDao) UpdateOnCredit(key string, data decimal.Decimal) (decimal.Decimal, bool) {
	c := rd.Client.ZIncrBy(context.Background(), "on_credit", data.InexactFloat64(), key)
	if c.Err() != nil {
		return decimal.Zero, false
	}
	return decimal.NewFromFloat(c.Val()), true
}

func (rd *RedisDao) HGet(key, attrKey string) (string, error) {
	return rd.Client.HGet(context.Background(), key, attrKey).Result()
}

func (rd *RedisDao) Set(key string, value interface{}, timeout int32) error {
	var to time.Duration
	if timeout > 0 {
		to = time.Duration(timeout) * time.Second
	} else {
		to = -1
	}

	return rd.Client.Set(context.Background(), key, value, to).Err()
}

func (rd *RedisDao) Publish(topic, msg string) error {
	return rd.Client.Publish(context.Background(), topic, msg).Err()
}

func (rd *RedisDao) Del(key string) error {
	return rd.Client.Del(context.Background(), key).Err()
}

func (rd *RedisDao) Get(key string) (string, error) {
	return rd.Client.Get(context.Background(), key).Result()
}

func (rd *RedisDao) HSet(key, value, attrKey string) error {
	return rd.Client.HSet(context.Background(), key, attrKey, value).Err()
}

func (rd *RedisDao) HDel(key, attrKey string) error {
	return rd.Client.HDel(context.Background(), key, attrKey).Err()
}

func (rd *RedisDao) SetKeyTimeOut(key string, timeout int32) (bool, error) {
	if timeout < 0 {
		return rd.Client.Persist(context.Background(), key).Result()
	}

	return rd.Client.PExpire(context.Background(), key, time.Duration(timeout)*time.Second).Result()
}

func (rd *RedisDao) Subscribe(channel string, registryInfo func() *event.EventMgr) {
	//watch
	sub := rd.Client.Subscribe(context.Background(), channel)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				zap.L().Error("panic", zap.Any("err", err))
			}
		}()
		//注册处理方法
		event := registryInfo()
		//关闭订阅
		defer sub.Close()
		for msg := range sub.Channel() {
			func() {
				defer func() {
					if err := recover(); err != nil {
						zap.L().Error("处理广播消息panic", zap.Any("error", err))
					}
				}()
				pack := &entity.Msg{}
				if err := jsoniter.UnmarshalFromString(msg.Payload, pack); err != nil {
					zap.L().Error("收到无法解析的推送", zap.Any("msg", msg.Payload), zap.Any("err", err))
					return
				}
				zap.L().Debug("收到事件广播", zap.Any("data", pack))
				event.OnEvent(pack)
			}()
		}
	}()
}

func LoadConfigs(client *RedisDao, prefix string) {
	//初始化加载所有配置
	iter := client.Client.Scan(context.Background(), 0, prefix, 0).Iterator()
	keys := make([]string, 0, 256)
	for iter.Next(context.Background()) {
		k := iter.Val()
		if !esindex.MatchKeyPrefix(k) {
			continue
		}
		keys = append(keys, k)
	}
	if err := iter.Err(); err != nil {
		panic(err)
	}
	tmpKey := make([]string, 0, 256)
	//加载数据
	pipe := client.Client.Pipeline()
	for _, k := range keys {
		pipe.Get(context.Background(), k)
		tmpKey = append(tmpKey, k)
		if len(tmpKey) >= 256 {
			result, err := pipe.Exec(context.Background())
			if err != nil {
				zap.L().Fatal("加载配置数据失败", zap.Any("err", err))
			}
			for p, r := range result {
				cmd := r.(*redis.StringCmd)
				ParseConfig(tmpKey[p], cmd.Val())
			}
			tmpKey = make([]string, 0, 256)
			pipe = client.Client.Pipeline()
		}
	}
	if len(tmpKey) > 0 {
		//加载数据
		result, err := pipe.Exec(context.Background())
		if err != nil {
			zap.L().Fatal("加载配置数据失败", zap.Any("err", err))
		}
		for p, r := range result {
			cmd := r.(*redis.StringCmd)
			ParseConfig(tmpKey[p], cmd.Val())
		}
	}
}

type PoolValueItem struct {
	EffValue decimal.Decimal
	ProValue decimal.Decimal
	Revenue  decimal.Decimal
}

func (pvi *PoolValueItem) GetValue(r decimal.Decimal) decimal.Decimal {
	return pvi.ProValue.Neg().Sub(pvi.Revenue).Add(r)
}

// 初始化配置信息
func ConfigsInit() {
	if config.CfgIns == nil {
		config.CfgIns = &config.Configs{
			Lock: &sync.RWMutex{},
			Pool: &config.PoolMgr{
				Default: make(map[string]*config.Pool),
				Agent:   make(map[string]*config.Pool),
			},
			Award: &config.AwardMgr{
				Data: make(map[string]*config.AwardConfig),
			},
			Coefficient: config.DefaultControlCoefficientConfig(),
			System:      &config.SystemConfig{},
			Currency: &config.CurrencyMgr{
				Data: make(map[string]decimal.Decimal),
			},
			AC: &config.AutoCtrlMgr{
				Ctrls: make([]*config.AutoCtrlItem, 0),
			},
		}
		//加载默认配置
		LoadConfigs(RedisIns(), esindex.ConfigPattern())
		//加载代理配置
		LoadConfigs(RedisIns(), esindex.AgentPattern())
	}
}

func LoadConfig() *config.SystemConfig {
	var sysConfig *config.SystemConfig = &config.SystemConfig{}
	value, err := RedisIns().Get(esindex.ConfigKey("system"))
	if err != nil {
		zap.L().Fatal("获取系统配置失败", zap.Any("err", err))
	}
	err = jsoniter.UnmarshalFromString(value, sysConfig)
	if err != nil {
		zap.L().Fatal("解析系统配置失败", zap.Any("err", err))
	}
	return sysConfig
}

// 解析配置
func ParseConfig(key string, value string) {
	cfgparse.Parse(key, value)
}

// 初始化基础配置
func InitBaseConfig(path string) *config.RunConfig {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		zap.L().Fatal("读取基础配置失败", zap.Any("error", err))
	}
	config := &config.RunConfig{}
	err = yaml.Unmarshal(yamlFile, config)
	if err != nil {
		zap.L().Fatal("解析基础配置异常", zap.Any("error", err))
	}
	zap.L().Debug("=====>", zap.Any("config", config))
	return config
}
