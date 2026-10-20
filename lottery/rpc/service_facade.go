package rpc

import (
	"context"
	"sort"
	"strings"

	"app/config"
	"app/tables/manager"
	"lottery/dao"

	jsoniter "github.com/json-iterator/go"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"micro_service/services"
)

func parseAmount(v string) decimal.Decimal {
	d, ok := parseAmountStrict(v)
	if !ok {
		return decimal.Zero
	}
	return d
}

// parseAmountStrict 空串视为 0；非法数字返回 false。
func parseAmountStrict(v string) (decimal.Decimal, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return decimal.Zero, true
	}
	d, err := decimal.NewFromString(v)
	if err != nil {
		return decimal.Zero, false
	}
	return d, true
}

func failSlotsDoBet(resp *services.SlotsDoBetResp, code services.ErrorCode, errName string) (*services.SlotsDoBetResp, error) {
	resp.Code = code
	resp.Ret = false
	if errName != "" {
		resp.Error = errName
	} else {
		resp.Error = slotErrorName(code)
	}
	return resp, nil
}

func validateFlowControl(fc *services.SlotDoBetFlowControl) (string, bool) {
	if fc == nil {
		return "", true
	}
	if refund, ok := parseAmountStrict(fc.RefundPoolAmount); !ok {
		return "invalid refundPoolAmount", false
	} else if refund.IsNegative() {
		return "refundPoolAmount can not be negative", false
	}
	for _, key := range fc.DeleteGameStorageKeys {
		if strings.TrimSpace(key) == "" {
			return "deleteGameStorageKeys contains empty key", false
		}
	}
	return "", true
}

func validateGameStorageItems(items []*services.GameStorageItem) (string, bool) {
	for _, item := range items {
		if item == nil {
			return "gameStorage item is nil", false
		}
		if strings.TrimSpace(item.StorageKey) == "" {
			return "gameStorage.storageKey required", false
		}
	}
	return "", true
}

// validateFacadeIdentity 结算门面共用的用户/代理/游戏/币种校验。
// agentId=0 视为测试代理：跳过代理存在/冻结校验，其余字段仍严格校验。
func validateFacadeIdentity(userId, agentId, gameId uint32, roundId, currencyType string) (services.ErrorCode, string) {
	if userId == 0 || gameId == 0 {
		return services.ErrorCode_PARAMS_INVALID, ""
	}
	if strings.TrimSpace(roundId) == "" || strings.TrimSpace(currencyType) == "" {
		return services.ErrorCode_PARAMS_INVALID, ""
	}
	if agentId > 0 && dao.AgentManagerIns().Get(int64(agentId)) == nil {
		return services.ErrorCode_AGENT_FROZEN, ""
	}
	if dao.GamesManagerIns().GetById(int64(gameId)) == nil {
		return services.ErrorCode_PARAMS_INVALID, ""
	}
	if _, ok := config.CfgIns.GetExchange(currencyType); !ok {
		return services.ErrorCode_SYSTEM_ERROR, "exchangeConfigMissing"
	}
	return services.ErrorCode_OK, ""
}

// validateSlotsFacadeBase 普通局/免费局门面共用的基础校验。
func validateSlotsFacadeBase(userId, agentId, gameId uint32, roundId, currencyType string, fc *services.SlotDoBetFlowControl, storages []*services.GameStorageItem) (services.ErrorCode, string) {
	if code, msg := validateFacadeIdentity(userId, agentId, gameId, roundId, currencyType); code != services.ErrorCode_OK {
		return code, msg
	}
	if msg, ok := validateFlowControl(fc); !ok {
		return services.ErrorCode_PARAMS_INVALID, msg
	}
	if msg, ok := validateGameStorageItems(storages); !ok {
		return services.ErrorCode_PARAMS_INVALID, msg
	}
	return services.ErrorCode_OK, ""
}

func failFruitDoBet(resp *services.FruitDoBetResp, code services.ErrorCode) (*services.FruitDoBetResp, error) {
	resp.Code = code
	resp.Ret = false
	return resp, nil
}

func failFruitDoBetMulti(resp *services.FruitDoBetMultiResp, code services.ErrorCode) (*services.FruitDoBetMultiResp, error) {
	resp.Code = code
	resp.Ret = false
	return resp, nil
}

func failFruitRefundMulti(resp *services.FruitRefundMultiResp, code services.ErrorCode) (*services.FruitRefundMultiResp, error) {
	resp.Code = code
	resp.Ret = false
	return resp, nil
}

func failFruitSettleRound(resp *services.FruitSettleRoundResp, code services.ErrorCode) (*services.FruitSettleRoundResp, error) {
	resp.Code = code
	resp.Ret = false
	return resp, nil
}

