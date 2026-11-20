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
	"math"
	"sync"
	"testing"
	"time"
)

func normalBillRound(mode string, payouts ...float64) *financeRound {
	round := reportRound(len(payouts))
	round.Settlement.Mode = mode
	round.Settlement.Status = financeSettlementStatusClaimed
	round.Settlement.CompletedAt = 0
	for i, payout := range payouts {
		round.Settlement.Reports[i].Win = payout
	}
	round.Settlement.Bills = normalSettlementBills(round, round.Settlement)
	return round
}

func completeNormalBillRound(round *financeRound) {
	round.Settlement.Status = financeSettlementStatusCompleted
	round.Settlement.CompletedAt = time.Now().Unix()
	if round.Settlement.Mode == financeModeVoidRefund {
		round.State = string(financeStateVoided)
	}
}

func TestNormalPayoutBillsAreFrozenBeforeCreditAndOnlyConfirmedBalancesAreProjected(t *testing.T) {
	for _, mode := range []string{financeModeNormal, financeModeVoidRefund} {
		t.Run(mode, func(t *testing.T) {
			round := normalBillRound(mode, 1, 0, 1)
			if len(round.Settlement.Bills) != 2 {
				t.Fatal("zero payout invented a cash movement")
			}
			first := *round.Settlement.Bills[0]
			if first.Bet != 1 || first.CurrentScore != 0 || first.CreateTime <= 0 || first.FlowingWaterOn == "" {
				t.Fatal(first)
			}
			if rows, err := settlementBillBatch(round); err != nil || len(rows) != 0 {
				t.Fatal("unconfirmed credit was projected", rows, err)
			}
			raw, _ := json.Marshal(round)
			var loaded financeRound
			if err := json.Unmarshal(raw, &loaded); err != nil {
				t.Fatal(err)
			}
			completeNormalBillRound(&loaded)
			rows, err := settlementBillBatch(&loaded)
			if err != nil || len(rows) != 2 || rows[0].FlowingWaterOn != first.FlowingWaterOn || rows[0].CreateTime != first.CreateTime || rows[0].CurrentScore != 12.34 {
				t.Fatal(rows, err)
			}
			if loaded.Settlement.Bills[0].CurrentScore != 0 {
				t.Fatal("projection changed original frozen payload")
			}
			want := "settlement"
			if mode == financeModeVoidRefund {
				want = "void_refund"
			}
			if rows[0].Desc != want {
				t.Fatal(rows[0])
			}
		})
	}
}

func TestNormalPayoutBillChecksOriginalModeAmountAndAllPayableAccounts(t *testing.T) {
	for _, mode := range []string{financeModeNormal, financeModeVoidRefund} {
		for _, variant := range []string{"amount", "negative", "description", "missing-account", "zero-payout", "duplicate", "state", "nonfinite"} {
			t.Run(mode+"/"+variant, func(t *testing.T) {
				round := normalBillRound(mode, 1, 1)
				completeNormalBillRound(round)
				switch variant {
				case "amount":
					round.Settlement.Bills[0].Bet = 2
				case "negative":
					round.Settlement.Bills[0].Bet = -1
				case "description":
					round.Settlement.Bills[0].Desc = "other"
				case "missing-account":
					round.Settlement.Bills = round.Settlement.Bills[:1]
				case "zero-payout":
					round.Settlement.Reports[0].Win = 0
				case "duplicate":
					round.Settlement.Bills = append(round.Settlement.Bills, round.Settlement.Bills[0])
				case "state":
					if mode == financeModeVoidRefund {
						round.State = string(financeStateSettled)
					} else {
						round.State = string(financeStateVoided)
					}
				case "nonfinite":
					round.Settlement.Reports[0].Win = math.NaN()
				}
				if _, err := settlementBillBatch(round); err == nil {
					t.Fatal("invalid payout bill accepted", variant)
				}
			})
		}
	}
}

