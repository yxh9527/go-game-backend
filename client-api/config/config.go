package config

import (
	"fmt"
	"strings"
)

type MysqlItem struct {
	Host     string `yaml:"host"`
	Port     int32  `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type RunConfig struct {
	Redis struct {
		Host []string `yaml:"host"`
		User string   `yaml:"user"`
		Pwd  string   `yaml:"pwd"`
	} `yaml:"redis"`
	Mysql map[string]MysqlItem `yaml:"mysql"`
	Elastic struct {
		Host     []string `yaml:"hosts"`
		UserName string   `yaml:"username"`
		Password string   `yaml:"password"`
	} `yaml:"elastic"`
	Cache struct {
		MemoryTTLSeconds int `yaml:"memory_ttl_seconds"`
	} `yaml:"cache"`
	RateLimit struct {
		QPS   int `yaml:"qps"`
		Burst int `yaml:"burst"`
	} `yaml:"rate_limit"`
	Games          map[string]string `yaml:"games"`
	ServerIp       string            `yaml:"server_ip"`
	ServerPort     int               `yaml:"server_port"`
	InternalAPIKey string            `yaml:"internal_api_key"`
	Log            string            `yaml:"log"`
}

func (c *RunConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if strings.TrimSpace(c.ServerIp) == "" {
		return fmt.Errorf("server_ip required")
	}
	if c.ServerPort <= 0 || c.ServerPort > 65535 {
		return fmt.Errorf("server_port invalid")
	}
	if len(c.Redis.Host) == 0 {
		return fmt.Errorf("redis.host required")
	}
	if c.Mysql == nil || c.Mysql["player"].Host == "" || c.Mysql["manager"].Host == "" {
		return fmt.Errorf("mysql.player/manager required")
	}
	if len(c.Elastic.Host) == 0 {
		return fmt.Errorf("elastic.hosts required")
	}
	if strings.TrimSpace(c.InternalAPIKey) == "" {
		return fmt.Errorf("internal_api_key required")
	}
	if c.Cache.MemoryTTLSeconds < 0 {
		return fmt.Errorf("cache.memory_ttl_seconds invalid")
	}
	if c.RateLimit.QPS < 0 || c.RateLimit.Burst < 0 {
		return fmt.Errorf("rate_limit invalid")
	}
	return nil
}
