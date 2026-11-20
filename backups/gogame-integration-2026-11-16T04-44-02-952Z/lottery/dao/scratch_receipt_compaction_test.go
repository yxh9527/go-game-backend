package dao

import (
	"context"
	"encoding/json"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"strings"
	"testing"
	"time"
)

func TestDeliveredScratchReceiptDropsPrivateBatchButPreservesResultAndRemainingExpiry(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()
	rd := NewFinanceRedis(client)
	ctx := context.Background()
	scope := ScratchScope{Agent: 7, GameID: 91061, UserID: 42, Currency: "CNY"}
	key := scratchBuyMarker(scope, "request")
	result := json.RawMessage(`{"code":0,"newTickets":[{"serialFull":"sold","faceValue":100,"gglKind":"GGL1"}],"walletAmount":9900}`)
	raw, _ := json.Marshal(map[string]any{"fingerprint": "original-hash", "result": result, "rounds": []string{strings.Repeat("private original snapshot", 500)}})
	client.Set(ctx, key, string(raw), 7*24*time.Hour)
	mr.FastForward(31 * time.Minute)
	before := client.PTTL(ctx, key).Val()
	if err := rd.CompactDeliveredScratchReceipt(ctx, scope, "request", string(raw)); err != nil {
		t.Fatal(err)
	}
	compact := client.Get(ctx, key).Val()
	var receipt map[string]json.RawMessage
	if json.Unmarshal([]byte(compact), &receipt) != nil || string(receipt["fingerprint"]) != `"original-hash"` || string(receipt["deliveryConfirmed"]) != "true" || receipt["rounds"] != nil || strings.Contains(compact, "private original") {
		t.Fatal("original receipt was not compacted", compact)
	}
	var actual, expected any
	json.Unmarshal(receipt["result"], &actual)
	json.Unmarshal(result, &expected)
	a, _ := json.Marshal(actual)
	b, _ := json.Marshal(expected)
	if string(a) != string(b) || len(compact) >= len(raw)/2 || client.PTTL(ctx, key).Val() != before {
		t.Fatal("compaction changed delivery or renewed expiry", compact, before, client.PTTL(ctx, key).Val())
	}
	mr.FastForward(time.Minute)
	if err := rd.CompactDeliveredScratchReceipt(ctx, scope, "request", compact); err != nil {
		t.Fatal(err)
	}
	if client.PTTL(ctx, key).Val() != before-time.Minute {
		t.Fatal("repeated delivery renewed original TTL")
	}
}

func TestScratchReceiptCompactionNeverRecreatesExpiredOrOverwritesChangedReceipts(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()
	rd := NewFinanceRedis(client)
	ctx := context.Background()
	scope := ScratchScope{Agent: 7, GameID: 91061, UserID: 42, Currency: "CNY"}
	key := scratchBuyMarker(scope, "request")
	raw := `{"fingerprint":"hash","result":{"code":0},"rounds":["original"]}`
	client.Set(ctx, key, "concurrent value", time.Hour)
	if err := rd.CompactDeliveredScratchReceipt(ctx, scope, "request", raw); err == nil || client.Get(ctx, key).Val() != "concurrent value" {
		t.Fatal("changed receipt overwritten", err)
	}
	client.Del(ctx, key)
	if err := rd.CompactDeliveredScratchReceipt(ctx, scope, "request", raw); err == nil || client.Exists(ctx, key).Val() != 0 {
		t.Fatal("expired receipt recreated", err)
	}
	for _, bad := range []string{`{}`, `{"fingerprint":"hash","result":null,"rounds":["original"]}`, `{"fingerprint":"hash","result":{}}`} {
		if err := rd.CompactDeliveredScratchReceipt(ctx, scope, "request", bad); err == nil {
			t.Fatal("invalid receipt accepted", bad)
		}
	}
}
