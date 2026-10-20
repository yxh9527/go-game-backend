package main

import (
	"app/config"
	"game-data-summary/consumer"
	"game-data-summary/dao"
	"game-data-summary/global"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

var (
	rootCmd = &cobra.Command{
		Use:  "game-data-summary",
		Long: "游戏数据统计",
	}
	runCmd = &cobra.Command{
		Use:   "run",
		Short: "游戏数据统计",
		Long:  "游戏数据统计",
	}

	EtcdClient *clientv3.Client
	EtcdHosts  []string
	EtcdUser   string
	EtcdPwd    string
)

// 初始化基础配置
func InitBaseConfig() *config.RunConfig {
	yamlFile, err := os.ReadFile(RunConfigPath)
	if err != nil {
		zap.L().Error("读取基础配置失败", zap.Any("error", err))
		return nil
	}
	config := &config.RunConfig{}
	err = yaml.Unmarshal(yamlFile, config)
	if err != nil {
		zap.L().Error("解析基础配置异常", zap.Any("error", err))
		return nil
	}
	return config
}

func run(_ *cobra.Command, _ []string) {
	rc := InitBaseConfig()
	dao.NewRedisDao(rc.Redis.Host, rc.Redis.User, rc.Redis.Pwd)
	dao.ConfigsInit()
	dao.LoadConfig()
	esClient := dao.NewEsInfo(rc)
	if esClient == nil {
		zap.L().Error("创建es链接失败")
		return
	}
	dao.NewDbInfo(rc)
	dao.InitGamesManager()
	consumer.UnmarshalFile()
	consumer.Games = dao.GamesMgr.List()
	go func() {
		defer func() {
			if err := recover(); err != nil {
				zap.L().Error("panic", zap.Any("err", err))
			}
		}()
		ticker := time.NewTicker(10 * time.Second)
		for range ticker.C {
			if c := InitBaseConfig(); c != nil {
				consumer.Games = dao.GamesMgr.List()
			}
			consumer.TaskByHour(dao.DB, esClient)
			consumer.TaskByDay(dao.DB, esClient)
			consumer.TaskByWeek(dao.DB, esClient)
		}
	}()
	zap.L().Debug("===>4")
	exitC := make(chan os.Signal, 1)
	signal.Notify(exitC, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	sig := <-exitC
	zap.L().Info("收到退出信号，服务器即将退出", zap.Stringer("Signal", sig))
}

var RunConfigPath = ""

func init() {
	runCmd.Flags().StringVar(&RunConfigPath, "config", "./config.yaml", "指定配置文件，默认使用当前目录下的config.yaml")
	rootCmd.AddCommand(runCmd)
}

func main() {
	_, _ = global.InitZapLogger()
	runCmd.Run = func(cmd *cobra.Command, args []string) {
		run(cmd, args)
	}
	if err := rootCmd.Execute(); err != nil {
		zap.L().Error("启动失败", zap.Error(err))
	}
}
