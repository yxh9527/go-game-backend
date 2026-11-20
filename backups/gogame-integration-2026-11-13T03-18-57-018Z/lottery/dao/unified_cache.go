package dao

import (
	"app/config"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"time"
)

func (gcm *GameCacheMgr) GetUnifiedPool(agentID int64, symbol string) decimal.Decimal {
	agent := gcm.GetAgent(agentID)
	agent.lock.Lock()
	defer agent.lock.Unlock()

	game := agent.GetGame(symbol)
	// 预留独立保存在 Redis，确保同一奖池的有效预留不会只存在某个实例内存中。
	reserved := RedisIns().GetAgentReserved(agentID, symbol)
	return availablePoolValue(game, reserved)
}

func poolValue(game *Game) decimal.Decimal {
	return game.TotalEffectBet.Sub(game.TotalProfLoss).Sub(game.TotalRevenue)
}

func availablePoolValue(game *Game, reserved decimal.Decimal) decimal.Decimal {
	return poolValue(game).Sub(reserved)
}

func poolType(pool decimal.Decimal, item *config.PoolItem) (int, decimal.Decimal) {
	if item == nil {
		return config.POOL_BREAK_DOWN, decimal.Zero
	}
	if pool.LessThanOrEqual(decimal.Zero) {
		return config.POOL_BREAK_DOWN, decimal.Zero
	}
	if pool.LessThanOrEqual(item.Normal) {
		return config.POOL_LOW, item.MinRate
	}
	if pool.LessThanOrEqual(item.Max) {
		return config.POOL_NORMAL, item.NormalRate
	}
	return config.POOL_HIGH, item.MaxRate
}

func (gcm *GameCacheMgr) CanAffordAward(
	agentID int64,
	userID uint32,
	item *config.PoolItem,
	symbol string,
	level uint32,
	award decimal.Decimal,
) bool {
	if !award.IsPositive() {
		return true
	}
	if item == nil || config.CfgIns == nil {
		zap.L().Debug("CanAffordAward:缺少水池或开奖配置",
			zap.Int64("agentId", agentID),
			zap.Uint32("userId", userID),
			zap.String("symbol", symbol),
			zap.Uint32("level", level),
			zap.String("award", award.String()),
		)
		return false
	}

	agent := gcm.GetAgent(agentID)
	agent.lock.Lock()
	defer agent.lock.Unlock()

	user := agent.GetUser(userID)
	game := agent.GetGame(symbol)
	pool := poolValue(game).Add(item.Base)

	t, r := poolType(pool, item)
	if t == config.POOL_BREAK_DOWN {
		zap.L().Debug("CanAffordAward:水池破产",
			zap.Int64("agentId", agentID),
			zap.Uint32("userId", userID),
			zap.String("symbol", symbol),
			zap.Uint32("level", level),
			zap.String("pool", pool.String()),
			zap.String("award", award.String()),
		)
		return false
	}

	rate := decimal.Zero
	if !user.TotalEffectBet.IsZero() {
		rate = user.TotalProfLoss.Div(user.TotalEffectBet)
	}
	cItems := config.CfgIns.GetAwardOddsConfigWithProfitOdds(symbol, rate)
	if cItems == nil {
		cItems = config.CfgIns.GetAwardOddsDefaultCtrlConfig(symbol)
	}
	if cItems == nil || t < 0 || t >= len(cItems.PoolOdds) || cItems.PoolOdds[t] == nil {
		zap.L().Debug("CanAffordAward:缺少水位开奖配置",
			zap.Int64("agentId", agentID),
			zap.Uint32("userId", userID),
			zap.String("symbol", symbol),
			zap.Uint32("level", level),
			zap.Int("poolType", t),
			zap.String("rate", rate.String()),
		)
		return false
	}
	oddsItem := cItems.PoolOdds[t]

	p1 := pool.Mul(r)
	cnt := decimal.NewFromInt(1)
	if !user.Count.IsZero() {
		cnt = user.Count.Add(decimal.NewFromInt(1))
	}
	p2 := user.TotalEffectBet.Div(cnt).Mul(oddsItem.M)
	p := p1
	if p1.GreaterThan(p2) {
		p = p2
		zap.L().Debug("CanAffordAward:使用平均值",
			zap.Int64("agentId", agentID),
			zap.Uint32("userId", userID),
			zap.String("symbol", symbol),
			zap.Uint32("level", level),
			zap.String("payable", p.String()),
			zap.String("award", award.String()),
			zap.String("totalEffectBet", user.TotalEffectBet.String()),
			zap.String("count", user.Count.String()),
			zap.Any("oddsItem", oddsItem),
		)
	} else {
		zap.L().Debug("CanAffordAward:使用水池值",
			zap.Int64("agentId", agentID),
			zap.Uint32("userId", userID),
			zap.String("symbol", symbol),
			zap.Uint32("level", level),
			zap.String("payable", p.String()),
			zap.String("award", award.String()),
			zap.String("pool", pool.String()),
			zap.String("poolRate", r.String()),
			zap.Any("oddsItem", oddsItem),
		)
	}

	if p.LessThan(award) {
		zap.L().Debug("CanAffordAward:赔付失败",
			zap.Int64("agentId", agentID),
			zap.Uint32("userId", userID),
			zap.String("symbol", symbol),
			zap.Uint32("level", level),
			zap.String("payable", p.String()),
			zap.String("award", award.String()),
		)
		return false
	}
	zap.L().Debug("CanAffordAward:赔付成功",
		zap.Int64("agentId", agentID),
		zap.Uint32("userId", userID),
		zap.String("symbol", symbol),
		zap.Uint32("level", level),
		zap.String("payable", p.String()),
		zap.String("award", award.String()),
	)
	return true
}

