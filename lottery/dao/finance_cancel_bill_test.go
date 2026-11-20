package dao

import (
	"context"
	"encoding/json"
	"testing"
)

func TestCancelBillConfirmationRequiresCompletedExactRefundAndNeverRecreatesSnapshot(t *testing.T) {
	for _, variant := range []string{"complete", "claimed", "not-paid", "failed", "wrong-id", "missing"} {
		t.Run(variant, func(t *testing.T) {
			rd, mr := currencyTestRedis(t)
			ctx := context.Background()
			claim := map[string]any{"requestId": "cancel", "billId": "original-output", "status": "COMPLETED", "success": true, "walletRefunded": true}
			switch variant {
			case "claimed":
				claim["status"] = "CLAIMED"
			case "not-paid":
				claim["walletRefunded"] = false
			case "failed":
				claim["success"] = false
			}
			round := map[string]any{"updatedAt": 1000, "cancelRequests": map[string]any{"cancel": claim}, "bets": map[string]any{"new-bet": map[string]any{"accepted": true}}}
			raw, _ := json.Marshal(round)
			if variant != "missing" {
				mr.HSet("lottery_finance_rounds", "round", string(raw))
			}
			id := "original-output"
			if variant == "wrong-id" {
				id = "wrong"
			}
			ok, err := rd.ConfirmFinanceCancelBill(ctx, "round", "cancel", id)
			if err != nil {
				t.Fatal(err)
			}
			if variant == "complete" {
				if !ok {
					t.Fatal("confirmed refund not acknowledged")
				}
				json.Unmarshal([]byte(mr.HGet("lottery_finance_rounds", "round")), &round)
				if round["updatedAt"] != float64(1000) || round["bets"].(map[string]any)["new-bet"] == nil || round["cancelRequests"].(map[string]any)["cancel"].(map[string]any)["billStored"] != true {
					t.Fatal(round)
				}
			} else if ok || variant == "missing" && mr.Exists("lottery_finance_rounds") || variant != "missing" && mr.HGet("lottery_finance_rounds", "round") != string(raw) {
				t.Fatal("unconfirmed refund mutated snapshot", ok)
			}
		})
	}
}

func TestCanceledDebitStillAcknowledgesItsOriginalCashMovement(t *testing.T) {
	rd, mr := currencyTestRedis(t)
	id := "test-round:rev:0:one"
	var round map[string]any
	json.Unmarshal([]byte(debitBillRaw(t, id, 123)), &round)
	bet := round["bets"].(map[string]any)[id].(map[string]any)
	bet["accepted"] = false
	bet["status"] = "CANCELED"
	raw, _ := json.Marshal(round)
	mr.HSet("lottery_finance_rounds", "test-round", string(raw))
	if ok, err := rd.ConfirmFinanceBetBill(context.Background(), "test-round", id, "original-output"); err != nil || !ok {
		t.Fatal("cancel erased original debit acknowledgement", ok, err)
	}
}

func TestNewDebitPreservesConcurrentRefundOutputConfirmation(t *testing.T) {
	rd, mr := currencyTestRedis(t)
	id := "test-round:rev:0:new"
	mr.HSet("player_42", "id", "42", "agent_id", "7", "currency_type", "CNY", "currency", "10000", "cash_revision", "3")
	var original map[string]any
	json.Unmarshal([]byte(debitBillRaw(t, id, 123)), &original)
	original["bets"] = map[string]any{}
	original["cancelRequests"] = map[string]any{"cancel": map[string]any{"billId": "refund-output", "status": "COMPLETED", "success": true, "walletRefunded": true}}
	raw, _ := json.Marshal(original)
	mr.HSet("lottery_finance_rounds", "test-round", string(raw))
	if ok, err := rd.ConfirmFinanceCancelBill(context.Background(), "test-round", "cancel", "refund-output"); err != nil || !ok {
		t.Fatal(ok, err)
	}
	var draft map[string]any
	json.Unmarshal([]byte(debitBillRaw(t, id, 123)), &draft)
	draft["cancelRequests"] = original["cancelRequests"]
	raw, _ = json.Marshal(draft)
	saved, already, err := rd.ApplyFinanceBet("test-round", id, string(raw), 42, 7, "CNY", 123)
	if err != nil || already {
		t.Fatal(already, err)
	}
	json.Unmarshal([]byte(saved), &draft)
	if draft["cancelRequests"].(map[string]any)["cancel"].(map[string]any)["billStored"] != true || mr.HGet("player_42", "currency") != "9877" {
		t.Fatal("old draft overwrote refund output acknowledgement", draft)
	}
}
