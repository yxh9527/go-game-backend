package unified

import (
	"sync"
	"testing"

	"micro_service/services"

	"github.com/shopspring/decimal"
)

func TestIsRoundArchivableRetention(t *testing.T) {
	svc := &LotteryService{}
	now := int64(1_700_000_000)

	settledFresh := &financeRound{
		State: string(financeStateSettled),
		Settlement: &financeSettlement{
			Status:      financeSettlementStatusCompleted,
			CompletedAt: now - 3600,
		},
	}
	if svc.isRoundArchivable(settledFresh, now) {
		t.Fatal("settled within 7d must not archive")
	}

	settledOld := &financeRound{
		State: string(financeStateSettled),
		Settlement: &financeSettlement{
			Status:      financeSettlementStatusCompleted,
			CompletedAt: now - financeRoundRetainSettledSeconds - 1,
		},
	}
	if !svc.isRoundArchivable(settledOld, now) {
		t.Fatal("settled beyond 7d must archive")
	}

	claimed := &financeRound{
		State: string(financeStateSettled),
		Settlement: &financeSettlement{
			Status:      financeSettlementStatusClaimed,
			CompletedAt: now - financeRoundRetainSettledSeconds - 1,
		},
	}
	if svc.isRoundArchivable(claimed, now) {
		t.Fatal("CLAIMED settlement must not archive")
	}

	expiredOld := &financeRound{
		State: string(financeStateExpired),
		Reservation: &financeReservation{
			ExpiresAt: now - financeRoundRetainExpiredSeconds - 1,
		},
	}
	if !svc.isRoundArchivable(expiredOld, now) {
		t.Fatal("expired beyond 30d must archive")
	}

	reserved := &financeRound{State: string(financeStateReserved), UpdatedAt: now - financeRoundRetainExpiredSeconds*2}
	if svc.isRoundArchivable(reserved, now) {
		t.Fatal("RESERVED must never archive")
	}
}

func TestArchiveStaleRoundsRemovesFromMemory(t *testing.T) {
	svc := &LotteryService{
		roundLock:               &sync.RWMutex{},
		rounds:                  map[string]*financeRound{},
		testSkipPoolSideEffects: true,
	}
	now := int64(1_700_000_000)
	old := &financeRound{
		RoundID: "old-settled",
		State:   string(financeStateSettled),
		Settlement: &financeSettlement{
			Status:      financeSettlementStatusCompleted,
			CompletedAt: now - financeRoundRetainSettledSeconds - 10,
		},
	}
	fresh := &financeRound{
		RoundID: "fresh-settled",
		State:   string(financeStateSettled),
		Settlement: &financeSettlement{
			Status:      financeSettlementStatusCompleted,
			CompletedAt: now - 10,
		},
	}
	svc.rounds[old.RoundID] = old
	svc.rounds[fresh.RoundID] = fresh
	n := svc.archiveStaleRoundsLocked(now, 10)
	if n != 1 {
		t.Fatalf("archived count=%d", n)
	}
	if _, ok := svc.rounds["old-settled"]; ok {
		t.Fatal("old round should be removed from memory")
	}
	if _, ok := svc.rounds["fresh-settled"]; !ok {
		t.Fatal("fresh round must remain")
	}
}

func TestCreditSettlementWalletAtomicIdempotent(t *testing.T) {
	balances := map[uint32]int64{42: 10000}
	svc := newCancelTestService(balances)
	testSettleWalletMarkers = sync.Map{}

	first, done, code := svc.creditSettlementWalletAtomic(42, 250, "settle-x")
	if code != services.ErrorCode_OK || done || first != 10250 {
		t.Fatalf("first credit bal=%d done=%v code=%v", first, done, code)
	}
	second, done, code := svc.creditSettlementWalletAtomic(42, 250, "settle-x")
	if code != services.ErrorCode_OK || !done || second != 10250 {
		t.Fatalf("second credit bal=%d done=%v code=%v", second, done, code)
	}
	if balances[42] != 10250 {
		t.Fatalf("balance mutated twice: %d", balances[42])
	}
}

func TestParseMoneyCentPrecision(t *testing.T) {
	value, err := parseMoney("1.230")
	if err != nil {
		t.Fatalf("parseMoney returned an error: %v", err)
	}
	if !value.Equal(decimal.RequireFromString("1.23")) {
		t.Fatalf("parseMoney returned %s", value)
	}

	if _, err := parseMoney("1.234"); err == nil {
		t.Fatal("parseMoney accepted precision below one cent")
	}
	if _, err := parseMoney("-1.00"); err == nil {
		t.Fatal("parseMoney accepted a negative amount")
	}
}

