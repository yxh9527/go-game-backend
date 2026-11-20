package unified

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestScratchBookReceiptMaintenanceOnlyUsesConfirmedOriginalMirrors(t *testing.T) {
	f := newScratchFixture(t)
	input, _ := f.purchase(true)
	f.mirrorFailure = true
	_, response := f.call("issue", input)
	if response.Success || f.balance(42) != 98000 {
		t.Fatal("original paid batch expected", response)
	}
	original, err := f.rd.ScratchReceipt(input.ScratchScope, input.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	_, n, err := f.rd.ShortenDeliveredScratchReceipts(0)
	if err != nil || n != 0 {
		t.Fatal("unmirrored batch retired", n, err)
	}
	current, _ := f.rd.ScratchReceipt(input.ScratchScope, input.RequestID)
	if current != original || len(f.mirrors) != 0 || f.balance(42) != 98000 {
		t.Fatal("maintenance delivered or charged an unconfirmed batch")
	}
	// Mirror the original committed tickets, then restore the old receipt to
	// represent a lost final compaction. No further purchase request is sent.
	f.mirrorFailure = false
	_, response = f.call("issue", input)
	if !response.Success || len(f.mirrors) != 20 {
		t.Fatal(response, len(f.mirrors))
	}
	ctx := context.Background()
	keys := f.client.Keys(ctx, "lottery:settle_wallet:scratch-buy:*").Val()
	if len(keys) != 1 {
		t.Fatal(keys)
	}
	f.client.Set(ctx, keys[0], original, 7*24*time.Hour)
	f.svc.rounds = map[string]*financeRound{}
	// A cleanup pass may not invoke persistence or revive in-memory rounds.
	f.svc.testScratchMirror = func(*financeRound) error { t.Fatal("maintenance attempted delivery"); return nil }
	before := f.client.HGetAll(ctx, financeRoundHash).Val()
	_, n, err = f.rd.ShortenDeliveredScratchReceipts(0)
	if err != nil || n != 1 {
		t.Fatal("already mirrored book receipt not retired", n, err)
	}
	after, _ := f.rd.ScratchReceipt(input.ScratchScope, input.RequestID)
	if strings.Contains(after, `"rounds"`) || !strings.Contains(after, `"deliveryConfirmed":true`) || f.client.PTTL(ctx, keys[0]).Val() > time.Hour {
		t.Fatal("receipt not bounded", after)
	}
	for id, raw := range before {
		if f.client.HGet(ctx, financeRoundHash, id).Val() != raw {
			t.Fatal("financial snapshot changed")
		}
	}
	if f.balance(42) != 98000 || len(f.mirrors) != 20 || len(f.svc.rounds) != 0 {
		t.Fatal("cleanup replayed financial or delivery work")
	}
	retry, response := f.call("issue", input)
	var receipt struct {
		Result map[string]any `json:"result"`
	}
	json.Unmarshal([]byte(original), &receipt)
	want, _ := json.Marshal(receipt.Result)
	got, _ := json.Marshal(retry)
	if !response.Success || string(want) != string(got) || f.balance(42) != 98000 || len(f.svc.rounds) != 0 {
		t.Fatal("confirmed retry recreated or charged a ticket", response)
	}
	state, response := f.call("state", f.input)
	if !response.Success || len(state["pendingTickets"].([]any)) != 20 {
		t.Fatal("paid unclaimed tickets lost", response, state)
	}
}