// validateFacadeRoom 百人房间维度校验（不含单个 userId）。
// agentId=0 视为测试代理：跳过代理存在/冻结校验。
func validateFacadeRoom(agentId, gameId uint32, roundId, currencyType string) (services.ErrorCode, string) {
	if gameId == 0 {
		return services.ErrorCode_PARAMS_INVALID, ""
	}
	if strings.TrimSpace(roundId) == "" || strings.TrimSpace(currencyType) == "" {
		return services.ErrorCode_PARAMS_INVALID, ""
	}
	if agentId > 0 && dao.AgentManagerIns().Get(int64(agentId)) == nil {
		return services.ErrorCode_AGENT_FROZEN, ""
	}
	if dao.GamesManagerIns().GetById(int64(gameId)) == nil {
		return services.ErrorCode_PARAMS_INVALID, ""
	}
	if _, ok := config.CfgIns.GetExchange(currencyType); !ok {
		return services.ErrorCode_SYSTEM_ERROR, "exchangeConfigMissing"
	}
	return services.ErrorCode_OK, ""
}

func (d *LotteryService) applyGameStorageWrites(userId, gameId uint32, currency string, items []*services.GameStorageItem, deleteKeys []string) services.ErrorCode {
	for _, key := range deleteKeys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if err := d.db.DeleteGameStorage(userId, gameId, currency, key); err != nil {
			zap.L().Error("facade delete game storage failed", zap.Uint32("userId", userId), zap.String("key", key), zap.Error(err))
			return services.ErrorCode_SYSTEM_ERROR
		}
	}
	for _, item := range items {
		if item == nil || strings.TrimSpace(item.StorageKey) == "" {
			continue
		}
		if err := d.db.SaveGameStorage(userId, gameId, currency, item.StorageKey, item.Payload, item.ExpireSeconds); err != nil {
			zap.L().Error("facade save game storage failed", zap.Uint32("userId", userId), zap.String("key", item.StorageKey), zap.Error(err))
			return services.ErrorCode_SYSTEM_ERROR
		}
	}
	return services.ErrorCode_OK
}

func (d *LotteryService) refundFacadePool(agentId, userId, gameId uint32, currencyType, amount string) services.ErrorCode {
	refund := parseAmount(amount)
	if !refund.GreaterThan(decimal.Zero) {
		return services.ErrorCode_OK
	}
	eGame := dao.GamesManagerIns().GetById(int64(gameId))
	if eGame == nil {
		return services.ErrorCode_PARAMS_INVALID
	}
	exchange, ok := config.CfgIns.GetExchange(currencyType)
	if !ok {
		return services.ErrorCode_SYSTEM_ERROR
	}
	dao.CacheIns().ReturnPool(int64(agentId), userId, eGame.ConfName, refund.Mul(exchange))
	d.pcr.Record(int64(agentId), eGame.ConfName, dao.CacheIns().GetPool(int64(agentId), eGame.ConfName))
	return services.ErrorCode_OK
}

func fillBalanceResp(currency string) (string, int64) {
	nc := parseAmount(currency)
	return nc.Truncate(2).StringFixed(2), nc.Mul(decimal.NewFromInt(100)).IntPart()
}

func slotErrorName(code services.ErrorCode) string {
	switch code {
	case services.ErrorCode_NO_ENOUGH_MONEY:
		return "balanceInsufficient"
	case services.ErrorCode_NO_ENOUGH_POOL_MONEY:
		return "poolInsufficient"
	case services.ErrorCode_PARAMS_INVALID:
		return "paramsInvalid"
	default:
		if code != services.ErrorCode_OK {
			return "systemError"
		}
		return ""
	}
}

