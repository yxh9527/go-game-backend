package dao

import (
	"app/tables/manager"
	"context"
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

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
