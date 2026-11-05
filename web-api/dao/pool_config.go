package dao

import (
	"app/entity/view"
	"app/tables/manager"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const resetPoolConfigKey = "reset_pool_config"

func InitPoolResetConfig() error {
	ctx := context.Background()
	_, err := RedisIns().Client.Get(ctx, resetPoolConfigKey).Result()
	if err == nil {
		return nil
	}
	if err != redis.Nil {
		return fmt.Errorf("read %s: %w", resetPoolConfigKey, err)
	}

	config := view.ResetPoolConfig{
		Interval:  7,
		ResetTime: time.Now().Unix() + 7*24*60*60,
	}
	value, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("encode %s: %w", resetPoolConfigKey, err)
	}
	if err := RedisIns().Client.SetNX(ctx, resetPoolConfigKey, value, 0).Err(); err != nil {
		return fmt.Errorf("initialize %s: %w", resetPoolConfigKey, err)
	}
	zap.L().Info("initialized pool reset config", zap.Int("interval_days", config.Interval))
	return nil
}

func SyncPoolConfigsToRedis() error {
	var poolConfigs []manager.PoolConfig
	if err := Mysql().Manager.Find(&poolConfigs).Error; err != nil {
		return err
	}

	pipe := RedisIns().Client.Pipeline()
	for _, item := range poolConfigs {
		if item.Key == "" {
			continue
		}
		pipe.Set(context.Background(), item.Key, item.Value, 0)
	}

	if _, err := pipe.Exec(context.Background()); err != nil {
		return err
	}

	zap.L().Info("pool_config loaded to redis", zap.Int("count", len(poolConfigs)))
	return nil
}

func UpsertPoolConfig(key, value string) error {
	var row manager.PoolConfig
	db := Mysql().Manager
	err := db.Where("`key` = ?", key).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(&manager.PoolConfig{Key: key, Value: value}).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&row).Update("value", value).Error
}
