package rpc

import (
	"context"
	"strings"

	"app/config"
	"app/tables/manager"
	"lottery/dao"

	jsoniter "github.com/json-iterator/go"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"micro_service/services"
)

type ug5SpinRecord struct {
	RecordId string `json:"recordId"`
	Cents    struct {
		BetCents       string `json:"betCents"`
		NormalWinCents string `json:"normalWinCents"`
	} `json:"cents"`
	ListItem struct {
		AllBets  any `json:"all_bets"`
		AllBonus any `json:"all_bonus"`
	} `json:"listItem"`
}

func failUg5Guess(resp *services.Ug5GuessResp, code services.ErrorCode, errName string) (*services.Ug5GuessResp, error) {
	resp.Code = code
	resp.Ret = false
	if errName != "" {
		resp.Error = errName
	} else {
		resp.Error = slotErrorName(code)
	}
	return resp, nil
}

func centsToYuanAmount(v string) (decimal.Decimal, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return decimal.Zero, false
	}
	d, err := decimal.NewFromString(v)
	if err != nil {
		return decimal.Zero, false
	}
	return d.Div(decimal.NewFromInt(100)), true
}

func anyToAmount(v any) (decimal.Decimal, bool) {
	if v == nil {
		return decimal.Zero, false
	}
	switch t := v.(type) {
	case string:
		return parseAmountStrict(t)
	case float64:
		return decimal.NewFromFloat(t), true
	case jsoniter.Number:
		return parseAmountStrict(t.String())
	default:
		return parseAmountStrict(jsoniter.Wrap(v).ToString())
	}
}

func parseUg5ChainRecord(recordJSON, roundId string) (bet, win decimal.Decimal, code services.ErrorCode, msg string) {
	raw := strings.TrimSpace(recordJSON)
	if raw == "" {
		return decimal.Zero, decimal.Zero, services.ErrorCode_PARAMS_INVALID, "complete record required"
	}
	rec := &ug5SpinRecord{}
	if err := jsoniter.UnmarshalFromString(raw, rec); err != nil {
		return decimal.Zero, decimal.Zero, services.ErrorCode_PARAMS_INVALID, "invalid recordJson"
	}
	if strings.TrimSpace(rec.RecordId) != strings.TrimSpace(roundId) {
		return decimal.Zero, decimal.Zero, services.ErrorCode_PARAMS_INVALID, "recordId must match roundId"
	}
	var ok bool
	if bet, ok = centsToYuanAmount(rec.Cents.BetCents); !ok {
		if bet, ok = anyToAmount(rec.ListItem.AllBets); !ok {
			return decimal.Zero, decimal.Zero, services.ErrorCode_PARAMS_INVALID, "record original bet missing"
		}
	}
	if win, ok = centsToYuanAmount(rec.Cents.NormalWinCents); !ok {
		if win, ok = anyToAmount(rec.ListItem.AllBonus); !ok {
			return decimal.Zero, decimal.Zero, services.ErrorCode_PARAMS_INVALID, "record final win missing"
		}
	}
	if bet.IsNegative() || win.IsNegative() {
		return decimal.Zero, decimal.Zero, services.ErrorCode_PARAMS_INVALID, "record amounts can not be negative"
	}
	return bet, win, services.ErrorCode_OK, ""
}

