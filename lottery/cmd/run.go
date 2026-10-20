package cmd

import (
	"errors"
	"lottery/dao"
	"lottery/util"
	"os"
	"os/signal"
	"syscall"

	"lottery/rpc"

	"github.com/spf13/cobra"

	"go.uber.org/zap"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "启动开奖逻辑服务器",
	Long:  "启动开奖逻辑服务器",
	RunE:  run,
}

var (
	Host          []string
	User          string
	Pwd           string
	log           string
	RunConfigPath string
)

func init() {
	runCmd.Flags().StringVar(&RunConfigPath, "config", "./config.yaml", "指定配置文件，默认使用当前目录下的config.yaml")
	rootCmd.AddCommand(runCmd)
}

func run(cmd *cobra.Command, args []string) error {
	util.InitZapLogger(log, "lottery")
	rc := util.InitBaseConfig(RunConfigPath)
	if rc == nil {
		return errors.New("读取启动配置失败")
	}
	dao.NewRedisDao(rc.Redis.Host, rc.Redis.User, rc.Redis.Pwd)
	dao.ConfigsInit()
	dao.LoadConfig()
	err := dao.InitDB(rc)
	if err != nil {
		return err
	}
	es, eErr := dao.InitES(rc)
	if eErr != nil {
		zap.L().Error("创建es客户端失败", zap.Error(err))
		return nil
	}
	dao.InitGameManager()
	dao.InitAgentManager()
	//缓存初始化
	dao.CahceInit()
	//加载对局数据
	dao.LoadAllRoundData()
	//同步注单数据
	dao.SyncRoundData()
	handler := rpc.NewLotteryService(es)
	rpcServer := rpc.NewServer(rc.ServerIp, rc.ServerPort, handler)
	go rpcServer.Serve(dao.RedisIns(), dao.NewDBDao())
	defer rpcServer.Stop()
	zap.L().Info("服务器启动成功")
	exitC := make(chan os.Signal, 1)
	signal.Notify(exitC, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	sig := <-exitC
	zap.L().Info("收到退出信号，服务器即将退出", zap.Stringer("Signal", sig))
	return nil
}
