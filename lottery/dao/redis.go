package dao

import (
	. "app/config"
	"app/entity"
	"app/esindex"
	"context"
	"fmt"
	"lottery/event"
	"lottery/util"
	"micro_service/services"
	"reflect"

	"strconv"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

var redisDao *RedisDao = nil

type RedisDao struct {
	cli redis.UniversalClient
}

func RedisIns() *RedisDao {
	return redisDao
}

func NewRedisDao(hosts []string, user, pwd string) {
	if redisDao == nil {
		cli := redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs:                 hosts,
			Password:              pwd,
			Username:              user,
			DB:                    0,
			PoolSize:              100,
			MinIdleConns:          30,
			MaxIdleConns:          0,
			ConnMaxIdleTime:       60 * time.Second,
			ConnMaxLifetime:       5 * time.Minute,
			ReadBufferSize:        32 * 1024,
			WriteBufferSize:       32 * 1024,
			PoolTimeout:           5 * time.Second,
			ReadTimeout:           10 * time.Second,
			WriteTimeout:          10 * time.Second,
			ContextTimeoutEnabled: true,
		})
		redisDao = &RedisDao{cli: cli}
		redisDao.Subscribe("message", func() *event.EventMgr {
			e := &event.EventMgr{
				Events: make(map[string]*event.Event),
			}
			e.Register("config", reflect.TypeOf(entity.ConfigMsg{}), ConfigHandler)
			e.Register("addSingleCtrl", reflect.TypeOf(entity.AddPlayerSingleCtrl{}), AddPlayerSingleCtrlHandler)
			e.Register("delSingleCtrl", reflect.TypeOf(entity.DelPlayerSingleCtrl{}), DelPlayerSingleCtrlHandler)
			e.Register("addGame", reflect.TypeOf(entity.AddGame{}), AddGame)
			e.Register("gameStatuChange", reflect.TypeOf(entity.GameStatuChange{}), GameStatusChange)
			e.Register("addAgent", reflect.TypeOf(entity.AddAgent{}), AgentStatusChange)
			e.Register("agentStatuChange", reflect.TypeOf(entity.AgentStatuChange{}), AgentStatusChange)
			e.Register("resetPool", reflect.TypeOf(entity.ResetPool{}), ResetPool)
			return e
		})
	}
}

// 配置同步
func ConfigHandler(data interface{}) {
	msg := data.(*entity.ConfigMsg)
	util.ParseConfig(msg.Key, msg.Data)
}

// 添加单控
func AddPlayerSingleCtrlHandler(data interface{}) {
	addCtrl := data.(*entity.AddPlayerSingleCtrl)
	SCIns().addSingleCtrl(addCtrl.Uid, addCtrl.Rate, addCtrl.CtrlScore, SYSTEM_CTRL)
}

// 移除单控
func DelPlayerSingleCtrlHandler(data interface{}) {
	addCtrl := data.(*entity.DelPlayerSingleCtrl)
	SCIns().delSingleCtrl(addCtrl.Uid)
}

func AddGame(data interface{}) {
	add := data.(*entity.AddGame)
	zap.L().Debug("添加游戏配置", zap.Any("add", add))
	GamesManagerIns().Add(add.Game)
}

func GameStatusChange(data interface{}) {
	change := data.(*entity.GameStatuChange)
	zap.L().Debug("修改游戏状态", zap.Any("change", change))
	if g := GamesManagerIns().Get(change.Symbol); g != nil {
		g.State = int16(change.Status)
	}
}

func AddAgent(data interface{}) {
	add := data.(*entity.AddAgent)
	zap.L().Debug("添加代理配置", zap.Any("add", add))
	AgentManagerIns().Add(add.Agent)
}

func AgentStatusChange(data interface{}) {
	change := data.(*entity.AgentStatuChange)
	zap.L().Debug("修改代理状态", zap.Any("change", change))
	AgentManagerIns().ChangeStatus(change.Id, change.Status)
}

func (rd *RedisDao) GetPlayer(playerId, factory uint32) (*services.HumanPlayer, error) {
	res, err := rd.cli.HGetAll(context.Background(), "player_"+strconv.FormatUint(uint64(playerId), 10)).Result()
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
		case "isTourist":
			isTourist, _ := strconv.Atoi(value)
			p.IsTourist = int32(isTourist)
		case "ex_data":
			p.ExData = value
		}
	}
	return &p, nil
}

