package dao

import (
	"context"
	"encoding/json"
	"testing"
)

func debitBillRaw(t *testing.T, id string, amount int) string {
	t.Helper()
	var draft map[string]any
	if err := json.Unmarshal([]byte(financeRaw(t, id, amount)), &draft); err != nil {
		t.Fatal(err)
	}
	bet := draft["bets"].(map[string]any)[id].(map[string]any)
	bet["code"], bet["billId"], bet["billCreatedAt"] = 0, "original-output", 1800000000
	raw, err := json.Marshal(draft)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestDebitBillMetadataIsAtomicWithMoneyAndRetryKeepsOriginalTime(t *testing.T) {
	rd, mr := currencyTestRedis(t)
	mr.HSet("player_42", "id", "42", "agent_id", "7", "currency_type", "CNY", "currency", "10000", "cash_revision", "3")
	id := "test-round:rev:0:one"
	saved, already, err := rd.ApplyFinanceBet("test-round", id, debitBillRaw(t, id, 123), 42, 7, "CNY", 123)
	if err != nil || already {
		t.Fatal(already, err)
	}
	var round map[string]any
	if err := json.Unmarshal([]byte(saved), &round); err != nil {
		t.Fatal(err)
	}
	bet := round["bets"].(map[string]any)[id].(map[string]any)
	if bet["billId"] != "original-output" || bet["billCreatedAt"] != float64(1800000000) || bet["currency"] != "98.77" || mr.HGet("player_42", "currency") != "9877" {
		t.Fatal(bet)
	}
	var retry map[string]any
	json.Unmarshal([]byte(debitBillRaw(t, id, 123)), &retry)
	retry["bets"].(map[string]any)[id].(map[string]any)["billCreatedAt"] = 1900000000
	raw, _ := json.Marshal(retry)
	second, already, err := rd.ApplyFinanceBet("test-round", id, string(raw), 42, 7, "CNY", 123)
	if err != nil || !already || second != saved || mr.HGet("player_42", "cash_revision") != "4" {
		t.Fatal("retry changed original output or money", already, err)
	}
}

func TestDebitBillConfirmationPreservesOtherBetsAndDoesNotRecreateMissingRounds(t *testing.T) {
	rd, mr := currencyTestRedis(t)
	ctx := context.Background()
	id := "test-round:rev:0:one"
	raw := debitBillRaw(t, id, 123)
	var round map[string]any
	json.Unmarshal([]byte(raw), &round)
	round["updatedAt"] = 1000
	round["bets"].(map[string]any)["other"] = map[string]any{"betId": "other", "accepted": true, "code": 0, "currency": "99.00"}
	encoded, _ := json.Marshal(round)
	mr.HSet("lottery_finance_rounds", "test-round", string(encoded))
	mr.HSet("player_42", "currency", "9999", "cash_revision", "9")
	if ok, err := rd.ConfirmFinanceBetBill(ctx, "test-round", id, "wrong-output"); err != nil || ok {
		t.Fatal(ok, err)
	}
	if mr.HGet("lottery_finance_rounds", "test-round") != string(encoded) {
		t.Fatal("wrong output modified snapshot")
	}
	if ok, err := rd.ConfirmFinanceBetBill(ctx, "test-round", id, "original-output"); err != nil || !ok {
		t.Fatal(ok, err)
	}
	json.Unmarshal([]byte(mr.HGet("lottery_finance_rounds", "test-round")), &round)
	if round["updatedAt"] != float64(1000) || round["bets"].(map[string]any)["other"] == nil || round["bets"].(map[string]any)[id].(map[string]any)["billStored"] != true {
		t.Fatal(round)
	}
	if mr.HGet("player_42", "currency") != "9999" || mr.HGet("player_42", "cash_revision") != "9" {
		t.Fatal("delivery confirmation changed money")
	}
	if ok, err := rd.ConfirmFinanceBetBill(ctx, "test-round", id, "original-output"); err != nil || !ok {
		t.Fatal("repeat confirmation failed", ok, err)
	}
	mr.HDel("lottery_finance_rounds", "test-round")
	if ok, err := rd.ConfirmFinanceBetBill(ctx, "test-round", id, "original-output"); err != nil || ok || mr.HGet("lottery_finance_rounds", "test-round") != "" {
		t.Fatal("missing snapshot recreated", ok, err)
	}
}

func TestNewDebitMergesConcurrentOutputAcknowledgementWithoutLosingOrRepeatingMoney(t *testing.T) {
	rd, mr := currencyTestRedis(t)
	ctx := context.Background()
	mr.HSet("player_42", "id", "42", "agent_id", "7", "currency_type", "CNY", "currency", "10000", "cash_revision", "3")
	first := "test-round:rev:0:one"
	saved, _, err := rd.ApplyFinanceBet("test-round", first, debitBillRaw(t, first, 123), 42, 7, "CNY", 123)
	if err != nil {
		t.Fatal(err)
	}
	var stale map[string]any
	json.Unmarshal([]byte(saved), &stale)
	if ok, err := rd.ConfirmFinanceBetBill(ctx, "test-round", first, "original-output"); err != nil || !ok {
		t.Fatal(ok, err)
	}
	second := "test-round:rev:0:two"
	var addition map[string]any
	json.Unmarshal([]byte(debitBillRaw(t, second, 200)), &addition)
	addition["bets"].(map[string]any)[second].(map[string]any)["billId"] = "second-output"
	stale["bets"].(map[string]any)[second] = addition["bets"].(map[string]any)[second]
	raw, _ := json.Marshal(stale)
	updated, already, err := rd.ApplyFinanceBet("test-round", second, string(raw), 42, 7, "CNY", 200)
	if err != nil || already {
		t.Fatal("concurrent delivery blocked the next legitimate debit", already, err)
	}
	var round map[string]any
	json.Unmarshal([]byte(updated), &round)
	if round["bets"].(map[string]any)[first].(map[string]any)["billStored"] != true || mr.HGet("player_42", "currency") != "9677" || mr.HGet("player_42", "cash_revision") != "5" {
		t.Fatal("acknowledgement lost or money replayed", round)
	}
}

func TestRejectedDebitNeverStoresOriginalOutputMetadata(t *testing.T) {
	rd, mr := currencyTestRedis(t)
	mr.HSet("player_42", "id", "42", "agent_id", "7", "currency_type", "CNY", "currency", "10", "cash_revision", "3")
	if _, _, err := rd.ApplyFinanceBet("test-round", "test:rev:0:one", debitBillRaw(t, "test:rev:0:one", 123), 42, 7, "CNY", 123); err == nil {
		t.Fatal("insufficient debit accepted")
	}
	if mr.Exists("lottery_finance_rounds") || mr.HGet("player_42", "currency") != "10" {
		t.Fatal("rejected debit emitted output or changed funds")
	}
}
