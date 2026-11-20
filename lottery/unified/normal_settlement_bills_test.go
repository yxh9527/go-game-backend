package unified

import (
	"app/entity"
	"encoding/json"
	"fmt"
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