func TestParseSignedMoneyAllowsNegativeProfit(t *testing.T) {
	value, err := parseSignedMoney("-1.25")
	if err != nil {
		t.Fatalf("parseSignedMoney returned an error: %v", err)
	}
	if !value.Equal(decimal.RequireFromString("-1.25")) {
		t.Fatalf("parseSignedMoney returned %s", value)
	}
}

func TestPoolSymbolIncludesLevel(t *testing.T) {
	poolSymbol := buildPoolSymbol("table_game", 3)
	if poolSymbol != "table_game_3" {
		t.Fatalf("buildPoolSymbol returned %q", poolSymbol)
	}
	if symbol := baseSymbolFromPoolSymbol(poolSymbol); symbol != "table_game" {
		t.Fatalf("baseSymbolFromPoolSymbol returned %q", symbol)
	}
	level, ok := levelFromPoolSymbol(poolSymbol)
	if !ok || level != 3 {
		t.Fatalf("levelFromPoolSymbol returned level=%d ok=%v", level, ok)
	}
}

func TestIsActiveFinanceBet(t *testing.T) {
	if isActiveFinanceBet(nil) {
		t.Fatal("nil bet must not be active")
	}
	if !isActiveFinanceBet(&financeBetSnapshot{Accepted: true}) {
		t.Fatal("legacy accepted bet without status must be active")
	}
	if !isActiveFinanceBet(&financeBetSnapshot{Accepted: true, Status: financeBetStatusActive}) {
		t.Fatal("explicit ACTIVE must be active")
	}
	if isActiveFinanceBet(&financeBetSnapshot{Accepted: true, Status: financeBetStatusCanceled}) {
		t.Fatal("CANCELED must not be active")
	}
	if isActiveFinanceBet(&financeBetSnapshot{Accepted: false, Status: financeBetStatusActive}) {
		t.Fatal("not accepted must not be active")
	}
}

func TestRoundBetDigestExcludesCanceled(t *testing.T) {
	svc := &LotteryService{}
	round := &financeRound{
		Bets: map[string]*financeBetSnapshot{
			"b1": {
				BetID: "b1", UserID: 1, CurrencyType: "CNY", Amount: "10.00", AreaID: "1",
				Accepted: true, Status: financeBetStatusActive,
			},
			"b2": {
				BetID: "b2", UserID: 2, CurrencyType: "CNY", Amount: "20.00", AreaID: "2",
				Accepted: true, Status: financeBetStatusActive,
			},
		},
	}
	before := svc.roundBetDigest(round)
	round.Bets["b1"].Status = financeBetStatusCanceled
	round.Bets["b1"].Accepted = false
	after := svc.roundBetDigest(round)
	if before == "" || after == "" || before == after {
		t.Fatalf("digest should change after cancel before=%s after=%s", before, after)
	}
}

func newCancelTestService(balances map[uint32]int64) *LotteryService {
	svc := &LotteryService{
		roundLock:               &sync.RWMutex{},
		rounds:                  map[string]*financeRound{},
		testSkipPoolSideEffects: true,
	}
	svc.testCurrencyHook = func(id uint32, delta int64) (int64, services.ErrorCode) {
		cur := balances[id]
		next := cur + delta
		if next < 0 {
			return cur, services.ErrorCode_NO_ENOUGH_MONEY
		}
		balances[id] = next
		return next, services.ErrorCode_OK
	}
	return svc
}

