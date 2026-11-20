package unified

import (
	"app/config"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"lottery/dao"
	"micro_service/services"
	"strings"
	"sync"
	"testing"
	"time"
)

type scratchFixture struct {
	svc           *LotteryService
	rd            *dao.RedisDao
	client        *redis.Client
	input         scratchInput
	mirrorFailure bool
	mirrors       []*scratchTicket
	t             *testing.T
}

func newScratchFixture(t *testing.T) *scratchFixture {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	f := &scratchFixture{rd: dao.NewFinanceRedis(client), client: client, t: t, input: scratchInput{ScratchScope: dao.ScratchScope{Agent: 7, GameID: 91061, UserID: 42, Currency: "CNY"}, Level: 1, RoomID: 61010026, GameRoomID: 62010026, RoomType: 61, Kind: "ggl1"}}
	old := config.CfgIns
	config.CfgIns = &config.Configs{Lock: &sync.RWMutex{}, Currency: &config.CurrencyMgr{Data: map[string]decimal.Decimal{"CNY": decimal.NewFromInt(1)}}, Pool: &config.PoolMgr{Default: map[string]*config.Pool{"test-scratch": {Symbol: "test-scratch", GameId: 91061, Pool: map[int32]*config.PoolItem{1: {Base: decimal.NewFromInt(1000000)}}}}, Agent: map[string]*config.Pool{}}}
	t.Cleanup(func() { config.CfgIns = old })
	f.svc = &LotteryService{rds: f.rd, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}, testSkipPoolSideEffects: true}
	f.svc.testResolveRuntime = func(agent, game, level uint32) (*roundRuntime, services.ErrorCode) {
		if agent != 7 || game != 91061 || level != 1 {
			return nil, services.ErrorCode_PARAMS_INVALID
		}
		return &roundRuntime{AgentID: 7, GameID: 91061, Level: 1, Symbol: "test-scratch", PoolSymbol: "test-scratch_1", RoomType: 61}, services.ErrorCode_OK
	}
	f.svc.testScratchMirror = func(r *financeRound) error {
		if f.mirrorFailure {
			return errors.New("mysql unavailable")
		}
		raw, _ := json.Marshal(r.Scratch)
		copy := &scratchTicket{}
		json.Unmarshal(raw, copy)
		f.mirrors = append(f.mirrors, copy)
		return nil
	}
	for _, uid := range []uint32{42, 43} {
		if err := client.HSet(context.Background(), fmt.Sprintf("player_%d", uid), "id", uid, "agent_id", 7, "currency_type", "CNY", "currency", 100000, "cash_revision", 0).Err(); err != nil {
			t.Fatal(err)
		}
	}
	return f
}
func (f *scratchFixture) call(op string, input scratchInput) (map[string]any, *services.ScratchActionResponse) {
	f.t.Helper()
	raw, _ := json.Marshal(input)
	response, err := f.svc.ScratchAction(context.Background(), &services.ScratchActionRequest{Operation: op, Payload: string(raw)})
	if err != nil {
		f.t.Fatal(err)
	}
	var value map[string]any
	if response.Payload != "" && json.Unmarshal([]byte(response.Payload), &value) != nil {
		f.t.Fatal("invalid response JSON")
	}
	return value, response
}
func (f *scratchFixture) balance(uid uint32) int64 {
	f.t.Helper()
	value, err := f.client.HGet(context.Background(), fmt.Sprintf("player_%d", uid), "currency").Int64()
	if err != nil {
		f.t.Fatal(err)
	}
	return value
}
func (f *scratchFixture) purchase(book bool) (scratchInput, string) {
	f.t.Helper()
	input := f.input
	op := "allocate"
	if book {
		op = "search"
		input.SearchType = "BOOK_TAIL"
		input.BookTail = "789"
	}
	data, response := f.call(op, input)
	if !response.Success {
		f.t.Fatalf("offer: %+v", response)
	}
	ref := data["matchedSerials"].([]any)[0].(string)
	offers, err := f.rd.ScratchOfferMap(f.input.ScratchScope)
	if err != nil {
		f.t.Fatal(err)
	}
	serials := offers[ref].Serials
	input = f.input
	input.RequestID = "purchase-" + ref
	input.Face = 100
	input.PurchaseMode = "RANDOM_SINGLE"
	if book {
		input.PurchaseMode = "FULL_BOOK"
	}
	input.Refs = []string{ref}
	for i, mode := range []string{"NATURAL", "PLAYER_WIN", "PLAYER_LOSE"} {
		candidate := scratchCandidate{ID: mode, Mode: mode}
		count := []int{3, 4, 0}[i]
		for n, serial := range serials {
			mask := make([]int, 20)
			for p := range mask {
				mask[p] = 2
				if p < count {
					mask[p] = 1
				}
			}
			candidate.Tickets = append(candidate.Tickets, scratchPlan{Serial: serial, RoundID: fmt.Sprintf("round-%s-%d-%d", ref, i, n), Mask: mask, ReelVersion: "0123456789012345678901234567890123456789012345678901234567890123"})
		}
		input.Candidates = append(input.Candidates, candidate)
	}
	return input, serials[0]
}
func TestScratchPaidTicketLifecycleIsPrivatePersistentAndIdempotent(t *testing.T) {
	f := newScratchFixture(t)
	input, serial := f.purchase(false)
	data, response := f.call("issue", input)
	if !response.Success {
		t.Fatalf("purchase: %+v", response)
	}
	if f.balance(42) != 99900 || len(data["newTickets"].([]any)) != 1 {
		t.Fatal("purchase did not debit exactly once")
	}
	public := data["newTickets"].([]any)[0].(map[string]any)
	if public["huMask"] != nil || public["winAmount"] != nil {
		t.Fatal("unrevealed ticket leaked prize")
	}
	_, response = f.call("issue", input)
	if !response.Success || f.balance(42) != 99900 || len(f.mirrors) != 1 {
		t.Fatal("purchase replay charged or mirrored twice")
	}
	id, _ := f.rd.ScratchSerialRound(input.ScratchScope, serial)
	round := f.svc.rounds[id]
	round.Reservation.ExpiresAt = time.Now().Unix() - 10
	if f.svc.expireRoundIfNeededLocked(round) || round.State != "RESERVED" {
		t.Fatal("paid unclaimed liability expired")
	}
	round.Reservation.ExpiresAt = 0
	wrong := f.input
	wrong.UserID = 43
	wrong.Serial = serial
	_, response = f.call("reveal", wrong)
	if response.Success || f.balance(43) != 100000 {
		t.Fatal("another user claimed a ticket")
	}
	// Reconstruct the handler after sale, using only persisted Redis state.
	old := f.svc
	f.svc = &LotteryService{rds: f.rd, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}, testSkipPoolSideEffects: true, testResolveRuntime: old.testResolveRuntime, testScratchMirror: old.testScratchMirror}
	state, response := f.call("state", f.input)
	if !response.Success || len(state["pendingTickets"].([]any)) != 1 {
		t.Fatal("paid ticket did not survive handler reconstruction")
	}
	claim := f.input
	claim.Serial = serial
	claim.Face = 10000
	data, response = f.call("reveal", claim)
	if !response.Success {
		t.Fatalf("claim: %+v", response)
	}
	if data["faceValue"] != float64(100) || data["winAmount"] != float64(200) || f.balance(42) != 100100 {
		t.Fatalf("claim result %v balance %d", data, f.balance(42))
	}
	_, response = f.call("reveal", claim)
	if !response.Success || f.balance(42) != 100100 {
		t.Fatal("claim replay credited twice")
	}
	totals, _ := f.rd.ScratchTotals(f.input.ScratchScope)
	if totals["GGL1"] != 200 {
		t.Fatal("replay changed accumulated prize")
	}
	if !f.rd.GetAgentReserved(7, "test-scratch_1").IsZero() {
		t.Fatal("reservation was not released")
	}
	state, response = f.call("state", f.input)
	if !response.Success || len(state["pendingTickets"].([]any)) != 0 {
		t.Fatal("claimed ticket still pending")
	}
	history, _ := f.rd.ScratchHistory(f.input.ScratchScope, f.input.RoomID)
	if len(history) != 1 {
		t.Fatal("claim replay duplicated history")
	}
	if len(f.mirrors) != 2 || !f.mirrors[1].Claimed {
		t.Fatal("claimed ownership was not mirrored")
	}
}

