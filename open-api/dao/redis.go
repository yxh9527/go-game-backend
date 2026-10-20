package dao

import (
	"app/cfgparse"
	"app/config"
	"app/entity"
	"app/esindex"
	"app/tables/player"
	"context"
	"fmt"
	"micro_service/services"
	"open-api/event"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type RedisDao struct {
	redis redis.UniversalClient
}

var redisDao *RedisDao = nil

func InitRedis(sc *config.RunConfig) error {
	c := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    sc.Redis.Host,
		Password: sc.Redis.Pwd,
		DB:       0,
	})
	redisDao = &RedisDao{redis: c}
	redisDao.Subscribe("message", func() *event.EventMgr {
		e := &event.EventMgr{
			Events: make(map[string]*event.Event),
		}
		e.Register("config", reflect.TypeOf(entity.ConfigMsg{}), ConfigHandler)
		e.Register("gameStatuChange", reflect.TypeOf(entity.GameStatuChange{}), FreezeGame)
		return e
	})
	return nil
}

func LoadConfigs(client *RedisDao, prefix string) {
	//初始化加载所有配置
	iter := client.redis.Scan(context.Background(), 0, prefix, 0).Iterator()
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
	pipe := client.redis.Pipeline()
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
			pipe = client.redis.Pipeline()
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
			Gateway: &config.GatewaysMgr{},
			AC:      &config.AutoCtrlMgr{},
		}
		//加载默认配置
		LoadConfigs(Redis(), esindex.ConfigPattern())
		//加载代理配置
		LoadConfigs(Redis(), esindex.AgentPattern())
	}
}

// 配置同步
func ConfigHandler(data interface{}) {
	msg := data.(*entity.ConfigMsg)
	ParseConfig(msg.Key, msg.Data)
}

// 配置同步
func FreezeGame(data interface{}) {
	msg := data.(*entity.GameStatuChange)
	GameCacheIns.lock.Lock()
	defer GameCacheIns.lock.Unlock()

	if g := GameCacheIns.Games[msg.Symbol]; g != nil {
		g.State = int16(msg.Status)
	}
}

func Redis() *RedisDao {
	return redisDao
}

// 解析配置
func ParseConfig(key string, value string) {
	cfgparse.Parse(key, value)
}