func TestCancelBetKeepsBettingAndRefundsActiveOnly(t *testing.T) {
	balances := map[uint32]int64{101: 90000, 202: 80000} // cents
	svc := newCancelTestService(balances)
	round := &financeRound{
		RoundID:    "round-1",
		GameID:     1,
		Agent:      1,
		Level:      1,
		Symbol:     "bjl",
		PoolSymbol: "bjl_1",
		State:      string(financeStateBetting),
		Bets: map[string]*financeBetSnapshot{
			"u101-a": {
				BetID: "u101-a", UserID: 101, CurrencyType: "CNY", Amount: "50.00", AmountCNY: "50.0000",
				AgentEffectAmountCNY: "50.0000", Accepted: true, Status: financeBetStatusActive, AreaID: "1",
			},
			"u101-b": {
				BetID: "u101-b", UserID: 101, CurrencyType: "CNY", Amount: "30.00", AmountCNY: "30.0000",
				AgentEffectAmountCNY: "30.0000", Accepted: true, Status: financeBetStatusActive, AreaID: "2",
			},
			"u202-a": {
				BetID: "u202-a", UserID: 202, CurrencyType: "CNY", Amount: "40.00", AmountCNY: "40.0000",
				AgentEffectAmountCNY: "40.0000", Accepted: true, Status: financeBetStatusActive, AreaID: "1",
			},
		},
		CancelRequests: map[string]*financeCancelRequest{},
		EffectCancels:  map[string]*financeEffectCancel{},
	}
	svc.rounds[round.RoundID] = round
	runtime := &roundRuntime{
		AgentID: 1, GameID: 1, Level: 1, Symbol: "bjl", PoolSymbol: "bjl_1",
		Revenue: decimal.RequireFromString("0.05"),
	}
	digestBefore := svc.roundBetDigest(round)

	svc.roundLock.Lock()
	resp := svc.applyCancelBetLocked(round, runtime, &services.CancelBetRequest{
		RequestId:         "cancel-1",
		RoundId:           "round-1",
		GameId:            1,
		Agent:             1,
		Level:             1,
		UserId:            101,
		CurrencyType:      "CNY",
		ExpectedBetDigest: digestBefore,
	})
	svc.roundLock.Unlock()

	if !resp.Success || resp.Code != services.ErrorCode_OK {
		t.Fatalf("cancel failed code=%v reason=%s", resp.Code, resp.Reason)
	}
	if resp.State != string(financeStateBetting) {
		t.Fatalf("state must stay BETTING, got %s", resp.State)
	}
	if resp.RefundAmount != "80" && resp.RefundAmount != "80.00" {
		t.Fatalf("refundAmount=%s", resp.RefundAmount)
	}
	if resp.BetRevision != 1 {
		t.Fatalf("betRevision=%d", resp.BetRevision)
	}
	if balances[101] != 98000 {
		t.Fatalf("wallet after cancel=%d", balances[101])
	}
	if balances[202] != 80000 {
		t.Fatalf("other player wallet changed: %d", balances[202])
	}
	if isActiveFinanceBet(round.Bets["u101-a"]) || isActiveFinanceBet(round.Bets["u101-b"]) {
		t.Fatal("canceled bets must not stay ACTIVE")
	}
	if !isActiveFinanceBet(round.Bets["u202-a"]) {
		t.Fatal("other player bets must remain ACTIVE")
	}
	if len(round.EffectCancels) != 2 {
		t.Fatalf("effect cancels=%d", len(round.EffectCancels))
	}
	for betID, effect := range round.EffectCancels {
		wantKey := cancelEffectRecordKey("round-1", betID)
		if effect.RecordKey != wantKey {
			t.Fatalf("recordKey=%s want=%s", effect.RecordKey, wantKey)
		}
	}
}

func TestCancelBetIdempotentByRequestID(t *testing.T) {
	balances := map[uint32]int64{101: 90000}
	svc := newCancelTestService(balances)
	round := &financeRound{
		RoundID: "round-idem", GameID: 1, Agent: 1, Level: 1, Symbol: "bjl", PoolSymbol: "bjl_1",
		State: string(financeStateBetting),
		Bets: map[string]*financeBetSnapshot{
			"b1": {
				BetID: "b1", UserID: 101, CurrencyType: "CNY", Amount: "10.00", AmountCNY: "10.0000",
				AgentEffectAmountCNY: "10.0000", Accepted: true, Status: financeBetStatusActive,
			},
		},
		CancelRequests: map[string]*financeCancelRequest{},
		EffectCancels:  map[string]*financeEffectCancel{},
	}
	runtime := &roundRuntime{AgentID: 1, GameID: 1, Level: 1, Symbol: "bjl", PoolSymbol: "bjl_1", Revenue: decimal.Zero}
	req := &services.CancelBetRequest{
		RequestId: "same-req", RoundId: "round-idem", GameId: 1, Agent: 1, Level: 1,
		UserId: 101, CurrencyType: "CNY",
	}
	svc.roundLock.Lock()
	first := svc.applyCancelBetLocked(round, runtime, req)
	second := svc.applyCancelBetLocked(round, runtime, req)
	svc.roundLock.Unlock()
	if !first.Success || !second.Success {
		t.Fatalf("idempotent cancel must succeed")
	}
	if first.RefundAmount != second.RefundAmount || first.BetRevision != second.BetRevision {
		t.Fatalf("idempotent response mismatch")
	}
	if balances[101] != 91000 {
		t.Fatalf("wallet refunded twice: %d", balances[101])
	}
	if len(round.EffectCancels) != 1 {
		t.Fatalf("effect canceled twice: %d", len(round.EffectCancels))
	}
}

