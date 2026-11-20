package dao

import "github.com/redis/go-redis/v9"

func NewRedisDaoWithClient(cli redis.UniversalClient) *RedisDao { return &RedisDao{cli: cli} }
func SetRedisDaoForTest(rd *RedisDao)                           { redisDao = rd }
