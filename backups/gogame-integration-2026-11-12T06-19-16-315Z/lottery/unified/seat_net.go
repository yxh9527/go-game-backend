package unified

import (
	"app/config"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/shopspring/decimal"
	"lottery/dao"
	"micro_service/services"
	"strconv"
	"time"
)

const financeModeSeatNet = "SEAT_NET"

func (d *LotteryService) settleSeatNetLocked(round *financeRound, runtime *roundRuntime, req *services.SettlementRequest) *services.SettlementResponse {
	resp := &services.SettlementResponse{Code: services.ErrorCode_PARAMS_INVALID, RoundId: req.RoundId, SettlementId: req.SettlementId}
	fingerprint := settlementRequestHash(req)
	if round.Settlement != nil {
		if round.Settlement.SettlementID != req.SettlementId || round.Settlement.RequestHash != fingerprint || round.Settlement.Mode != financeModeSeatNet {
			return resp
		}
		if round.Settlement.Status == financeSettlementStatusCompleted {
			resp.Code = services.ErrorCode_OK
			resp.State = round.State
			for _, result := range round.Settlement.Results {
				resp.Items = append(resp.Items, d.buildSettlementResponse(result))
			}
			return resp
		}
		return resp
	}
	if d.rds == nil || round.State != string(financeStateBetting) || round.Reservation != nil || req.ReservationId != "" || req.OutcomeHash == "" || req.BetDigest != d.roundBetDigest(round) || !validSettlementCarries(round, req) {
		return resp
	}
	active := map[string]*services.FinanceCarryBinding{}
	for _, bet := range round.Bets {
		if isActiveFinanceBet(bet) {
			amount, err := parseMoney(bet.Amount)
			if err != nil || !amount.IsZero() || bet.Carry == nil {
				return resp
			}
			active[fmt.Sprintf("%d|%s", bet.UserID, bet.CurrencyType)] = bet.Carry
		}
	}
	if len(req.Items) != len(active) || len(active) == 0 || len(active) > 16 {
		return resp
	}
	batch := dao.SeatNetBatch{ID: req.SettlementId}
	seen := map[string]bool{}
	type economic struct {
		item                             *services.SettlementItem
		effective, payout, net, exchange decimal.Decimal
	}
	amounts := make([]economic, 0, len(req.Items))
	totalNet := decimal.Zero
	terminalRaw, _ := jsoniter.MarshalToString(round)
	terminal := &financeRound{}
	if jsoniter.UnmarshalFromString(terminalRaw, terminal) != nil {
		return resp
	}
	terminal.State = string(financeStateSettled)
	terminal.UpdatedAt = time.Now().Unix()
	terminal.Settlement = &financeSettlement{SettlementID: req.SettlementId, BetDigest: req.BetDigest, OutcomeHash: req.OutcomeHash, Mode: financeModeSeatNet, Status: financeSettlementStatusCompleted, RequestHash: fingerprint, CompletedAt: time.Now().Unix()}
	for i, item := range req.Items {
		if item == nil || item.Carry == nil {
			return resp
		}
		key := fmt.Sprintf("%d|%s", item.UserId, item.CurrencyType)
		binding := active[key]
		if seen[key] || binding == nil || binding.Token != item.Carry.Token || binding.TableKey != item.Carry.TableKey {
			return resp
		}
		seen[key] = true
		net, err := parseSignedMoney(item.NetScore)
		if err != nil {
			return resp
		}
		effective, err := parseMoney(item.BetAmount)
		if err != nil || !effective.Equal(net.Abs()) {
			return resp
		}
		profit, err := parseSignedMoney(item.Profit)
		if err != nil || !profit.Equal(net) {
			return resp
		}
		payout, err := parseMoney(item.Payout)
		if err != nil || !payout.Equal(effective.Add(net)) {
			return resp
		}
		pump, err := parseMoney(item.Pump)
		if err != nil || pump.IsNegative() {
			return resp
		}
		exchange, ok := config.CfgIns.GetExchange(item.CurrencyType)
		if !ok || !exchange.IsPositive() {
			resp.Code = services.ErrorCode_SYSTEM_ERROR
			return resp
		}
		revisionRaw, err := d.rds.HGet(fmt.Sprintf("player_%d", item.UserId), "cash_revision")
		if err != nil {
			resp.Code = services.ErrorCode_SYSTEM_ERROR
			return resp
		}
		revision, err := strconv.ParseInt(revisionRaw, 10, 64)
		if err != nil || revision < 0 {
			return resp
		}
		batch.Accounts = append(batch.Accounts, dao.SeatNetAccount{UserID: item.UserId, AgentID: req.Agent, Currency: item.CurrencyType, Revision: revision})
		batch.Steps = append(batch.Steps, dao.SeatNetStep{ID: fmt.Sprintf("%s:%d", req.SettlementId, item.UserId), Account: i + 1, Kind: "carryDelta", Amount: toCentDelta(net), Token: item.Carry.Token, TableKey: item.Carry.TableKey, Revision: item.Carry.Revision})
		terminal.Settlement.Results = append(terminal.Settlement.Results, &financeSettlementResult{UserID: item.UserId, Code: int32(services.ErrorCode_OK), Currency: "0.00"})
		totalNet = totalNet.Add(net.Mul(exchange).Truncate(4))
		amounts = append(amounts, economic{item, effective, payout, net, exchange})
	}
	poolItem := config.CfgIns.GetPoolItem(int64(req.Agent), runtime.Symbol, runtime.Level)
	if poolItem == nil {
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		return resp
	}
	if !d.testSkipPoolSideEffects {
		funded, err := dao.CacheIns().CanFundSeatNet(int64(req.Agent), runtime.PoolSymbol, poolItem.Base, totalNet)
		if err != nil {
			resp.Code = services.ErrorCode_SYSTEM_ERROR
			return resp
		}
		if !funded {
			resp.Code = services.ErrorCode_NO_ENOUGH_POOL_MONEY
			return resp
		}
	}
	for _, bet := range terminal.Bets {
		if isActiveFinanceBet(bet) {
			bet.Status = financeBetStatusSettled
		}
	}
	raw, err := jsoniter.MarshalToString(terminal)
	if err != nil {
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		return resp
	}
	before, err := d.rds.HGet(financeRoundHash, req.RoundId)
	if err != nil {
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		return resp
	}
	receipt, err := d.rds.SettleSeatNet(req.RoundId, fingerprint, before, raw, batch)
	if err != nil {
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		return resp
	}
	var updated struct {
		Accounts []struct {
			UserID  uint32 `json:"userId"`
			Balance int64  `json:"balance"`
		}
	}
	if jsoniter.Unmarshal(receipt, &updated) != nil || len(updated.Accounts) != len(req.Items) {
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		return resp
	}
	for i, account := range updated.Accounts {
		if account.UserID != req.Items[i].UserId {
			resp.Code = services.ErrorCode_SYSTEM_ERROR
			return resp
		}
		terminal.Settlement.Results[i].Currency = decimalFromCent(account.Balance).StringFixed(2)
	}
	d.rounds[req.RoundId] = terminal
	for _, amount := range amounts {
		if !d.testSkipPoolSideEffects {
			effectiveCny := amount.effective.Mul(amount.exchange).Truncate(4)
			payoutCny := amount.payout.Mul(amount.exchange).Truncate(4)
			dao.CacheIns().ApplyPoolChange(int64(req.Agent), amount.item.UserId, runtime.PoolSymbol, amount.item.CurrencyType, req.RoundId, effectiveCny, payoutCny, decimal.Zero)
			dao.CacheIns().RecordSettlement(int64(req.Agent), amount.item.UserId, runtime.PoolSymbol, amount.effective, amount.payout)
			balance := decimal.RequireFromString(terminal.Settlement.Results[findSeatResult(terminal.Settlement.Results, amount.item.UserId)].Currency)
			record := ConvertRecord(req.Agent, amount.item.UserId, req.Level, req.RoundId, amount.item.CurrencyType, runtime.Symbol, "", amount.item.Record, balance, runtime.WebID, true, amount.effective.InexactFloat64(), amount.payout.InexactFloat64(), mustSeatMoney(amount.item.Pump).InexactFloat64())
			record.Revenue = 0
			record.ExRevenue = 0
			d.SaveRecord(record)
			d.SaveBill(req.Agent, amount.item.UserId, amount.net, balance.InexactFloat64(), runtime.Symbol, "seat_net", amount.item.CurrencyType, req.RoundId)
		}
	}
	terminal.Settlement.PoolApplied = true
	// Money and terminal snapshot already committed together. Statistics/ES retain
	// their documented asynchronous boundary; do not replay transfers on error.
	_ = d.saveRoundLocked(terminal)
	resp.Code = services.ErrorCode_OK
	resp.State = terminal.State
	for _, result := range terminal.Settlement.Results {
		resp.Items = append(resp.Items, d.buildSettlementResponse(result))
	}
	return resp
}
func findSeatResult(results []*financeSettlementResult, userID uint32) int {
	for i, r := range results {
		if r.UserID == userID {
			return i
		}
	}
	return 0
}
func mustSeatMoney(value string) decimal.Decimal { v, _ := parseMoney(value); return v }