func TestCancelBetRejectedWhenNotBetting(t *testing.T) {
	balances := map[uint32]int64{101: 90000}
	svc := newCancelTestService(balances)
	round := &financeRound{
		RoundID: "round-settled", GameID: 1, Agent: 1, Level: 1, Symbol: "bjl", PoolSymbol: "bjl_1",
		State: string(financeStateSettled),
		Bets: map[string]*financeBetSnapshot{
			"b1": {BetID: "b1", UserID: 101, CurrencyType: "CNY", Amount: "10.00", AmountCNY: "10", Accepted: true, Status: financeBetStatusActive},
		},
		CancelRequests: map[string]*financeCancelRequest{},
		EffectCancels:  map[string]*financeEffectCancel{},
	}
	runtime := &roundRuntime{AgentID: 1, GameID: 1, Level: 1, Symbol: "bjl", PoolSymbol: "bjl_1"}
	svc.roundLock.Lock()
	resp := svc.applyCancelBetLocked(round, runtime, &services.CancelBetRequest{
		RequestId: "c1", RoundId: "round-settled", GameId: 1, Agent: 1, UserId: 101, CurrencyType: "CNY",
	})
	svc.roundLock.Unlock()
	if resp.Success || resp.Reason != "ROUND_NOT_BETTING" {
		t.Fatalf("expected ROUND_NOT_BETTING, got success=%v reason=%s", resp.Success, resp.Reason)
	}
	if balances[101] != 90000 {
		t.Fatal("wallet must not change")
	}
}

func TestCancelBetWalletFailureRollsNothing(t *testing.T) {
	balances := map[uint32]int64{} // missing user -> hook returns NO_ENOUGH if we set negative path; use custom hook
	svc := newCancelTestService(balances)
	svc.testCurrencyHook = func(id uint32, delta int64) (int64, services.ErrorCode) {
		return 0, services.ErrorCode_SYSTEM_ERROR
	}
	round := &financeRound{
		RoundID: "round-fail", GameID: 1, Agent: 1, Level: 1, Symbol: "bjl", PoolSymbol: "bjl_1",
		State: string(financeStateBetting),
		Bets: map[string]*financeBetSnapshot{
			"b1": {BetID: "b1", UserID: 101, CurrencyType: "CNY", Amount: "10.00", AmountCNY: "10", AgentEffectAmountCNY: "10", Accepted: true, Status: financeBetStatusActive},
		},
		CancelRequests: map[string]*financeCancelRequest{},
		EffectCancels:  map[string]*financeEffectCancel{},
	}
	runtime := &roundRuntime{AgentID: 1, GameID: 1, Level: 1, Symbol: "bjl", PoolSymbol: "bjl_1"}
	svc.roundLock.Lock()
	resp := svc.applyCancelBetLocked(round, runtime, &services.CancelBetRequest{
		RequestId: "c-fail", RoundId: "round-fail", GameId: 1, Agent: 1, UserId: 101, CurrencyType: "CNY",
	})
	svc.roundLock.Unlock()
	if resp.Success || resp.Reason != "BALANCE_UPDATE_FAILED" {
		t.Fatalf("expected balance failure, got %+v", resp)
	}
	if !isActiveFinanceBet(round.Bets["b1"]) {
		t.Fatal("bet must remain ACTIVE after wallet failure")
	}
	if round.BetRevision != 0 || len(round.EffectCancels) != 0 {
		t.Fatal("revision/effect must not change on wallet failure")
	}
}

func TestParseBetIDRevision(t *testing.T) {
	rev, ok := parseBetIDRevision("round:101:rev:3:hash")
	if !ok || rev != 3 {
		t.Fatalf("parse rev got %d ok=%v", rev, ok)
	}
	if _, ok := parseBetIDRevision("round:101:hash"); ok {
		t.Fatal("missing rev must fail")
	}
}

