package dao

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPeriodicPendingScratchCompactionPreservesDeadlineAndUnfinishedRights(t *testing.T) {
	for _, ttl := range []time.Duration{0, 37 * time.Minute} {
		t.Run(ttl.String(), func(t *testing.T) {
			rd, client := retentionRedis(t)
			ctx := context.Background()
			scope := ScratchScope{7, 91061, 42, "CNY"}
			key := scratchBuyMarker(scope, "original-purchase")
			original := pendingScratchReceipt(t, nil)
			client.Set(ctx, key, original, ttl)
			client.HSet(ctx, "lottery_finance_rounds", "original", "original unpaid or undelivered financial responsibility")
			pendingKey := "gogame:scratch:pending:" + scope.Key()
			soldKey := "gogame:scratch:sold:" + scope.Market() + ":ticket"
			client.SAdd(ctx, pendingKey, "original")
			client.Set(ctx, soldKey, "original", 0)
			before := client.PTTL(ctx, key).Val()
			_, n, err := rd.ShortenDeliveredScratchReceipts(0)
			if err != nil || n != 1 {
				t.Fatal("pending receipt not compacted", n, err)
			}
			slim := client.Get(ctx, key).Val()
			var got map[string]json.RawMessage
			if json.Unmarshal([]byte(slim), &got) != nil || string(got["referencesOnly"]) != "true" || got["deliveryConfirmed"] != nil || strings.Contains(slim, "huMask") || strings.Contains(slim, "wallet original") {
				t.Fatal("invalid minimal receipt", slim)
			}
			if client.PTTL(ctx, key).Val() != before {
				t.Fatal("pending receipt deadline changed")
			}
			if !client.SIsMember(ctx, pendingKey, "original").Val() || client.Get(ctx, soldKey).Val() != "original" || client.HGet(ctx, "lottery_finance_rounds", "original").Val() != "original unpaid or undelivered financial responsibility" {
				t.Fatal("unfinished rights changed")
			}
			_, n, err = rd.ShortenDeliveredScratchReceipts(0)
			if err != nil || n != 0 || client.Get(ctx, key).Val() != slim || client.PTTL(ctx, key).Val() != before {
				t.Fatal("repeat maintenance renewed or changed receipt", n, err)
			}
		})
	}
}

func TestPeriodicPendingScratchCompactionRejectsUntrustedScopeAndIdentity(t *testing.T) {
	rd, client := retentionRedis(t)
	ctx := context.Background()
	scope := ScratchScope{7, 91061, 42, "CNY"}
	for _, test := range []struct{ key, raw string }{
		{"lottery:settle_wallet:scratch-buy:7:91061:42:CNY:not-a-request-hash", pendingScratchReceipt(t, nil)},
		{scratchBuyMarker(scope, "other-owner"), pendingScratchReceipt(t, func(round map[string]any) { round["agent"] = 8 })},
		{scratchBuyMarker(scope, "missing-identity"), `{"fingerprint":"paid","result":{"code":0},"rounds":["missing"]}`},
	} {
		client.Set(ctx, test.key, test.raw, 0)
	}
	_, n, err := rd.ShortenDeliveredScratchReceipts(0)
	if err != nil || n != 0 {
		t.Fatal("untrusted receipt changed", n, err)
	}
}