func TestFirstScratchPurchaseIsRejectedBeforeDebitWhenReportingSQLUnavailable(t *testing.T) {
	f := newScratchFixture(t)
	input, serial := f.purchase(false)
	f.svc.testScratchMirror = nil
	before := f.balance(42)
	_, err := f.svc.issueScratch(context.Background(), input, &roundRuntime{AgentID: 7, GameID: 91061, Level: 1, Symbol: "test-scratch", PoolSymbol: "test-scratch_1", RoomType: 61})
	if err == nil || !strings.Contains(err.Error(), "scratch purchase reporting storage unavailable") {
		t.Fatal("new purchase did not check reporting storage before ownership storage", err)
	}
	_, response := f.call("issue", input)
	if response.Success || f.balance(42) != before || len(f.svc.rounds) != 0 {
		t.Fatal("unavailable SQL accepted a fresh purchase", response)
	}
	if _, err := f.rd.ScratchReceipt(input.ScratchScope, input.RequestID); err != redis.Nil {
		t.Fatal("rejected purchase persisted a receipt", err)
	}
	if _, err := f.rd.ScratchSerialRound(input.ScratchScope, serial); err != redis.Nil {
		t.Fatal("rejected purchase acquired ticket ownership", err)
	}
}

func TestOriginalScratchReceiptIsReturnedBeforeNewPurchaseReportingGate(t *testing.T) {
	f := newScratchFixture(t)
	input, serial := f.purchase(false)
	first, response := f.call("issue", input)
	if !response.Success {
		t.Fatal(response)
	}
	balance := f.balance(42)
	f.svc.testScratchMirror = nil
	repeated, response := f.call("issue", input)
	if !response.Success || f.balance(42) != balance {
		t.Fatal("reporting outage invalidated an original paid receipt", response)
	}
	one, _ := json.Marshal(first["newTickets"])
	two, _ := json.Marshal(repeated["newTickets"])
	if string(one) != string(two) {
		t.Fatal("retry changed sold tickets")
	}
	owner, err := f.rd.ScratchSerialRound(input.ScratchScope, serial)
	if err != nil || owner == "" {
		t.Fatal(owner, err)
	}
}
func TestPaidScratchTicketCannotBeMutatedByOrdinaryFinanceRPCs(t *testing.T) {
	f := newScratchFixture(t)
	input, serial := f.purchase(false)
	if _, response := f.call("issue", input); !response.Success {
		t.Fatal(response)
	}
	id, _ := f.rd.ScratchSerialRound(input.ScratchScope, serial)
	round := f.svc.rounds[id]
	before, _ := f.client.HGet(context.Background(), "lottery_finance_rounds", id).Result()
	reserved := f.rd.GetAgentReserved(7, "test-scratch_1")
	ctx := context.Background()
	bet, err := f.svc.Bet(ctx, &services.BetRequest{RequestId: "extra", RoundId: id, Agent: 7, GameId: 91061, Level: 1,
		Items: []*services.BetItem{{BetId: "extra", UserId: 42, CurrencyType: "CNY", Amount: "1.00"}}})
	if err != nil || bet.Code == services.ErrorCode_OK {
		t.Fatal("ordinary Bet changed an owned ticket", bet, err)
	}
	prepay, err := f.svc.PrePay(ctx, &services.PrePayRequest{RequestId: "extra", RoundId: id, Agent: 7, GameId: 91061, Level: 1,
		BetDigest: round.Scratch.Digest, OutcomeHash: round.Scratch.OutcomeHash,
		Items: []*services.PrePayItem{{UserId: 42, CurrencyType: "CNY", Payout: "2.00"}}})
	if err != nil || prepay.Code == services.ErrorCode_OK {
		t.Fatal("ordinary PrePay changed an owned ticket", prepay, err)
	}
	renew, err := f.svc.RenewReservation(ctx, &services.RenewReservationRequest{RequestId: "extra", RoundId: id, Agent: 7, GameId: 91061, Level: 1,
		ReservationId: round.Reservation.ReservationID, ExtendSeconds: 300})
	if err != nil || renew.Code == services.ErrorCode_OK {
		t.Fatal("ordinary renewal changed an owned ticket", renew, err)
	}
	cancel, err := f.svc.CancelBet(ctx, &services.CancelBetRequest{RequestId: "extra", RoundId: id, Agent: 7, GameId: 91061, Level: 1,
		UserId: 42, CurrencyType: "CNY"})
	if err != nil || cancel.Code == services.ErrorCode_OK {
		t.Fatal("ordinary cancellation refunded an owned ticket", cancel, err)
	}
	after, _ := f.client.HGet(ctx, "lottery_finance_rounds", id).Result()
	if before != after || f.balance(42) != 99900 || !reserved.Equal(f.rd.GetAgentReserved(7, "test-scratch_1")) {
		t.Fatal("rejected mutation changed ownership, funds, or pool reservation")
	}
	claim := f.input
	claim.Serial = serial
	if _, response := f.call("reveal", claim); !response.Success || f.balance(42) != 100100 {
		t.Fatal("rejected mutation blocked original prize", response)
	}
}

