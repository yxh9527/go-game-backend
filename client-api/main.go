package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"app/esindex"
	"client-api/cache"
	"client-api/common"
	"client-api/config"
	"client-api/controller"
	"client-api/dao"
	"client-api/middleware"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

var (
	rootCmd = &cobra.Command{
		Use:  "client-api",
		Long: "client-api: 面向客户端的鉴权与注单查询服务",
	}
	runCmd = &cobra.Command{
		Use:   "run",
		Short: "启动 client-api",
		Run:   run,
	}
	RunConfigPath string
)

func init() {
	runCmd.Flags().StringVar(&RunConfigPath, "config", "./config.yaml", "指定配置文件")
	rootCmd.AddCommand(runCmd)
}

func loadConfig() *config.RunConfig {
	raw, err := os.ReadFile(RunConfigPath)
	if err != nil {
		zap.L().Fatal("读取配置失败", zap.Error(err))
	}
	cfg := &config.RunConfig{}
	if err := yaml.Unmarshal(raw, cfg); err != nil {
		zap.L().Fatal("解析配置失败", zap.Error(err))
	}
	if err := cfg.Validate(); err != nil {
		zap.L().Fatal("配置校验失败", zap.Error(err))
	}
	return cfg
}

func run(_ *cobra.Command, _ []string) {
	cfg := loadConfig()
	if err := dao.InitRedis(cfg); err != nil {
		zap.L().Fatal("初始化 Redis 失败", zap.Error(err))
	}
	if err := dao.Redis().Ping(); err != nil {
		zap.L().Fatal("Redis 自检失败", zap.Error(err))
	}
	if err := dao.InitDB(cfg); err != nil {
		zap.L().Fatal("初始化 MySQL 失败", zap.Error(err))
	}
	if err := dao.DB().Ping(); err != nil {
		zap.L().Fatal("MySQL 自检失败", zap.Error(err))
	}
	dao.InitApiConfigMgr()
	if err := dao.InitES(cfg); err != nil {
		zap.L().Fatal("初始化 ES 失败", zap.Error(err))
	}
	if err := dao.ES().Ping(); err != nil {
		zap.L().Fatal("ES 自检失败", zap.Error(err))
	}

	guard := cache.NewGuard(cache.Options{
		MemoryTTLSeconds: cfg.Cache.MemoryTTLSeconds,
	})
	limiter := middleware.NewRateLimiter(cfg.RateLimit.QPS, cfg.RateLimit.Burst)

	// 配置 games 优先；否则从 MySQL gp_game 加载 number->confName
	gameMapRaw := cfg.Games
	if len(gameMapRaw) == 0 {
		gameMapRaw = dao.DB().LoadGameMap()
	}
	gameMap := dao.NewGameMap(gameMapRaw)

	zap.L().Info("client-api start",
		zap.String("ip", cfg.ServerIp),
		zap.Int("port", cfg.ServerPort),
		zap.Strings("redis", cfg.Redis.Host),
		zap.Strings("elastic", cfg.Elastic.Host),
		zap.Int("memoryTTL", cfg.Cache.MemoryTTLSeconds),
		zap.Int("rateQPS", cfg.RateLimit.QPS),
		zap.Int("gameMapSize", len(gameMapRaw)))

	ns, err := dao.NewDefNamingService(dao.Redis(), esindex.ServiceName("clientapi"), cfg.ServerIp, int32(cfg.ServerPort))
	if err != nil {
		zap.L().Fatal("注册 client-api 失败", zap.Error(err))
	}

	r := controller.NewRouter(controller.RouterDeps{
		Guard:          guard,
		Limiter:        limiter,
		GameMap:        gameMap,
		InternalAPIKey: cfg.InternalAPIKey,
	})
	go func() {
		if runErr := r.Run(fmt.Sprintf(":%d", cfg.ServerPort)); runErr != nil {
			ns.ClearRegistryInfo()
			zap.L().Fatal("HTTP 启动失败", zap.Error(runErr))
		}
	}()

	exitC := make(chan os.Signal, 1)
	signal.Notify(exitC, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	sig := <-exitC
	zap.L().Info("收到退出信号", zap.String("signal", sig.String()))
	ns.ClearRegistryInfo()
}

func main() {
	common.InitZapLogger()
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("%v\n", err)
	}
}