// SlotsDoBet 新服普通局门面。
// 规则 A：内部走原 SlotsLottery（下注 + Complete 结算/ES 注单），再提交 GameStorage；
// preWinAmount 与本局 winAmount 一起预占水池；未使用部分只通过 flowControl.refundPoolAmount 退回。
func (d *LotteryService) SlotsDoBet(ctx context.Context, req *services.SlotsDoBetReq) (resp *services.SlotsDoBetResp, err error) {
	resp = &services.SlotsDoBetResp{Code: services.ErrorCode_OK, Ret: false}
	if req == nil {
		return failSlotsDoBet(resp, services.ErrorCode_PARAMS_INVALID, "")
	}
	if code, msg := validateSlotsFacadeBase(req.UserId, req.AgentId, req.GameId, req.RoundId, req.CurrencyType, req.FlowControl, req.GameStorage); code != services.ErrorCode_OK {
		return failSlotsDoBet(resp, code, msg)
	}

	bet, betOK := parseAmountStrict(req.BetAmount)
	if !betOK {
		return failSlotsDoBet(resp, services.ErrorCode_PARAMS_INVALID, "invalid betAmount")
	}
	if bet.IsNegative() {
		return failSlotsDoBet(resp, services.ErrorCode_PARAMS_INVALID, "betAmount can not be negative")
	}
	win, winOK := parseAmountStrict(req.WinAmount)
	if !winOK {
		return failSlotsDoBet(resp, services.ErrorCode_PARAMS_INVALID, "invalid winAmount")
	}
	if win.IsNegative() {
		return failSlotsDoBet(resp, services.ErrorCode_PARAMS_INVALID, "winAmount can not be negative")
	}
	preWin, preOK := parseAmountStrict(req.PreWinAmount)
	if !preOK {
		return failSlotsDoBet(resp, services.ErrorCode_PARAMS_INVALID, "invalid preWinAmount")
	}
	if preWin.IsNegative() {
		return failSlotsDoBet(resp, services.ErrorCode_PARAMS_INVALID, "preWinAmount can not be negative")
	}

	idemKey := buildIdempotencyKey("slotsDoBet", u32Str(req.UserId), u32Str(req.GameId), req.CurrencyType, req.RoundId)
	idemSig := idempotencySignature(bet.String(), win.String(), preWin.String(), req.RecordJson, req.Account, decimal.NewFromFloat(req.AverageBet).String())
	if hit, payload, code := d.beginIdempotency(idemKey, idemSig); code != services.ErrorCode_OK {
		msg := ""
		if code == services.ErrorCode_PARAMS_INVALID {
			msg = "idempotencyConflict"
		}
		return failSlotsDoBet(resp, code, msg)
	} else if hit {
		if cached, rCode := restoreSlotsDoBetResp(payload); rCode != services.ErrorCode_OK {
			return failSlotsDoBet(resp, rCode, "")
		} else {
			return cached, nil
		}
	}

	slotsResp, callErr := d.SlotsLottery(ctx, &slotsLotteryReq{
		PlayerId:      req.UserId,
		CurrencyType:  req.CurrencyType,
		AgentId:       int64(req.AgentId),
		GameId:        req.GameId,
		ProfitLoss:    win.String(),
		Bet:           bet.String(),
		State:         req.RecordJson,
		RoundID:       req.RoundId,
		MaxProfitLoss: preWin.String(),
		Account:       req.Account,
		AverageBet:    req.AverageBet,
	})
	if callErr != nil {
		d.abortIdempotency(idemKey)
		zap.L().Error("SlotsDoBet call SlotsLottery failed",
			zap.Uint32("userId", req.UserId),
			zap.String("roundId", req.RoundId),
			zap.Error(callErr))
		return failSlotsDoBet(resp, services.ErrorCode_SYSTEM_ERROR, "")
	}
	if slotsResp == nil {
		d.abortIdempotency(idemKey)
		return failSlotsDoBet(resp, services.ErrorCode_SYSTEM_ERROR, "")
	}
	if !slotsResp.Result || slotsResp.Code != services.ErrorCode_OK {
		d.abortIdempotency(idemKey)
		resp.Code = slotsResp.Code
		resp.Ret = false
		resp.Error = slotErrorName(slotsResp.Code)
		if slotsResp.NewCurrency != "" {
			resp.Currency, resp.CurrencyCent = fillBalanceResp(slotsResp.NewCurrency)
		}
		return resp, nil
	}

	var deleteKeys []string
	if req.FlowControl != nil {
		deleteKeys = req.FlowControl.DeleteGameStorageKeys
		if code := d.refundFacadePool(req.AgentId, req.UserId, req.GameId, req.CurrencyType, req.FlowControl.RefundPoolAmount); code != services.ErrorCode_OK {
			d.abortIdempotency(idemKey)
			return failSlotsDoBet(resp, code, "")
		}
	}
	if code := d.applyGameStorageWrites(req.UserId, req.GameId, req.CurrencyType, req.GameStorage, deleteKeys); code != services.ErrorCode_OK {
		d.abortIdempotency(idemKey)
		return failSlotsDoBet(resp, code, "")
	}

	resp.Ret = true
	resp.Currency, resp.CurrencyCent = fillBalanceResp(slotsResp.NewCurrency)
	if raw, mErr := jsoniter.MarshalToString(resp); mErr == nil {
		d.commitIdempotency(idemKey, idemSig, raw)
	} else {
		d.abortIdempotency(idemKey)
	}
	zap.L().Debug("SlotsDoBet success",
		zap.Uint32("userId", req.UserId),
		zap.Uint32("agentId", req.AgentId),
		zap.Uint32("gameId", req.GameId),
		zap.String("roundId", req.RoundId),
		zap.String("bet", bet.String()),
		zap.String("win", win.String()),
		zap.String("currency", resp.Currency))
	return resp, nil
}