func (gcm *GameCacheMgr) ApplyPoolChange(
	agentID int64,
	userID uint32,
	symbol string,
	currencyType string,
	recordID string,
	betDelta decimal.Decimal,
	payoutDelta decimal.Decimal,
	revenueDelta decimal.Decimal,
) {
	agent := gcm.GetAgent(agentID)
	agent.lock.Lock()
	defer agent.lock.Unlock()

	user := agent.GetUser(userID)
	if user.IsTourist != 0 {
		return
	}

	game := agent.GetGame(symbol)
	before := poolValue(game)
	game.TotalEffectBet = game.TotalEffectBet.Add(betDelta.Truncate(4))
	// TotalProfLoss 只累计实际赔付，不混入下注或玩家净利润。
	game.TotalProfLoss = game.TotalProfLoss.Add(payoutDelta.Truncate(4))
	game.TotalRevenue = game.TotalRevenue.Add(revenueDelta.Truncate(4))
	game.UpdateTime = time.Now().Unix()

	user.TotalEffectBet = user.TotalEffectBet.Add(betDelta.Truncate(4))
	user.UpdateTime = time.Now().Unix()
	after := poolValue(game)
	zap.L().Debug("pool change",
		zap.Int64("agentId", agentID),
		zap.Uint32("userId", userID),
		zap.String("symbol", symbol),
		zap.String("currencyType", currencyType),
		zap.String("recordId", recordID),
		zap.String("betDelta", betDelta.String()),
		zap.String("payoutDelta", payoutDelta.String()),
		zap.String("revenueDelta", revenueDelta.String()),
		zap.String("before", before.String()),
		zap.String("after", after.String()),
	)
}

func (gcm *GameCacheMgr) TryReservePool(agentID int64, symbol string, amount decimal.Decimal) (bool, error) {
	agent := gcm.GetAgent(agentID)
	agent.lock.Lock()
	defer agent.lock.Unlock()

	basePool := poolValue(agent.GetGame(symbol))
	return RedisIns().TryReserveAgentPool(agentID, symbol, basePool, amount)
}

func (gcm *GameCacheMgr) TryReservePoolWithLedger(
	agentID int64,
	symbol string,
	amount decimal.Decimal,
	reservationID, roundID string,
) (ReserveWithLedgerResult, error) {
	agent := gcm.GetAgent(agentID)
	agent.lock.Lock()
	defer agent.lock.Unlock()

	basePool := poolValue(agent.GetGame(symbol))
	return RedisIns().TryReserveAgentPoolWithLedger(
		agentID, symbol, basePool, amount, reservationID, roundID,
	)
}

