package unified

import (
	"app/config"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"io"
	"lottery/dao"
	"micro_service/services"
	"strconv"
	"strings"
	"time"
)

var scratchOdds = []int64{50, 100, 200, 300, 500, 800, 1000, 1500, 2000, 3000, 5000, 10000, 15000, 20000, 30000, 50000, 100000, 200000, 500000, 1000000}
var scratchFaces = []int64{100, 200, 500, 1000, 2000, 3000, 5000, 10000}

type scratchTicket struct {
	Claimed         bool    `json:"claimed"`
	ClaimedAt       int64   `json:"claimedAt"`
	Refunded        bool    `json:"refunded"`
	Serial          string  `json:"serialFull"`
	RoundID         string  `json:"roundId"`
	UserID          uint32  `json:"userId"`
	Currency        string  `json:"currency"`
	Kind            string  `json:"kind"`
	Face            int64   `json:"faceValue"`
	Win             int64   `json:"winAmount"`
	Mask            []int   `json:"huMask"`
	Count           int     `json:"huCount"`
	Multiplier      float64 `json:"multiplier"`
	PrizeTier       string  `json:"prizeTier"`
	RoomID          int64   `json:"roomId"`
	GameRoomID      int64   `json:"gameRoomId"`
	RoomType        int     `json:"roomType"`
	ReelVersion     string  `json:"reelVersion"`
	DecisionID      string  `json:"decisionId"`
	ClaimID         string  `json:"claimId"`
	Digest          string  `json:"digest"`
	OutcomeHash     string  `json:"outcomeHash"`
	Record          string  `json:"record"`
	Mirrored        bool    `json:"mirrored"`
	HistoryRecorded bool    `json:"historyRecorded"`
	PurchaseBalance int64   `json:"purchaseBalance"`
}
type scratchPlan struct {
	RoundID     string `json:"roundId"`
	Serial      string `json:"serialFull"`
	Mask        []int  `json:"huMask"`
	ReelVersion string `json:"reelVersion"`
}
type scratchCandidate struct {
	ID      string        `json:"id"`
	Mode    string        `json:"mode"`
	Tickets []scratchPlan `json:"tickets"`
}
type scratchInput struct {
	dao.ScratchScope
	Level        uint32             `json:"level"`
	RoomID       int64              `json:"roomId"`
	GameRoomID   int64              `json:"gameRoomId"`
	RoomType     int                `json:"roomType"`
	Kind         string             `json:"kind"`
	RequestID    string             `json:"requestId"`
	Face         int64              `json:"faceValue"`
	Serial       string             `json:"serialFull"`
	Refs         []string           `json:"refs"`
	PurchaseMode string             `json:"purchaseMode"`
	SearchType   string             `json:"searchType"`
	BookTail     string             `json:"bookTail"`
	TicketTail   string             `json:"ticketTail"`
	Candidates   []scratchCandidate `json:"candidates"`
}

func scratchHash(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%x", sum)
}
func scratchReply(value any, err error) (*services.ScratchActionResponse, error) {
	if err != nil {
		code := services.ErrorCode_SYSTEM_ERROR
		reason := "SCRATCH_UNAVAILABLE"
		if strings.Contains(err.Error(), "CASH_INSUFFICIENT") {
			code = services.ErrorCode_NO_ENOUGH_MONEY
			reason = "INSUFFICIENT_BALANCE"
		}
		if strings.Contains(err.Error(), "SCRATCH_POOL") {
			code = services.ErrorCode_NO_ENOUGH_POOL_MONEY
			reason = "INSUFFICIENT_POOL"
		}
		zap.L().Error("scratch operation failed", zap.Error(err))
		return &services.ScratchActionResponse{Code: code, Reason: reason}, nil
	}
	raw, e := json.Marshal(value)
	if e != nil {
		return nil, e
	}
	return &services.ScratchActionResponse{Code: services.ErrorCode_OK, Success: true, Payload: string(raw)}, nil
}