func TestRefundCancelWalletAtomicMarksOnlyAfterCredit(t *testing.T) {
	testCancelWalletMarkers = sync.Map{}
	balances := map[uint32]int64{101: 90000}
	svc := newCancelTestService(balances)
	// 第一次：入账并写 marker
	bal, done, code := svc.refundCancelWalletAtomic(101, 1000, "req-atomic-1", "10.00")
	if code != services.ErrorCode_OK || done || bal != 91000 {
		t.Fatalf("first refund bal=%d done=%v code=%v", bal, done, code)
	}
	// 第二次：marker 已存在，不得再次入账
	bal2, done2, code2 := svc.refundCancelWalletAtomic(101, 1000, "req-atomic-1", "10.00")
	if code2 != services.ErrorCode_OK || !done2 || bal2 != 91000 || balances[101] != 91000 {
		t.Fatalf("replay bal=%d done=%v code=%v wallet=%d", bal2, done2, code2, balances[101])
	}
}

func TestPendingClaimBlocksUntilResumed(t *testing.T) {
	testCancelWalletMarkers = sync.Map{}
	balances := map[uint32]int64{101: 90000}
	svc := newCancelTestService(balances)
	round := &financeRound{
		RoundID: "round-claim-gate", GameID: 1, Agent: 1, Level: 1, Symbol: "bjl", PoolSymbol: "bjl_1",
		State: string(financeStateBetting), BetRevision: 1,
		Bets: map[string]*financeBetSnapshot{
			"canceled": {
				BetID: "canceled", UserID: 101, CurrencyType: "CNY", Amount: "10.00", AmountCNY: "10",
				Accepted: false, Status: financeBetStatusCanceled,
			},
		},
		CancelRequests: map[string]*financeCancelRequest{
			"claim-1": {
				RequestID: "claim-1", UserID: 101, CurrencyType: "CNY",
				Status: financeCancelStatusClaimed, RefundAmount: "10.00",
				BetRevision: 1, CanceledBetIDs: []string{"canceled"},
			},
		},
		EffectCancels: map[string]*financeEffectCancel{},
	}
	runtime := &roundRuntime{AgentID: 1, GameID: 1, Level: 1, Symbol: "bjl", PoolSymbol: "bjl_1"}
	svc.roundLock.Lock()
	ok, reason := svc.ensureNoPendingCancelClaimsLocked(round, runtime)
	svc.roundLock.Unlock()
	if !ok {
		t.Fatalf("resume should complete claim, reason=%s", reason)
	}
	claim := round.CancelRequests["claim-1"]
	if claim == nil || claim.Status != financeCancelStatusCompleted || !claim.Success {
		t.Fatalf("claim not completed: %+v", claim)
	}
	if balances[101] != 91000 {
		t.Fatalf("wallet after resume=%d", balances[101])
	}
}

func TestCancelBetRequestIdIdentityConflict(t *testing.T) {
	balances := map[uint32]int64{101: 90000, 202: 90000}
	svc := newCancelTestService(balances)
	round := &financeRound{
		RoundID: "round-conflict", GameID: 1, Agent: 1, Level: 1, Symbol: "bjl", PoolSymbol: "bjl_1",
		State: string(financeStateBetting),
		Bets: map[string]*financeBetSnapshot{
			"b1": {
				BetID: "b1", UserID: 101, CurrencyType: "CNY", Amount: "10.00", AmountCNY: "10",
				AgentEffectAmountCNY: "10", AgentRevenueAmountCNY: "0.5", Accepted: true, Status: financeBetStatusActive,
			},
		},
		CancelRequests: map[string]*financeCancelRequest{},
		EffectCancels:  map[string]*financeEffectCancel{},
	}
	runtime := &roundRuntime{AgentID: 1, GameID: 1, Level: 1, Symbol: "bjl", PoolSymbol: "bjl_1"}
	svc.roundLock.Lock()
	first := svc.applyCancelBetLocked(round, runtime, &services.CancelBetRequest{
		RequestId: "shared-id", RoundId: "round-conflict", GameId: 1, Agent: 1, UserId: 101, CurrencyType: "CNY",
	})
	conflict := svc.applyCancelBetLocked(round, runtime, &services.CancelBetRequest{
		RequestId: "shared-id", RoundId: "round-conflict", GameId: 1, Agent: 1, UserId: 202, CurrencyType: "CNY",
	})
	svc.roundLock.Unlock()
	if !first.Success {
		t.Fatalf("first cancel failed: %+v", first)
	}
	if conflict.Success || conflict.Reason != "REQUEST_ID_CONFLICT" {
		t.Fatalf("expected REQUEST_ID_CONFLICT, got %+v", conflict)
	}
}

