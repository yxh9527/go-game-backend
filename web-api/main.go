package main

import (
	"app/config"
	"app/tables"
	"fmt"
	"os"
	"web-api/common"
	. "web-api/common"
	"web-api/controller"
	"web-api/crontab"
	"web-api/dao"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var (
	rootCmd = &cobra.Command{
		Use:  "web-api",
		Long: "web-api",
	}
	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Launch the server",
		Run:   run,
	}
)

func init() {
	runCmd.Flags().String("config", "./config.yaml", "Path to config file, defaults to ./config.yaml")
	rootCmd.AddCommand(runCmd)
}

func run(cmd *cobra.Command, args []string) {
	path, err := cmd.Flags().GetString("config")
	if err != nil {
		return
	}

	yamlFile, err := os.ReadFile(path)
	if err != nil {
		zap.L().Fatal("failed to read config file", zap.Any("error", err))
	}

	rc := &config.RunConfig{}
	if err = yaml.Unmarshal(yamlFile, rc); err != nil {
		zap.L().Fatal("failed to parse config file", zap.Any("error", err))
	}

	// Init Redis.
	dao.NewRedisDao(rc.Redis.Host, rc.Redis.User, rc.Redis.Pwd)

	// Init MySQL.
	if err := dao.InitDB(rc); err != nil {
		panic(err)
	}

	// Init database tables and seed data.
	tables.InitMysqlDb(dao.Mysql().Manager, dao.Mysql().Player)

	// Load pool_config table data into Redis on startup.
	if err := dao.SyncPoolConfigsToRedis(); err != nil {
		panic(err)
	}

	// Init in-memory configs from Redis.
	dao.ConfigsInit()

	// Init Elasticsearch.
	if err := dao.InitES(rc); err != nil {
		panic(err)
	}

	// Init permission policy and scheduled jobs.
	common.NewUrlPloy()
	crontab.NewCrontab()

	r := controller.NewRouter()
	zap.L().Info("server started")
	if err := r.Run(fmt.Sprintf(":%d", rc.ServerPort)); err != nil {
		zap.L().Fatal("failed to start HTTP server", zap.Error(err))
	}
}

func main() {
	// Init logger.
	InitZapLogger()
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("%v", err)
	}
}