// SlotsDoBetFree 免费/奖励局门面。
// 不扣本金、不预扣水池、不走 MaxProfitLoss 对账。每次调用都完局（入账 + 注单 + Complete 统计）。
// 无赢分且无注单时只处理 storage/退池。未使用预占通过 flowControl.refundPoolAmount 退回。
func (d *LotteryService) SlotsDoBetFree(ctx context.Context, req *services.SlotsDoBetFreeReq) (resp *services.SlotsDoBetResp, err error) {
	resp = &services.SlotsDoBetResp{Code: services.ErrorCode_OK, Ret: false}
	if req == nil {
		return failSlotsDoBet(resp, services.ErrorCode_PARAMS_INVALID, "")
	}
	if code, msg := validateSlotsFacadeBase(req.UserId, req.AgentId, req.GameId, req.RoundId, req.CurrencyType, req.FlowControl, req.GameStorage); code != services.ErrorCode_OK {
		return failSlotsDoBet(resp, code, msg)
	}

	win, winOK := parseAmountStrict(req.WinAmount)
	if !winOK {
		return failSlotsDoBet(resp, services.ErrorCode_PARAMS_INVALID, "invalid winAmount")
	}
	if win.IsNegative() {
		return failSlotsDoBet(resp, services.ErrorCode_PARAMS_INVALID, "winAmount can not be negative")
	}
	totalBet, betOK := parseAmountStrict(req.TotalBetAmount)
	if !betOK {
		return failSlotsDoBet(resp, services.ErrorCode_PARAMS_INVALID, "invalid totalBetAmount")
	}
	if totalBet.IsNegative() {
		return failSlotsDoBet(resp, services.ErrorCode_PARAMS_INVALID, "totalBetAmount can not be negative")
	}

	hasRecord := strings.TrimSpace(req.RecordJson) != ""
	if !hasRecord && win.GreaterThan(decimal.Zero) {
		return failSlotsDoBet(resp, services.ErrorCode_PARAMS_INVALID, "recordless winAmount must be zero")
	}

	idemKey := buildIdempotencyKey("slotsDoBetFree", u32Str(req.UserId), u32Str(req.GameId), req.CurrencyType, req.RoundId)
	idemSig := idempotencySignature(win.String(), totalBet.String(), req.RecordJson, req.Account)
	if hit, payload, code := d.beginIdempotency(idemKey, idemSig); code != services.ErrorCode_OK {
		msg := ""
		if code == services.ErrorCode_PARAMS_INVALID {
			msg = "idempotencyConflict"
		}
		return failSlotsDoBet(resp, code, msg)
	} else if hit {
		if cached, rCode := restoreSlotsDoBetResp(payload); rCode != services.ErrorCode_OK {
			return failSlotsDoBet(resp, rCode, "")
		} else {
			return cached, nil
		}
	}

	currency := ""
	if win.GreaterThan(decimal.Zero) || hasRecord {
		nc, code := d.slotsFreeAward(req, win, totalBet)
		if code != services.ErrorCode_OK {
			d.abortIdempotency(idemKey)
			return failSlotsDoBet(resp, code, "")
		}
		currency = nc.Truncate(2).StringFixed(2)
	} else {
		cent, code := d.getPlayerCurrency(req.UserId)
		if code != services.ErrorCode_OK {
			d.abortIdempotency(idemKey)
			return failSlotsDoBet(resp, code, "")
		}
		currency = currencyFromCent(cent).Truncate(2).StringFixed(2)
	}

	var deleteKeys []string
	if req.FlowControl != nil {
		deleteKeys = req.FlowControl.DeleteGameStorageKeys
	}
	if code := d.applyGameStorageWrites(req.UserId, req.GameId, req.CurrencyType, req.GameStorage, deleteKeys); code != services.ErrorCode_OK {
		d.abortIdempotency(idemKey)
		return failSlotsDoBet(resp, code, "")
	}
	if req.FlowControl != nil {
		if code := d.refundFacadePool(req.AgentId, req.UserId, req.GameId, req.CurrencyType, req.FlowControl.RefundPoolAmount); code != services.ErrorCode_OK {
			d.abortIdempotency(idemKey)
			return failSlotsDoBet(resp, code, "")
		}
	}

	resp.Ret = true
	resp.Currency, resp.CurrencyCent = fillBalanceResp(currency)
	if raw, mErr := jsoniter.MarshalToString(resp); mErr == nil {
		d.commitIdempotency(idemKey, idemSig, raw)
	} else {
		d.abortIdempotency(idemKey)
	}
	zap.L().Debug("SlotsDoBetFree success",
		zap.Uint32("userId", req.UserId),
		zap.Uint32("agentId", req.AgentId),
		zap.Uint32("gameId", req.GameId),
		zap.String("roundId", req.RoundId),
		zap.String("currency", resp.Currency))
	return resp, nil
}

