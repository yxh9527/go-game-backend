package v1

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"client-api/cache"
	"client-api/common"
	"client-api/dao"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AuthHandler struct {
	Guard *cache.Guard
}

type authenticateReq struct {
	Token       string `json:"token"`
	GameId      uint32 `json:"gameId"`
	EntryUserId uint32 `json:"entryUserId"`
}

type validateReq struct {
	Token  string `json:"token"`
	UserId uint32 `json:"userId"`
}

type authResultCache struct {
	Success      bool   `json:"success"`
	UserId       uint32 `json:"userId"`
	GameId       uint32 `json:"gameId"`
	IsReEnter    bool   `json:"isReEnter"`
	AttemptCount int64  `json:"attemptCount"`
	Message      string `json:"message"`
	Unauthorized bool   `json:"unauthorized"`
}

func (h *AuthHandler) Authenticate(c *gin.Context) {
	var req authenticateReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Token) == "" {
		common.Fail(c, http.StatusOK, common.CodeBadRequest, "参数错误")
		return
	}
	payload := &authResultCache{}
	key := cache.BuildKey("authAuthenticate", req.Token, fmt.Sprintf("%d", req.GameId), fmt.Sprintf("%d", req.EntryUserId))
	err := h.Guard.Do(key, payload, func() (interface{}, error) {
		resp := dao.Authenticate(req.Token, req.GameId, req.EntryUserId)
		if resp.SystemError {
			return nil, fmt.Errorf("auth system error")
		}
		return &authResultCache{
			Success:      resp.Success,
			UserId:       resp.UserId,
			GameId:       resp.GameId,
			IsReEnter:    resp.IsReEnter,
			AttemptCount: resp.AttemptCount,
			Message:      resp.Message,
			Unauthorized: resp.Unauthorized || !resp.Success,
		}, nil
	})
	if err != nil {
		zap.L().Error("Authenticate failed", zap.Error(err))
		common.Fail(c, http.StatusOK, common.CodeSystemError, "鉴权服务不可用")
		return
	}
	if payload.Unauthorized || !payload.Success {
		common.Fail(c, http.StatusOK, common.CodeUnauthorized, firstNonEmpty(payload.Message, "未登录"))
		return
	}
	out := gin.H{
		"success":      payload.Success,
		"userId":       payload.UserId,
		"gameId":       payload.GameId,
		"isReEnter":    payload.IsReEnter,
		"attemptCount": payload.AttemptCount,
		"message":      payload.Message,
	}
	if node := dao.PickGateway(payload.UserId); node != nil {
		out["gateway"] = node
	}
	common.OK(c, out)
}

func (h *AuthHandler) ValidateToken(c *gin.Context) {
	var req validateReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Token) == "" || req.UserId == 0 {
		common.Fail(c, http.StatusOK, common.CodeBadRequest, "参数错误")
		return
	}
	type validateCache struct {
		Valid bool `json:"valid"`
	}
	payload := &validateCache{}
	key := cache.BuildKey("authValidate", req.Token, fmt.Sprintf("%d", req.UserId))
	err := h.Guard.Do(key, payload, func() (interface{}, error) {
		valid, sysErr := dao.ValidateToken(req.Token, req.UserId)
		if sysErr {
			return nil, fmt.Errorf("validate system error")
		}
		return &validateCache{Valid: valid}, nil
	})
	if err != nil {
		zap.L().Error("ValidateToken failed", zap.Error(err))
		common.Fail(c, http.StatusOK, common.CodeSystemError, "鉴权服务不可用")
		return
	}
	common.OK(c, gin.H{"valid": payload.Valid})
}

func (h *AuthHandler) GetLoginData(c *gin.Context) {
	userId64, _ := strconv.ParseUint(c.Query("userId"), 10, 32)
	userId := uint32(userId64)
	if userId == 0 {
		common.Fail(c, http.StatusOK, common.CodeBadRequest, "参数错误")
		return
	}
	type loginCache struct {
		UserId     uint32  `json:"userId"`
		UserName   string  `json:"userName"`
		AgentId    uint32  `json:"agentId"`
		TopAgentId uint32  `json:"topAgentId"`
		UserIcon   int32   `json:"userIcon"`
		LoginType  int32   `json:"loginType"`
		Gm         int32   `json:"gm"`
		Rate       float64 `json:"rate"`
		Symbol     string  `json:"symbol"`
		NotFound   bool    `json:"notFound"`
	}
	payload := &loginCache{}
	key := cache.BuildKey("authLoginData", fmt.Sprintf("%d", userId))
	err := h.Guard.Do(key, payload, func() (interface{}, error) {
		profile, ok, sysErr := dao.GetLoginProfile(userId)
		if sysErr {
			return nil, fmt.Errorf("login data system error")
		}
		if !ok || profile == nil {
			return &loginCache{NotFound: true}, nil
		}
		return &loginCache{
			UserId:     profile.UserId,
			UserName:   profile.UserName,
			AgentId:    profile.AgentId,
			TopAgentId: profile.TopAgentId,
			UserIcon:   profile.UserIcon,
			LoginType:  profile.LoginType,
			Gm:         profile.Gm,
			Rate:       profile.Rate,
			Symbol:     profile.Symbol,
		}, nil
	})
	if err != nil {
		zap.L().Error("GetLoginData failed", zap.Error(err))
		common.Fail(c, http.StatusOK, common.CodeSystemError, "资料服务不可用")
		return
	}
	if payload.NotFound {
		common.Fail(c, http.StatusOK, common.CodeBadRequest, "用户不存在")
		return
	}
	out := gin.H{
		"userId":     payload.UserId,
		"userName":   payload.UserName,
		"agentId":    payload.AgentId,
		"topAgentId": payload.TopAgentId,
		"userIcon":   payload.UserIcon,
		"loginType":  payload.LoginType,
		"gm":         payload.Gm,
		"rate":       payload.Rate,
		"currency": gin.H{
			"currencyId":   0,
			"symbol":       payload.Symbol,
			"currencyRate": 1,
		},
	}
	if node := dao.PickGateway(payload.UserId); node != nil {
		out["gateway"] = node
	}
	common.OK(c, out)
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
