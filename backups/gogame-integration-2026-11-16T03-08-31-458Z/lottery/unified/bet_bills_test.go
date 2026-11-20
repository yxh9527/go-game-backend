package unified

import (
	"app/entity"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"lottery/dao"
	"micro_service/services"
	"sync"
	"testing"
	"time"
)

func debitOutputFixture() *financeRound {
	round := &financeRound{RoundID: "original-debit", Agent: 7, GameID: 92058, Level: 1, Symbol: "baccarat", State: "BETTING", UpdatedAt: time.Now().Unix(), Bets: map[string]*financeBetSnapshot{}}
	bet := &financeBetSnapshot{BetID: "original-debit:rev:0:first", UserID: 42, CurrencyType: "CNY", Amount: "1.00", Currency: "999.00", Accepted: true, Status: financeBetStatusActive, Code: 0, BillCreatedAt: time.Now().Unix()}
	bet.BillID = originalBetBillID(round, bet)
	round.Bets[bet.BetID] = bet
	return round
}

func TestActualUnifiedBetStoresOriginalOutputWithDebitAndDuplicateCannotRegenerateIt(t *testing.T) {
	f := newScratchFixture(t)
	request := &services.BetRequest{RequestId: "debit-request", RoundId: "original-debit", Agent: 7, GameId: 91061, Level: 1, Items: []*services.BetItem{{BetId: "original-debit:rev:0:first", UserId: 42, CurrencyType: "CNY", Amount: "1.00", AreaId: "1"}}}
	response, err := f.svc.Bet(context.Background(), request)
	if err != nil || response.Code != 0 || len(response.Items) != 1 || !response.Items[0].Accepted || f.balance(42) != 99900 {
		t.Fatal(response, err)
	}
	round := f.svc.rounds[request.RoundId]
	bet := round.Bets[request.Items[0].BetId]
	if bet.BillID == "" || bet.BillCreatedAt <= 0 || bet.BillStored {
		t.Fatal("successful debit did not capture output identity", bet)
	}
	id, created := bet.BillID, bet.BillCreatedAt
	raw, err := f.rd.HGet(financeRoundHash, request.RoundId)
	if err != nil {
		t.Fatal(err)
	}
	var saved financeRound
	if err := json.Unmarshal([]byte(raw), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Bets[bet.BetID].BillID != id || saved.Bets[bet.BetID].Currency != "999.00" {
		t.Fatal("money and original debit metadata diverged", saved)
	}
	response, err = f.svc.Bet(context.Background(), request)
	if err != nil || !response.Items[0].Accepted || f.balance(42) != 99900 || round.Bets[bet.BetID].BillID != id || round.Bets[bet.BetID].BillCreatedAt != created {
		t.Fatal("retry debited or regenerated output", response, err)
	}
	f.client.HSet(context.Background(), "player_42", "currency", 88888)
	replacement := &LotteryService{rds: f.rd, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}, testSkipPoolSideEffects: true}
	replacement.loadRoundStates()
	if count, err := replacement.projectBetBills(func(rows []*entity.CacheBillsReq) error {
		if len(rows) != 1 || rows[0].Bet != -1 || rows[0].CurrentScore != 999 || rows[0].FlowingWaterOn != id || rows[0].CreateTime != created || rows[0].Desc != "bet" {
			t.Fatal("bill used current wallet or changed original", rows)
		}
		return nil
	}); err != nil || count != 1 {
		t.Fatal(count, err)
	}
	if f.balance(42) != 88888 {
		t.Fatal("output storage replayed money")
	}
	if _, err := replacement.projectBetBills(func([]*entity.CacheBillsReq) error { t.Fatal("confirmed debit re-emitted"); return nil }); err != nil {
		t.Fatal(err)
	}
}

func TestDebitOutputValidationDoesNotInventLegacyZeroRejectedOrChangedOriginals(t *testing.T) {
	for _, variant := range []string{"legacy", "zero", "rejected", "amount", "balance", "time", "identity", "already-stored", "canceled"} {
		t.Run(variant, func(t *testing.T) {
			round := debitOutputFixture()
			bet := round.Bets["original-debit:rev:0:first"]
			switch variant {
			case "legacy":
				bet.BillID = ""
			case "zero":
				bet.Amount = "0.00"
			case "rejected":
				bet.Accepted = false
			case "amount":
				bet.Amount = "-1.00"
			case "balance":
				bet.Currency = ""
			case "time":
				bet.BillCreatedAt = 0
			case "identity":
				bet.BillID = "different"
			case "already-stored":
				bet.BillStored = true
			case "canceled":
				bet.Status = financeBetStatusCanceled
				bet.Accepted = false
			}
			row, err := originalBetBill(round, bet)
			if variant == "legacy" || variant == "already-stored" {
				if err != nil || row != nil {
					t.Fatal(row, err)
				}
				return
			}
			if variant == "canceled" {
				if err != nil || row == nil || row.Bet != -1 {
					t.Fatal("original debit disappeared after cancellation", row, err)
				}
				return
			}
			if err == nil || row != nil {
				t.Fatal("unconfirmed debit emitted", row, err)
			}
		})
	}
}