// slotsFreeAward 免费局入账并完局统计。不扣本金、不预扣、不对账 RoundCache。
func (d *LotteryService) slotsFreeAward(req *services.SlotsDoBetFreeReq, win, totalBet decimal.Decimal) (decimal.Decimal, services.ErrorCode) {
	eAgent := dao.AgentManagerIns().Get(int64(req.AgentId))
	if eAgent == nil {
		if req.AgentId == 0 {
			eAgent = &manager.Agent{Id: 0, WebId: 0}
		} else {
			return decimal.Zero, services.ErrorCode_AGENT_FROZEN
		}
	}
	eGame := dao.GamesManagerIns().GetById(int64(req.GameId))
	if eGame == nil {
		return decimal.Zero, services.ErrorCode_PARAMS_INVALID
	}

	exchange, ok := config.CfgIns.GetExchange(req.CurrencyType)
	if !ok {
		zap.L().Error("SlotsDoBetFree get exchange failed",
			zap.String("currencyType", req.CurrencyType),
			zap.String("roundId", req.RoundId))
		return decimal.Zero, services.ErrorCode_SYSTEM_ERROR
	}
	pc := config.CfgIns.GetPoolCfg(int64(req.AgentId), eGame.ConfName)
	if pc == nil {
		zap.L().Error("SlotsDoBetFree get pool cfg failed",
			zap.Uint32("agentId", req.AgentId),
			zap.String("roundId", req.RoundId),
			zap.String("symbol", eGame.ConfName))
		return decimal.Zero, services.ErrorCode_SYSTEM_ERROR
	}

	var newCurrency int64
	var code services.ErrorCode
	if win.GreaterThan(decimal.Zero) {
		newCurrency, code = d.updatePlayerCurrency(req.UserId, win.Mul(decimal.NewFromInt(100)).IntPart())
		if code != services.ErrorCode_OK {
			return decimal.Zero, code
		}
	} else {
		newCurrency, code = d.getPlayerCurrency(req.UserId)
		if code != services.ErrorCode_OK {
			return decimal.Zero, code
		}
	}

	nc := currencyFromCent(newCurrency)
	user := dao.CacheIns().GetUser(int64(req.AgentId), int64(req.UserId))
	if user != nil && user.IsTourist == 0 {
		account := req.Account
		if account == "" {
			account = user.Account
		}
		record := ConvertRecord(
			req.AgentId,
			req.UserId,
			req.RoundId,
			req.CurrencyType,
			eGame.ConfName,
			account,
			req.RecordJson,
			nc,
			uint32(eAgent.WebId),
			totalBet.InexactFloat64(),
			win.InexactFloat64(),
		)
		d.SaveRecord(record)
		zap.L().Info("SlotsDoBetFree:注单信息",
			zap.Uint32("agentId", req.AgentId),
			zap.Uint32("playerId", req.UserId),
			zap.Uint32("gameId", req.GameId),
			zap.String("symbol", eGame.ConfName),
			zap.String("roundId", req.RoundId),
			zap.Any("record", record))
		billBet := decimal.Zero
		if totalBet.GreaterThan(decimal.Zero) {
			billBet = totalBet.Neg()
		}
		billAward := decimal.Zero
		if win.GreaterThan(decimal.Zero) {
			billAward = win
		}
		d.SaveBill(req.AgentId, req.UserId, billBet, billAward, nc.Truncate(2).InexactFloat64(), eGame.ConfName, "返奖", req.CurrencyType, req.RoundId)
	}
	dao.CacheIns().Complete(int64(req.AgentId), req.UserId, eGame.ConfName, totalBet.Mul(exchange), win.Mul(exchange), pc.Pool[1].Revenue)
	return nc, services.ErrorCode_OK
}

// FruitDoBet 单人 Fruit 门面：复用 doSingleBet（规则 A：水池/注单/Complete 均走原逻辑）。
func (d *LotteryService) FruitDoBet(ctx context.Context, req *services.FruitDoBetReq) (resp *services.FruitDoBetResp, err error) {
	resp = &services.FruitDoBetResp{Code: services.ErrorCode_OK, Ret: false}
	if req == nil {
		return failFruitDoBet(resp, services.ErrorCode_PARAMS_INVALID)
	}
	if code, _ := validateFacadeIdentity(req.UserId, req.AgentId, req.GameId, req.RoundId, req.CurrencyType); code != services.ErrorCode_OK {
		return failFruitDoBet(resp, code)
	}

	bet, betOK := parseAmountStrict(req.BetAmount)
	if !betOK {
		return failFruitDoBet(resp, services.ErrorCode_PARAMS_INVALID)
	}
	if bet.IsNegative() {
		return failFruitDoBet(resp, services.ErrorCode_PARAMS_INVALID)
	}
	if strings.TrimSpace(req.RecordJson) == "" {
		return failFruitDoBet(resp, services.ErrorCode_PARAMS_INVALID)
	}
	win, winOK := parseAmountStrict(req.WinAmount)
	if !winOK {
		return failFruitDoBet(resp, services.ErrorCode_PARAMS_INVALID)
	}
	if win.IsNegative() {
		return failFruitDoBet(resp, services.ErrorCode_PARAMS_INVALID)
	}

	idemKey := buildIdempotencyKey("fruitDoBet", u32Str(req.UserId), u32Str(req.GameId), req.CurrencyType, req.RoundId)
	idemSig := idempotencySignature(bet.String(), win.String(), req.RecordJson)
	if hit, payload, code := d.beginIdempotency(idemKey, idemSig); code != services.ErrorCode_OK {
		return failFruitDoBet(resp, code)
	} else if hit {
		if cached, rCode := restoreFruitDoBetResp(payload); rCode != services.ErrorCode_OK {
			return failFruitDoBet(resp, rCode)
		} else {
			return cached, nil
		}
	}

	betResp, callErr := d.doSingleBet(&singleBetReq{
		UserId:       req.UserId,
		GameId:       req.GameId,
		Win:          win.String(),
		RoundID:      req.RoundId,
		Result:       req.RecordJson,
		Bet:          bet.String(),
		AgentId:      req.AgentId,
		CurrencyType: req.CurrencyType,
	})
	if callErr != nil {
		d.abortIdempotency(idemKey)
		zap.L().Error("FruitDoBet call doSingleBet failed",
			zap.Uint32("userId", req.UserId),
			zap.String("roundId", req.RoundId),
			zap.Error(callErr))
		return failFruitDoBet(resp, services.ErrorCode_SYSTEM_ERROR)
	}
	if betResp == nil {
		d.abortIdempotency(idemKey)
		return failFruitDoBet(resp, services.ErrorCode_SYSTEM_ERROR)
	}
	resp.Code = betResp.Code
	if betResp.Currency != "" {
		resp.Currency, resp.CurrencyCent = fillBalanceResp(betResp.Currency)
	}
	if betResp.Code == services.ErrorCode_OK {
		resp.Ret = true
		if raw, mErr := jsoniter.MarshalToString(resp); mErr == nil {
			d.commitIdempotency(idemKey, idemSig, raw)
		} else {
			d.abortIdempotency(idemKey)
		}
		zap.L().Debug("FruitDoBet success",
			zap.Uint32("userId", req.UserId),
			zap.Uint32("agentId", req.AgentId),
			zap.Uint32("gameId", req.GameId),
			zap.String("roundId", req.RoundId),
			zap.String("currency", resp.Currency))
	} else {
		d.abortIdempotency(idemKey)
	}
	return resp, nil
}

