package unified

import (
	"app/entity"
	"context"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"lottery/dao"
	"micro_service/services"
	"sync"
	"testing"
	"time"
)

func completedCancelOutput() (*financeRound, *financeCancelRequest) {
	round := debitOutputFixture()
	bet := round.Bets["original-debit:rev:0:first"]
	bet.Status = financeBetStatusCanceled
	bet.Accepted = false
	claim := &financeCancelRequest{RequestID: "original-cancel", UserID: 42, CurrencyType: "CNY", Status: financeCancelStatusCompleted, Success: true, WalletRefunded: true, RefundAmount: "1.00", Currency: "1000.00", CanceledBetIDs: []string{bet.BetID}, CompletedAt: time.Now().Unix(), BillCreatedAt: time.Now().Unix()}
	claim.BillID = originalCancelBillID(round, claim)
	round.CancelRequests = map[string]*financeCancelRequest{claim.RequestID: claim}
	return round, claim
}

func TestActualBetAndCancelRetainBothOriginalMovementsAndNeverRefundTwice(t *testing.T) {
	f := newScratchFixture(t)
	betRequest := &services.BetRequest{RequestId: "bet", RoundId: "original-debit", Agent: 7, GameId: 91061, Level: 1, Items: []*services.BetItem{{BetId: "original-debit:rev:0:first", UserId: 42, CurrencyType: "CNY", Amount: "1.00", AreaId: "1"}}}
	betResponse, err := f.svc.Bet(context.Background(), betRequest)
	if err != nil || len(betResponse.Items) != 1 || !betResponse.Items[0].Accepted {
		t.Fatal(betResponse, err)
	}
	cancel := &services.CancelBetRequest{RequestId: "original-cancel", RoundId: betRequest.RoundId, Agent: 7, GameId: 91061, Level: 1, UserId: 42, CurrencyType: "CNY", ExpectedBetDigest: betResponse.BetDigest}
	response, err := f.svc.CancelBet(context.Background(), cancel)
	if err != nil || !response.Success || response.RefundAmount != "1" || f.balance(42) != 100000 {
		t.Fatal(response, err)
	}
	round := f.svc.rounds[betRequest.RoundId]
	claim := round.CancelRequests[cancel.RequestId]
	bet := round.Bets[betRequest.Items[0].BetId]
	if bet.Accepted || bet.Status != financeBetStatusCanceled || claim.BillID == "" || claim.BillCreatedAt <= 0 || claim.BillStored {
		t.Fatal(bet, claim)
	}
	id, created := claim.BillID, claim.BillCreatedAt
	if response, err = f.svc.CancelBet(context.Background(), cancel); err != nil || !response.Success || f.balance(42) != 100000 || claim.BillID != id || claim.BillCreatedAt != created {
		t.Fatal("cancel replay changed money or original output", response, err)
	}
	f.client.HSet(context.Background(), "player_42", "currency", 88888)
	loaded := &LotteryService{rds: f.rd, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}, testSkipPoolSideEffects: true}
	loaded.loadRoundStates()
	seen := map[string]*entity.CacheBillsReq{}
	if count, err := loaded.projectBetBills(func(rows []*entity.CacheBillsReq) error {
		for _, row := range rows {
			seen[row.Desc] = row
		}
		return nil
	}); err != nil || count != 2 {
		t.Fatal(count, err)
	}
	if len(seen) != 2 || seen["bet"].Bet != -1 || seen["bet"].CurrentScore != 999 || seen["bet_cancel"].Bet != 1 || seen["bet_cancel"].CurrentScore != 1000 || seen["bet_cancel"].FlowingWaterOn != id || seen["bet_cancel"].CreateTime != created {
		t.Fatal(seen)
	}
	if f.balance(42) != 88888 {
		t.Fatal("storing refund output changed wallet")
	}
	if _, err := loaded.projectBetBills(func([]*entity.CacheBillsReq) error { t.Fatal("confirmed movements repeated"); return nil }); err != nil {
		t.Fatal(err)
	}
}

func TestCanceledOutputRequiresCompletedRefundAndExactOriginalCanceledStakes(t *testing.T) {
	for _, variant := range []string{"complete", "legacy", "claimed", "not-paid", "failed", "amount", "balance", "time", "identity", "missing-bet", "duplicate-bet", "wrong-user", "active-bet", "rejected-code"} {
		t.Run(variant, func(t *testing.T) {
			round, claim := completedCancelOutput()
			bet := round.Bets[claim.CanceledBetIDs[0]]
			switch variant {
			case "legacy":
				claim.BillID = ""
			case "claimed":
				claim.Status = financeCancelStatusClaimed
			case "not-paid":
				claim.WalletRefunded = false
			case "failed":
				claim.Success = false
			case "amount":
				claim.RefundAmount = "2.00"
			case "balance":
				claim.Currency = ""
			case "time":
				claim.CompletedAt = 0
			case "identity":
				claim.BillID = "another"
			case "missing-bet":
				round.Bets = map[string]*financeBetSnapshot{}
			case "duplicate-bet":
				claim.CanceledBetIDs = append(claim.CanceledBetIDs, claim.CanceledBetIDs[0])
			case "wrong-user":
				bet.UserID = 43
			case "active-bet":
				bet.Status = financeBetStatusActive
				bet.Accepted = true
			case "rejected-code":
				bet.Code = 1
			}
			row, err := originalCancelBill(round, claim)
			if variant == "complete" {
				if err != nil || row == nil || row.Bet != 1 || row.CurrentScore != 1000 {
					t.Fatal(row, err)
				}
				return
			}
			if variant == "legacy" || variant == "claimed" || variant == "not-paid" || variant == "failed" {
				if err != nil || row != nil {
					t.Fatal(row, err)
				}
				return
			}
			if err == nil || row != nil {
				t.Fatal("invalid refund output accepted", row, err)
			}
		})
	}
}

func TestRefundScanDoesNotStarveBehindThousandConfirmedDebitsAndExpiresWithOriginalRound(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	rd := dao.NewFinanceRedis(client)
	round, claim := completedCancelOutput()
	for _, bet := range round.Bets {
		bet.BillStored = true
	}
	for i := 0; i < 1100; i++ {
		round.Bets[fmt.Sprint(i)] = &financeBetSnapshot{BillStored: true}
	}
	svc := &LotteryService{rds: rd, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}}
	if err := svc.saveRoundLocked(round); err != nil {
		t.Fatal(err)
	}
	svc.billScanCancelsFirst = true // First pass starts with the debit collection.
	count := 0
	for pass := 0; pass < 2; pass++ {
		n, err := svc.projectBetBills(func(rows []*entity.CacheBillsReq) error {
			if len(rows) != 1 || rows[0].Desc != "bet_cancel" {
				t.Fatal(rows)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		count += n
	}
	if count != 1 || !claim.BillStored {
		t.Fatal("refund starved behind confirmed debits", count)
	}
	claim.BillStored = false
	round.State = string(financeStateExpired)
	round.Reservation = &financeReservation{ExpiresAt: time.Now().Unix() - 3600}
	if n, err := svc.projectBetBills(func([]*entity.CacheBillsReq) error { t.Fatal("expired snapshot was read"); return nil }); err != nil || n != 0 {
		t.Fatal(n, err)
	}
}
