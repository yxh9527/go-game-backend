package view

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
)

func TestBillNormalizeAwardAndZeroBetOnRebate(t *testing.T) {
	raw := []byte(`{"desc":"返奖","bet":-2.5,"award":1.5,"currentScore":99952.95}`)
	bill := &Bill{}
	if err := json.Unmarshal(raw, bill); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	bill.Normalize()
	if !bill.Bets.IsZero() {
		t.Fatalf("bet want 0, got %s", bill.Bets)
	}
	wantAward := decimal.RequireFromString("1.5")
	if !bill.Award.Equal(wantAward) {
		t.Fatalf("award want 1.5, got %s", bill.Award)
	}

	out, err := json.Marshal(bill)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(out, &probe); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if _, ok := probe["award"]; !ok {
		t.Fatal("award missing from json")
	}
}

func TestBillNormalizeKeepsBetOnWager(t *testing.T) {
	bill := &Bill{Msg: "下注", Bets: decimal.RequireFromString("-2.5")}
	bill.Normalize()
	if !bill.Bets.Equal(decimal.RequireFromString("-2.5")) {
		t.Fatalf("bet want -2.5, got %s", bill.Bets)
	}
	if !bill.Award.IsZero() {
		t.Fatalf("award want 0, got %s", bill.Award)
	}
}