func (gcm *GameCacheMgr) ReleasePoolReservation(agentID int64, symbol string, amount decimal.Decimal) bool {
	released, err := RedisIns().ReleaseAgentReserved(agentID, symbol, amount)
	if err != nil {
		zap.L().Error("释放奖池有效预留失败", zap.Int64("agentId", agentID), zap.String("symbol", symbol), zap.String("amount", amount.String()), zap.Error(err))
		return false
	}
	if !released {
		zap.L().Error("奖池有效预留不足，拒绝重复释放", zap.Int64("agentId", agentID), zap.String("symbol", symbol), zap.String("amount", amount.String()))
	}
	return released
}

func (gcm *GameCacheMgr) ReleasePoolReservationWithLedger(
	agentID int64,
	symbol string,
	amount decimal.Decimal,
	reservationID, roundID, releaseID, reason string,
	allowMigrate bool,
) ReleaseWithLedgerResult {
	qBefore := RedisIns().GetAgentReserved(agentID, symbol)
	result, err := RedisIns().ReleaseAgentReservedWithLedger(
		agentID, symbol, amount, reservationID, roundID, releaseID, reason, allowMigrate,
	)
	if err != nil {
		zap.L().Error("预扣回退水池失败",
			zap.Int64("agentId", agentID),
			zap.String("poolSymbol", symbol),
			zap.String("amountCny", amount.String()),
			zap.String("reservationId", reservationID),
			zap.String("roundId", roundID),
			zap.String("releaseId", releaseID),
			zap.String("reason", reason),
			zap.String("qBefore", qBefore.String()),
			zap.Error(err),
		)
		return ReleaseWithLedgerResult{}
	}
	if !result.Ok {
		zap.L().Error("预扣回退水池被拒绝",
			zap.Int64("agentId", agentID),
			zap.String("poolSymbol", symbol),
			zap.String("amountCny", amount.String()),
			zap.String("reservationId", reservationID),
			zap.String("roundId", roundID),
			zap.String("releaseId", releaseID),
			zap.String("reason", reason),
			zap.String("qBefore", qBefore.String()),
		)
		return result
	}
	qAfter := RedisIns().GetAgentReserved(agentID, symbol)
	if result.AlreadyReleased {
		zap.L().Info("预扣回退水池幂等跳过(ledger已RELEASED)",
			zap.Int64("agentId", agentID),
			zap.String("poolSymbol", symbol),
			zap.String("amountCny", amount.String()),
			zap.String("reservationId", reservationID),
			zap.String("roundId", roundID),
			zap.String("releaseId", releaseID),
			zap.String("reason", reason),
			zap.String("qBefore", qBefore.String()),
			zap.String("qAfter", qAfter.String()),
		)
		return result
	}
	zap.L().Info("预扣回退水池成功",
		zap.Int64("agentId", agentID),
		zap.String("poolSymbol", symbol),
		zap.String("amountCny", amount.String()),
		zap.String("reservationId", reservationID),
		zap.String("roundId", roundID),
		zap.String("releaseId", releaseID),
		zap.String("reason", reason),
		zap.String("qBefore", qBefore.String()),
		zap.String("qAfter", qAfter.String()),
	)
	return result
}

