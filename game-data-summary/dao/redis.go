package dao

import (
	"app/config"
	"app/esindex"
	"context"
	"fmt"
	"game-data-summary/entity"
	"game-data-summary/event"
	"game-data-summary/util"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	jsoniter "github.com/json-iterator/go"

	"micro_service/services"

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
	util.ParseConfig(msg.Key, msg.Data)
}

func (rd *RedisDao) GetPlayerOrigin(playerId uint32) (*services.HumanPlayer, string, error) {
	res, err := rd.Client.HGetAll(context.Background(), "player_"+strconv.FormatUint(uint64(playerId), 10)).Result()
	if err == redis.Nil {
		return nil, "", nil
	} else if err != nil {
		return nil, "", err
	}

	if len(res) == 0 {
		return nil, "", nil
	}

	var p services.HumanPlayer
	var exp string
	zap.L().Info("Redis:GetPlayer", zap.Uint32("player_id", playerId), zap.Any("player", res))

	for key, value := range res {
		switch key {
		case "id":
			id, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return nil, "", err
			}
			p.Id = uint32(id)
		case "nickname":
			p.Nickname = value
		case "currency":
			strCurrency, err := decimal.NewFromString(value)
			if err != nil {
				return nil, "", err
			}
			// redis中存储的单位为分，这里需要转换一下
			p.GameCurrency = strCurrency.Div(decimal.New(1, 2)).StringFixed(2)
		case "avatar":
			p.Avatar = value
		case "gender":
			gender, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return nil, "", err
			}
			p.Gender = uint32(gender)
		case "exp":
			exp = value
		case "agent_id":
			agentId, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return nil, "", err
			}
			p.AgentId = uint32(agentId)
		case "login_ip":
			p.LoginIP = value
		case "login_time":
			loginTimestamp, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, "", err
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
				return nil, "", err
			}
			p.WebSiteId = uint32(websiteId)
		case "all_times":
			allTimes, _ := strconv.Atoi(value)
			p.AllTimes = int32(allTimes)
		case "ex_data":
			p.ExData = value
		}
	}

	return &p, exp, nil
}

func (rd *RedisDao) GetPlayer(playerId, factory uint32) (*services.HumanPlayer, error) {
	res, err := rd.Client.HGetAll(context.Background(), "player_"+strconv.FormatUint(uint64(playerId), 10)).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
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
			pack := &entity.Msg{}
			if err := jsoniter.UnmarshalFromString(msg.Payload, pack); err != nil {
				zap.L().Error("收到无法解析的推送", zap.Any("msg", msg))
			} else {
				event.OnEvent(pack)
			}
		}
	}()
}

func (rd *RedisDao) ChangeOnlineCount(agentId, count int32) error {
	effectBetKey := "online_count"
	itemKey := fmt.Sprintf("%d", agentId)
	if err := rd.Client.ZIncrBy(context.Background(), effectBetKey, float64(count), itemKey).Err(); err != nil {
		zap.L().Error("更新在线人数失败", zap.Error(err))
		return err
	}
	return nil
}

func (rd *RedisDao) UpdatePlayerAvatarAndGender(playerId uint32, avatar, name string, gender uint32) error {
	if len(avatar) == 0 && gender == 0 {
		return nil
	}
	pID := "player_" + strconv.FormatUint(uint64(playerId), 10)
	exist, err := rd.Client.HExists(context.Background(), pID, "id").Result() // 完整的player信息是否存在
	if err == redis.Nil || (err == nil && !exist) {
		return redis.Nil
	} else if err != nil {
		return err
	}
	piple := rd.Client.Pipeline()
	if len(avatar) != 0 {
		if err := piple.HSet(context.Background(), pID, "avatar", avatar).Err(); err != nil {
			return err
		}
	}
	if len(name) != 0 {
		if err := piple.HSet(context.Background(), pID, "nickname", name).Err(); err != nil {
			return err
		}
	}
	if gender != 0 {
		if err := piple.HSet(context.Background(), pID, "gender", gender).Err(); err != nil {
			return err
		}
	}
	piple.Expire(context.Background(), pID, time.Minute*20)
	piple.SAdd(context.Background(), "dirty_list", playerId)
	piple.Exec(context.Background())
	return err
}

func (rd *RedisDao) IncrSerialIdSeed() (int64, error) {
	return rd.Client.Incr(context.Background(), "serial_id_seed").Result()
}

func (rd *RedisDao) UpdatePlayerCurrencyAndExp(playerId uint32, currencyDelta int64, exp, factory uint32, source int) (newCurrency int64, err error) {
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
		return 0, fmt.Errorf("玩家分数更新后不能小于0。玩家ID:%s，玩家在redis中的游戏币数量：%d，更新量：%d", pID, currentCurrency, currencyDelta)
	}
	newCurrency, err = rd.Client.HIncrBy(context.Background(), pID, "currency", currencyDelta).Result()
	if err != nil {
		return 0, err
	}
	rd.Client.HIncrBy(context.Background(), pID, "exp", int64(exp))
	rd.Client.SAdd(context.Background(), "dirty_list_imp", playerId)
	return newCurrency, nil
}

