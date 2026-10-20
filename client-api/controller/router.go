package controller

import (
	"context"
	"net/http"
	"time"

	"client-api/cache"
	v1 "client-api/controller/v1"
	"client-api/dao"
	"client-api/middleware"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type RouterDeps struct {
	Guard          *cache.Guard
	Limiter        *middleware.RateLimiter
	GameMap        *dao.GameMap
	InternalAPIKey string
}

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Cache-Control", "no-store")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func RequestTimeout(d time.Duration) gin.HandlerFunc {
	if d <= 0 {
		d = 8 * time.Second
	}
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func NewRouter(deps RouterDeps) *gin.Engine {
	engine := gin.New()
	engine.Use(ginzap.RecoveryWithZap(zap.L(), true))
	engine.Use(Cors())
	engine.Use(RequestTimeout(8 * time.Second))
	engine.MaxMultipartMemory = 1 << 20
	if deps.Limiter != nil {
		engine.Use(deps.Limiter.Middleware())
	}

	auth := &v1.AuthHandler{Guard: deps.Guard}
	record := &v1.RecordHandler{Guard: deps.Guard, GameMap: deps.GameMap}
	balance := &v1.BalanceHandler{Guard: deps.Guard}

	api := engine.Group("/api/client/v1")
	{
		api.POST("/auth/authenticate", auth.Authenticate)
		api.POST("/auth/validate", auth.ValidateToken)
		api.GET("/auth/login-data", auth.GetLoginData)
		api.GET("/balance", balance.GetBalance)

		api.POST("/record/authenticate", record.AuthenticateRecordPage)
		api.GET("/record/list", record.ListRecords)
		api.GET("/record/detail", record.GetRecordDetail)

		internal := api.Group("/internal", middleware.InternalAuth(deps.InternalAPIKey))
		{
			internal.GET("/record/list", record.InternalListRecords)
			internal.GET("/record/detail", record.InternalGetRecordDetail)
			internal.GET("/statement/list", record.InternalListStatements)
			internal.GET("/balance", balance.GetBalance)
		}
	}

	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return engine
}