// FruitDoBetMulti 百人扣款门面。
// 规则 A：复用 doMultiBet（只扣余额，不改水池）。
// 每次调用都是一次独立加注；roundId 常为 mp-{gameId}-{userId}，不能按局号做完成态幂等。
func (d *LotteryService) FruitDoBetMulti(ctx context.Context, req *services.FruitDoBetMultiReq) (resp *services.FruitDoBetMultiResp, err error) {
	resp = &services.FruitDoBetMultiResp{Code: services.ErrorCode_OK, Ret: false}
	if req == nil {
		return failFruitDoBetMulti(resp, services.ErrorCode_PARAMS_INVALID)
	}
	if code, _ := validateFacadeIdentity(req.UserId, req.AgentId, req.GameId, req.RoundId, req.CurrencyType); code != services.ErrorCode_OK {
		return failFruitDoBetMulti(resp, code)
	}

	bet, betOK := parseAmountStrict(req.BetAmount)
	if !betOK {
		return failFruitDoBetMulti(resp, services.ErrorCode_PARAMS_INVALID)
	}
	if bet.IsNegative() {
		return failFruitDoBetMulti(resp, services.ErrorCode_PARAMS_INVALID)
	}

	lockKey := buildIdempotencyKey("fruitDoBetMultiLock", u32Str(req.UserId), u32Str(req.GameId), req.CurrencyType, req.RoundId, u32Str(req.AreaId), bet.String())
	if code := d.beginMultiOpLock(lockKey); code != services.ErrorCode_OK {
		return failFruitDoBetMulti(resp, code)
	}

	betResp, callErr := d.doMultiBet(&multiBetReq{
		UserId:       req.UserId,
		GameId:       req.GameId,
		RoundID:      req.RoundId,
		AreaId:       req.AreaId,
		InitBet:      bet.String(),
		AgentId:      req.AgentId,
		CurrencyType: req.CurrencyType,
	})
	d.endMultiOpLock(lockKey)
	if callErr != nil {
		zap.L().Error("FruitDoBetMulti call doMultiBet failed",
			zap.Uint32("userId", req.UserId),
			zap.String("roundId", req.RoundId),
			zap.Error(callErr))
		return failFruitDoBetMulti(resp, services.ErrorCode_SYSTEM_ERROR)
	}
	if betResp == nil {
		return failFruitDoBetMulti(resp, services.ErrorCode_SYSTEM_ERROR)
	}
	// 在 deductBet 失败时可能仍返回 OK 且 currency 为空，门面按失败处理。
	if betResp.Code != services.ErrorCode_OK {
		resp.Code = betResp.Code
		return resp, nil
	}
	if strings.TrimSpace(betResp.Currency) == "" {
		zap.L().Error("FruitDoBetMulti empty currency after bet",
			zap.Uint32("userId", req.UserId),
			zap.String("roundId", req.RoundId),
			zap.String("bet", bet.String()))
		return failFruitDoBetMulti(resp, services.ErrorCode_SYSTEM_ERROR)
	}

	resp.Ret = true
	resp.Currency, resp.CurrencyCent = fillBalanceResp(betResp.Currency)
	zap.L().Debug("FruitDoBetMulti success",
		zap.Uint32("userId", req.UserId),
		zap.Uint32("agentId", req.AgentId),
		zap.Uint32("gameId", req.GameId),
		zap.String("roundId", req.RoundId),
		zap.String("bet", bet.String()),
		zap.String("currency", resp.Currency))
	return resp, nil
}