func (gcm *GameCacheMgr) AdjustPoolReservationWithLedger(
	agentID int64,
	symbol string,
	oldAmount, newAmount decimal.Decimal,
	reservationID, roundID string,
) (bool, error) {
	agent := gcm.GetAgent(agentID)
	agent.lock.Lock()
	defer agent.lock.Unlock()

	basePool := poolValue(agent.GetGame(symbol))
	qBefore := RedisIns().GetAgentReserved(agentID, symbol)
	ok, err := RedisIns().AdjustAgentReservedWithLedger(
		agentID, symbol, basePool, oldAmount, newAmount, reservationID, roundID,
	)
	if err != nil {
		zap.L().Error("预扣额度调整失败",
			zap.Int64("agentId", agentID),
			zap.String("poolSymbol", symbol),
			zap.String("oldAmountCny", oldAmount.String()),
			zap.String("newAmountCny", newAmount.String()),
			zap.String("reservationId", reservationID),
			zap.String("roundId", roundID),
			zap.String("qBefore", qBefore.String()),
			zap.Error(err),
		)
		return false, err
	}
	if !ok {
		// ledger 缺失：补建 ACTIVE（不改 Q，旧预留已在汇总里），再调一次。
		if seedErr := RedisIns().SeedReservationLedgerActive(
			agentID, symbol, oldAmount, reservationID, roundID,
		); seedErr != nil {
			zap.L().Error("预扣 ledger 补建失败",
				zap.Int64("agentId", agentID),
				zap.String("poolSymbol", symbol),
				zap.String("reservationId", reservationID),
				zap.String("roundId", roundID),
				zap.Error(seedErr),
			)
			return false, seedErr
		}
		zap.L().Info("预扣 ledger 缺失已补建ACTIVE",
			zap.Int64("agentId", agentID),
			zap.String("poolSymbol", symbol),
			zap.String("amountCny", oldAmount.String()),
			zap.String("reservationId", reservationID),
			zap.String("roundId", roundID),
		)
		ok, err = RedisIns().AdjustAgentReservedWithLedger(
			agentID, symbol, basePool, oldAmount, newAmount, reservationID, roundID,
		)
		if err != nil {
			return false, err
		}
	}
	if !ok {
		zap.L().Error("预扣额度调整被拒绝",
			zap.Int64("agentId", agentID),
			zap.String("poolSymbol", symbol),
			zap.String("oldAmountCny", oldAmount.String()),
			zap.String("newAmountCny", newAmount.String()),
			zap.String("reservationId", reservationID),
			zap.String("roundId", roundID),
			zap.String("qBefore", qBefore.String()),
		)
		return false, nil
	}
	qAfter := RedisIns().GetAgentReserved(agentID, symbol)
	delta := newAmount.Sub(oldAmount)
	if delta.IsNegative() {
		zap.L().Info("预扣差额回退水池成功",
			zap.Int64("agentId", agentID),
			zap.String("poolSymbol", symbol),
			zap.String("oldAmountCny", oldAmount.String()),
			zap.String("newAmountCny", newAmount.String()),
			zap.String("releasedCny", delta.Abs().String()),
			zap.String("reservationId", reservationID),
			zap.String("roundId", roundID),
			zap.String("qBefore", qBefore.String()),
			zap.String("qAfter", qAfter.String()),
		)
	} else if delta.IsPositive() {
		zap.L().Info("预扣额度上调成功",
			zap.Int64("agentId", agentID),
			zap.String("poolSymbol", symbol),
			zap.String("oldAmountCny", oldAmount.String()),
			zap.String("newAmountCny", newAmount.String()),
			zap.String("addedCny", delta.String()),
			zap.String("reservationId", reservationID),
			zap.String("roundId", roundID),
			zap.String("qBefore", qBefore.String()),
			zap.String("qAfter", qAfter.String()),
		)
	}
	return true, nil
}

func (gcm *GameCacheMgr) RecordSettlement(agentID int64, userID uint32, symbol string, bet, payout decimal.Decimal) {
	agent := gcm.GetAgent(agentID)
	agent.lock.Lock()
	defer agent.lock.Unlock()

	user := agent.GetUser(userID)
	if user.IsTourist != 0 {
		return
	}

	game := agent.GetGame(symbol)
	chips := payout
	if payout.LessThan(bet) {
		chips = bet
	}
	game.TotalChips = game.TotalChips.Add(chips.Truncate(4))
	game.UpdateTime = time.Now().Unix()
	user.Count = user.Count.Add(decimal.NewFromInt(1))
	user.TotalProfLoss = user.TotalProfLoss.Add(payout.Truncate(4))
	user.UpdateTime = time.Now().Unix()
}
