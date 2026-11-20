package rpc

import (
	"app/config"
	"app/entity"
	"app/esindex"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"lottery/dao"
	"math"
	"regexp"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"micro_service/services"
)

var luck5AmountPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]{1,2})?$`)

// 拒绝小数分、指数与超出 int64 的金额，资金计算始终使用整数分。
func parseLuck5Amount(raw string) (int64, bool) {
	if !luck5AmountPattern.MatchString(raw) {
		return 0, false
	}
	amount, err := decimal.NewFromString(raw)
	if err != nil {
		return 0, false
	}
	cents := amount.Shift(2).BigInt()
	if !cents.IsInt64() {
		return 0, false
	}
	return cents.Int64(), true
}

// 幸运5只接受自身游戏的完整身份，零代理也是合法的测试代理。
func validateLuck5Identity(round *dao.Luck5Round) bool {
	return round.UserId > 0 && round.GameId == 7012 && strings.TrimSpace(round.RoundId) != "" &&
		strings.TrimSpace(round.CurrencyType) != ""
}

// 业务拒绝通过协议码返回，系统错误记录上下文但不暴露原始记录。
func luck5Response(resp *services.Luck5Resp, err error, roundId string) (*services.Luck5Resp, error) {
	if err != nil {
		zap.L().Error("Luck5 RPC failed", zap.String("roundId", roundId), zap.Error(err))
	}
	if resp == nil {
		resp = &services.Luck5Resp{Code: services.ErrorCode_SYSTEM_ERROR, Error: "systemError"}
	}
	if resp.Code == services.ErrorCode_NO_ENOUGH_POOL_MONEY {
		zap.L().Info("Luck5 pool insufficient", zap.String("roundId", roundId))
	}
	return resp, nil
}

// 首牌尚未展示，沿用现有 RPC 占位和代理锁扣注、扣除最高毛赔付。
func (d *LotteryService) Luck5Deal(ctx context.Context, req *services.Luck5DealReq) (*services.Luck5Resp, error) {
	if req == nil {
		return &services.Luck5Resp{Code: services.ErrorCode_PARAMS_INVALID, Error: "invalidRequest"}, nil
	}
	round := &dao.Luck5Round{UserId: req.UserId, AgentId: req.AgentId, GameId: req.GameId, CurrencyType: req.CurrencyType, RoundId: req.RoundId}
	bet, betOK := parseLuck5Amount(req.BetAmount)
	reserve, reserveOK := parseLuck5Amount(req.ReserveAmount)
	if !validateLuck5Identity(round) || !betOK || !reserveOK {
		return &services.Luck5Resp{Code: services.ErrorCode_PARAMS_INVALID, Error: "invalidDeal"}, nil
	}
	lockKey := buildIdempotencyKey("luck5Lock", req.RoundId)
	if code := d.beginMultiOpLock(lockKey); code != services.ErrorCode_OK {
		return &services.Luck5Resp{Code: code, Error: "roundBusy"}, nil
	}
	defer d.endMultiOpLock(lockKey)
	round.BetCent, round.ReserveCent = bet, reserve
	if response, err := d.rds.Luck5DealResult(ctx, round); err != nil || response != nil {
		return luck5Response(response, err, req.RoundId)
	}
	eGame := dao.GamesManagerIns().GetById(int64(req.GameId))
	if eGame == nil {
		return &services.Luck5Resp{Code: services.ErrorCode_PARAMS_INVALID, Error: "invalidGame"}, nil
	}
	if req.AgentId > 0 {
		agent := dao.AgentManagerIns().Get(int64(req.AgentId))
		if agent == nil {
			return &services.Luck5Resp{Code: services.ErrorCode_PARAMS_INVALID, Error: "invalidAgent"}, nil
		}
		round.WebId = uint32(agent.WebId)
	}
	exchange, ok := config.CfgIns.GetExchange(req.CurrencyType)
	pool := config.CfgIns.GetPoolCfg(int64(req.AgentId), eGame.ConfName)
	if !ok || exchange.IsNegative() || exchange.IsZero() || pool == nil || pool.Pool[1] == nil {
		return &services.Luck5Resp{Code: services.ErrorCode_SYSTEM_ERROR, Error: "poolConfigMissing"}, nil
	}
	if _, code := d.getPlayerCurrency(req.UserId); code != services.ErrorCode_OK {
		return &services.Luck5Resp{Code: code, Error: "balanceLoadFailed"}, nil
	}
	player, err := d.rds.GetPlayer(req.UserId, 0)
	if err != nil {
		return luck5Response(nil, err, req.RoundId)
	}
	if player == nil || player.AgentId != req.AgentId || player.CurrencyType != req.CurrencyType {
		return &services.Luck5Resp{Code: services.ErrorCode_PARAMS_INVALID, Error: "playerIdentityMismatch"}, nil
	}
	round.Symbol, round.Exchange, round.GameName = eGame.ConfName, exchange, pool.Name
	resp, err := dao.CacheIns().Luck5Deal(ctx, round)
	return luck5Response(resp, err, req.RoundId)
}

// 结算核对完整记录身份，不生成牌面或从金额重造游戏记录。
func validateLuck5Record(req *services.Luck5SettleReq) bool {
	var record struct {
		RecordId       string  `json:"recordId"`
		UserId         uint32  `json:"userId"`
		AgentId        *uint32 `json:"agentId"`
		GameId         uint32  `json:"gameId"`
		CurrencySymbol string  `json:"currencySymbol"`
		CreatedAtMs    int64   `json:"createdAtMs"`
	}
	if err := jsoniter.UnmarshalFromString(req.RecordJson, &record); err != nil {
		return false
	}
	return record.CreatedAtMs >= 0 && record.CreatedAtMs <= math.MaxInt64/1000000 &&
		record.RecordId == req.RoundId && record.UserId == req.UserId && record.GameId == req.GameId &&
		record.CurrencySymbol == req.CurrencyType && (record.AgentId == nil || *record.AgentId == req.AgentId)
}

// 已预扣局直接按原池派奖，记录保存失败时资金阶段保留供原请求补完。
func (d *LotteryService) Luck5Settle(ctx context.Context, req *services.Luck5SettleReq) (*services.Luck5Resp, error) {
	if req == nil {
		return &services.Luck5Resp{Code: services.ErrorCode_PARAMS_INVALID, Error: "invalidRequest"}, nil
	}
	round := &dao.Luck5Round{UserId: req.UserId, AgentId: req.AgentId, GameId: req.GameId, CurrencyType: req.CurrencyType, RoundId: req.RoundId, RecordJson: req.RecordJson}
	win, ok := parseLuck5Amount(req.WinAmount)
	if !validateLuck5Identity(round) || !ok || !validateLuck5Record(req) {
		return &services.Luck5Resp{Code: services.ErrorCode_PARAMS_INVALID, Error: "invalidSettle"}, nil
	}
	lockKey := buildIdempotencyKey("luck5Lock", req.RoundId)
	if code := d.beginMultiOpLock(lockKey); code != services.ErrorCode_OK {
		return &services.Luck5Resp{Code: code, Error: "roundBusy"}, nil
	}
	defer d.endMultiOpLock(lockKey)
	round.WinCent = win
	round.Signature = idempotencySignature(fmt.Sprint(win), req.RecordJson)
	resp, err := dao.CacheIns().Luck5Settle(ctx, round, func(stored *dao.Luck5Round) error {
		return d.saveLuck5Record(ctx, stored)
	})
	return luck5Response(resp, err, req.RoundId)
}

// 使用原局固定文档 ID 同步保存，写入失败或回包丢失不会生成第二条记录。
func (d *LotteryService) saveLuck5Record(ctx context.Context, round *dao.Luck5Round) error {
	if d.es == nil || d.es.Client == nil {
		return errors.New("record service unavailable")
	}
	bet := decimal.NewFromInt(round.BetCent).Shift(-2)
	win := decimal.NewFromInt(round.WinCent).Shift(-2)
	balance := decimal.NewFromInt(round.Balance).Shift(-2)
	var payload struct {
		CreatedAtMs int64 `json:"createdAtMs"`
	}
	if err := jsoniter.UnmarshalFromString(round.RecordJson, &payload); err != nil {
		return err
	}
	if payload.CreatedAtMs < 0 || payload.CreatedAtMs > math.MaxInt64/1000000 {
		return errors.New("invalid record timestamp")
	}
	hash := md5.Sum([]byte(fmt.Sprintf("%d|%d|%s", round.AgentId, round.UserId, round.RoundId)))
	record := &entity.CacheRecordsReq{
		WebId: round.WebId, UserId: round.UserId, AgentId: round.AgentId, GameId: round.GameId,
		Account: round.Account, NickName: round.Account, Currency: round.CurrencyType, CurrencySymbol: round.CurrencyType,
		Bet: bet.InexactFloat64(), BaseBet: bet.InexactFloat64(), ExBet: bet.Mul(round.Exchange).InexactFloat64(),
		Win: win.InexactFloat64(), ExWin: win.Mul(round.Exchange).InexactFloat64(),
		Balance: balance.InexactFloat64(), BalanceCash: balance.InexactFloat64(),
		Chips:      decimal.Max(bet, win).Mul(round.Exchange).InexactFloat64(),
		PlayedDate: payload.CreatedAtMs, RowVersion: payload.CreatedAtMs * 1000000,
		RoundID: round.RoundId, Symbol: round.Symbol, GameName: round.GameName,
		Log: round.RecordJson, Hash: fmt.Sprintf("%x", hash), Complete: true, IsTourist: round.IsTourist,
	}
	saveCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, err := d.es.Client.Index().Index(esindex.Settlement()).Id(record.Hash).BodyJson(record).Do(saveCtx)
	return err
}
