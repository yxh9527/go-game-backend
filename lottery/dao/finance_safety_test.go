package dao

import (
	"encoding/json"
	"fmt"
	"testing"
)

func financeRaw(t *testing.T, id string, amount int) string {
	t.Helper()
	v := map[string]any{"roundId": "test-round", "agent": 7, "gameId": 91001, "level": 1, "state": "BETTING", "betRevision": 0,
		"bets": map[string]any{id: map[string]any{"betId": id, "userId": 42, "currencyType": "CNY", "amount": fmt.Sprintf("%.2f", float64(amount)/100), "accepted": true, "status": "ACTIVE"}}}
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}
func TestFinanceBetAtomicallyStoresSnapshotAndDebitsOnlyOnce(t *testing.T) {
	rd, mr := currencyTestRedis(t)
	mr.HSet("player_42", "id", "42", "agent_id", "7", "currency_type", "CNY", "currency", "10000", "cash_revision", "3")
	id := "test-round:rev:0:one"
	raw := financeRaw(t, id, 123)
	saved, already, err := rd.ApplyFinanceBet("test-round", id, raw, 42, 7, "CNY", 123)
	if err != nil || already {
		t.Fatalf("commit: %v %v", already, err)
	}
	if mr.HGet("player_42", "currency") != "9877" || mr.HGet("player_42", "cash_revision") != "4" || mr.HGet("lottery_finance_rounds", "test-round") != saved {
		t.Fatal("snapshot and debit diverged")
	}
	_, already, err = rd.ApplyFinanceBet("test-round", id, raw, 42, 7, "CNY", 123)
	if err != nil || !already || mr.HGet("player_42", "currency") != "9877" || mr.HGet("player_42", "cash_revision") != "4" {
		t.Fatal("retry changed money")
	}
	_, _, err = rd.ApplyFinanceBet("test-round", id, financeRaw(t, id, 124), 42, 7, "CNY", 124)
	if err == nil || mr.HGet("player_42", "currency") != "9877" {
		t.Fatal("changed bet ID accepted")
	}
}
func TestFinanceBetPreflightFailureLeavesNoDeductionOrSnapshot(t *testing.T) {
	for _, testCase := range []string{"wrong_type", "wrong_agent", "insufficient", "carry", "revision"} {
		t.Run(testCase, func(t *testing.T) {
			rd, mr := currencyTestRedis(t)
			mr.HSet("player_42", "id", "42", "agent_id", "7", "currency_type", "CNY", "currency", "100", "cash_revision", "3")
			switch testCase {
			case "wrong_type":
				mr.Set("lottery_finance_rounds", "broken")
			case "carry":
				mr.Set("gogame:cash-carry:7:42:CNY", `{"amount":99}`)
			case "revision":
				mr.HSet("player_42", "cash_revision", "9007199254740991")
			}
			agent := uint32(7)
			if testCase == "wrong_agent" {
				agent = 8
			}
			amount := int64(2)
			if testCase == "insufficient" {
				amount = 101
			}
			_, _, err := rd.ApplyFinanceBet("test-round", "test:rev:0:a", financeRaw(t, "test:rev:0:a", int(amount)), 42, agent, "CNY", amount)
			if err == nil || mr.HGet("player_42", "currency") != "100" {
				t.Fatalf("preflight mutated funds: %v", err)
			}
		})
	}
}
func TestFinanceCreditUpdatesCashRevisionAndPreflightsOverflow(t *testing.T) {
	rd, mr := currencyTestRedis(t)
	mr.HSet("player_42", "id", "42", "currency", "10000", "cash_revision", "3")
	a, e := rd.CreditPlayerCurrencyWithSettleMarker(42, 200, "test-revision")
	if e != nil || a.NewCurrency != 10200 || mr.HGet("player_42", "cash_revision") != "4" {
		t.Fatalf("credit: %+v %v", a, e)
	}
	_, e = rd.CreditPlayerCurrencyWithSettleMarker(42, 200, "test-revision")
	if e != nil || mr.HGet("player_42", "cash_revision") != "4" {
		t.Fatal("duplicate advanced revision")
	}
	mr.HSet("player_42", "currency", "9007199254740991")
	_, e = rd.CreditPlayerCurrencyWithSettleMarker(42, 1, "test-overflow")
	if e == nil || mr.Exists(SettleWalletMarkerKey("test-overflow", 42)) || mr.HGet("player_42", "currency") != "9007199254740991" {
		t.Fatal("overflow wrote balance or marker")
	}
}
