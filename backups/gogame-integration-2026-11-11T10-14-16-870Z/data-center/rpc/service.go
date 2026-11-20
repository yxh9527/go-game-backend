package rpc

import (
	"app/entity"
	"context"
	"data-center/dao"
	"micro_service/services"
	"strings"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DataCenterService struct {
	services.UnimplementedDataCenterServiceServer
	db  *dao.DBDao
	rds *dao.RedisDao
	es  *dao.ESDao

	RecordChan chan *entity.CacheRecordsReq
}

func NewDataCenterService(db, mdb *gorm.DB, es *elastic.Client) *DataCenterService {
	tmp := &DataCenterService{
		db:         dao.NewDBDao(db, mdb),
		rds:        dao.RedisIns(),
		es:         dao.NewESDao(es),
		RecordChan: make(chan *entity.CacheRecordsReq, 10240),
	}

	tmp.syncRecords()
	return tmp
}

// 注单入库
func (dc *DataCenterService) syncRecords() {
	go func() {
		defer func() {
			if e := recover(); e != nil {
				zap.L().Error("syncRecords,数据落地协程panic", zap.Any("recover", e))
			}
		}()
		data := make([]*entity.CacheRecordsReq, 0, 16)
		//十秒写一次
		t := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-t.C:
				if len(data) > 0 {
					dc.es.BulkRecordsSave(data)
					ids := make([]string, 0, 64)
					for _, v := range data {
						ids = append(ids, v.RoundID)
					}
					zap.L().Debug("批量写入注单信息", zap.Any("roundIds", ids))
					data = make([]*entity.CacheRecordsReq, 0, 16)
				}
			case req := <-dc.RecordChan:
				data = append(data, req)
				if len(data) >= 8 {
					dc.es.BulkRecordsSave(data)
					ids := make([]string, 0, 64)
					for _, v := range data {
						ids = append(ids, v.RoundID)
					}
					zap.L().Debug("批量写入注单信息", zap.Any("roundIds", ids))
					data = make([]*entity.CacheRecordsReq, 0, 16)
				}
			}
		}
	}()
}

func (d *DataCenterService) GetPlayer(ctx context.Context, req *services.GetPlayerReq) (resp *services.GetPlayerResp, err error) {
	p, err := d.rds.GetPlayer(req.GetPlayerId(), req.Factory)
	if err != nil {
		return nil, err
	}
	resp = &services.GetPlayerResp{}
	if p != nil {
		resp.HumanPlayer = p
		return resp, nil
	}
	// 在redis中没有找到玩家信息，从db中加载
	player, err := d.db.GetPlayer(ctx, req.PlayerId)
	if err != nil {
		return nil, err
	}
	p = ConvertUserEntityToHumanPlayer(player)
	if err := d.rds.SetPlayer(p, false); err != nil {
		zap.L().Warn("向redis写入玩家信息缓存失败，下一次获取该玩家信息时还将从数据库中读取", zap.Uint32("player_id", p.Id))
	}
	resp.HumanPlayer = p
	zap.L().Debug("get player", zap.Any("resp", resp))
	return resp, nil
}

func (d *DataCenterService) UpdatePlayerAvatarAndGender(ctx context.Context, req *services.UpdatePlayerAvatarAndGenderReq) (resp *services.UpdatePlayerAvatarAndGenderResp, err error) {
	err = d.rds.UpdatePlayerAvatarAndGender(req.PlayerId, req.Avatar, req.Name, req.Gender)
	if err == redis.Nil {
		// 在redis中没有找到玩家信息，从db中加载
		player, err := d.db.GetPlayer(ctx, req.PlayerId)
		if err != nil {
			zap.L().Error("从数据库加载玩家信息失败", zap.Stringer("req", req), zap.Error(err))
			return nil, err
		}
		if err := d.rds.SetPlayer(ConvertUserEntityToHumanPlayer(player), true); err != nil {
			zap.L().Error("向redis写入玩家信息缓存失败", zap.Stringer("req", req), zap.Error(err))
			return nil, err
		}
	} else if err != nil {
		zap.L().Error("更新玩家头像或性别失败", zap.Stringer("req", req), zap.Error(err))
		return nil, err
	}
	return &services.UpdatePlayerAvatarAndGenderResp{}, nil
}

func (d *DataCenterService) GetValue(_ context.Context, req *services.GetValueReq) (resp *services.GetValueResp, err error) {
	resp = &services.GetValueResp{Code: services.ErrorCode_OK}
	res, err := d.rds.Get(req.Key, req.TimeOut)
	if err != nil && err != redis.Nil {
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		return resp, err
	}
	resp.Value = res
	return resp, nil
}