func (rd *RedisDao) Subscribe(channel string, registryInfo func() *event.EventMgr) {
	//watch
	sub := rd.redis.Subscribe(context.Background(), channel)
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

// 获取缓存中玩家积分
func (r *RedisDao) GetGameCurrency(id int64) (decimal.Decimal, bool) {
	if result, err := r.redis.HGet(context.Background(), fmt.Sprintf("player_%d", id), "currency").Result(); err == nil {
		if currency, e := decimal.NewFromString(result); e != nil {
			return decimal.Zero, false
		} else {
			return currency, true
		}
	} else {
		return decimal.Zero, false
	}
}

func (rd *RedisDao) Get(key string) (string, error) {
	return rd.redis.Get(context.Background(), key).Result()
}

func (r *RedisDao) InGame(uid int64) bool {
	if _, err := r.redis.Get(context.Background(), fmt.Sprintf("user_lock_%d", uid)).Result(); err == redis.Nil {
		return false
	}
	return true
}

func (r *RedisDao) Exists(key string) bool {
	if result, err := r.redis.Exists(context.Background(), key).Result(); err == nil {
		if result == 1 {
			return true
		}
	}
	return false
}

func (r *RedisDao) TTL(key string) int64 {
	if result, err := r.redis.TTL(context.Background(), key).Result(); err == nil {
		return int64(result)
	}
	return 0
}

func (r *RedisDao) HSet(key, attr, value string) bool {
	if _, err := r.redis.HSet(context.Background(), key, attr, value).Result(); err == nil {
		return true
	}
	return false
}

func (r *RedisDao) Set(key, value string, timeout int32) error {
	var to time.Duration
	if timeout > 0 {
		to = time.Duration(timeout) * time.Second
	} else {
		to = -1
	}

	return r.redis.Set(context.Background(), key, value, to).Err()
}

// 批量删除指定前缀
func (r *RedisDao) BulkDel(prefix string) error {
	iter := r.redis.Scan(context.Background(), 0, prefix, 0).Iterator()
	keys := make([]string, 0, 256)
	for iter.Next(context.Background()) {
		k := iter.Val()
		keys = append(keys, k)
	}
	if err := iter.Err(); err != nil {
		return err
	}
	return r.redis.Del(context.Background(), keys...).Err()
}

func (r *RedisDao) Del(key string) bool {
	if _, err := r.redis.Del(context.Background(), key).Result(); err == nil {
		return true
	}
	return false
}

// 获取缓存中玩家积分 open-api 锁不能重试
func (r *RedisDao) UserLock(id, timeOut int64) string {
	uid, _ := uuid.NewV4()
	success, _ := r.redis.SetNX(context.Background(), fmt.Sprintf("user_lock_%d", id), uid.String(), 120*time.Second).Result()
	if success {
		return uid.String()
	}
	return ""
}

// 获取缓存中玩家积分
func (r *RedisDao) UserUnlock(userId int64, token string) bool {
	script := redis.NewScript(`
	if redis.call('get', KEYS[1]) == ARGV[1] then
		return redis.call('del', KEYS[1])
	else
		return 0
	end`)
	n, err := script.Run(context.Background(), r.redis, []string{fmt.Sprintf("user_lock_%d", userId)}, []string{token}).Int()
	if err != nil {
		zap.L().Debug("用户锁 解锁失败", zap.Any("userId", userId), zap.Any("n", n), zap.Any("token", token), zap.Any("err", err))
		return false
	}
	return true
}

func ConvertUserEntityToHumanPlayer(p *player.Player) *services.HumanPlayer {
	return &services.HumanPlayer{
		Id:             uint32(p.UserId),
		Nickname:       p.NickName,
		GameCurrency:   p.Score.StringFixed(2),
		Avatar:         p.Pic,
		Gender:         uint32(p.Sex),
		Exp:            p.Exp,
		AgentId:        uint32(p.ProxyId),
		LoginIP:        p.LoginIp,
		LoginTimeStamp: p.LoginTime,
		CurrencyLimit:  p.MoneyLimit.StringFixed(2),
		WebSiteId:      uint32(p.WebsiteId),
		Account:        p.Account,
		CurrencyType:   p.CurrencyType,
		AllTimes:       p.AllTimes,
		IsTourist:      p.IsTourist,
		ExData:         p.ExData,
	}
}

func (r *RedisDao) SetPlayer(p *services.HumanPlayer) error {
	pID := "player_" + strconv.FormatUint(uint64(p.Id), 10)
	pipe := r.redis.Pipeline()
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

func (r *RedisDao) UpdatePlayerCurrency(playerId uint32, currencyDelta int64) (newCurrency int64, err error) {
	pID := "player_" + strconv.FormatUint(uint64(playerId), 10)
	pipe := r.redis.Pipeline()
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
	pipe = r.redis.Pipeline()
	pipe.HIncrBy(context.Background(), pID, "currency", currencyDelta)
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

// 加载gateway配置
func (r *RedisDao) LoadAllGateways() []string {
	gs, err := r.redis.Get(context.Background(), esindex.ConfigKey("gateways")).Result()
	if err != nil {
		zap.L().Error("获取网关信息失败", zap.Any("err", err))
		return []string{}
	} else {
		return strings.Split(gs, ",")
	}
}

func LoadConfig() *config.SystemConfig {
	var sysConfig *config.SystemConfig = &config.SystemConfig{}
	value, err := Redis().Get(esindex.ConfigKey("system"))
	if err != nil {
		zap.L().Fatal("获取系统配置失败", zap.Any("err", err))
	}
	err = jsoniter.UnmarshalFromString(value, sysConfig)
	if err != nil {
		zap.L().Fatal("解析系统配置失败", zap.Any("err", err))
	}
	return sysConfig
}
