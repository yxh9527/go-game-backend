package dao

import (
	"encoding/json"
	"testing"
)

func TestCarryWagerAndPayoutAtomicallyUpdateWalletAndTheOriginalReservation(t *testing.T) {
	rd, mr := currencyTestRedis(t)
	mr.HSet("player_42", "id", "42", "agent_id", "7", "currency_type", "CNY", "currency", "10000", "cash_revision", "3")
	mr.Set("gogame:cash-carry:7:42:CNY", `{"amount":9000,"token":"seat-token","tableKey":"7:holdem:1","revision":5}`)
	id := "r:rev:0:1"
	var raw map[string]any
	json.Unmarshal([]byte(financeRaw(t, id, 2000)), &raw)
	raw["bets"].(map[string]any)[id].(map[string]any)["carry"] = map[string]any{"token": "seat-token", "tableKey": "7:holdem:1", "revision": 5}
	encoded, _ := json.Marshal(raw)
	_, _, err := rd.ApplyFinanceBet("test-round", id, string(encoded), 42, 7, "CNY", 2000)
	if err != nil {
		t.Fatal(err)
	}
	if mr.HGet("player_42", "currency") != "8000" {
		t.Fatal("wager did not debit once")
	}
	var carry struct{ Amount, Revision int64 }
	stored, _ := mr.Get("gogame:cash-carry:7:42:CNY")
	json.Unmarshal([]byte(stored), &carry)
	if carry.Amount != 7000 || carry.Revision != 6 {
		t.Fatal("wager did not reduce the original reservation")
	}
	binding := FinanceCarryCredit{Token: "seat-token", TableKey: "7:holdem:1", Revision: 6, Agent: 7, Currency: "CNY"}
	result, err := rd.CreditPlayerCurrencyWithSettleMarker(42, 3000, "seat-settle", binding)
	if err != nil || result.NewCurrency != 11000 {
		t.Fatalf("payout: %+v %v", result, err)
	}
	_, err = rd.CreditPlayerCurrencyWithSettleMarker(42, 3000, "seat-settle", binding)
	if err != nil {
		t.Fatal(err)
	}
	stored, _ = mr.Get("gogame:cash-carry:7:42:CNY")
	json.Unmarshal([]byte(stored), &carry)
	if carry.Amount != 10000 || carry.Revision != 7 || mr.HGet("player_42", "currency") != "11000" || mr.HGet("player_42", "cash_revision") != "5" {
		t.Fatal("payout replay changed carry or wallet")
	}
}