// FruitRefundMulti 百人退款门面。
// 规则 A：复用 doMultiRefund（只退余额，不改水池）。
// 与加注一样按次独立退款，不做 roundId 完成态幂等。
func (d *LotteryService) FruitRefundMulti(ctx context.Context, req *services.FruitRefundMultiReq) (resp *services.FruitRefundMultiResp, err error) {
	resp = &services.FruitRefundMultiResp{Code: services.ErrorCode_OK, Ret: false}
	if req == nil {
		return failFruitRefundMulti(resp, services.ErrorCode_PARAMS_INVALID)
	}
	if code, _ := validateFacadeIdentity(req.UserId, req.AgentId, req.GameId, req.RoundId, req.CurrencyType); code != services.ErrorCode_OK {
		return failFruitRefundMulti(resp, code)
	}

	bet, betOK := parseAmountStrict(req.BetAmount)
	if !betOK {
		return failFruitRefundMulti(resp, services.ErrorCode_PARAMS_INVALID)
	}
	if bet.IsNegative() {
		return failFruitRefundMulti(resp, services.ErrorCode_PARAMS_INVALID)
	}

	lockKey := buildIdempotencyKey("fruitRefundMultiLock", u32Str(req.UserId), u32Str(req.GameId), req.CurrencyType, req.RoundId, bet.String())
	if code := d.beginMultiOpLock(lockKey); code != services.ErrorCode_OK {
		return failFruitRefundMulti(resp, code)
	}

	betResp, callErr := d.doMultiRefund(&multiRefundReq{
		UserId:       req.UserId,
		GameId:       req.GameId,
		Bet:          bet.String(),
		RoundID:      req.RoundId,
		AgentId:      req.AgentId,
		CurrencyType: req.CurrencyType,
	})
	d.endMultiOpLock(lockKey)
	if callErr != nil {
		zap.L().Error("FruitRefundMulti call doMultiRefund failed",
			zap.Uint32("userId", req.UserId),
			zap.String("roundId", req.RoundId),
			zap.Error(callErr))
		return failFruitRefundMulti(resp, services.ErrorCode_SYSTEM_ERROR)
	}
	if betResp == nil {
		return failFruitRefundMulti(resp, services.ErrorCode_SYSTEM_ERROR)
	}
	if betResp.Code != services.ErrorCode_OK {
		resp.Code = betResp.Code
		return resp, nil
	}
	if strings.TrimSpace(betResp.Currency) == "" {
		zap.L().Error("FruitRefundMulti empty currency after refund",
			zap.Uint32("userId", req.UserId),
			zap.String("roundId", req.RoundId),
			zap.String("bet", bet.String()))
		return failFruitRefundMulti(resp, services.ErrorCode_SYSTEM_ERROR)
	}

	resp.Ret = true
	resp.Currency, resp.CurrencyCent = fillBalanceResp(betResp.Currency)
	zap.L().Debug("FruitRefundMulti success",
		zap.Uint32("userId", req.UserId),
		zap.Uint32("agentId", req.AgentId),
		zap.Uint32("gameId", req.GameId),
		zap.String("roundId", req.RoundId),
		zap.String("bet", bet.String()),
		zap.String("currency", resp.Currency))
	return resp, nil
}

// fruitSettleRecordID 只解析百人玩家记录：其注单号独立于房间 roundId，不能复用单人结算的相等校验。
func fruitSettleRecordID(req *services.FruitSettleRoundReq, player *services.FruitSettlePlayer) (string, bool) {
	var record struct {
		RecordID       string  `json:"recordId"`
		UserId         uint32  `json:"userId"`
		AgentId        *uint32 `json:"agentId"`
		GameId         uint32  `json:"gameId"`
		CurrencySymbol string  `json:"currencySymbol"`
	}
	if err := jsoniter.UnmarshalFromString(player.RecordJson, &record); err != nil {
		return "", false
	}
	if strings.TrimSpace(record.RecordID) == "" || record.RecordID != strings.TrimSpace(record.RecordID) ||
		record.UserId != player.UserId || record.GameId != req.GameId || record.CurrencySymbol != req.CurrencyType ||
		(record.AgentId != nil && *record.AgentId != req.AgentId) {
		return "", false
	}
	return record.RecordID, true
}