func TestDebitOutputSurvivesSQLFailureAndLostDeliveryAckWithoutReplayingMoney(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	rd := dao.NewFinanceRedis(client)
	round := debitOutputFixture()
	bet := round.Bets["original-debit:rev:0:first"]
	svc := &LotteryService{rds: rd, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}}
	if err := svc.saveRoundLocked(round); err != nil {
		t.Fatal(err)
	}
	client.HSet(context.Background(), "player_42", "currency", 88888, "cash_revision", 5)
	if _, err := svc.projectBetBills(func([]*entity.CacheBillsReq) error { return errors.New("SQL unavailable") }); err == nil || bet.BillStored {
		t.Fatal("failed SQL acknowledged original output", err)
	}
	stored := map[string]string{}
	calls := 0
	store := func(rows []*entity.CacheBillsReq) error {
		calls++
		for _, row := range rows {
			raw, _ := json.Marshal(row)
			if old, ok := stored[row.FlowingWaterOn]; ok && old != string(raw) {
				t.Fatal("repeated original output changed")
			}
			stored[row.FlowingWaterOn] = string(raw)
		}
		return nil
	}
	// SQL accepted the original, but confirmation was lost. A new handler must
	// resend the identical output; it cannot debit or read a newer wallet balance.
	if err := store([]*entity.CacheBillsReq{func() *entity.CacheBillsReq {
		row, err := originalBetBill(round, bet)
		if err != nil {
			t.Fatal(err)
		}
		return row
	}()}); err != nil {
		t.Fatal(err)
	}
	replacement := &LotteryService{rds: rd, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}}
	replacement.loadRoundStates()
	if n, err := replacement.projectBetBills(store); err != nil || n != 1 || calls != 2 || len(stored) != 1 {
		t.Fatal(n, err, calls, stored)
	}
	if client.HGet(context.Background(), "player_42", "currency").Val() != "88888" || client.HGet(context.Background(), "player_42", "cash_revision").Val() != "5" {
		t.Fatal("delivery modified money")
	}
}

func TestDebitOutputRespectsBatchLimitsTerminalExpiryAndBackpressure(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	rd := dao.NewFinanceRedis(client)
	round := debitOutputFixture()
	round.Bets = map[string]*financeBetSnapshot{}
	for i := 0; i < 145; i++ {
		bet := &financeBetSnapshot{BetID: fmt.Sprintf("original-debit:rev:0:%d", i), UserID: 42, CurrencyType: "CNY", Amount: "1.00", Currency: "999.00", Accepted: true, Code: 0, BillCreatedAt: time.Now().Unix()}
		bet.BillID = originalBetBillID(round, bet)
		round.Bets[bet.BetID] = bet
	}
	svc := &LotteryService{rds: rd, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}}
	if err := svc.saveRoundLocked(round); err != nil {
		t.Fatal(err)
	}
	counts := []int{}
	store := func(rows []*entity.CacheBillsReq) error { counts = append(counts, len(rows)); return nil }
	if n, err := svc.projectBetBills(store); err != nil || n != 100 {
		t.Fatal(n, err)
	}
	if n, err := svc.projectBetBills(store); err != nil || n != 45 {
		t.Fatal(n, err)
	}
	if fmt.Sprint(counts) != "[100 45]" {
		t.Fatal(counts)
	}
	for _, bet := range round.Bets {
		bet.BillStored = false
	}
	round.State = string(financeStateSettled)
	round.Settlement = &financeSettlement{CompletedAt: time.Now().Unix() - 3600}
	if n, err := svc.projectBetBills(func([]*entity.CacheBillsReq) error { t.Fatal("expired snapshot output was read"); return nil }); err != nil || n != 0 {
		t.Fatal(n, err)
	}
	round.State = string(financeStateBetting)
	round.Settlement = nil
	round.Bets = map[string]*financeBetSnapshot{}
	for i := 0; i < 50000; i++ {
		round.Bets[fmt.Sprint(i)] = &financeBetSnapshot{BillID: "original"}
	}
	if !svc.reportBacklogFullLocked(time.Now().Unix()) {
		t.Fatal("pending debit outputs bypassed original reporting backpressure")
	}
}