func (d *DataCenterService) SetValue(_ context.Context, req *services.SetValueReq) (resp *services.SetValueResp, err error) {
	resp = &services.SetValueResp{Code: services.ErrorCode_OK}
	if err := d.rds.Set(req.Key, req.Value, req.TimeOut); err != nil {
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		return nil, err
	}
	return resp, nil
}

// 获取注单信息（ES 旧注单列表，规则 A：不另落 SlotSpinRecord）
func (d *DataCenterService) GetRecords(_ context.Context, req *services.GetRecordsReq) (resp *services.GetRecordsResp, err error) {
	resp = &services.GetRecordsResp{Code: services.ErrorCode_OK, Data: make([]*services.RecordItem, 0)}
	if req == nil {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		return resp, nil
	}

	symbol := strings.TrimSpace(req.Symbol)
	if symbol == "" && req.GameId > 0 {
		if g := d.db.GetGameByNumber(int64(req.GameId)); g != nil {
			symbol = g.ConfName
		}
	}

	// 兼容旧调用：仅 Hash 时走原路径语义
	if strings.TrimSpace(req.Hash) != "" && req.Page == 0 && req.Size == 0 && req.StartTime == 0 && req.EndTime == 0 && strings.TrimSpace(req.Currency) == "" {
		resp.Data = d.es.GetRecords(req.UserId, symbol, req.Hash, "")
		resp.Total = int64(len(resp.Data))
		return resp, nil
	}

	items, total, qErr := d.es.ListRecords(dao.RecordListQuery{
		UserId:   req.UserId,
		Symbol:   symbol,
		Hash:     req.Hash,
		Currency: req.Currency,
		GameId:   req.GameId,
		StartMs:  req.StartTime,
		EndMs:    req.EndTime,
		Page:     int(req.Page),
		Size:     int(req.Size),
	})
	if qErr != nil {
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		return resp, nil
	}
	resp.Data = items
	resp.Total = total
	return resp, nil
}

// GetRecordDetail 详情必须匹配 recordId + userId + gameId；不匹配返回 found=false。
func (d *DataCenterService) GetRecordDetail(_ context.Context, req *services.GetRecordDetailReq) (resp *services.GetRecordDetailResp, err error) {
	resp = &services.GetRecordDetailResp{Code: services.ErrorCode_OK, Found: false}
	if req == nil || strings.TrimSpace(req.RecordId) == "" || req.UserId == 0 || req.GameId == 0 {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		return resp, nil
	}
	symbol := ""
	if g := d.db.GetGameByNumber(int64(req.GameId)); g != nil {
		symbol = g.ConfName
	}
	item, found, qErr := d.es.GetRecordDetailByKeys(req.RecordId, req.UserId, req.GameId, symbol)
	if qErr != nil {
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		return resp, nil
	}
	resp.Found = found
	resp.Data = item
	return resp, nil
}

// 玩家锁 加锁
func (d *DataCenterService) UserLock(ctx context.Context, req *services.UserLockReq) (resp *services.UserLockResp, err error) {
	resp = &services.UserLockResp{}
	resp.Result = d.rds.UserLock(req.UserId)
	return resp, nil
}

// 玩家锁 解锁
func (d *DataCenterService) UserUnLock(ctx context.Context, req *services.UserUnLockReq) (resp *services.UserUnLockResp, err error) {
	resp = &services.UserUnLockResp{}
	resp.Result = d.rds.UserUnLock(req.UserId, req.Token)
	return resp, nil
}

// 获取游戏列表
func (d *DataCenterService) GetGameList(ctx context.Context, req *services.GetGameListReq) (resp *services.GetGameListResp, err error) {
	resp = d.db.GetGameList()
	// zap.L().Debug("游戏列表", zap.Any("data", resp), zap.Any("count", len(resp.All)))
	return resp, nil
}

func (d *DataCenterService) GetSesson(ctx context.Context, req *services.GetSessionReq) (resp *services.GetSessionResp, err error) {
	resp = &services.GetSessionResp{Code: services.ErrorCode_OK}
	res, err := d.rds.Get(req.Key, req.TimeOut)
	if err != nil && err != redis.Nil {
		zap.L().Debug("====>getSession", zap.Any("req", req))
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		return resp, err
	}
	resp.Value = res
	return resp, nil
}