func TestScratchWholeBookPurchaseIsAtomicAndMirrorFailureKeepsOwnership(t *testing.T) {
	f := newScratchFixture(t)
	input, serial := f.purchase(true)
	f.mirrorFailure = true
	_, response := f.call("issue", input)
	if response.Success || f.balance(42) != 98000 {
		t.Fatalf("expected committed ownership with mirror failure: %+v", response)
	}
	ids, _ := f.rd.ScratchPending(input.ScratchScope)
	if len(ids) != 20 {
		t.Fatal("paid book disappeared after mysql error")
	}
	f.mirrorFailure = false
	_, response = f.call("issue", input)
	if !response.Success || f.balance(42) != 98000 || len(f.mirrors) != 20 {
		t.Fatal("mirror recovery redrew or recharged the book")
	}
	changed := input
	changed.Candidates = append([]scratchCandidate(nil), input.Candidates...)
	changed.Candidates[0].Tickets = append([]scratchPlan(nil), input.Candidates[0].Tickets...)
	changed.Candidates[0].Tickets[0].Mask = append([]int(nil), input.Candidates[0].Tickets[0].Mask...)
	changed.Candidates[0].Tickets[0].Mask[4] = 1
	_, response = f.call("issue", changed)
	if response.Success || f.balance(42) != 98000 {
		t.Fatal("same purchase ID accepted changed masks")
	}
	claim := f.input
	claim.Serial = serial
	f.mirrorFailure = true
	_, response = f.call("reveal", claim)
	if response.Success || f.balance(42) != 98200 {
		t.Fatal("prize not committed exactly once before mirror error")
	}
	f.mirrorFailure = false
	_, response = f.call("reveal", claim)
	if !response.Success || f.balance(42) != 98200 {
		t.Fatal("claim mirror recovery changed money")
	}
}
func TestScratchSalePreflightDoesNotSpendReservedSeatChipsOrPartiallySell(t *testing.T) {
	f := newScratchFixture(t)
	input, _ := f.purchase(true)
	f.client.Set(context.Background(), "gogame:cash-carry:7:42:CNY", `{"token":"seat","tableKey":"7:holdem:1","revision":1,"amount":99000}`, time.Minute)
	_, response := f.call("issue", input)
	if response.Success || f.balance(42) != 100000 {
		t.Fatal("scratch spent seat-reserved balance")
	}
	ids, _ := f.rd.ScratchPending(input.ScratchScope)
	if len(ids) != 0 {
		t.Fatal("failed purchase installed partial tickets")
	}
	f.client.Del(context.Background(), "gogame:cash-carry:7:42:CNY")
	last := input.Candidates[0].Tickets[19].Serial
	f.client.Set(context.Background(), "gogame:scratch:sold:"+input.ScratchScope.Market()+":"+last, "other-owner", 0)
	_, response = f.call("issue", input)
	if response.Success || f.balance(42) != 100000 {
		t.Fatal("last sold ticket caused partial debit")
	}
}