func (d *LotteryService) issueScratch(ctx context.Context, input scratchInput, runtime *roundRuntime) (any, error) {
	if input.RequestID == "" || len(input.Refs) == 0 || len(input.Refs) > 100 || len(input.Candidates) != 3 {
		return nil, fmt.Errorf("invalid scratch purchase")
	}
	fingerprint := scratchHash(input)
	if previous, err := d.rds.ScratchReceipt(input.ScratchScope, input.RequestID); err == nil {
		return d.finishScratchIssue(ctx, previous, fingerprint, false, runtime)
	} else if err != redis.Nil {
		return nil, err
	}
	if d.testScratchMirror == nil {
		readyCtx, cancel := context.WithTimeout(ctx, time.Second)
		err := d.db.FinalRecordStorageReady(readyCtx)
		cancel()
		if err != nil {
			return nil, fmt.Errorf("scratch purchase reporting storage unavailable: %w", err)
		}
		if err := d.db.ScratchStorageReady(ctx); err != nil {
			return nil, err
		}
	}
	faceValid := false
	for _, face := range scratchFaces {
		if face == input.Face {
			faceValid = true
		}
	}
	if !faceValid {
		return nil, fmt.Errorf("invalid denomination")
	}
	if input.PurchaseMode != "RANDOM_SINGLE" && input.PurchaseMode != "FULL_BOOK" {
		return nil, fmt.Errorf("invalid purchase mode")
	}
	offers, err := d.rds.ScratchOfferMap(input.ScratchScope)
	if err != nil {
		return nil, err
	}
	serials := []string{}
	seen := map[string]bool{}
	for _, ref := range input.Refs {
		offer, ok := offers[ref]
		if !ok || offer.Expires <= time.Now().Unix() || offer.Book != (input.PurchaseMode == "FULL_BOOK") {
			return nil, fmt.Errorf("ticket offer expired")
		}
		for _, serial := range offer.Serials {
			if seen[serial] {
				return nil, fmt.Errorf("overlapping ticket selection")
			}
			seen[serial] = true
			serials = append(serials, serial)
		}
	}
	if len(serials) == 0 || len(serials) > 100 {
		return nil, fmt.Errorf("invalid ticket quantity")
	}
	cost := input.Face * int64(len(serials))
	control := &services.GetPoolControlDecisionRequest{RequestId: input.RequestID + ":decision", RoundId: input.RequestID, GameId: input.GameID, Agent: input.Agent, Level: input.Level, CurrencyType: input.Currency, UserId: input.UserID}
	modes := map[string]bool{}
	byID := map[string]scratchCandidate{}
	for _, candidate := range input.Candidates {
		if candidate.ID == "" || modes[candidate.Mode] || len(candidate.Tickets) != len(serials) || (candidate.Mode != "NATURAL" && candidate.Mode != "PLAYER_WIN" && candidate.Mode != "PLAYER_LOSE") {
			return nil, fmt.Errorf("invalid scratch candidates")
		}
		modes[candidate.Mode] = true
		if _, ok := byID[candidate.ID]; ok {
			return nil, fmt.Errorf("duplicate candidate")
		}
		total := int64(0)
		roundIDs := map[string]bool{}
		for i, ticket := range candidate.Tickets {
			if ticket.Serial != serials[i] || !scratchSafeID(ticket.RoundID, 100) || roundIDs[ticket.RoundID] || !scratchVersion(ticket.ReelVersion) {
				return nil, fmt.Errorf("candidate ticket mismatch")
			}
			roundIDs[ticket.RoundID] = true
			_, prize, _, err := scratchPrize(ticket.Mask, input.Face)
			if err != nil {
				return nil, err
			}
			total += prize
		}
		if candidate.Mode == "PLAYER_WIN" && total <= cost || candidate.Mode == "PLAYER_LOSE" && total >= cost {
			return nil, fmt.Errorf("candidate economic mode mismatch")
		}
		control.Candidates = append(control.Candidates, &services.PoolControlCandidate{Id: candidate.ID, Mode: candidate.Mode, BetAmount: decimalFromCent(cost).StringFixed(2), Payout: decimalFromCent(total).StringFixed(2)})
		byID[candidate.ID] = candidate
	}
	decision, err := d.GetPoolControlDecision(ctx, control)
	if err != nil {
		return nil, err
	}
	if decision.Code != services.ErrorCode_OK {
		return nil, fmt.Errorf("pool decision rejected")
	}
	selected, ok := byID[decision.SelectedCandidateId]
	if !ok {
		return nil, fmt.Errorf("pool selected unknown candidate")
	}
	exchange, ok := config.CfgIns.GetExchange(input.Currency)
	if !ok || !exchange.IsPositive() {
		return nil, fmt.Errorf("currency exchange missing")
	}
	balance, code := d.getPlayerCurrency(input.UserID)
	if code != services.ErrorCode_OK || balance < cost {
		return nil, fmt.Errorf("CASH_INSUFFICIENT")
	}
	revision, err := d.rds.ScratchRevision(input.UserID)
	if err != nil {
		return nil, err
	}
	cash := dao.SeatNetBatch{ID: input.RequestID, Accounts: []dao.SeatNetAccount{{UserID: input.UserID, AgentID: input.Agent, Currency: input.Currency, Revision: revision}}, Steps: []dao.SeatNetStep{{ID: input.RequestID, Account: 1, Kind: "debit", Amount: cost}}}
	item := config.CfgIns.GetPoolItem(int64(input.Agent), runtime.Symbol, input.Level)
	if item == nil {
		return nil, fmt.Errorf("pool level missing")
	}
	totalBetCny := decimalFromCent(cost).Mul(exchange).Truncate(4)
	revenue := totalBetCny.Mul(runtime.Revenue).Truncate(4)
	base := item.Base
	if !d.testSkipPoolSideEffects {
		base = dao.CacheIns().FinanceBasePool(int64(input.Agent), runtime.PoolSymbol, item.Base)
	}
	issue := dao.ScratchIssue{Fingerprint: fingerprint, BasePoolAfterBet: base.Add(totalBetCny).Sub(revenue).String(), PoolMember: fmt.Sprintf("%d_%s", input.Agent, runtime.PoolSymbol)}
	newTickets := []any{}
	for _, plan := range selected.Tickets {
		count, win, tier, err := scratchPrize(plan.Mask, input.Face)
		if err != nil {
			return nil, err
		}
		ticket := &scratchTicket{Serial: plan.Serial, RoundID: plan.RoundID, UserID: input.UserID, Currency: input.Currency, Kind: input.Kind, Face: input.Face, Win: win, Mask: plan.Mask, Count: count, Multiplier: float64(win) / float64(input.Face), PrizeTier: tier, RoomID: input.RoomID, GameRoomID: input.GameRoomID, RoomType: input.RoomType, ReelVersion: plan.ReelVersion, DecisionID: decision.DecisionId, ClaimID: "scratch:" + plan.RoundID + ":claim"}
		ticket.OutcomeHash = scratchHash(map[string]any{"serial": ticket.Serial, "face": ticket.Face, "mask": ticket.Mask, "reelVersion": ticket.ReelVersion})
		record := map[string]any{"roomId": ticket.RoomID, "gameRoomId": ticket.GameRoomID, "roomType": ticket.RoomType, "serialNo": ticket.Serial, "faceValue": ticket.Face, "huMask": ticket.Mask, "huCount": ticket.Count, "multiplier": ticket.Multiplier, "amount": ticket.Win - ticket.Face, "reelVersion": ticket.ReelVersion, "decisionId": ticket.DecisionID}
		rawRecord, _ := json.Marshal(record)
		ticket.Record = string(rawRecord)
		betID := plan.RoundID + ":rev:0:buy"
		amount := decimalFromCent(input.Face)
		cny := amount.Mul(exchange).Truncate(4)
		payout := decimalFromCent(win).Mul(exchange).Truncate(4)
		round := &financeRound{RoundID: plan.RoundID, Agent: input.Agent, GameID: input.GameID, Level: input.Level, Symbol: runtime.Symbol, PoolSymbol: runtime.PoolSymbol, State: string(financeStateReserved), Bets: map[string]*financeBetSnapshot{betID: {BetID: betID, UserID: input.UserID, CurrencyType: input.Currency, Amount: amount.StringFixed(2), AmountCNY: cny.String(), AgentEffectAmountCNY: cny.String(), AgentRevenueAmountCNY: cny.Mul(runtime.Revenue).Truncate(4).String(), Accepted: true, Status: financeBetStatusActive, Code: 0}}, Scratch: ticket, UpdatedAt: time.Now().Unix()}
		ticket.Digest = d.roundBetDigest(round)
		resID := "scratch-res:" + plan.RoundID
		round.Reservation = &financeReservation{RequestID: input.RequestID, ReservationID: resID, BetDigest: ticket.Digest, OutcomeHash: ticket.OutcomeHash, TotalPayoutCNY: payout.String(), Status: string(financeStateReserved), ExpiresAt: 0}
		raw, _ := json.Marshal(round)
		issue.Rounds = append(issue.Rounds, dao.ScratchIssuedRound{RoundID: plan.RoundID, Serial: plan.Serial, Raw: string(raw), ReservationID: resID, PayoutCNY: payout.String(), Ledger: dao.EncodeReservationLedger(dao.ReservationLedgerActive, payout.String(), "", plan.RoundID, int64(input.Agent), runtime.PoolSymbol, "", time.Now().Unix())})
		newTickets = append(newTickets, scratchPublic(ticket))
	}
	result, _ := json.Marshal(map[string]any{"code": 0, "purchaseMode": input.PurchaseMode, "newTickets": newTickets, "failedItems": []any{}})
	issue.Result = result
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	receipt, err := d.rds.IssueScratch(input.ScratchScope, input.RequestID, issue, cash)
	if err != nil {
		return nil, err
	}
	return d.finishScratchIssue(ctx, string(receipt), fingerprint, true, runtime)
}
func (d *LotteryService) finishScratchIssue(ctx context.Context, raw, fingerprint string, fresh bool, runtime *roundRuntime) (any, error) {
	var receipt struct {
		Fingerprint string         `json:"fingerprint"`
		Result      map[string]any `json:"result"`
		Rounds      []string       `json:"rounds"`
	}
	if json.Unmarshal([]byte(raw), &receipt) != nil || receipt.Fingerprint != fingerprint {
		return nil, fmt.Errorf("purchase request changed")
	}
	for _, encoded := range receipt.Rounds {
		round := &financeRound{}
		if json.Unmarshal([]byte(encoded), round) != nil || round.Scratch == nil {
			return nil, fmt.Errorf("invalid paid ticket snapshot")
		}
		if existing, ok := d.loadRoundLocked(round.RoundID); ok {
			round = existing
		} else {
			d.rounds[round.RoundID] = round
		}
		if fresh && !d.testSkipPoolSideEffects {
			ticket := round.Scratch
			exchange, _ := config.CfgIns.GetExchange(ticket.Currency)
			bet := decimalFromCent(ticket.Face).Mul(exchange).Truncate(4)
			dao.CacheIns().ApplyPoolChange(int64(round.Agent), ticket.UserID, round.PoolSymbol, ticket.Currency, round.RoundID, bet, decimal.Zero, bet.Mul(runtime.Revenue).Truncate(4))
			d.SaveBill(round.Agent, ticket.UserID, decimalFromCent(ticket.Face).Neg(), decimalFromCent(ticket.PurchaseBalance).InexactFloat64(), runtime.Symbol, "scratch_purchase", ticket.Currency, round.RoundID)
		}
	}
	// Account for the entire already-committed book before a slow or failed SQL
	// mirror can stop delivery. Receipt retries must not add those wagers again.
	for _, encoded := range receipt.Rounds {
		var snapshot financeRound
		if err := json.Unmarshal([]byte(encoded), &snapshot); err != nil {
			return nil, err
		}
		round := d.rounds[snapshot.RoundID]
		if err := d.mirrorScratch(ctx, round); err != nil {
			return nil, err
		}
	}
	return receipt.Result, nil
}
func (d *LotteryService) revealScratch(ctx context.Context, input scratchInput) (*services.ScratchActionResponse, error) {
	if len(input.Serial) == 0 || len(input.Serial) > 80 {
		return scratchReply(nil, fmt.Errorf("invalid serial"))
	}
	d.roundLock.Lock()
	id, err := d.rds.ScratchSerialRound(input.ScratchScope, input.Serial)
	if err != nil {
		d.roundLock.Unlock()
		return scratchReply(nil, err)
	}
	round, ok := d.loadRoundLocked(id)
	if !ok || round.Scratch == nil || round.Agent != input.Agent || round.GameID != input.GameID || round.Scratch.UserID != input.UserID || round.Scratch.Currency != input.Currency {
		d.roundLock.Unlock()
		return scratchReply(nil, fmt.Errorf("ticket owner mismatch"))
	}
	ticket := round.Scratch
	request := &services.SettlementRequest{SettlementId: ticket.ClaimID, RoundId: id, GameId: round.GameID, Agent: round.Agent, Level: round.Level, ReservationId: round.Reservation.ReservationID, BetDigest: ticket.Digest, OutcomeHash: ticket.OutcomeHash, Mode: financeModeNormal, Items: []*services.SettlementItem{{UserId: ticket.UserID, CurrencyType: ticket.Currency, BetAmount: decimalFromCent(ticket.Face).StringFixed(2), Payout: decimalFromCent(ticket.Win).StringFixed(2), Profit: decimalFromCent(ticket.Win - ticket.Face).StringFixed(2), Pump: "0.00", Record: ticket.Record}}}
	d.roundLock.Unlock()
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	response, err := d.Settlement(ctx, request)
	if err != nil {
		return nil, err
	}
	if response.Code != services.ErrorCode_OK {
		return &services.ScratchActionResponse{Code: response.Code, Reason: "CLAIM_REJECTED"}, nil
	}
	d.roundLock.Lock()
	defer d.roundLock.Unlock()
	round = d.rounds[id]
	ticket = round.Scratch
	if !ticket.Claimed {
		ticket.Claimed = true
		ticket.ClaimedAt = round.Settlement.CompletedAt * 1000
		ticket.Mirrored = false
	}
	if err = d.mirrorScratch(ctx, round); err != nil {
		return scratchReply(nil, err)
	}
	if err = d.rds.FinishScratch(input.ScratchScope, id, ticket.Serial, strings.ToUpper(ticket.Kind), ticket.Win); err != nil {
		return scratchReply(nil, err)
	}
	ticket.HistoryRecorded = true
	balance, code := d.getPlayerCurrency(input.UserID)
	if code != services.ErrorCode_OK {
		return scratchReply(nil, fmt.Errorf("balance query failed"))
	}
	totals, err := d.rds.ScratchTotals(input.ScratchScope)
	if err != nil {
		return scratchReply(nil, err)
	}
	return scratchReply(map[string]any{"code": 0, "serialFull": ticket.Serial, "faceValue": ticket.Face, "huCount": ticket.Count, "multiplier": ticket.Multiplier, "winAmount": ticket.Win, "prizeTier": ticket.PrizeTier, "huMask": ticket.Mask, "gglKind": strings.ToUpper(ticket.Kind), "walletAmount": balance, "accumulatedWinByKind": totals, "accumulatedWin": totals["GGL1"] + totals["GGL2"], "claimedAt": round.Settlement.CompletedAt * 1000, "purchaseRoomId": ticket.RoomID, "purchaseGameRoomId": ticket.GameRoomID, "finalReportStored": scratchDisplaySnapshotStored(round)}, nil)
}
func (d *LotteryService) ScratchAction(ctx context.Context, req *services.ScratchActionRequest) (response *services.ScratchActionResponse, transportErr error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			zap.L().Error("scratch request panic", zap.Any("error", recovered))
			response = &services.ScratchActionResponse{Code: services.ErrorCode_SYSTEM_ERROR, Reason: "SCRATCH_UNAVAILABLE"}
			transportErr = nil
		}
	}()
	if d.rds == nil {
		return scratchReply(nil, fmt.Errorf("scratch storage unavailable"))
	}
	if req == nil || len(req.Payload) > 512*1024 {
		return &services.ScratchActionResponse{Code: services.ErrorCode_PARAMS_INVALID, Reason: "INVALID_REQUEST"}, nil
	}
	var input scratchInput
	dec := json.NewDecoder(strings.NewReader(req.Payload))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&input); err != nil {
		return &services.ScratchActionResponse{Code: services.ErrorCode_PARAMS_INVALID, Reason: "INVALID_PAYLOAD"}, nil
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return &services.ScratchActionResponse{Code: services.ErrorCode_PARAMS_INVALID, Reason: "TRAILING_PAYLOAD"}, nil
	}
	if input.UserID == 0 || !scratchCurrency(input.Currency) || input.RoomID <= 0 || input.GameRoomID <= 0 || len(input.RequestID) > 160 || (input.RequestID != "" && !scratchSafeID(input.RequestID, 160)) {
		return &services.ScratchActionResponse{Code: services.ErrorCode_PARAMS_INVALID, Reason: "INVALID_SCOPE"}, nil
	}
	runtime, code := d.resolveRuntime(input.Agent, input.GameID, input.Level)
	if code != services.ErrorCode_OK {
		return &services.ScratchActionResponse{Code: code, Reason: "INVALID_RUNTIME"}, nil
	}
	if int(runtime.RoomType) != input.RoomType || !((input.RoomType == 61 && input.Kind == "ggl1") || (input.RoomType == 63 && input.Kind == "ggl2")) {
		return &services.ScratchActionResponse{Code: services.ErrorCode_PARAMS_INVALID, Reason: "WRONG_GAME"}, nil
	}
	if req.Operation == "reveal" {
		return d.revealScratch(ctx, input)
	}
	d.roundLock.Lock()
	defer d.roundLock.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch req.Operation {
	case "state":
		value, err := d.scratchState(ctx, input)
		return scratchReply(value, err)
	case "search", "allocate":
		value, err := d.searchScratch(ctx, input, req.Operation == "allocate")
		return scratchReply(value, err)
	case "issue":
		value, err := d.issueScratch(ctx, input, runtime)
		return scratchReply(value, err)
	}
	return &services.ScratchActionResponse{Code: services.ErrorCode_PARAMS_INVALID, Reason: "INVALID_OPERATION"}, nil
}
func (d *LotteryService) scratchState(ctx context.Context, input scratchInput) (any, error) {
	ids, err := d.rds.ScratchPending(input.ScratchScope)
	if err != nil {
		return nil, err
	}
	if len(ids) > 100 {
		return nil, fmt.Errorf("invalid pending ticket count")
	}
	pending := []any{}
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		round, ok := d.loadRoundLocked(id)
		if !ok || round.Scratch == nil {
			return nil, fmt.Errorf("paid ticket snapshot missing")
		}
		ticket := round.Scratch
		if ticket.UserID != input.UserID || ticket.Currency != input.Currency || round.Agent != input.Agent || round.GameID != input.GameID {
			return nil, fmt.Errorf("ticket owner mismatch")
		}
		if round.State == string(financeStateVoided) {
			if !ticket.Refunded {
				ticket.Refunded = true
				ticket.Mirrored = false
			}
			if err := d.mirrorScratch(ctx, round); err != nil {
				return nil, err
			}
			if err := d.rds.DropScratchPending(input.ScratchScope, id, ticket.Serial); err != nil {
				return nil, err
			}
			continue
		}
		if round.State == string(financeStateSettled) && ticket.Claimed && ticket.Mirrored {
			if err := d.rds.FinishScratch(input.ScratchScope, id, ticket.Serial, strings.ToUpper(ticket.Kind), ticket.Win); err != nil {
				return nil, err
			}
			ticket.HistoryRecorded = true
			continue
		}
		pending = append(pending, scratchPublic(ticket))
	}
	balance, code := d.getPlayerCurrency(input.UserID)
	if code != services.ErrorCode_OK {
		return nil, fmt.Errorf("authoritative balance unavailable")
	}
	totals, err := d.rds.ScratchTotals(input.ScratchScope)
	if err != nil {
		return nil, err
	}
	return map[string]any{"code": 0, "walletAmount": balance, "pendingTickets": pending, "denominations": scratchFaces, "winMultipliers": []float64{.5, 1, 2, 3, 5, 8, 10, 15, 20, 30, 50, 100, 150, 200, 300, 500, 1000, 2000, 5000, 10000}, "accumulatedWinByKind": totals, "accumulatedWin": totals["GGL1"] + totals["GGL2"]}, nil
}
func scratchPublic(ticket *scratchTicket) any {
	return map[string]any{"serialFull": ticket.Serial, "faceValue": ticket.Face, "gglKind": strings.ToUpper(ticket.Kind)}
}

