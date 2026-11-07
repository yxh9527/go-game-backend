package dao

import (
	"context"
	"testing"
)

func TestCashRejectsInvalidAndOverBalanceAmounts(t *testing.T) {
	for _, tc := range []struct {
		name, kind string
		amount     int64
	}{
		{"negative debit", "debit", -1},
		{"negative credit", "credit", -1},
		{"negative reserve", "reserve", -1},
		{"over balance", "debit", 1001},
		{"over balance reserve", "reserve", 1001},
		{"unsafe amount", "debit", 9007199254740992},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, server, account := cashFixture(t)
			request := CashRequest{ID: "invalid-amount", Accounts: []CashAccount{account}, Steps: []CashStep{{ID: "step", Account: 1, Kind: tc.kind, Amount: tc.amount, Token: "token", TableKey: "7:holdem:1"}}}
			if _, err := CashApply(context.Background(), client, request); err == nil {
				t.Fatal("invalid amount accepted")
			}
			if server.HGet("player_42", "currency") != "1000" || server.Exists(cashKeys(account)[1]) {
				t.Fatal("rejected request modified balance or carry")
			}
		})
	}
}

func TestCashRPCRejectsNonIntegerAmountTypes(t *testing.T) {
	for _, raw := range []string{
		`{"id":"x","accounts":[],"steps":[{"amount":1.5}]}`,
		`{"id":"x","accounts":[],"steps":[{"amount":"100"}]}`,
		`{"id":"x","accounts":[],"steps":[{"amount":true}]}`,
		`{"id":"x","accounts":[],"steps":[{"amount":1e309}]}`,
		`{"id":"x","accounts":[],"steps":[{"amount":[]}]}`,
	} {
		var request CashRequest
		if err := DecodeGogameJSON(raw, &request); err == nil {
			t.Fatal("malformed monetary type accepted", raw)
		}
	}
}
