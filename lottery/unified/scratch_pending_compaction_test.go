package unified

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Emulate a receipt produced by the currently deployed full-snapshot writer.
// Test fixtures never turn these copies back into financial state.
func legacyFullPendingReceipt(t *testing.T, f *scratchFixture, input scratchInput) (string, string) {
	t.Helper()
	ctx := context.Background()
	original, err := f.rd.ScratchReceipt(input.ScratchScope, input.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	var receipt map[string]any
	if json.Unmarshal([]byte(original), &receipt) != nil {
		t.Fatal("bad fixture receipt")
	}
	var rounds []string
	for _, encoded := range receipt["rounds"].([]any) {
		var ref struct {
			RoundID string `json:"roundId"`
		}
		json.Unmarshal([]byte(encoded.(string)), &ref)
		raw, err := f.client.HGet(ctx, financeRoundHash, ref.RoundID).Result()
		if err != nil {
			t.Fatal(err)
		}
		rounds = append(rounds, raw)
	}
	receipt["rounds"] = rounds
	delete(receipt, "referencesOnly")
	raw, _ := json.Marshal(receipt)
	keys := f.client.Keys(ctx, "lottery:settle_wallet:scratch-buy:*").Val()
	if len(keys) != 1 {
		t.Fatal(keys)
	}
	ttl := f.client.PTTL(ctx, keys[0]).Val()
	if err = f.client.Set(ctx, keys[0], raw, ttl).Err(); err != nil {
		t.Fatal(err)
	}
	return keys[0], string(raw)
}

func TestNewUnconfirmedBookReceiptHasNoPrivateFinancialSnapshotCopies(t *testing.T) {
	f := newScratchFixture(t)
	input, _ := f.purchase(true)
	f.mirrorFailure = true
	if _, response := f.call("issue", input); response.Success || f.balance(42) != 98000 {
		t.Fatal(response)
	}
	raw, err := f.rd.ScratchReceipt(input.ScratchScope, input.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, "huMask") || strings.Contains(raw, "winAmount") || strings.Contains(raw, "bets") || strings.Contains(raw, "reservation") || !strings.Contains(raw, `"referencesOnly":true`) {
		t.Fatal("unconfirmed receipt duplicated private state", raw)
	}
	var receipt struct {
		Rounds    []string `json:"rounds"`
		Confirmed bool     `json:"deliveryConfirmed"`
	}
	json.Unmarshal([]byte(raw), &receipt)
	if len(receipt.Rounds) != 20 || receipt.Confirmed {
		t.Fatal("paid batch lost or falsely acknowledged")
	}
	// The previous finishScratchIssue reader expects rounds []string with a
	// non-nil scratch object; these references preserve that contract.
	for _, encoded := range receipt.Rounds {
		var old financeRound
		if json.Unmarshal([]byte(encoded), &old) != nil || old.Scratch == nil || old.RoundID == "" {
			t.Fatal("old reader cannot decode identity reference")
		}
		stored, ok := f.svc.loadRoundLocked(old.RoundID)
		if !ok || stored.Scratch == nil || len(stored.Scratch.Mask) != 20 {
			t.Fatal("original financial state unavailable")
		}
	}
}

func TestLegacyUnconfirmedBookSlimsWhileSQLOfflineWithoutRenewingOrChangingOriginalResult(t *testing.T) {
	f := newScratchFixture(t)
	input, _ := f.purchase(true)
	f.mirrorFailure = true
	if _, response := f.call("issue", input); response.Success {
		t.Fatal("expected delivery fault")
	}
	key, legacy := legacyFullPendingReceipt(t, f, input)
	ctx := context.Background()
	f.client.PExpire(ctx, key, 23*time.Minute)
	ids, _ := f.rd.ScratchPending(input.ScratchScope)
	snapshots := map[string]string{}
	for _, id := range ids {
		snapshots[id] = f.client.HGet(ctx, financeRoundHash, id).Val()
	}
	if _, response := f.call("issue", input); response.Success || f.balance(42) != 98000 {
		t.Fatal("SQL outage reported delivered or charged twice", response)
	}
	slim := f.client.Get(ctx, key).Val()
	if len(slim) >= len(legacy)/2 || strings.Contains(slim, "huMask") || strings.Contains(slim, `"deliveryConfirmed":true`) {
		t.Fatal("old undelivered batch not slimmed", len(legacy), len(slim))
	}
	if ttl := f.client.PTTL(ctx, key).Val(); ttl <= 0 || ttl > 23*time.Minute {
		t.Fatal("pending expiry changed", ttl)
	}
	var before, after map[string]json.RawMessage
	json.Unmarshal([]byte(legacy), &before)
	json.Unmarshal([]byte(slim), &after)
	for _, field := range []string{"fingerprint", "result", "createdAt"} {
		if string(before[field]) != string(after[field]) {
			t.Fatal("original receipt changed", field)
		}
	}
	for id, raw := range snapshots {
		if f.client.HGet(ctx, financeRoundHash, id).Val() != raw {
			t.Fatal("pending compaction changed financial state")
		}
	}
	t.Logf("pending book receipt reduced from %d to %d bytes", len(legacy), len(slim))
	f.mirrorFailure = false
	first, response := f.call("issue", input)
	if !response.Success || len(f.mirrors) != 20 || f.balance(42) != 98000 {
		t.Fatal("original pending delivery did not resume", response, len(f.mirrors))
	}
	second, response := f.call("issue", input)
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if !response.Success || string(a) != string(b) || len(f.mirrors) != 20 || f.balance(42) != 98000 {
		t.Fatal("delivered retry changed result or repeated money", response)
	}
}

func TestSlimPendingReceiptNeverRestoresMissingOriginalMoneyState(t *testing.T) {
	f := newScratchFixture(t)
	input, _ := f.purchase(true)
	f.mirrorFailure = true
	f.call("issue", input)
	key, _ := legacyFullPendingReceipt(t, f, input)
	ids, _ := f.rd.ScratchPending(input.ScratchScope)
	f.client.HDel(context.Background(), financeRoundHash, ids[0])
	f.svc.rounds = map[string]*financeRound{}
	f.mirrorFailure = false
	if _, response := f.call("issue", input); response.Success || f.balance(42) != 98000 || f.client.HExists(context.Background(), financeRoundHash, ids[0]).Val() {
		t.Fatal("receipt restored missing financial state or charged again", response)
	}
	if raw := f.client.Get(context.Background(), key).Val(); !strings.Contains(raw, `"referencesOnly":true`) || strings.Contains(raw, "huMask") {
		t.Fatal("failed delivery left original private duplicates")
	}
}

func TestPendingReceiptReferenceMustMatchActualOriginalTicket(t *testing.T) {
	f := newScratchFixture(t)
	input, _ := f.purchase(false)
	f.mirrorFailure = true
	f.call("issue", input)
	raw, _ := f.rd.ScratchReceipt(input.ScratchScope, input.RequestID)
	var receipt map[string]any
	json.Unmarshal([]byte(raw), &receipt)
	refs := receipt["rounds"].([]any)
	var ref map[string]any
	json.Unmarshal([]byte(refs[0].(string)), &ref)
	ref["scratch"].(map[string]any)["serialFull"] = "different-original-ticket"
	encoded, _ := json.Marshal(ref)
	refs[0] = string(encoded)
	receipt["rounds"] = refs
	encoded, _ = json.Marshal(receipt)
	key := f.client.Keys(context.Background(), "lottery:settle_wallet:scratch-buy:*").Val()[0]
	f.client.Set(context.Background(), key, encoded, time.Hour)
	f.mirrorFailure = false
	if _, response := f.call("issue", input); response.Success || len(f.mirrors) != 0 || f.balance(42) != 99900 {
		t.Fatal("wrong original ticket reference accepted", response)
	}
}