func scratchSettlementMatches(round *financeRound, request *services.SettlementRequest) bool {
	t := round.Scratch
	mode := normalizeFinanceMode(request.Mode)
	claimID := t.ClaimID
	prizeAmount := t.Win
	profitAmount := t.Win - t.Face
	if mode == financeModeVoidRefund {
		claimID = "scratch:" + t.RoundID + ":void"
		prizeAmount = t.Face
		profitAmount = 0
	} else if mode != financeModeNormal {
		return false
	}
	if request.SettlementId != claimID || request.BetDigest != t.Digest || request.OutcomeHash != t.OutcomeHash || round.Reservation == nil || request.ReservationId != round.Reservation.ReservationID || len(request.Items) != 1 {
		return false
	}
	item := request.Items[0]
	if item == nil || item.UserId != t.UserID || item.CurrencyType != t.Currency || item.Carry != nil || item.NetScore != "" || item.Record != t.Record {
		return false
	}
	bet, e := parseMoney(item.BetAmount)
	if e != nil || !bet.Equal(decimalFromCent(t.Face)) {
		return false
	}
	prize, e := parseMoney(item.Payout)
	if e != nil || !prize.Equal(decimalFromCent(prizeAmount)) {
		return false
	}
	profit, e := parseSignedMoney(item.Profit)
	if e != nil || !profit.Equal(decimalFromCent(profitAmount)) {
		return false
	}
	pump, e := parseMoney(item.Pump)
	return e == nil && pump.IsZero()
}
func digits(value string, min, max int) bool {
	if len(value) < min || len(value) > max {
		return false
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
func scratchCurrency(value string) bool {
	if len(value) < 1 || len(value) > 12 {
		return false
	}
	for _, c := range value {
		if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}
func scratchSafeID(value string, max int) bool {
	if len(value) < 1 || len(value) > max {
		return false
	}
	for _, c := range value {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("_.:/-", c)) {
			return false
		}
	}
	return true
}
func scratchVersion(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, c := range value {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func (d *LotteryService) searchScratch(ctx context.Context, input scratchInput, direct bool) (any, error) {
	if direct && input.BookTail != "" {
		return nil, fmt.Errorf("invalid allocation suffix")
	}
	if input.SearchType == "SINGLE_TICKET" {
		number, err := strconv.Atoi(input.TicketTail)
		if err != nil || number < 1 || number > 20 {
			return nil, fmt.Errorf("invalid ticket suffix")
		}
	}
	if !direct && (!digits(input.BookTail, 1, 7) || (input.SearchType != "BOOK_TAIL" && input.SearchType != "SINGLE_TICKET") || (input.SearchType == "SINGLE_TICKET" && (!digits(input.TicketTail, 1, 2)))) {
		return nil, fmt.Errorf("invalid ticket suffix")
	}
	offers, err := d.rds.ScratchOfferMap(input.ScratchScope)
	if err != nil {
		return nil, err
	}
	matched := []string{}
	seen := map[string]bool{}
	for attempt := 0; attempt < 10 && len(offers) < 100; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		book, err := d.rds.NextScratchBook(input.ScratchScope, input.Kind, input.BookTail)
		if err != nil {
			return nil, err
		}
		serials := make([]string, 20)
		available := true
		for i := 1; i <= 20; i++ {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			serials[i-1] = dao.ScratchSerial(book, i)
			_, err = d.rds.ScratchSerialRound(input.ScratchScope, serials[i-1])
			if err == nil {
				available = false
			} else if err != redis.Nil {
				return nil, err
			}
		}
		refs := map[string][]string{}
		if direct {
			refs[serials[0]] = []string{serials[0]}
		} else if input.SearchType == "BOOK_TAIL" {
			if available {
				refs[book] = serials
			}
		} else {
			for i, serial := range serials {
				if strings.HasSuffix(fmt.Sprintf("%02d", i+1), input.TicketTail) {
					_, e := d.rds.ScratchSerialRound(input.ScratchScope, serial)
					if e == redis.Nil {
						refs[serial] = []string{serial}
					}
				}
			}
		}
		for ref, values := range refs {
			if !seen[ref] && len(offers) < 100 {
				seen[ref] = true
				matched = append(matched, ref)
				offers[ref] = dao.ScratchOffer{Serials: values, Book: !direct && input.SearchType == "BOOK_TAIL", Expires: time.Now().Unix() + 60}
			}
		}
		if direct {
			break
		}
	}
	if err = d.rds.SaveScratchOffers(input.ScratchScope, offers); err != nil {
		return nil, err
	}
	return map[string]any{"code": 0, "searchType": input.SearchType, "matchedSerials": matched, "bookTail": input.BookTail, "ticketTail": input.TicketTail}, nil
}
func scratchPrize(mask []int, face int64) (int, int64, string, error) {
	if len(mask) != 20 {
		return 0, 0, "", fmt.Errorf("twenty symbols required")
	}
	count := 0
	for _, symbol := range mask {
		if symbol < 1 || symbol > 10 {
			return 0, 0, "", fmt.Errorf("invalid scratch symbol")
		}
		if symbol == 1 {
			count++
		}
	}
	if count == 0 {
		return 0, 0, "NONE", nil
	}
	prize := face * scratchOdds[count-1] / 100
	tier := "NORMAL"
	if scratchOdds[count-1] >= 5000 {
		tier = "SUPER"
	} else if scratchOdds[count-1] >= 3500 {
		tier = "HUGE"
	} else if scratchOdds[count-1] >= 2000 {
		tier = "BIG"
	}
	return count, prize, tier, nil
}
func (d *LotteryService) mirrorScratch(ctx context.Context, round *financeRound) error {
	if round.Scratch.Mirrored {
		return nil
	}
	if d.testScratchMirror != nil {
		if err := d.testScratchMirror(round); err != nil {
			return err
		}
	} else {
		raw, err := json.Marshal(round.Scratch)
		if err != nil {
			return err
		}
		if err = d.db.SaveScratchGameStorage(ctx, round.Scratch.UserID, round.GameID, round.Scratch.Currency, "scratch:"+round.Scratch.Serial, string(raw)); err != nil {
			return err
		}
	}
	round.Scratch.Mirrored = true
	return d.saveRoundLocked(round)
}
