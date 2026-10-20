package main

import (
	"app/config"
	"fmt"
	. "open-api/common"
	"open-api/controller"
	"open-api/dao"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

var (
	rootCmd = &cobra.Command{
		Use:  "open-api",
		Long: "open-api",
	}
	runCmd = &cobra.Command{
		Use:   "run",
		Short: "鍚姩open-api",
		Long:  "鍚姩open-api",
		Run:   run,
	}
)

var (
	RunConfigPath string
)

func init() {
	runCmd.Flags().StringVar(&RunConfigPath, "config", "./config.yaml", "鎸囧畾閰嶇疆鏂囦欢锛岄粯璁や娇鐢ㄥ綋鍓嶇洰褰曚笅鐨刢onfig.yaml")
	rootCmd.AddCommand(runCmd)
}

// 鍒濆鍖栧熀纭€閰嶇疆
func InitBaseConfig() *config.RunConfig {
	yamlFile, err := os.ReadFile(RunConfigPath)
	if err != nil {
		zap.L().Fatal("璇诲彇鍩虹閰嶇疆澶辫触", zap.Any("error", err))
	}
	c := &config.RunConfig{}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		zap.L().Fatal("瑙ｆ瀽鍩虹閰嶇疆寮傚父", zap.Any("error", err))
	}
	return c
}

func run(cmd *cobra.Command, args []string) {
	rc := InitBaseConfig()
	dao.InitRedis(rc)
	dao.ConfigsInit()
	dao.LoadConfig()
	if err := dao.InitES(rc); err != nil {
		panic(err)
	}
	if err := dao.InitDB(rc); err != nil {
		panic(err)
	}
	//鍒濆鍖栦唬鐞嗙紦瀛?
	dao.InitAgentMgr()
	//鍔ㄦ€佸煙鍚嶅姞杞?
	InitApiConfigMgr()
	//鍔犺浇娓告垙閰嶇疆
	dao.InitGameCacheMgr()
	zap.L().Info("Server ", zap.String("name", "open-api"))
	zap.L().Info("Server start ok")
	r := controller.NewRouter()
	if err := r.Run(fmt.Sprintf(":%d", rc.ServerPort)); err != nil {
		zap.L().Fatal("HTTP Server鍚姩澶辫触", zap.Error(err))
	}
}

func main() {
	// 鍒濆鍖栨棩蹇楀簱
	InitZapLogger()

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("%v", err)
	}
}