func (rd *RedisDao) UpdatePlayerLoginIPAndTime(playerId uint32, ip string) error {
	pID := "player_" + strconv.FormatUint(uint64(playerId), 10)
	exist, err := rd.Client.HExists(context.Background(), pID, "id").Result() // 完整的player信息是否存在
	if err == redis.Nil || (err == nil && !exist) {
		return redis.Nil
	} else if err != nil {
		return err
	}
	if err := rd.Client.HSet(context.Background(), pID, map[string]interface{}{
		"login_ip":   ip,
		"login_time": time.Now().Unix(),
	}).Err(); err != nil {
		return err
	}
	rd.Client.Expire(context.Background(), pID, time.Minute*20)
	rd.Client.SAdd(context.Background(), "dirty_list", playerId)
	return err
}

func (rd *RedisDao) SetPlayer(p *services.HumanPlayer, refreshDB bool) error {
	pID := "player_" + strconv.FormatUint(uint64(p.Id), 10)
	exist, err := rd.Client.HExists(context.Background(), pID, "id").Result()
	if err != nil && err != redis.Nil {
		return err
	}
	if exist {
		return nil
	}
	err = rd.Client.HSet(context.Background(), pID, map[string]interface{}{
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
		// "revenue":        p.Revenue,
	}).Err()
	if err == nil {
		rd.Client.Expire(context.Background(), pID, time.Minute*20)
		rd.Client.SAdd(context.Background(), "dirty_list", p.Id)
	}
	return err
}

func (rd *RedisDao) GetPlayerStatistics(key string) *services.PlayerStatistics {
	res, err := rd.Client.HGetAll(context.Background(), key).Result()
	if err == redis.Nil {
		return nil
	} else if err != nil {
		return nil
	}
	if len(res) == 0 {
		return nil
	}
	var ps services.PlayerStatistics
	zap.L().Info("Redis:PlayerStatistics", zap.String("key", key), zap.Any("PlayerStatistics", res))
	for key, value := range res {
		switch key {
		case "times":
			times, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return nil
			}
			ps.Times = uint32(times)
		case "bet_times":
			bet, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return nil
			}
			ps.Bet = uint32(bet)
		case "incr_times":
			incr, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return nil
			}
			ps.Incr = uint32(incr)
		}
	}
	return &ps
}

func (rd *RedisDao) SetPlayerStatistics(key string, p *services.PlayerStatistics) error {
	exist, err := rd.Client.Exists(context.Background(), key).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	if exist <= 0 {
		return nil
	}
	err = rd.Client.HSet(context.Background(), key, map[string]interface{}{
		"times":      p.Times,
		"bet_times":  p.Bet,
		"incr_times": p.Incr,
	}).Err()
	if err == nil {
		rd.Client.Expire(context.Background(), key, time.Minute*20)
		// rd.cli.SAdd(context.Background(), "dirty_list", p.Id)
	}
	return err
}

// 获取token字符串
func (rd *RedisDao) GetTokenString(key string) (string, error) {
	return rd.Client.Get(context.Background(), key).Result()
}

func (rd *RedisDao) HGet(key, attrKey string) (string, error) {
	return rd.Client.HGet(context.Background(), key, attrKey).Result()
}

func (rd *RedisDao) Set(key, value string, timeout int32) error {
	var to time.Duration
	if timeout > 0 {
		to = time.Duration(timeout) * time.Second
	} else {
		to = -1
	}

	return rd.Client.Set(context.Background(), key, value, to).Err()
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

func (rd *RedisDao) ScoreLock(userId, timeOut uint32) string {
	key := fmt.Sprintf("score_lock_%d", userId)
	uid, _ := uuid.NewV4()
	success, _ := rd.Client.SetNX(context.Background(), key, uid.String(), time.Duration(timeOut)*time.Second).Result()
	if success {
		return uid.String()
	}
	return ""
}

func (rd *RedisDao) ScoreUnLock(userId uint32, token string) bool {
	script := redis.NewScript(`
	if redis.call('get', KEYS[1]) == ARGV[1] then
		return redis.call('del', KEYS[1])
	else
		return 0
	end`)
	n, err := script.Run(context.Background(), rd.Client, []string{fmt.Sprintf("score_lock_%d", userId)}, []string{token}).Int()
	if err != nil {
		zap.L().Debug("积分锁 解锁失败", zap.Any("userId", userId), zap.Any("n", n), zap.Any("err", err))
		return false
	} else {
		if n > 0 {
			zap.L().Debug("积分锁 解锁成功", zap.Any("userId", userId), zap.Any("n", n))
		} else {
			zap.L().Debug("积分锁 解锁失败", zap.Any("userId", userId), zap.Any("n", n), zap.Any("err", err))
		}
	}
	return true
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
				util.ParseConfig(tmpKey[p], cmd.Val())
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
			util.ParseConfig(tmpKey[p], cmd.Val())
		}
	}
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
			System: &config.SystemConfig{},
			Currency: &config.CurrencyMgr{
				Data: make(map[string]decimal.Decimal),
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
