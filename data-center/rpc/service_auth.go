package rpc

import (
	"context"
	"strconv"
	"strings"
	"time"

	"app/entity"

	jsoniter "github.com/json-iterator/go"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"micro_service/services"
)

const sessionAuthTTLSeconds int32 = 20 * 60

func normalizeSessionKey(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if strings.HasPrefix(token, "SESSION@") {
		return token
	}
	return "SESSION@" + token
}

func (d *DataCenterService) loadSession(token string) (*entity.Session, string, error) {
	key := normalizeSessionKey(token)
	if key == "" {
		return nil, "", nil
	}
	raw, err := d.rds.Get(key, 0)
	if err == redis.Nil || raw == "" {
		return nil, key, nil
	}
	if err != nil {
		return nil, key, err
	}
	session := &entity.Session{}
	if err := jsoniter.UnmarshalFromString(raw, session); err != nil {
		zap.L().Error("parse session failed", zap.Error(err))
		return nil, key, err
	}
	return session, key, nil
}

func (d *DataCenterService) saveSession(key string, session *entity.Session) error {
	raw, err := jsoniter.MarshalToString(session)
	if err != nil {
		return err
	}
	return d.rds.Set(key, raw, sessionAuthTTLSeconds)
}

// Authenticate 基于 open-api 写入的 SESSION@token 做正式鉴权。
func (d *DataCenterService) Authenticate(_ context.Context, req *services.AuthenticateReq) (resp *services.AuthenticateResp, err error) {
	resp = &services.AuthenticateResp{
		Code:    services.ErrorCode_OK,
		Success: false,
		Message: "auth failed",
	}
	if req == nil || strings.TrimSpace(req.Token) == "" {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Message = "token required"
		return resp, nil
	}

	session, key, err := d.loadSession(req.Token)
	if err != nil {
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		resp.Message = "session unavailable"
		return resp, nil
	}
	if session == nil || session.UserId <= 0 {
		resp.Code = services.ErrorCode_AUTH_TOKEN_INVALID
		resp.Message = "token invalid"
		return resp, nil
	}

	if req.EntryUserId > 0 && uint32(session.UserId) != req.EntryUserId {
		resp.Code = services.ErrorCode_AUTH_TOKEN_INVALID
		resp.Message = "entry user mismatch"
		return resp, nil
	}

	if req.GameId > 0 {
		game := d.db.GetGameByNumber(int64(req.GameId))
		if game == nil || game.State != 1 {
			resp.Code = services.ErrorCode_GAME_FROZEN
			resp.Message = "game unavailable"
			return resp, nil
		}
		if session.GameId > 0 && session.GameId != int64(req.GameId) && session.Symbol != "" && session.Symbol != game.ConfName {
			resp.Code = services.ErrorCode_AUTH_TOKEN_INVALID
			resp.Message = "game mismatch"
			return resp, nil
		}
		session.GameId = int64(req.GameId)
		session.Symbol = game.ConfName
	}

	session.AuthCount++
	session.LastAuthTime = time.Now().Unix()
	if err := d.saveSession(key, session); err != nil {
		zap.L().Error("save session after authenticate failed", zap.Error(err))
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		resp.Message = "session update failed"
		return resp, nil
	}

	resp.Success = true
	resp.UserId = uint32(session.UserId)
	resp.AttemptCount = session.AuthCount
	resp.IsReEnter = session.AuthCount > 1
	if resp.IsReEnter {
		resp.Message = "re-enter auth success"
	} else {
		resp.Message = "auth success"
	}
	return resp, nil
}

// GetLoginData 返回 Gateway 登录资料；余额仍以 lottery.GetBalance 为准。
func (d *DataCenterService) GetLoginData(ctx context.Context, req *services.GetLoginDataReq) (resp *services.GetLoginDataResp, err error) {
	resp = &services.GetLoginDataResp{Code: services.ErrorCode_OK}
	if req == nil || req.UserId == 0 {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		return resp, nil
	}

	p, err := d.rds.GetPlayer(req.UserId, 0)
	if err != nil {
		zap.L().Error("get player cache failed", zap.Uint32("userId", req.UserId), zap.Error(err))
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		return resp, nil
	}
	if p == nil {
		playerInfo, dbErr := d.db.GetPlayer(ctx, req.UserId)
		if dbErr != nil {
			if dbErr == gorm.ErrRecordNotFound {
				resp.Code = services.ErrorCode_PARAMS_INVALID
				return resp, nil
			}
			zap.L().Error("get player from db failed", zap.Uint32("userId", req.UserId), zap.Error(dbErr))
			resp.Code = services.ErrorCode_SYSTEM_ERROR
			return resp, nil
		}
		p = ConvertUserEntityToHumanPlayer(playerInfo)
		_ = d.rds.SetPlayer(p, false)
	}

	userIcon := int32(0)
	if p.Avatar != "" {
		if v, parseErr := strconv.ParseInt(p.Avatar, 10, 32); parseErr == nil {
			userIcon = int32(v)
		}
	}

	topAgentId := d.db.ResolveTopAgentId(int64(p.AgentId))
	userName := p.Nickname
	if userName == "" {
		userName = p.Account
	}

	resp.Profile = &services.LoginProfile{
		UserId:     p.Id,
		UserName:   userName,
		AgentId:    p.AgentId,
		TopAgentId: topAgentId,
		UserIcon:   userIcon,
		LoginType:  0,
		Gm:         0,
		Rate:       1,
		Currency: &services.LoginCurrency{
			CurrencyId:   0,
			Symbol:       p.CurrencyType,
			CurrencyRate: 1,
		},
	}
	return resp, nil
}

// ValidateToken 仅判断 token 与 userId 绑定是否有效；系统错误与业务无效分开。
func (d *DataCenterService) ValidateToken(_ context.Context, req *services.ValidateTokenReq) (resp *services.ValidateTokenResp, err error) {
	resp = &services.ValidateTokenResp{
		Code:  services.ErrorCode_OK,
		Valid: false,
	}
	if req == nil || strings.TrimSpace(req.Token) == "" || req.UserId == 0 {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		return resp, nil
	}

	session, _, err := d.loadSession(req.Token)
	if err != nil {
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		return resp, nil
	}
	if session == nil || session.UserId <= 0 {
		return resp, nil
	}
	resp.Valid = uint32(session.UserId) == req.UserId
	return resp, nil
}