func (rd *RedisDao) SetPlayer(p *services.HumanPlayer) error {
	pID := "player_" + strconv.FormatUint(uint64(p.Id), 10)
	pipe := rd.cli.Pipeline()
	pipe.HSet(context.Background(), pID, map[string]interface{}{
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
		"isTourist":      p.IsTourist,
		"ex_data":        p.ExData,
	})
	pipe.Expire(context.Background(), pID, time.Minute*20)
	pipe.SAdd(context.Background(), "dirty_list", p.Id)
	_, err := pipe.Exec(context.Background())
	return err
}

func (rd *RedisDao) UpdatePlayerExData(playerId uint32, exData string) error {
	pID := "player_" + strconv.FormatUint(uint64(playerId), 10)
	exist, err := rd.cli.HExists(context.Background(), pID, "id").Result()
	if err == redis.Nil || (err == nil && !exist) {
		return redis.Nil
	} else if err != nil {
		return err
	}
	pipe := rd.cli.Pipeline()
	pipe.HSet(context.Background(), pID, "ex_data", exData)
	pipe.Expire(context.Background(), pID, time.Minute*20)
	pipe.SAdd(context.Background(), "dirty_list", playerId)
	_, err = pipe.Exec(context.Background())
	return err
}

func (rd *RedisDao) UpdatePlayerCurrency(playerId uint32, currencyDelta int64, exp, factory uint32, source int) (newCurrency int64, err error) {
	pID := "player_" + strconv.FormatUint(uint64(playerId), 10)
	pipe := rd.cli.Pipeline()
	pipe.HExists(context.Background(), pID, "id").Result() // 完整的player信息是否存在
	pipe.Expire(context.Background(), pID, time.Minute*20)
	pipe.HGet(context.Background(), pID, "currency")
	result, err := pipe.Exec(context.Background())
	if err != nil {
		return 0, err
	}
	rb := result[0].(*redis.BoolCmd)
	exist, err := rb.Result()
	if err == redis.Nil || (err == nil && !exist) {
		return 0, redis.Nil
	} else if err != nil {
		return 0, err
	}
	cr := result[2].(*redis.StringCmd)
	currentCurrency, err := cr.Int64()
	if err != nil {
		return 0, err
	}
	if currentCurrency+currencyDelta < 0 {
		newCurrency = currentCurrency
		return 0, fmt.Errorf("玩家分数更新后不能小于0。玩家ID:%s,玩家在redis中的游戏币数量:%d,更新量：%d", pID, currentCurrency, currencyDelta)
	}
	pipe = rd.cli.Pipeline()
	pipe.HIncrBy(context.Background(), pID, "currency", currencyDelta)
	pipe.HIncrBy(context.Background(), pID, "exp", int64(exp))
	pipe.SAdd(context.Background(), "dirty_list_imp", playerId)
	result, err = pipe.Exec(context.Background())
	if err != nil {
		return 0, err
	}
	crr := result[0].(*redis.IntCmd)
	newCurrency, err = crr.Result()
	if err != nil {
		return 0, err
	}
	return newCurrency, nil
}

func (rd *RedisDao) GetPlayerCurrency(playerId uint32) (newCurrency int64, err error) {
	pID := "player_" + strconv.FormatUint(uint64(playerId), 10)
	pipe := rd.cli.Pipeline()
	pipe.HExists(context.Background(), pID, "id").Result() // 完整的player信息是否存在
	pipe.Expire(context.Background(), pID, time.Minute*20)
	pipe.HGet(context.Background(), pID, "currency")
	result, e := pipe.Exec(context.Background())
	if e != nil {
		return 0, e
	}
	sc := result[2].(*redis.StringCmd)
	nc, err := sc.Result()
	if err != nil {
		return 0, err
	}
	newCurrency, err = strconv.ParseInt(nc, 10, 64)
	if err != nil {
		return 0, err
	}
	return newCurrency, nil
}

// 批量获取玩家金币信息，只有再玩家登录后才能使用该接口 且 返回金币是未换算的值
func (rd *RedisDao) BatchGetPlayerCurrencys(ids []uint32) (map[uint32]int64, error) {
	currencys := make(map[uint32]int64)
	pipe := rd.cli.Pipeline()
	for _, id := range ids {
		pID := fmt.Sprintf("player_%d", id)
		pipe.HGet(context.Background(), pID, "currency")
	}
	result, e := pipe.Exec(context.Background())
	if e != nil {
		return nil, e
	}
	for i, id := range ids {
		sc := result[i].(*redis.StringCmd)
		nc, err := sc.Result()
		if err != nil {
			continue
		}
		newCurrency, _ := strconv.ParseInt(nc, 10, 64)
		currencys[id] = newCurrency
	}
	return currencys, nil
}

