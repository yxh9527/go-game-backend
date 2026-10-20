package cmd

import (
	"app/config"
	"data-center/dao"
	"data-center/rpc"
	"data-center/util"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"go.uber.org/zap"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "启动数据中心服务器",
	Long:  "启动数据中心服务器",
	RunE:  run,
}

var (
	Host          []string
	User          string
	Pwd           string
	log           string
	Ip            string
	RunConfigPath string
)

func init() {
	runCmd.Flags().StringVar(&RunConfigPath, "config", "./config.yaml", "指定配置文件，默认使用当前目录下的config.yaml")
	rootCmd.AddCommand(runCmd)
}

// 初始化基础配置
func InitBaseConfig() *config.RunConfig {
	yamlFile, err := os.ReadFile(RunConfigPath)
	if err != nil {
		zap.L().Fatal("读取基础配置失败", zap.Any("error", err))
	}
	config := &config.RunConfig{}
	err = yaml.Unmarshal(yamlFile, config)
	if err != nil {
		zap.L().Fatal("解析基础配置异常", zap.Any("error", err))
	}
	return config
}

func run(cmd *cobra.Command, args []string) error {
	util.InitZapLogger(log, "data-center", Ip)
	rc := InitBaseConfig()
	if rc == nil {
		return errors.New("读取启动配置失败")
	}
	dao.NewRedisDao(rc.Redis.Host, rc.Redis.User, rc.Redis.Pwd)
	dao.ConfigsInit()
	dao.LoadConfig()
	db, mdb, err := util.InitDB(rc)
	if err != nil {
		return err
	}
	es, eErr := util.InitES(rc)
	if eErr != nil {
		zap.L().Error("创建es客户端失败", zap.Error(err))
		return nil
	}
	syncer := dao.NewDataSyncer(db)
	syncer.Start()
	defer syncer.Stop()
	handler := rpc.NewDataCenterService(db, mdb, es)
	rpcServer := rpc.NewServer(rc.ServerIp, rc.ServerPort, handler)
	go rpcServer.Serve(dao.RedisIns(), dao.NewDBDao(db, mdb))
	defer rpcServer.Stop()
	zap.L().Info("服务器启动成功")
	exitC := make(chan os.Signal, 1)
	signal.Notify(exitC, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	sig := <-exitC

	zap.L().Info("收到退出信号，服务器即将退出", zap.Stringer("Signal", sig))
	return nil
}
