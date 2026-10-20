package rpc

import (
	"context"
	"strings"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"micro_service/services"
)

// GetBalance 返回权威余额；不改动下注/水池/注单链路。
func (d *LotteryService) GetBalance(_ context.Context, req *services.GetBalanceReq) (resp *services.GetBalanceResp, err error) {
	resp = &services.GetBalanceResp{Code: services.ErrorCode_OK}
	if req == nil || req.UserId == 0 {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		return resp, nil
	}

	cent, code := d.getPlayerCurrency(req.UserId)
	if code != services.ErrorCode_OK {
		zap.L().Error("GetBalance failed", zap.Uint32("userId", req.UserId), zap.Any("code", code))
		resp.Code = code
		return resp, nil
	}
	resp.CurrencyCent = cent
	resp.Currency = decimal.NewFromInt(cent).Div(decimal.NewFromInt(100)).Truncate(2).StringFixed(2)
	zap.L().Info("GetBalance result", zap.Uint32("userId", req.UserId), zap.Int64("currencyCent", cent), zap.String("currency", resp.Currency))
	return resp, nil
}

// SaveGameStorage 长期状态落 MySQL，不写入注单缓存。
func (d *LotteryService) SaveGameStorage(_ context.Context, req *services.SaveGameStorageReq) (resp *services.SaveGameStorageResp, err error) {
	resp = &services.SaveGameStorageResp{Code: services.ErrorCode_OK}
	if req == nil ||
		req.UserId == 0 ||
		req.GameId == 0 ||
		strings.TrimSpace(req.CurrencySymbol) == "" ||
		strings.TrimSpace(req.StorageKey) == "" {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		return resp, nil
	}
	if err := d.db.SaveGameStorage(req.UserId, req.GameId, req.CurrencySymbol, req.StorageKey, req.Payload, req.ExpireSeconds); err != nil {
		zap.L().Error("SaveGameStorage failed",
			zap.Uint32("userId", req.UserId),
			zap.Uint32("gameId", req.GameId),
			zap.String("storageKey", req.StorageKey),
			zap.Error(err))
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		return resp, nil
	}
	return resp, nil
}

// LoadGameStorage 按联合键读取长期状态；不存在返回 found=false。
func (d *LotteryService) LoadGameStorage(_ context.Context, req *services.LoadGameStorageReq) (resp *services.LoadGameStorageResp, err error) {
	resp = &services.LoadGameStorageResp{Code: services.ErrorCode_OK}
	if req == nil ||
		req.UserId == 0 ||
		req.GameId == 0 ||
		strings.TrimSpace(req.CurrencySymbol) == "" ||
		strings.TrimSpace(req.StorageKey) == "" {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		return resp, nil
	}
	payload, found, err := d.db.LoadGameStorage(req.UserId, req.GameId, req.CurrencySymbol, req.StorageKey)
	if err != nil {
		zap.L().Error("LoadGameStorage failed",
			zap.Uint32("userId", req.UserId),
			zap.Uint32("gameId", req.GameId),
			zap.String("storageKey", req.StorageKey),
			zap.Error(err))
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		return resp, nil
	}
	resp.Found = found
	resp.Payload = payload
	return resp, nil
}

// DeleteGameStorage 幂等删除长期状态。
func (d *LotteryService) DeleteGameStorage(_ context.Context, req *services.DeleteGameStorageReq) (resp *services.DeleteGameStorageResp, err error) {
	resp = &services.DeleteGameStorageResp{Code: services.ErrorCode_OK}
	var userID, gameID uint32
	var currencySymbol, storageKey string
	if req != nil {
		userID = req.UserId
		gameID = req.GameId
		currencySymbol = req.CurrencySymbol
		storageKey = req.StorageKey
	}
	zap.L().Info("DeleteGameStorage request",
		zap.Uint32("userId", userID),
		zap.Uint32("gameId", gameID),
		zap.String("currencySymbol", currencySymbol),
		zap.String("storageKey", storageKey),
	)
	if req == nil ||
		req.UserId == 0 ||
		req.GameId == 0 ||
		strings.TrimSpace(req.CurrencySymbol) == "" ||
		strings.TrimSpace(req.StorageKey) == "" {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		return resp, nil
	}
	if err := d.db.DeleteGameStorage(req.UserId, req.GameId, req.CurrencySymbol, req.StorageKey); err != nil {
		zap.L().Error("DeleteGameStorage failed",
			zap.Uint32("userId", req.UserId),
			zap.Uint32("gameId", req.GameId),
			zap.String("storageKey", req.StorageKey),
			zap.Error(err))
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		return resp, nil
	}
	return resp, nil
}
