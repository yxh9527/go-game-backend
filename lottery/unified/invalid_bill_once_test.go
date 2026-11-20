package unified

import (
	"app/entity"
	"context"
	"encoding/json"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"lottery/dao"
	"sync"
	"testing"
	"time"
)

func TestInvalidBillLogsOnceWithoutBlockingValidOutputOrChangingMoney(t *testing.T) {
	for _, variant := range []string{"debit-balance", "debit-amount", "debit-key", "debit-code", "refund-balance", "refund-amount", "refund-key", "refund-reference"} {
		t.Run(variant, func(t *testing.T) {
			core, logs := observer.New(zap.ErrorLevel)
			restore := zap.ReplaceGlobals(zap.New(core))
			defer restore()
			mr := miniredis.RunT(t)
			client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
			t.Cleanup(func() { client.Close() })
			rd := dao.NewFinanceRedis(client)
			round := debitOutputFixture()
			originalID := "original-debit:rev:0:first"
			bad := round.Bets[originalID]
			var claim *financeCancelRequest
			if len(variant) >= 6 && variant[:6] == "refund" {
				round, claim = completedCancelOutput()
				bad = round.Bets[originalID]
				bad.BillStored = true
			}
			switch variant {
			case "debit-balance":
				bad.Currency = "unknown"
			case "debit-amount":
				bad.Amount = "-1.00"
			case "debit-key":
				bad.BetID = "different-key"
			case "debit-code":
				bad.Code = 1
			case "refund-balance":
				claim.Currency = "unknown"
			case "refund-amount":
				claim.RefundAmount = "2.00"
			case "refund-key":
				claim.RequestID = "different-key"
			case "refund-reference":
				claim.CanceledBetIDs = []string{"missing-original-bet"}
			}
			beforeBad := *bad
			var beforeClaim financeCancelRequest
			if claim != nil {
				beforeClaim = *claim
			}
			good := &financeBetSnapshot{BetID: "original-debit:rev:0:good", UserID: 43, CurrencyType: "CNY", Amount: "2.00", Currency: "998.00", Accepted: true, Code: 0, Status: financeBetStatusActive, BillCreatedAt: time.Now().Unix()}
			good.BillID = originalBetBillID(round, good)
			round.Bets[good.BetID] = good
			svc := &LotteryService{rds: rd, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}}
			if err := svc.saveRoundLocked(round); err != nil {
				t.Fatal(err)
			}
			client.HSet(context.Background(), "player_42", "currency", 76543, "cash_revision", 9)
			calls := 0
			store := func(rows []*entity.CacheBillsReq) error {
				calls++
				if len(rows) != 1 || rows[0].UserId != 43 || rows[0].Bet != -2 || rows[0].CurrentScore != 998 {
					t.Fatal("invalid record reached ES or blocked valid output", rows)
				}
				return nil
			}
			if n, err := svc.projectBetBills(store); err != nil || n != 2 || calls != 1 || logs.Len() != 1 {
				t.Fatal(n, err, calls, logs.All())
			}
			if !good.BillStored || !good.BillAttempted {
				t.Fatal("valid output was not confirmed")
			}
			if claim == nil {
				if !bad.BillAttempted || bad.BillStored {
					t.Fatal("invalid output counted as stored", bad)
				}
				actual := *bad
				actual.BillAttempted = false
				a, _ := json.Marshal(actual)
				b, _ := json.Marshal(beforeBad)
				if string(a) != string(b) {
					t.Fatal("output validation changed original financial data")
				}
			} else {
				if !claim.BillAttempted || claim.BillStored {
					t.Fatal("invalid refund counted as stored", claim)
				}
				actual := *claim
				actual.BillAttempted = false
				a, _ := json.Marshal(actual)
				b, _ := json.Marshal(beforeClaim)
				if string(a) != string(b) {
					t.Fatal("output validation changed original refund")
				}
			}
			for i := 0; i < 2; i++ {
				if n, err := svc.projectBetBills(store); err != nil || n != 0 {
					t.Fatal(n, err)
				}
			}
			loaded := &LotteryService{rds: rd, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}}
			loaded.loadRoundStates()
			if n, err := loaded.projectBetBills(store); err != nil || n != 0 || calls != 1 || logs.Len() != 1 {
				t.Fatal("invalid bill was retried after handler replacement", n, err, calls, logs.All())
			}
			if client.HGet(context.Background(), "player_42", "currency").Val() != "76543" || client.HGet(context.Background(), "player_42", "cash_revision").Val() != "9" {
				t.Fatal("skipping output changed wallet")
			}
		})
	}
}

func TestInvalidOnlyBatchesAreBoundedAndNeverSendEmptyESWrites(t *testing.T) {
	core, logs := observer.New(zap.ErrorLevel)
	restore := zap.ReplaceGlobals(zap.New(core))
	defer restore()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	rd := dao.NewFinanceRedis(client)
	round := debitOutputFixture()
	round.Bets = map[string]*financeBetSnapshot{}
	for i := 0; i < 105; i++ {
		bet := &financeBetSnapshot{BetID: fmt.Sprintf("original-debit:rev:0:%d", i), UserID: 42, CurrencyType: "CNY", Amount: "1.00", Currency: "invalid", Accepted: true, Code: 0, BillCreatedAt: time.Now().Unix()}
		bet.BillID = originalBetBillID(round, bet)
		round.Bets[bet.BetID] = bet
	}
	svc := &LotteryService{rds: rd, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}}
	if err := svc.saveRoundLocked(round); err != nil {
		t.Fatal(err)
	}
	store := func([]*entity.CacheBillsReq) error { t.Fatal("invalid or empty batch reached ES"); return nil }
	for _, want := range []int{100, 5, 0} {
		if n, err := svc.projectBetBills(store); err != nil || n != want {
			t.Fatal(n, err, want)
		}
	}
	if logs.Len() != 105 {
		t.Fatal("invalid records were logged more than once", logs.Len())
	}
}
