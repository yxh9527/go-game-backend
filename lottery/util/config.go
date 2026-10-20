package util

import (
	"app/cfgparse"
	"app/config"
	"os"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

// ParseConfig 解析 Redis 配置；key 第一段必须等于 esindex.Prefix。
func ParseConfig(key string, value string) {
	cfgparse.Parse(key, value)
}

// InitBaseConfig 初始化基础配置。
func InitBaseConfig(path string) *config.RunConfig {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		zap.L().Fatal("读取基础配置失败", zap.Any("error", err))
	}
	cfg := &config.RunConfig{}
	err = yaml.Unmarshal(yamlFile, cfg)
	if err != nil {
		zap.L().Fatal("解析基础配置异常", zap.Any("error", err))
	}
	zap.L().Debug("=====>", zap.Any("config", cfg))
	return cfg
}
