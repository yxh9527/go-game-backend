package unified

import (
	"app/config"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/shopspring/decimal"
	"lottery/dao"
	"micro_service/services"
	"strconv"
	"strings"
	"time"
)

const financeModeBankerNet = "BANKER_NET"

// Bettors' original paid wagers and bankers' zero-wager carry registrations
// share one round, one reservation and one atomic terminal money transfer.
func (d *LotteryService) settleBankerNetLocked(round *financeRound, runtime *roundRuntime, req *services.SettlementRequest) *services.SettlementResponse {
	resp := &services.SettlementResponse{Code: services.ErrorCode_PARAMS_INVALID, RoundId: req.RoundId, SettlementId: req.SettlementId}
	fingerprint := settlementRequestHash(req)
	if round.Settlement != nil {
		saved := round.Settlement
		if saved.SettlementID != req.SettlementId || saved.RequestHash != fingerprint || saved.Mode != financeModeBankerNet || saved.Status != financeSettlementStatusCompleted {
			return resp
		}
		resp.Code = services.ErrorCode_OK
		resp.State = round.State
		for _, item := range saved.Results {
			resp.Items = append(resp.Items, d.buildSettlementResponse(item))
		}
		return resp
	}
	d.expireRoundIfNeededLocked(round)
	reservation := round.Reservation
	if d.rds == nil || round.State != string(financeStateReserved) || reservation == nil || reservation.Status != string(financeStateReserved) || reservation.ReservationID != req.ReservationId ||
		reservation.BetDigest != req.BetDigest || reservation.OutcomeHash != req.OutcomeHash || req.BetDigest != d.roundBetDigest(round) || !validSettlementCarries(round, req) {
		return resp
	}
	type registered struct {
		stake  decimal.Decimal
		carry  *services.FinanceCarryBinding
		cap    int64
		banker bool
	}
	active := map[string]*registered{}
	for _, bet := range round.Bets {
		if !isActiveFinanceBet(bet) {
			continue
		}
		key := fmt.Sprintf("%d|%s", bet.UserID, bet.CurrencyType)
		value := active[key]
		if value == nil {
			value = &registered{}
			active[key] = value
		}
		amount, err := parseMoney(bet.Amount)
		if err != nil {
			return resp
		}
		if strings.HasPrefix(bet.AreaID, "banker:") {
			cap, err := strconv.ParseInt(strings.TrimPrefix(bet.AreaID, "banker:"), 10, 64)
			if err != nil || cap <= 0 || cap > 200000000 || !amount.IsZero() || bet.Carry == nil || value.banker || !value.stake.IsZero() {
				return resp
			}
			value.banker = true
			value.carry = bet.Carry
			value.cap = cap
		} else {
			if bet.Carry != nil || value.banker {
				return resp
			}
			value.stake = value.stake.Add(amount)
		}
	}
	if len(active) == 0 || len(active) > 1024 || len(active) != len(req.Items) {
		return resp
	}
	before, err := d.rds.HGet(financeRoundHash, req.RoundId)
	if err != nil {
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		return resp
	}
	raw, _ := jsoniter.MarshalToString(round)
	terminal := &financeRound{}
	if jsoniter.UnmarshalFromString(raw, terminal) != nil {
		return resp
	}
	terminal.State = string(financeStateSettled)
	terminal.UpdatedAt = time.Now().Unix()
	terminal.Reservation.Status = "RELEASED"
	terminal.Reservation.ExpiresAt = 0
	terminal.Settlement = &financeSettlement{SettlementID: req.SettlementId, Mode: financeModeBankerNet, Status: financeSettlementStatusCompleted, RequestHash: fingerprint, BetDigest: req.BetDigest, OutcomeHash: req.OutcomeHash, CompletedAt: time.Now().Unix()}
	type economic struct {
		item                               *services.SettlementItem
		effective, payout, delta, exchange decimal.Decimal
		banker                             bool
	}
	amounts := []economic{}
	batch := dao.SeatNetBatch{ID: req.SettlementId}
	seen := map[string]bool{}
	bankerCount := 0
	totalReserved := decimal.Zero
	for i, item := range req.Items {
		if item == nil {
			return resp
		}
		key := fmt.Sprintf("%d|%s", item.UserId, item.CurrencyType)
		value := active[key]
		if value == nil || seen[key] {
			return resp
		}
		seen[key] = true
		effective, e := parseMoney(item.BetAmount)
		if e != nil {
			return resp
		}
		payout, e := parseMoney(item.Payout)
		if e != nil {
			return resp
		}
		profit, e := parseSignedMoney(item.Profit)
		if e != nil {
			return resp
		}
		pump, e := parseMoney(item.Pump)
		if e != nil || !pump.IsZero() {
			return resp
		}
		delta := payout
		kind := "credit"
		if value.banker {
			bankerCount++
			net, e := parseSignedMoney(item.NetScore)
			if e != nil || !effective.Equal(net.Abs()) || !profit.Equal(net) || !payout.Equal(effective.Add(net)) || net.Abs().GreaterThan(decimalFromCent(value.cap)) || item.Carry == nil {
				return resp
			}
			delta = net
			kind = "carryDelta"
		} else if item.NetScore != "" || item.Carry != nil || !effective.Equal(value.stake) || !profit.Equal(payout.Sub(effective)) {
			return resp
		}
		exchange, ok := config.CfgIns.GetExchange(item.CurrencyType)
		if !ok || !exchange.IsPositive() {
			resp.Code = services.ErrorCode_SYSTEM_ERROR
			return resp
		}
		reservedPayout := payout
		if value.banker {
			reservedPayout = decimal.Max(decimal.Zero, delta)
		}
		totalReserved = totalReserved.Add(reservedPayout.Mul(exchange).Truncate(4))
		revisionRaw, e := d.rds.HGet(fmt.Sprintf("player_%d", item.UserId), "cash_revision")
		if e != nil {
			resp.Code = services.ErrorCode_SYSTEM_ERROR
			return resp
		}
		revision, e := strconv.ParseInt(revisionRaw, 10, 64)
		if e != nil {
			return resp
		}
		batch.Accounts = append(batch.Accounts, dao.SeatNetAccount{UserID: item.UserId, AgentID: req.Agent, Currency: item.CurrencyType, Revision: revision})
		step := dao.SeatNetStep{ID: fmt.Sprintf("%s:%d", req.SettlementId, item.UserId), Account: i + 1, Kind: kind, Amount: toCentDelta(delta)}
		if item.Carry != nil {
			step.Token = item.Carry.Token
			step.TableKey = item.Carry.TableKey
			step.Revision = item.Carry.Revision
		}
		batch.Steps = append(batch.Steps, step)
		terminal.Settlement.Results = append(terminal.Settlement.Results, &financeSettlementResult{UserID: item.UserId, Code: 0, Currency: "0.00"})
		amounts = append(amounts, economic{item, effective, payout, delta, exchange, value.banker})
	}
	reserved, e := decimal.NewFromString(reservation.TotalPayoutCNY)
	if e != nil || bankerCount == 0 || totalReserved.GreaterThan(reserved) {
		return resp
	}
	for _, bet := range terminal.Bets {
		if isActiveFinanceBet(bet) {
			bet.Status = financeBetStatusSettled
		}
	}
	if !d.testSkipPoolSideEffects {
		for _, a := range amounts {
			report := ConvertRecord(req.Agent, a.item.UserId, req.Level, req.RoundId, a.item.CurrencyType, runtime.Symbol, "", a.item.Record, decimal.Zero, runtime.WebID, true, a.effective.InexactFloat64(), a.payout.InexactFloat64(), 0)
			terminal.Settlement.Reports = append(terminal.Settlement.Reports, report)
			terminal.Settlement.Bills = append(terminal.Settlement.Bills, settlementBill(terminal, runtime.Symbol, a.item.UserId, a.item.CurrencyType, a.delta, "banker_net"))
		}
	}
	raw, err = jsoniter.MarshalToString(terminal)
	if err != nil {
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		return resp
	}
	receipt, err := d.rds.SettleBankerBatch(req.RoundId, fingerprint, before, raw, batch, reservation.ReservationID, fmt.Sprintf("%d_%s", req.Agent, runtime.PoolSymbol), reservation.TotalPayoutCNY)
	if err != nil {
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		return resp
	}
	var updated struct {
		Accounts []struct {
			UserID  uint32 `json:"userId"`
			Balance int64  `json:"balance"`
		} `json:"accounts"`
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
	if !d.testSkipPoolSideEffects {
		for _, a := range amounts {
			betDelta := decimal.Zero
			if a.banker {
				betDelta = a.effective.Mul(a.exchange).Truncate(4)
			}
			dao.CacheIns().ApplyPoolChange(int64(req.Agent), a.item.UserId, runtime.PoolSymbol, a.item.CurrencyType, req.RoundId, betDelta, a.payout.Mul(a.exchange).Truncate(4), decimal.Zero)
			dao.CacheIns().RecordSettlement(int64(req.Agent), a.item.UserId, runtime.PoolSymbol, a.effective, a.payout)
		}
	}
	terminal.Settlement.PoolApplied = true
	_ = d.saveRoundLocked(terminal)
	resp.Code = 0
	resp.State = terminal.State
	for _, item := range terminal.Settlement.Results {
		resp.Items = append(resp.Items, d.buildSettlementResponse(item))
	}
	return resp
}
