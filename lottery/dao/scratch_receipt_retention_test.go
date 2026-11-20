package dao

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestDeliveredScratchReceiptExpiresAtOriginalPurchaseHourWithoutRemovingTicketRights(t *testing.T) {
	rd, client := retentionRedis(t)
	ctx := context.Background()
	scope := ScratchScope{7, 91061, 42, "CNY"}
	key := scratchBuyMarker(scope, "original")
	issued := time.Now().Unix() - 900
	raw, _ := json.Marshal(map[string]any{"fingerprint": "original-fingerprint", "createdAt": issued, "result": map[string]any{"code": 0, "walletAmount": 9900}, "rounds": []string{"original-paid-ticket"}})
	client.Set(ctx, key, raw, 7*24*time.Hour)
	client.SAdd(ctx, "gogame:scratch:pending:"+scope.Key(), "original-round")
	client.Set(ctx, "gogame:scratch:sold:"+scope.Market()+":ticket", "original-round", 0)
	if err := rd.CompactDeliveredScratchReceipt(ctx, scope, "original", string(raw)); err != nil {
		t.Fatal(err)
	}
	ttl := client.PTTL(ctx, key).Val()
	if ttl > 45*time.Minute || ttl < 44*time.Minute {
		t.Fatal("original purchase deadline not used", ttl)
	}
	compact := client.Get(ctx, key).Val()
	client.PExpire(ctx, key, time.Minute)
	if err := rd.CompactDeliveredScratchReceipt(ctx, scope, "original", compact); err != nil || client.PTTL(ctx, key).Val() > time.Minute {
		t.Fatal("receipt read renewed original deadline", err)
	}
	if !client.SIsMember(ctx, "gogame:scratch:pending:"+scope.Key(), "original-round").Val() || client.Get(ctx, "gogame:scratch:sold:"+scope.Market()+":ticket").Val() != "original-round" {
		t.Fatal("delivery receipt retirement deleted paid ticket rights")
	}
}

func TestOldConfirmedScratchReceiptCleanupIsBoundedAndNeverTouchesUnconfirmedBooks(t *testing.T) {
	rd, client := retentionRedis(t)
	ctx := context.Background()
	scope := ScratchScope{7, 91061, 42, "CNY"}
	old := scratchBuyMarker(scope, "old")
	fresh := scratchBuyMarker(scope, "fresh")
	pending := scratchBuyMarker(scope, "pending")
	oldRaw, _ := json.Marshal(map[string]any{"fingerprint": "old", "result": map[string]any{"code": 0}, "deliveryConfirmed": true, "createdAt": time.Now().Unix() - 3601})
	freshRaw, _ := json.Marshal(map[string]any{"fingerprint": "fresh", "result": map[string]any{"code": 0}, "deliveryConfirmed": true, "createdAt": time.Now().Unix() - 600})
	pendingRaw := `{"fingerprint":"pending","result":{"code":0},"rounds":["original-paid-book"]}`
	client.Set(ctx, old, oldRaw, 7*24*time.Hour)
	client.Set(ctx, fresh, freshRaw, 7*24*time.Hour)
	client.Set(ctx, pending, pendingRaw, 7*24*time.Hour)
	client.Set(ctx, "scratch-history:v2:query", "query", 180*time.Second)
	cursor := uint64(0)
	changed := 0
	for pass := 0; pass < 10; pass++ {
		next, n, err := rd.ShortenDeliveredScratchReceipts(cursor)
		if err != nil {
			t.Fatal(err)
		}
		cursor = next
		changed += n
		if cursor == 0 {
			break
		}
	}
	if changed != 2 || client.Exists(ctx, old).Val() != 0 {
		t.Fatal("old confirmed receipt remains", changed)
	}
	if ttl := client.PTTL(ctx, fresh).Val(); ttl > 50*time.Minute || ttl < 49*time.Minute {
		t.Fatal(ttl)
	}
	if client.Get(ctx, pending).Val() != pendingRaw || client.PTTL(ctx, pending).Val() != 7*24*time.Hour || client.PTTL(ctx, "scratch-history:v2:query").Val() != 180*time.Second {
		t.Fatal("cleanup changed unfinished book or unrelated query cache")
	}
}

func TestLateDeliveryReceiptRetirementDoesNotRecreateExpiredOrChangedValue(t *testing.T) {
	rd, client := retentionRedis(t)
	ctx := context.Background()
	scope := ScratchScope{7, 91061, 42, "CNY"}
	key := scratchBuyMarker(scope, "old")
	raw, _ := json.Marshal(map[string]any{"fingerprint": "old", "result": map[string]any{"code": 0}, "rounds": []string{"original-paid-ticket"}, "createdAt": time.Now().Unix() - 3601})
	client.Set(ctx, key, raw, 7*24*time.Hour)
	if err := rd.CompactDeliveredScratchReceipt(ctx, scope, "old", string(raw)); err != nil || client.Exists(ctx, key).Val() != 0 {
		t.Fatal("overdue receipt was retained", err)
	}
	client.Set(ctx, key, "new receipt", time.Hour)
	if err := rd.CompactDeliveredScratchReceipt(ctx, scope, "old", string(raw)); err == nil || client.Get(ctx, key).Val() != "new receipt" {
		t.Fatal("stale retirement removed changed receipt", err)
	}
}