// FruitSettleRound 百人整局结算门面。
// 规则 A：复用 doMultiSettle（只入账 winAmount；totalBet 只用于记录）。
func (d *LotteryService) FruitSettleRound(ctx context.Context, req *services.FruitSettleRoundReq) (resp *services.FruitSettleRoundResp, err error) {
	resp = &services.FruitSettleRoundResp{
		Code:    services.ErrorCode_OK,
		Ret:     false,
		Players: make([]*services.FruitSettlePlayerResult, 0),
	}
	if req == nil {
		return failFruitSettleRound(resp, services.ErrorCode_PARAMS_INVALID)
	}
	if code, _ := validateFacadeRoom(req.AgentId, req.GameId, req.RoundId, req.CurrencyType); code != services.ErrorCode_OK {
		return failFruitSettleRound(resp, code)
	}
	if req.Period < 0 {
		return failFruitSettleRound(resp, services.ErrorCode_PARAMS_INVALID)
	}

	records := make([]*settleRecord, 0, len(req.Players))
	totalWin := decimal.Zero
	seen := make(map[uint32]struct{}, len(req.Players))
	seenRecords := make(map[string]struct{}, len(req.Players))
	for _, p := range req.Players {
		if p == nil || p.UserId == 0 {
			return failFruitSettleRound(resp, services.ErrorCode_PARAMS_INVALID)
		}
		if _, dup := seen[p.UserId]; dup {
			zap.L().Error("FruitSettleRound duplicate user",
				zap.Uint32("userId", p.UserId),
				zap.String("roundId", req.RoundId))
			return failFruitSettleRound(resp, services.ErrorCode_PARAMS_INVALID)
		}
		seen[p.UserId] = struct{}{}

		win, winOK := parseAmountStrict(p.WinAmount)
		if !winOK || win.IsNegative() {
			return failFruitSettleRound(resp, services.ErrorCode_PARAMS_INVALID)
		}
		bet, betOK := parseAmountStrict(p.TotalBet)
		if !betOK || bet.IsNegative() {
			return failFruitSettleRound(resp, services.ErrorCode_PARAMS_INVALID)
		}
		recordId, recordOK := fruitSettleRecordID(req, p)
		if !recordOK {
			return failFruitSettleRound(resp, services.ErrorCode_PARAMS_INVALID)
		}
		if _, duplicate := seenRecords[recordId]; duplicate {
			return failFruitSettleRound(resp, services.ErrorCode_PARAMS_INVALID)
		}
		seenRecords[recordId] = struct{}{}
		totalWin = totalWin.Add(win)

		records = append(records, &settleRecord{
			UserId:       p.UserId,
			GameId:       req.GameId,
			Win:          win.String(),
			RecordID:     recordId,
			Log:          p.RecordJson,
			Bet:          bet.String(),
			CurrencyType: req.CurrencyType,
			AgentId:      req.AgentId,
			Account:      p.Account,
		})
	}

	idemKey := buildIdempotencyKey("fruitSettleRound", u32Str(req.AgentId), u32Str(req.GameId), req.CurrencyType, i64Str(req.Period), req.RoundId)
	signaturePlayers := append([]*services.FruitSettlePlayer(nil), req.Players...)
	sort.Slice(signaturePlayers, func(left, right int) bool {
		return signaturePlayers[left].UserId < signaturePlayers[right].UserId
	})
	sigParts := make([]string, 0, len(records)*6+1)
	sigParts = append(sigParts, totalWin.String())
	for _, player := range signaturePlayers {
		sigParts = append(sigParts, u32Str(player.UserId), player.UserName, i64Str(int64(player.UserIcon)),
			parseAmount(player.TotalBet).String(), parseAmount(player.WinAmount).String(), player.RecordJson)
	}
	idemSig := idempotencySignature(sigParts...)
	if hit, payload, code := d.beginIdempotency(idemKey, idemSig); code != services.ErrorCode_OK {
		return failFruitSettleRound(resp, code)
	} else if hit {
		if cached, rCode := restoreFruitSettleRoundResp(payload); rCode != services.ErrorCode_OK {
			return failFruitSettleRound(resp, rCode)
		} else {
			return cached, nil
		}
	}

	betResp, callErr := d.doMultiSettle(&multiSettleReq{
		Records:  records,
		TotalWin: totalWin.String(),
	})
	if callErr != nil {
		d.abortIdempotency(idemKey)
		zap.L().Error("FruitSettleRound call doMultiSettle failed",
			zap.Uint32("agentId", req.AgentId),
			zap.Uint32("gameId", req.GameId),
			zap.String("roundId", req.RoundId),
			zap.Error(callErr))
		return failFruitSettleRound(resp, services.ErrorCode_SYSTEM_ERROR)
	}
	if betResp == nil {
		d.abortIdempotency(idemKey)
		return failFruitSettleRound(resp, services.ErrorCode_SYSTEM_ERROR)
	}
	resp.Code = betResp.Code
	if betResp.Code != services.ErrorCode_OK {
		d.abortIdempotency(idemKey)
		return resp, nil
	}

	for _, item := range betResp.Currencys {
		if item == nil {
			continue
		}
		currency, cent := fillBalanceResp(item.Currency)
		resp.Players = append(resp.Players, &services.FruitSettlePlayerResult{
			UserId:       item.UserId,
			Currency:     currency,
			CurrencyCent: cent,
		})
	}
	resp.Ret = true
	if raw, mErr := jsoniter.MarshalToString(resp); mErr == nil {
		d.commitIdempotency(idemKey, idemSig, raw)
	} else {
		d.abortIdempotency(idemKey)
	}
	zap.L().Debug("FruitSettleRound success",
		zap.Uint32("agentId", req.AgentId),
		zap.Uint32("gameId", req.GameId),
		zap.String("roundId", req.RoundId),
		zap.Int64("period", req.Period),
		zap.Int("playerCount", len(resp.Players)),
		zap.String("totalWin", totalWin.String()))
	return resp, nil
}