// 批量修改玩家金币信息，只有再玩家登录后才能使用该接口 且 返回金币是未换算的值
func (rd *RedisDao) BatchUpdatePlayerCurrencys(deltas map[uint32]int64) (map[uint32]int64, error) {
	currencys := make(map[uint32]int64)
	pipe := rd.cli.Pipeline()
	ids := make([]uint32, 0, len(deltas))
	for id, delta := range deltas {
		ids = append(ids, id)
		pID := fmt.Sprintf("player_%d", id)
		pipe.HIncrBy(context.Background(), pID, "currency", delta)
	}
	result, e := pipe.Exec(context.Background())
	if e != nil {
		return nil, e
	}
	for i, id := range ids {
		sc := result[i].(*redis.IntCmd)
		nc, err := sc.Result()
		if err != nil {
			continue
		}
		currencys[id] = nc
	}

	//触发数据落地
	pipe = rd.cli.Pipeline()
	for id := range deltas {
		pipe.SAdd(context.Background(), "dirty_list_imp", id)
	}
	pipe.Exec(context.Background())

	return currencys, nil
}

func (rd *RedisDao) GetUserStatData(id uint32) (decimal.Decimal, decimal.Decimal, decimal.Decimal, string, int32) {
	itemKey := fmt.Sprintf("%d", id)
	piple := rd.cli.Pipeline()
	piple.ZIncrBy(context.Background(), "userTotalEffBet", 0, itemKey)
	piple.ZIncrBy(context.Background(), "userTotalProfLoss", 0, itemKey)
	piple.ZIncrBy(context.Background(), "userBetCount", 0, itemKey)
	piple.HGet(context.Background(), fmt.Sprintf("player_%d", id), "account")
	piple.HGet(context.Background(), fmt.Sprintf("player_%d", id), "isTourist")
	result, err := piple.Exec(context.Background())
	if err == nil {
		eff, effErr := result[0].(*redis.FloatCmd).Result()
		if effErr != nil {
			zap.L().Error("获取有效投注失败", zap.Error(effErr))
			return decimal.Zero, decimal.Zero, decimal.Zero, "", 0
		}
		pro, proErr := result[1].(*redis.FloatCmd).Result()
		if proErr != nil {
			zap.L().Error("获取盈亏信息失败", zap.Error(proErr))
			return decimal.Zero, decimal.Zero, decimal.Zero, "", 0
		}
		count, cError := result[2].(*redis.FloatCmd).Result()
		if cError != nil {
			zap.L().Error("获取下注次数失败", zap.Error(proErr))
			return decimal.Zero, decimal.Zero, decimal.Zero, "", 0
		}
		account, accError := result[3].(*redis.StringCmd).Result()
		if accError != nil {
			zap.L().Error("获取下注次数失败", zap.Error(proErr))
			return decimal.Zero, decimal.Zero, decimal.Zero, "", 0
		}
		isTouristStr, touristErr := result[4].(*redis.StringCmd).Result()
		if touristErr != nil {
			zap.L().Error("获取游客标志失败", zap.Error(proErr))
			return decimal.Zero, decimal.Zero, decimal.Zero, "", 0
		}
		isTourist, _ := strconv.Atoi(isTouristStr)
		return decimal.NewFromFloat(eff), decimal.NewFromFloat(pro), decimal.NewFromFloat(count), account, int32(isTourist)
	}
	return decimal.Zero, decimal.Zero, decimal.Zero, "", 0
}

