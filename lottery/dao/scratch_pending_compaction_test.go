package dao

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func pendingScratchReceipt(t *testing.T, change func(map[string]any)) string {
	t.Helper()
	round := map[string]any{"roundId": "original", "agent": 7, "gameId": 91061, "state": "RESERVED", "bets": map[string]any{"private": "wallet original"}, "scratch": map[string]any{"serialFull": "ticket", "userId": 42, "currency": "CNY", "huMask": []int{1, 2}, "winAmount": 10000}}
	if change != nil {
		change(round)
	}
	raw, _ := json.Marshal(round)
	receipt, _ := json.Marshal(map[string]any{"fingerprint": "original-hash", "result": map[string]any{"code": 0, "newTickets": []any{map[string]any{"serialFull": "ticket"}}}, "rounds": []string{string(raw)}, "createdAt": time.Now().Unix() - 600})
	return string(receipt)
}

func TestPendingScratchReceiptKeepsOnlyIdentityWithoutChangingExpiryOrRights(t *testing.T) {
	rd, client := retentionRedis(t)
	ctx := context.Background()
	scope := ScratchScope{7, 91061, 42, "CNY"}
	key := scratchBuyMarker(scope, "purchase")
	original := pendingScratchReceipt(t, nil)
	client.Set(ctx, key, original, 37*time.Minute)
	client.HSet(ctx, "lottery_finance_rounds", "original", "authoritative original snapshot")
	client.SAdd(ctx, "gogame:scratch:pending:"+scope.Key(), "original")
	before := client.PTTL(ctx, key).Val()
	slim, err := rd.CompactPendingScratchReceipt(ctx, scope, "purchase", original)
	if err != nil || client.Get(ctx, key).Val() != slim || client.PTTL(ctx, key).Val() > before {
		t.Fatal("pending compaction changed TTL", err)
	}
	if strings.Contains(slim, "huMask") || strings.Contains(slim, "winAmount") || strings.Contains(slim, "wallet original") || strings.Contains(slim, "RESERVED") {
		t.Fatal("private recovery copy remains", slim)
	}
	var a, b map[string]json.RawMessage
	json.Unmarshal([]byte(original), &a)
	json.Unmarshal([]byte(slim), &b)
	if string(a["result"]) != string(b["result"]) || string(a["fingerprint"]) != string(b["fingerprint"]) || string(a["createdAt"]) != string(b["createdAt"]) || string(b["referencesOnly"]) != "true" || b["deliveryConfirmed"] != nil {
		t.Fatal("receipt identity, result or confirmation changed", slim)
	}
	if client.HGet(ctx, "lottery_finance_rounds", "original").Val() != "authoritative original snapshot" || !client.SIsMember(ctx, "gogame:scratch:pending:"+scope.Key(), "original").Val() {
		t.Fatal("compaction changed financial rights")
	}
	if again, err := rd.CompactPendingScratchReceipt(ctx, scope, "purchase", slim); err != nil || again != slim || client.PTTL(ctx, key).Val() > before {
		t.Fatal("repeat pending compaction renewed deadline", err)
	}
}

func TestPendingScratchReceiptRejectsChangedAndExpiredKeysWithoutRecreatingThem(t *testing.T) {
	for _, replacement := range []string{"", "another original receipt"} {
		rd, client := retentionRedis(t)
		ctx := context.Background()
		scope := ScratchScope{7, 91061, 42, "CNY"}
		key := scratchBuyMarker(scope, "purchase")
		original := pendingScratchReceipt(t, nil)
		if replacement != "" {
			client.Set(ctx, key, replacement, time.Minute)
		}
		if _, err := rd.CompactPendingScratchReceipt(ctx, scope, "purchase", original); err == nil {
			t.Fatal("old receipt replaced newer or missing key")
		}
		if replacement == "" && client.Exists(ctx, key).Val() != 0 {
			t.Fatal("expired key recreated")
		}
		if replacement != "" && (client.Get(ctx, key).Val() != replacement || client.PTTL(ctx, key).Val() > time.Minute) {
			t.Fatal("concurrent receipt changed")
		}
	}
}

func TestPendingScratchReceiptRejectsForeignAndDuplicateReferencesBeforeMutation(t *testing.T) {
	for _, edit := range []func(map[string]any){
		func(r map[string]any) { r["agent"] = 8 }, func(r map[string]any) { r["gameId"] = 91063 }, func(r map[string]any) { r["roundId"] = "" }, func(r map[string]any) { r["scratch"].(map[string]any)["userId"] = 43 }, func(r map[string]any) { r["scratch"].(map[string]any)["currency"] = "USD" },
	} {
		rd, client := retentionRedis(t)
		ctx := context.Background()
		scope := ScratchScope{7, 91061, 42, "CNY"}
		key := scratchBuyMarker(scope, "purchase")
		raw := pendingScratchReceipt(t, edit)
		client.Set(ctx, key, raw, time.Hour)
		if _, err := rd.CompactPendingScratchReceipt(ctx, scope, "purchase", raw); err == nil || client.Get(ctx, key).Val() != raw {
			t.Fatal("foreign receipt mutated")
		}
	}
	rd, client := retentionRedis(t)
	ctx := context.Background()
	scope := ScratchScope{7, 91061, 42, "CNY"}
	key := scratchBuyMarker(scope, "purchase")
	raw := pendingScratchReceipt(t, nil)
	var receipt map[string]any
	json.Unmarshal([]byte(raw), &receipt)
	refs := receipt["rounds"].([]any)
	receipt["rounds"] = append(refs, refs[0])
	encoded, _ := json.Marshal(receipt)
	raw = string(encoded)
	client.Set(ctx, key, raw, time.Hour)
	if _, err := rd.CompactPendingScratchReceipt(ctx, scope, "purchase", raw); err == nil || client.Get(ctx, key).Val() != raw {
		t.Fatal("duplicate round accepted")
	}
}