// Ug5Guess UG5 竞猜增量结算。
// 不消费免费预扣。中间步只扣派+改水池+过程态；complete 才写整链注单，统计取记录里的原始本金和最终奖金。
func (d *LotteryService) Ug5Guess(ctx context.Context, req *services.Ug5GuessReq) (resp *services.Ug5GuessResp, err error) {
	resp = &services.Ug5GuessResp{Code: services.ErrorCode_OK, Ret: false}
	if req == nil {
		return failUg5Guess(resp, services.ErrorCode_PARAMS_INVALID, "")
	}
	if code, msg := validateFacadeIdentity(req.UserId, req.AgentId, req.GameId, req.RoundId, req.CurrencyType); code != services.ErrorCode_OK {
		return failUg5Guess(resp, code, msg)
	}
	if msg, ok := validateGameStorageItems(req.GameStorage); !ok {
		return failUg5Guess(resp, services.ErrorCode_PARAMS_INVALID, msg)
	}
	for _, key := range req.DeleteGameStorageKeys {
		if strings.TrimSpace(key) == "" {
			return failUg5Guess(resp, services.ErrorCode_PARAMS_INVALID, "deleteGameStorageKeys contains empty key")
		}
	}

	bet, betOK := parseAmountStrict(req.BetAmount)
	if !betOK {
		return failUg5Guess(resp, services.ErrorCode_PARAMS_INVALID, "invalid betAmount")
	}
	if bet.IsNegative() {
		return failUg5Guess(resp, services.ErrorCode_PARAMS_INVALID, "betAmount can not be negative")
	}
	win, winOK := parseAmountStrict(req.WinAmount)
	if !winOK {
		return failUg5Guess(resp, services.ErrorCode_PARAMS_INVALID, "invalid winAmount")
	}
	if win.IsNegative() {
		return failUg5Guess(resp, services.ErrorCode_PARAMS_INVALID, "winAmount can not be negative")
	}

	recordJSON := strings.TrimSpace(req.RecordJson)
	var chainBet, chainWin decimal.Decimal
	if req.Complete {
		if recordJSON == "" {
			return failUg5Guess(resp, services.ErrorCode_PARAMS_INVALID, "complete record required")
		}
		var code services.ErrorCode
		var msg string
		chainBet, chainWin, code, msg = parseUg5ChainRecord(recordJSON, req.RoundId)
		if code != services.ErrorCode_OK {
			return failUg5Guess(resp, code, msg)
		}
	} else if recordJSON != "" {
		return failUg5Guess(resp, services.ErrorCode_PARAMS_INVALID, "intermediate guess must not contain record")
	}

	idemKey := ""
	idemSig := ""
	if req.Complete {
		idemKey = buildIdempotencyKey("ug5Guess", u32Str(req.UserId), u32Str(req.GameId), req.CurrencyType, req.RoundId)
		idemSig = idempotencySignature(bet.String(), win.String(), recordJSON)
		if hit, payload, code := d.beginIdempotency(idemKey, idemSig); code != services.ErrorCode_OK {
			msg := ""
			if code == services.ErrorCode_PARAMS_INVALID {
				msg = "idempotencyConflict"
			}
			return failUg5Guess(resp, code, msg)
		} else if hit {
			if cached, rCode := restoreUg5GuessResp(payload); rCode != services.ErrorCode_OK {
				return failUg5Guess(resp, rCode, "")
			} else {
				return cached, nil
			}
		}
	}

	lockKey := buildIdempotencyKey("ug5GuessLock", u32Str(req.UserId), u32Str(req.GameId), req.CurrencyType, req.RoundId, bet.String(), win.String())
	if code := d.beginMultiOpLock(lockKey); code != services.ErrorCode_OK {
		if idemKey != "" {
			d.abortIdempotency(idemKey)
		}
		return failUg5Guess(resp, code, "")
	}
	defer d.endMultiOpLock(lockKey)

	eGame := dao.GamesManagerIns().GetById(int64(req.GameId))
	eAgent := dao.AgentManagerIns().Get(int64(req.AgentId))
	if eAgent == nil && req.AgentId == 0 {
		eAgent = &manager.Agent{Id: 0, WebId: 0}
	}
	if eGame == nil || eAgent == nil {
		if idemKey != "" {
			d.abortIdempotency(idemKey)
		}
		return failUg5Guess(resp, services.ErrorCode_PARAMS_INVALID, "")
	}
	exchange, ok := config.CfgIns.GetExchange(req.CurrencyType)
	pc := config.CfgIns.GetPoolCfg(int64(req.AgentId), eGame.ConfName)
	if !ok || pc == nil {
		if idemKey != "" {
			d.abortIdempotency(idemKey)
		}
		return failUg5Guess(resp, services.ErrorCode_SYSTEM_ERROR, "exchangeConfigMissing")
	}

	balance, code := d.getPlayerCurrency(req.UserId)
	if code != services.ErrorCode_OK {
		if idemKey != "" {
			d.abortIdempotency(idemKey)
		}
		return failUg5Guess(resp, code, "")
	}
	resp.Currency, resp.CurrencyCent = fillBalanceResp(currencyFromCent(balance).String())
	betCents := bet.Mul(decimal.NewFromInt(100)).IntPart()
	winCents := win.Mul(decimal.NewFromInt(100)).IntPart()
	if balance < betCents {
		if idemKey != "" {
			d.abortIdempotency(idemKey)
		}
		return failUg5Guess(resp, services.ErrorCode_NO_ENOUGH_MONEY, "")
	}

	user := dao.CacheIns().GetUser(int64(req.AgentId), int64(req.UserId))
	if user == nil {
		if idemKey != "" {
			d.abortIdempotency(idemKey)
		}
		return failUg5Guess(resp, services.ErrorCode_PARAMS_INVALID, "")
	}

	poolChanged := false
	balanceDelta := int64(0)
	failAfterMutate := func(failCode services.ErrorCode, errName string) (*services.Ug5GuessResp, error) {
		if balanceDelta != 0 {
			if _, rollbackCode := d.updatePlayerCurrency(req.UserId, -balanceDelta); rollbackCode != services.ErrorCode_OK {
				zap.L().Error("Ug5Guess rollback balance failed", zap.Uint32("userId", req.UserId), zap.Any("code", rollbackCode))
			}
		}
		if poolChanged {
			d.rollbackPoolChange(req.AgentId, req.UserId, eGame.ConfName, req.CurrencyType, req.RoundId,
				bet.Mul(exchange), win.Mul(exchange), pc.Pool[1].Revenue)
		}
		if idemKey != "" {
			d.abortIdempotency(idemKey)
		}
		return failUg5Guess(resp, failCode, errName)
	}

	if !bet.IsZero() || !win.IsZero() {
		revenue := bet.Mul(exchange).Mul(pc.Pool[1].Revenue).Truncate(4)
		if !dao.CacheIns().CheckPoolWithChange(int64(req.AgentId), eGame.ConfName, req.RoundId, req.CurrencyType,
			win.Mul(exchange), bet.Mul(exchange), revenue, req.UserId) {
			if idemKey != "" {
				d.abortIdempotency(idemKey)
			}
			return failUg5Guess(resp, services.ErrorCode_NO_ENOUGH_POOL_MONEY, "")
		}
		poolChanged = true
	}

	if betCents > 0 {
		balance, code = d.updatePlayerCurrency(req.UserId, -betCents)
		if code != services.ErrorCode_OK {
			return failAfterMutate(code, "")
		}
		balanceDelta -= betCents
	}
	if winCents > 0 {
		balance, code = d.updatePlayerCurrency(req.UserId, winCents)
		if code != services.ErrorCode_OK {
			return failAfterMutate(code, "")
		}
		balanceDelta += winCents
	}
	newCurrency := currencyFromCent(balance)

	if user.IsTourist == 0 {
		if betCents > 0 {
			d.SaveBill(req.AgentId, req.UserId, bet.Neg(), decimal.Zero, currencyFromCent(balance-winCents).Truncate(2).InexactFloat64(),
				eGame.ConfName, "竞猜下注", req.CurrencyType, req.RoundId)
		}
		if winCents > 0 {
			d.SaveBill(req.AgentId, req.UserId, bet.Neg(), win, newCurrency.Truncate(2).InexactFloat64(),
				eGame.ConfName, "竞猜派奖", req.CurrencyType, req.RoundId)
		}
		if req.Complete {
			account := user.Account
			record := ConvertRecord(req.AgentId, req.UserId, req.RoundId, req.CurrencyType, eGame.ConfName,
				account, recordJSON, newCurrency, uint32(eAgent.WebId), chainBet.InexactFloat64(), chainWin.InexactFloat64())
			d.SaveRecord(record)
			zap.L().Info("Ug5Guess:注单信息",
				zap.Uint32("agentId", req.AgentId),
				zap.Uint32("playerId", req.UserId),
				zap.Uint32("gameId", req.GameId),
				zap.String("symbol", eGame.ConfName),
				zap.String("roundId", req.RoundId),
				zap.Any("record", record))
			dao.CacheIns().Complete(int64(req.AgentId), req.UserId, eGame.ConfName, chainBet.Mul(exchange), chainWin.Mul(exchange), pc.Pool[1].Revenue)
		}
		if poolChanged {
			d.pcr.Record(int64(req.AgentId), eGame.ConfName, dao.CacheIns().GetPool(int64(req.AgentId), eGame.ConfName))
		}
	}

	if code := d.applyGameStorageWrites(req.UserId, req.GameId, req.CurrencyType, req.GameStorage, req.DeleteGameStorageKeys); code != services.ErrorCode_OK {
		return failAfterMutate(code, "")
	}

	resp.Ret = true
	resp.Code = services.ErrorCode_OK
	resp.Error = ""
	resp.Currency, resp.CurrencyCent = fillBalanceResp(newCurrency.Truncate(2).StringFixed(2))
	if idemKey != "" {
		if raw, mErr := jsoniter.MarshalToString(resp); mErr == nil {
			d.commitIdempotency(idemKey, idemSig, raw)
		} else {
			d.abortIdempotency(idemKey)
		}
	}
	zap.L().Debug("Ug5Guess success",
		zap.Uint32("userId", req.UserId),
		zap.Uint32("agentId", req.AgentId),
		zap.Uint32("gameId", req.GameId),
		zap.String("roundId", req.RoundId),
		zap.Bool("complete", req.Complete),
		zap.String("bet", bet.String()),
		zap.String("win", win.String()),
		zap.String("currency", resp.Currency))
	return resp, nil
}