func TestScratchManualWholeRefundClosesTheTicketWithoutASecondClaim(t *testing.T) {
	f := newScratchFixture(t)
	input, serial := f.purchase(false)
	_, response := f.call("issue", input)
	if !response.Success {
		t.Fatal(response)
	}
	id, _ := f.rd.ScratchSerialRound(input.ScratchScope, serial)
	round := f.svc.rounds[id]
	ticket := round.Scratch
	req := &services.SettlementRequest{SettlementId: "scratch:" + id + ":void", RoundId: id, GameId: round.GameID, Agent: round.Agent, Level: round.Level, ReservationId: round.Reservation.ReservationID, BetDigest: ticket.Digest, OutcomeHash: ticket.OutcomeHash, Mode: financeModeVoidRefund,
		Items: []*services.SettlementItem{{UserId: 42, CurrencyType: "CNY", BetAmount: "1.00", Payout: "1.00", Profit: "0.00", Pump: "0.00", Record: ticket.Record}}}
	refund, err := f.svc.Settlement(context.Background(), req)
	if err != nil || refund.Code != services.ErrorCode_OK || f.balance(42) != 100000 {
		t.Fatalf("refund: %+v %v", refund, err)
	}
	_, response = f.call("state", f.input)
	if !response.Success {
		t.Fatal(response)
	}
	ids, _ := f.rd.ScratchPending(f.input.ScratchScope)
	if len(ids) != 0 {
		t.Fatal("refunded ticket still pending")
	}
	claim := f.input
	claim.Serial = serial
	_, response = f.call("reveal", claim)
	if response.Success || f.balance(42) != 100000 {
		t.Fatal("refunded ticket paid again")
	}
	if !f.rd.GetAgentReserved(7, "test-scratch_1").IsZero() {
		t.Fatal("refund left prize reserved")
	}
}

func TestScratchMissingOwnershipStorageAndInsufficientPoolCannotDebit(t *testing.T) {
	f := newScratchFixture(t)
	input, _ := f.purchase(true)
	mirror := f.svc.testScratchMirror
	f.svc.testScratchMirror = nil
	_, response := f.call("issue", input)
	if response.Success || f.balance(42) != 100000 {
		t.Fatal("purchase charged before ownership schema was ready")
	}
	f.svc.testScratchMirror = mirror
	config.CfgIns.Pool.Default["test-scratch"].Pool[1].Base = decimal.Zero
	_, response = f.call("issue", input)
	if response.Success || response.Code != services.ErrorCode_NO_ENOUGH_POOL_MONEY || f.balance(42) != 100000 {
		t.Fatal("purchase did not preflight the full prize reservation")
	}
	ids, _ := f.rd.ScratchPending(input.ScratchScope)
	if len(ids) != 0 {
		t.Fatal("failed preflight installed paid tickets")
	}
}
