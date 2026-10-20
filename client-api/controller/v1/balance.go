package v1

import (
	"fmt"
	"net/http"
	"strconv"

	"client-api/cache"
	"client-api/common"
	"client-api/dao"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type BalanceHandler struct {
	Guard *cache.Guard
}

type balanceCache struct {
	Currency     string `json:"currency"`
	CurrencyCent int64  `json:"currencyCent"`
}

func (h *BalanceHandler) GetBalance(c *gin.Context) {
	userId64, _ := strconv.ParseUint(c.Query("userId"), 10, 32)
	userId := uint32(userId64)
	if userId == 0 {
		common.Fail(c, http.StatusOK, common.CodeBadRequest, "参数错误")
		return
	}

	payload := &balanceCache{}
	key := cache.BuildKey("balance", fmt.Sprintf("%d", userId))
	err := h.Guard.Do(key, payload, func() (interface{}, error) {
		cent, err := dao.GetBalanceCent(userId)
		if err == redis.Nil {
			return nil, fmt.Errorf("player currency missing")
		}
		if err != nil {
			return nil, err
		}
		return &balanceCache{
			Currency:     decimal.NewFromInt(cent).Div(decimal.NewFromInt(100)).Truncate(2).StringFixed(2),
			CurrencyCent: cent,
		}, nil
	})
	zap.L().Info("GetBalance result", zap.Uint32("userId", userId), zap.Int64("currencyCent", payload.CurrencyCent), zap.Error(err))
	if err != nil {
		zap.L().Error("GetBalance failed", zap.Uint32("userId", userId), zap.Error(err))
		common.Fail(c, http.StatusOK, common.CodeSystemError, "余额查询失败")
		return
	}
	common.OK(c, gin.H{
		"userId":       userId,
		"currency":     payload.Currency,
		"currencyCent": payload.CurrencyCent,
	})
}