func (rd *RedisDao) GetGameStatData(id int64, symbol string) (decimal.Decimal, decimal.Decimal, decimal.Decimal, decimal.Decimal, bool) {
	itemKey := fmt.Sprintf("%d_%s", id, symbol)
	piple := rd.cli.Pipeline()
	piple.ZIncrBy(context.Background(), esindex.AgentEffectData(), 0, itemKey)
	piple.ZIncrBy(context.Background(), esindex.AgentChipsData(), 0, itemKey)
	piple.ZIncrBy(context.Background(), esindex.AgentProfitLossData(), 0, itemKey)
	piple.ZIncrBy(context.Background(), esindex.AgentRevenueData(), 0, itemKey)
	result, err := piple.Exec(context.Background())
	if err == nil {
		eff, effErr := result[0].(*redis.FloatCmd).Result()
		if effErr != nil {
			zap.L().Error("获取有效投注失败", zap.Error(effErr))
			return decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, false
		}
		chips, chipsErr := result[1].(*redis.FloatCmd).Result()
		if chipsErr != nil {
			zap.L().Error("获取盈亏信息失败", zap.Error(chipsErr))
			return decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, false
		}
		pro, proErr := result[2].(*redis.FloatCmd).Result()
		if proErr != nil {
			zap.L().Error("获取盈亏信息失败", zap.Error(proErr))
			return decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, false
		}
		revenue, rError := result[3].(*redis.FloatCmd).Result()
		if rError != nil {
			zap.L().Error("获取盈亏信息失败", zap.Error(proErr))
			return decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, false
		}
		return decimal.NewFromFloat(eff), decimal.NewFromFloat(chips), decimal.NewFromFloat(pro), decimal.NewFromFloat(revenue), true
	}
	return decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, false
}

func (rd *RedisDao) HGet(key, attrKey string) (string, error) {
	return rd.cli.HGet(context.Background(), key, attrKey).Result()
}

func (rd *RedisDao) Set(key, value string, timeout int32) error {
	var to time.Duration
	if timeout > 0 {
		to = time.Duration(timeout) * time.Second
	} else {
		to = -1
	}

	return rd.cli.Set(context.Background(), key, value, to).Err()
}

// SetNX 仅当 key 不存在时写入，用于结算幂等占位。
func (rd *RedisDao) SetNX(key, value string, timeout int32) (bool, error) {
	var to time.Duration
	if timeout > 0 {
		to = time.Duration(timeout) * time.Second
	} else {
		to = -1
	}
	return rd.cli.SetNX(context.Background(), key, value, to).Result()
}

func (rd *RedisDao) Del(key string) error {
	return rd.cli.Del(context.Background(), key).Err()
}

func (rd *RedisDao) Get(key string) (string, error) {
	return rd.cli.Get(context.Background(), key).Result()
}

func (rd *RedisDao) HSet(key, value, attrKey string) error {
	return rd.cli.HSet(context.Background(), key, attrKey, value).Err()
}

func (rd *RedisDao) HDel(key, attrKey string) error {
	return rd.cli.HDel(context.Background(), key, attrKey).Err()
}

func (rd *RedisDao) SetKeyTimeOut(key string, timeout int32) (bool, error) {
	if timeout < 0 {
		return rd.cli.Persist(context.Background(), key).Result()
	}
	return rd.cli.Expire(context.Background(), key, time.Duration(timeout)*time.Second).Result()
}

func (rd *RedisDao) Subscribe(channel string, registryInfo func() *event.EventMgr) {
	//watch
	sub := rd.cli.Subscribe(context.Background(), channel)
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
					zap.L().Error("收到无法解析的推送", zap.Any("msg", msg))
					return
				}
				event.OnEvent(pack)
			}()
		}
	}()
}

func LoadConfigs(client *RedisDao, prefix string) {
	//初始化加载所有配置
	iter := client.cli.Scan(context.Background(), 0, prefix, 0).Iterator()
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
	pipe := client.cli.Pipeline()
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
			pipe = client.cli.Pipeline()
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
	if CfgIns == nil {
		CfgIns = &Configs{
			Lock: &sync.RWMutex{},
			Pool: &PoolMgr{
				Default: make(map[string]*Pool),
				Agent:   make(map[string]*Pool),
			},
			Award: &AwardMgr{
				Data: make(map[string]*AwardConfig),
			},
			Coefficient: DefaultControlCoefficientConfig(),
			System:      &SystemConfig{},
			Currency: &CurrencyMgr{
				Data: make(map[string]decimal.Decimal),
			},
			AC: &AutoCtrlMgr{
				Ctrls: make([]*AutoCtrlItem, 0),
			},
		}
		//加载默认配置
		LoadConfigs(RedisIns(), esindex.ConfigPattern())
		//加载代理配置
		LoadConfigs(RedisIns(), esindex.AgentPattern())
	}
}

func LoadConfig() *SystemConfig {
	var sysConfig *SystemConfig = &SystemConfig{}
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
