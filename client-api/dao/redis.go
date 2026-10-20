package dao

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"app/entity"
	"client-api/config"

	jsoniter "github.com/json-iterator/go"
	"github.com/redis/go-redis/v9"
)

const sessionAuthTTLSeconds = 20 * 60

type RedisDao struct {
	cli redis.UniversalClient
}

var redisIns *RedisDao

type PlayerCache struct {
	Id           uint32
	Nickname     string
	Account      string
	Avatar       string
	AgentId      uint32
	CurrencyType string
	CurrencyCent int64
}

func InitRedis(c *config.RunConfig) error {
	if c == nil || len(c.Redis.Host) == 0 {
		return errors.New("redis config missing")
	}
	cli := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    c.Redis.Host,
		Password: c.Redis.Pwd,
		DB:       0,
	})
	if err := cli.Ping(context.Background()).Err(); err != nil {
		return err
	}
	redisIns = &RedisDao{cli: cli}
	return nil
}

func Redis() *RedisDao {
	return redisIns
}

func (r *RedisDao) Ping() error {
	if r == nil || r.cli == nil {
		return errors.New("redis not initialized")
	}
	return r.cli.Ping(context.Background()).Err()
}

func (r *RedisDao) Set(key, value string, timeout int32) error {
	if r == nil || r.cli == nil {
		return errors.New("redis not initialized")
	}
	var to time.Duration
	if timeout > 0 {
		to = time.Duration(timeout) * time.Second
	}
	return r.cli.Set(context.Background(), key, value, to).Err()
}

func (r *RedisDao) Del(key string) error {
	if r == nil || r.cli == nil {
		return errors.New("redis not initialized")
	}
	return r.cli.Del(context.Background(), key).Err()
}

func (r *RedisDao) SetKeyTimeOut(key string, timeout int32) (bool, error) {
	if r == nil || r.cli == nil {
		return false, errors.New("redis not initialized")
	}
	if timeout < 0 {
		return r.cli.Persist(context.Background(), key).Result()
	}
	return r.cli.Expire(context.Background(), key, time.Duration(timeout)*time.Second).Result()
}

func normalizeSessionKey(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if strings.HasPrefix(token, "SESSION@") {
		return token
	}
	return "SESSION@" + token
}

func (r *RedisDao) LoadSession(token string) (*entity.Session, string, error) {
	key := normalizeSessionKey(token)
	if key == "" {
		return nil, "", nil
	}
	raw, err := r.cli.Get(context.Background(), key).Result()
	if err == redis.Nil || raw == "" {
		return nil, key, nil
	}
	if err != nil {
		return nil, key, err
	}
	session := &entity.Session{}
	if err := jsoniter.UnmarshalFromString(raw, session); err != nil {
		return nil, key, err
	}
	return session, key, nil
}

func (r *RedisDao) SaveSession(key string, session *entity.Session) error {
	raw, err := jsoniter.MarshalToString(session)
	if err != nil {
		return err
	}
	return r.cli.Set(context.Background(), key, raw, time.Duration(sessionAuthTTLSeconds)*time.Second).Err()
}

func (r *RedisDao) GetPlayer(userId uint32) (*PlayerCache, error) {
	if userId == 0 {
		return nil, nil
	}
	key := "player_" + strconv.FormatUint(uint64(userId), 10)
	res, err := r.cli.HGetAll(context.Background(), key).Result()
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}
	p := &PlayerCache{Id: userId}
	if v := res["nickname"]; v != "" {
		p.Nickname = v
	}
	if v := res["account"]; v != "" {
		p.Account = v
	}
	if v := res["avatar"]; v != "" {
		p.Avatar = v
	}
	if v := res["currency_type"]; v != "" {
		p.CurrencyType = v
	}
	if v := res["agent_id"]; v != "" {
		id, e := strconv.ParseUint(v, 10, 32)
		if e == nil {
			p.AgentId = uint32(id)
		}
	}
	if v := res["currency"]; v != "" {
		cent, e := strconv.ParseInt(v, 10, 64)
		if e == nil {
			p.CurrencyCent = cent
		}
	}
	_ = r.cli.Expire(context.Background(), key, 20*time.Minute).Err()
	return p, nil
}

func (r *RedisDao) GetUserTotalEffBet(userId uint32) float64 {
	if r == nil || r.cli == nil || userId == 0 {
		return 0
	}
	score, err := r.cli.ZScore(context.Background(), "userTotalEffBet", strconv.FormatUint(uint64(userId), 10)).Result()
	if err != nil {
		return 0
	}
	return score
}

func (r *RedisDao) GetPlayerCurrencyCent(userId uint32) (int64, error) {
	if userId == 0 {
		return 0, fmt.Errorf("userId required")
	}
	key := "player_" + strconv.FormatUint(uint64(userId), 10)
	pipe := r.cli.Pipeline()
	pipe.HExists(context.Background(), key, "id")
	pipe.Expire(context.Background(), key, 20*time.Minute)
	pipe.HGet(context.Background(), key, "currency")
	result, err := pipe.Exec(context.Background())
	if err != nil && err != redis.Nil {
		return 0, err
	}
	exist, e1 := result[0].(*redis.BoolCmd).Result()
	if e1 != nil && e1 != redis.Nil {
		return 0, e1
	}
	if !exist {
		return 0, redis.Nil
	}
	raw, e2 := result[2].(*redis.StringCmd).Result()
	if e2 != nil {
		return 0, e2
	}
	return strconv.ParseInt(raw, 10, 64)
}
