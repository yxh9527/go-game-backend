package rpc

import (
	"context"
	"strconv"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"micro_service/services"
)

func mergePlayerGameIcon(exData string, gameId, icon uint32) (string, error) {
	root := map[string]map[string]any{}
	if strings.TrimSpace(exData) != "" {
		if err := jsoniter.UnmarshalFromString(exData, &root); err != nil {
			return "", err
		}
	}
	key := strconv.FormatUint(uint64(gameId), 10)
	item := root[key]
	if item == nil {
		item = map[string]any{}
	}
	item["icon"] = icon
	root[key] = item
	return jsoniter.MarshalToString(root)
}

// SetUserIcon 写入 UG5 游戏内头像到 player.ex_data：{"gameId":{"icon":id}}。
func (d *LotteryService) SetUserIcon(_ context.Context, req *services.SetUserIconReq) (resp *services.SetUserIconResp, err error) {
	resp = &services.SetUserIconResp{Code: services.ErrorCode_OK}
	if req == nil || req.UserId == 0 || req.GameId == 0 {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		return resp, nil
	}
	p, err := d.rds.GetPlayer(req.UserId, 0)
	if err != nil {
		zap.L().Error("SetUserIcon get player failed", zap.Uint32("userId", req.UserId), zap.Error(err))
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		return resp, nil
	}
	if p == nil {
		if code := d.loadPlayerToCache(req.UserId); code != services.ErrorCode_OK {
			resp.Code = code
			return resp, nil
		}
		p, err = d.rds.GetPlayer(req.UserId, 0)
		if err != nil || p == nil {
			zap.L().Error("SetUserIcon reload player failed", zap.Uint32("userId", req.UserId), zap.Error(err))
			resp.Code = services.ErrorCode_SYSTEM_ERROR
			return resp, nil
		}
	}
	exData, err := mergePlayerGameIcon(p.ExData, req.GameId, req.Icon)
	if err != nil {
		zap.L().Error("SetUserIcon merge ex_data failed", zap.Uint32("userId", req.UserId), zap.Error(err))
		resp.Code = services.ErrorCode_PARAMS_INVALID
		return resp, nil
	}
	if err := d.rds.UpdatePlayerExData(req.UserId, exData); err != nil {
		if err == redis.Nil {
			if code := d.loadPlayerToCache(req.UserId); code != services.ErrorCode_OK {
				resp.Code = code
				return resp, nil
			}
			if err := d.rds.UpdatePlayerExData(req.UserId, exData); err != nil {
				zap.L().Error("SetUserIcon save ex_data failed", zap.Uint32("userId", req.UserId), zap.Error(err))
				resp.Code = services.ErrorCode_SYSTEM_ERROR
				return resp, nil
			}
			return resp, nil
		}
		zap.L().Error("SetUserIcon save ex_data failed", zap.Uint32("userId", req.UserId), zap.Error(err))
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		return resp, nil
	}
	return resp, nil
}