func TestCancelUsesStoredRevenueNotCurrentRuntime(t *testing.T) {
	balances := map[uint32]int64{101: 90000}
	svc := newCancelTestService(balances)
	round := &financeRound{
		RoundID: "round-rev-tax", GameID: 1, Agent: 1, Level: 1, Symbol: "bjl", PoolSymbol: "bjl_1",
		State: string(financeStateBetting),
		Bets: map[string]*financeBetSnapshot{
			"b1": {
				BetID: "b1", UserID: 101, CurrencyType: "CNY", Amount: "100.00", AmountCNY: "100",
				AgentEffectAmountCNY: "100", AgentRevenueAmountCNY: "7.0000",
				Accepted: true, Status: financeBetStatusActive,
			},
		},
		CancelRequests: map[string]*financeCancelRequest{},
		EffectCancels:  map[string]*financeEffectCancel{},
	}
	// runtime.Revenue 故意不同；冲销必须用下注时保存的 7.0000。
	runtime := &roundRuntime{
		AgentID: 1, GameID: 1, Level: 1, Symbol: "bjl", PoolSymbol: "bjl_1",
		Revenue: decimal.RequireFromString("0.99"),
	}
	svc.roundLock.Lock()
	resp := svc.applyCancelBetLocked(round, runtime, &services.CancelBetRequest{
		RequestId: "tax-1", RoundId: "round-rev-tax", GameId: 1, Agent: 1, UserId: 101, CurrencyType: "CNY",
	})
	svc.roundLock.Unlock()
	if !resp.Success {
		t.Fatalf("cancel failed: %+v", resp)
	}
	effect := round.EffectCancels["b1"]
	if effect == nil {
		t.Fatal("missing effect cancel")
	}
	if effect.RevenueAmountCNY != "-7" && effect.RevenueAmountCNY != "-7.0000" {
		t.Fatalf("revenue rollback=%q", effect.RevenueAmountCNY)
	}
}

func TestRebetAfterCancelNeedsNewBetIDViaRevision(t *testing.T) {
	// 资金侧保证取消后 revision++；游戏服 betId 必须带新 revision。
	balances := map[uint32]int64{101: 90000}
	svc := newCancelTestService(balances)
	round := &financeRound{
		RoundID: "round-rev", GameID: 1, Agent: 1, Level: 1, Symbol: "bjl", PoolSymbol: "bjl_1",
		State: string(financeStateBetting), BetRevision: 0,
		Bets: map[string]*financeBetSnapshot{
			"round-rev:101:rev:0:hash": {
				BetID: "round-rev:101:rev:0:hash", UserID: 101, CurrencyType: "CNY", Amount: "10.00", AmountCNY: "10",
				AgentEffectAmountCNY: "10", Accepted: true, Status: financeBetStatusActive,
			},
		},
		CancelRequests: map[string]*financeCancelRequest{},
		EffectCancels:  map[string]*financeEffectCancel{},
	}
	runtime := &roundRuntime{AgentID: 1, GameID: 1, Level: 1, Symbol: "bjl", PoolSymbol: "bjl_1"}
	svc.roundLock.Lock()
	resp := svc.applyCancelBetLocked(round, runtime, &services.CancelBetRequest{
		RequestId: "c-rev", RoundId: "round-rev", GameId: 1, Agent: 1, UserId: 101, CurrencyType: "CNY",
	})
	svc.roundLock.Unlock()
	if !resp.Success || resp.BetRevision != 1 {
		t.Fatalf("revision after cancel=%d success=%v", resp.BetRevision, resp.Success)
	}
	// 模拟取消后重新下注：新 betId 带 rev:1，可并存于同一 round。
	round.Bets["round-rev:101:rev:1:hash"] = &financeBetSnapshot{
		BetID: "round-rev:101:rev:1:hash", UserID: 101, CurrencyType: "CNY", Amount: "10.00", AmountCNY: "10",
		AgentEffectAmountCNY: "10", Accepted: true, Status: financeBetStatusActive,
	}
	if isActiveFinanceBet(round.Bets["round-rev:101:rev:0:hash"]) {
		t.Fatal("old revision bet must stay canceled")
	}
	if !isActiveFinanceBet(round.Bets["round-rev:101:rev:1:hash"]) {
		t.Fatal("new revision bet must be active")
	}
}