func TestNormalPayoutSQLAndESFailureRetriesOriginalBillsWithoutReplayingCredits(t *testing.T) {
	for _, mode := range []string{financeModeNormal, financeModeVoidRefund} {
		t.Run(mode, func(t *testing.T) {
			mr := miniredis.RunT(t)
			client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
			t.Cleanup(func() { client.Close() })
			rd := dao.NewFinanceRedis(client)
			round := normalBillRound(mode, 1, 0, 1)
			completeNormalBillRound(round)
			old := &LotteryService{rds: rd, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}}
			if err := old.saveRoundLocked(round); err != nil {
				t.Fatal(err)
			}
			client.HSet(context.Background(), "player_42", "currency", 9999, "cash_revision", 8)
			es, outage := billES(t)
			loaded := &LotteryService{rds: rd, es: es, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}}
			loaded.loadRoundStates()
			sqlDown := true
			stored := map[string]string{}
			calls := 0
			store := func(rows []*entity.CacheBillsReq) error {
				calls++
				if sqlDown {
					return errors.New("SQL unavailable")
				}
				for _, row := range rows {
					raw, _ := json.Marshal(row)
					if previous, ok := stored[row.FlowingWaterOn]; ok && previous != string(raw) {
						t.Fatal("retry changed original payout bill")
					}
					if row.CurrentScore != 12.34 {
						t.Fatal("original credit balance replaced by current wallet", row)
					}
					stored[row.FlowingWaterOn] = string(raw)
				}
				return nil
			}
			if _, err := loaded.projectSettlementReportsWithBills(1, store); err == nil {
				t.Fatal("SQL outage was hidden")
			}
			if loaded.rounds[round.RoundID].Settlement.BillOffset != 0 || loaded.rounds[round.RoundID].Settlement.ReportsIndexed {
				t.Fatal("failed bill save advanced progress")
			}
			sqlDown = false
			*outage = true
			if _, err := loaded.projectSettlementReportsWithBills(1, store); err == nil {
				t.Fatal("ES outage was hidden")
			}
			if len(stored) != 2 {
				t.Fatal("bills not received before ES", stored)
			}
			replacement := &LotteryService{rds: rd, es: es, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}}
			replacement.loadRoundStates()
			*outage = false
			if _, err := replacement.projectSettlementReportsWithBills(1, store); err != nil {
				t.Fatal(err)
			}
			completed := replacement.rounds[round.RoundID].Settlement
			if !completed.ReportsIndexed || completed.BillOffset != 2 || len(stored) != 2 {
				t.Fatal(completed, stored)
			}
			before := calls
			if _, err := replacement.projectSettlementReportsWithBills(1, store); err != nil || calls != before {
				t.Fatal("completed bill repeated", err, calls)
			}
			if client.HGet(context.Background(), "player_42", "currency").Val() != "9999" || client.HGet(context.Background(), "player_42", "cash_revision").Val() != "8" {
				t.Fatal("original bill persistence replayed credit")
			}
		})
	}
}

func TestNormalPayoutBillBatchIsBoundedAndLegacyOrZeroPayoutDoesNotInventBills(t *testing.T) {
	payouts := make([]float64, 145)
	for i := range payouts {
		payouts[i] = 1
	}
	round := normalBillRound(financeModeNormal, payouts...)
	completeNormalBillRound(round)
	es, _ := billES(t)
	svc := &LotteryService{es: es, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{round.RoundID: round}}
	counts := []int{}
	store := func(rows []*entity.CacheBillsReq) error { counts = append(counts, len(rows)); return nil }
	if _, err := svc.projectSettlementReportsWithBills(1, store); err != nil {
		t.Fatal(err)
	}
	if round.Settlement.BillOffset != 100 || round.Settlement.ReportsIndexed {
		t.Fatal(round.Settlement)
	}
	if !svc.isRoundExpired(round, time.Now().Unix()+financeRoundRetainSeconds+1) {
		t.Fatal("pending payout bills extended snapshot retention")
	}
	if _, err := svc.projectSettlementReportsWithBills(1, store); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(counts) != "[100 45]" || !round.Settlement.ReportsIndexed {
		t.Fatal(counts, round.Settlement)
	}
	legacy := reportRound(1)
	if rows, err := settlementBillBatch(legacy); err != nil || len(rows) != 0 {
		t.Fatal("legacy bill reconstructed", rows, err)
	}
	zero := normalBillRound(financeModeNormal, 0)
	completeNormalBillRound(zero)
	if rows, err := settlementBillBatch(zero); err != nil || len(rows) != 0 {
		t.Fatal("zero payout invented a bill", rows, err)
	}
}
