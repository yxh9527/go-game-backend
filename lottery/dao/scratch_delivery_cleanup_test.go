package dao

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func mirroredScratchReceipt(t *testing.T, rd *RedisDao, edit func(map[string]any)) (ScratchScope, string, string, []string) {
	t.Helper()
	ctx := context.Background()
	scope := ScratchScope{7, 91061, 42, "CNY"}
	refs := []string{}
	ids := []string{"paid-one", "paid-two"}
	for i, id := range ids {
		serial := "ticket-" + id
		round := map[string]any{"roundId": id, "agent": scope.Agent, "gameId": scope.GameID, "state": "RESERVED", "reservation": map[string]any{"requestId": "purchase"}, "scratch": map[string]any{"roundId": id, "serialFull": serial, "userId": scope.UserID, "currency": scope.Currency, "mirrored": true, "soldMirrored": true}}
		if i == 1 && edit != nil {
			edit(round)
		}
		raw, _ := json.Marshal(round)
		rd.cli.HSet(ctx, "lottery_finance_rounds", id, raw)
		ref, _ := json.Marshal(scratchPaidReference{RoundID: id, Agent: scope.Agent, GameID: scope.GameID, Scratch: &struct {
			Serial   string `json:"serialFull"`
			UserID   uint32 `json:"userId"`
			Currency string `json:"currency"`
		}{serial, scope.UserID, scope.Currency}})
		refs = append(refs, string(ref))
		rd.cli.SAdd(ctx, "gogame:scratch:pending:"+scope.Key(), id)
		rd.cli.Set(ctx, "gogame:scratch:sold:"+scope.Market()+":"+serial, id, 0)
	}
	raw, _ := json.Marshal(map[string]any{"fingerprint": "original", "result": map[string]any{"code": 0, "walletAmount": 9800}, "rounds": refs, "referencesOnly": true, "createdAt": time.Now().Unix() - 600})
	key := scratchBuyMarker(scope, "purchase")
	rd.cli.Set(ctx, key, raw, 7*24*time.Hour)
	return scope, key, string(raw), ids
}

func TestReceiptMaintenanceConfirmsAlreadyMirroredBatchWithoutAPlayerRetry(t *testing.T) {
	rd, client := retentionRedis(t)
	scope, key, original, ids := mirroredScratchReceipt(t, rd, nil)
	ctx := context.Background()
	before := client.HGetAll(ctx, "lottery_finance_rounds").Val()
	_, changed, err := rd.ShortenDeliveredScratchReceipts(0)
	if err != nil || changed != 1 {
		t.Fatal("mirrored purchase not retired", changed, err)
	}
	after := client.Get(ctx, key).Val()
	if strings.Contains(after, `"rounds"`) || !strings.Contains(after, `"deliveryConfirmed":true`) {
		t.Fatal("unnecessary paid references retained", after)
	}
	var a, b map[string]json.RawMessage
	json.Unmarshal([]byte(original), &a)
	json.Unmarshal([]byte(after), &b)
	for _, field := range []string{"fingerprint", "result", "createdAt"} {
		if string(a[field]) != string(b[field]) {
			t.Fatal("original receipt changed", field)
		}
	}
	if ttl := client.PTTL(ctx, key).Val(); ttl < 49*time.Minute || ttl > 50*time.Minute {
		t.Fatal("purchase deadline not used", ttl)
	}
	for _, id := range ids {
		if client.HGet(ctx, "lottery_finance_rounds", id).Val() != before[id] || !client.SIsMember(ctx, "gogame:scratch:pending:"+scope.Key(), id).Val() || client.Get(ctx, "gogame:scratch:sold:"+scope.Market()+":ticket-"+id).Val() != id {
			t.Fatal("paid ticket rights changed")
		}
	}
	client.PExpire(ctx, key, time.Minute)
	_, _, err = rd.ShortenDeliveredScratchReceipts(0)
	if err != nil || client.PTTL(ctx, key).Val() > time.Minute {
		t.Fatal("maintenance renewed receipt", err)
	}
}

func TestReceiptMaintenanceKeepsPartialOrUntrustedDelivery(t *testing.T) {
	for name, edit := range map[string]func(map[string]any){
		"ticket-not-mirrored": func(r map[string]any) { r["scratch"].(map[string]any)["mirrored"] = false },
		"market-not-mirrored": func(r map[string]any) { delete(r["scratch"].(map[string]any), "soldMirrored") },
		"wrong-agent":         func(r map[string]any) { r["agent"] = 8 },
		"wrong-game":          func(r map[string]any) { r["gameId"] = 91063 },
		"wrong-player":        func(r map[string]any) { r["scratch"].(map[string]any)["userId"] = 43 },
		"wrong-currency":      func(r map[string]any) { r["scratch"].(map[string]any)["currency"] = "USD" },
		"wrong-serial":        func(r map[string]any) { r["scratch"].(map[string]any)["serialFull"] = "another" },
		"wrong-round":         func(r map[string]any) { r["scratch"].(map[string]any)["roundId"] = "another" },
		"other-purchase":      func(r map[string]any) { r["reservation"].(map[string]any)["requestId"] = "other-purchase" },
		"missing-reservation": func(r map[string]any) { delete(r, "reservation") },
	} {
		t.Run(name, func(t *testing.T) {
			rd, client := retentionRedis(t)
			_, key, raw, _ := mirroredScratchReceipt(t, rd, edit)
			_, changed, err := rd.ShortenDeliveredScratchReceipts(0)
			if err != nil || changed != 0 || client.Get(context.Background(), key).Val() != raw || client.PTTL(context.Background(), key).Val() != 7*24*time.Hour {
				t.Fatal("unconfirmed rights retired", changed, err)
			}
		})
	}
}

func TestReceiptMaintenanceDoesNotRebuildMissingOrDamagedFinancialSnapshots(t *testing.T) {
	for _, raw := range []string{"", "damaged", "{}"} {
		rd, client := retentionRedis(t)
		_, key, receipt, ids := mirroredScratchReceipt(t, rd, nil)
		ctx := context.Background()
		if raw == "" {
			client.HDel(ctx, "lottery_finance_rounds", ids[1])
		} else {
			client.HSet(ctx, "lottery_finance_rounds", ids[1], raw)
		}
		_, changed, err := rd.ShortenDeliveredScratchReceipts(0)
		if err != nil || changed != 0 || client.Get(ctx, key).Val() != receipt || client.HGet(ctx, "lottery_finance_rounds", ids[1]).Val() != raw {
			t.Fatal("missing evidence was replaced or ignored", changed, err)
		}
	}
}

func TestReceiptMaintenanceHandlesOriginalDeadlineAndRejectsUntrustedReferences(t *testing.T) {
	for _, mode := range []string{"overdue", "shorter-ttl", "full-legacy", "duplicate", "foreign-reference", "missing-time", "future-time", "too-many", "oversized-round"} {
		t.Run(mode, func(t *testing.T) {
			rd, client := retentionRedis(t)
			_, key, original, ids := mirroredScratchReceipt(t, rd, nil)
			ctx := context.Background()
			var receipt map[string]any
			json.Unmarshal([]byte(original), &receipt)
			expectChange := false
			switch mode {
			case "overdue":
				receipt["createdAt"] = time.Now().Unix() - 3601
				expectChange = true
			case "shorter-ttl":
				expectChange = true
			case "full-legacy":
				receipt["rounds"] = []string{client.HGet(ctx, "lottery_finance_rounds", ids[0]).Val(), client.HGet(ctx, "lottery_finance_rounds", ids[1]).Val()}
				delete(receipt, "referencesOnly")
				expectChange = true
			case "duplicate":
				receipt["rounds"] = []any{receipt["rounds"].([]any)[0], receipt["rounds"].([]any)[0]}
			case "foreign-reference":
				var ref map[string]any
				json.Unmarshal([]byte(receipt["rounds"].([]any)[0].(string)), &ref)
				ref["agent"] = 8
				raw, _ := json.Marshal(ref)
				receipt["rounds"].([]any)[0] = string(raw)
			case "missing-time":
				delete(receipt, "createdAt")
			case "future-time":
				receipt["createdAt"] = time.Now().Unix() + 3600
			case "too-many":
				refs := []any{}
				for i := 0; i < 101; i++ {
					refs = append(refs, receipt["rounds"].([]any)[0])
				}
				receipt["rounds"] = refs
			case "oversized-round":
				client.HSet(ctx, "lottery_finance_rounds", ids[1], strings.Repeat("x", 128*1024+1))
			}
			raw, _ := json.Marshal(receipt)
			original = string(raw)
			client.Set(ctx, key, original, 7*24*time.Hour)
			if mode == "shorter-ttl" {
				client.PExpire(ctx, key, time.Minute)
			}
			_, n, err := rd.ShortenDeliveredScratchReceipts(0)
			if err != nil || (n == 1) != expectChange {
				t.Fatal("unexpected maintenance result", n, err)
			}
			if mode == "overdue" {
				if client.Exists(ctx, key).Val() != 0 {
					t.Fatal("overdue receipt not deleted")
				}
				return
			}
			if !expectChange && (client.Get(ctx, key).Val() != original || client.PTTL(ctx, key).Val() != 7*24*time.Hour) {
				t.Fatal("untrusted receipt changed")
			}
			if mode == "shorter-ttl" && client.PTTL(ctx, key).Val() > time.Minute {
				t.Fatal("shorter TTL extended")
			}
		})
	}
}

func TestReceiptMaintenanceFinalComparePreservesChangedReceiptAndDeliveryEvidence(t *testing.T) {
	for _, mode := range []string{"expired-receipt", "changed-receipt", "changed-round", "removed-round"} {
		t.Run(mode, func(t *testing.T) {
			rd, client := retentionRedis(t)
			_, key, original, ids := mirroredScratchReceipt(t, rd, nil)
			ctx := context.Background()
			proofs := []scratchDeliveryProof{}
			for _, id := range ids {
				proofs = append(proofs, scratchDeliveryProof{RoundID: id, Raw: client.HGet(ctx, "lottery_finance_rounds", id).Val()})
			}
			switch mode {
			case "expired-receipt":
				client.Del(ctx, key)
			case "changed-receipt":
				client.Set(ctx, key, "new receipt", time.Minute)
			case "changed-round":
				client.HSet(ctx, "lottery_finance_rounds", ids[1], "changed proof")
			case "removed-round":
				client.HDel(ctx, "lottery_finance_rounds", ids[1])
			}
			before := client.Get(ctx, key).Val()
			evidence, _ := json.Marshal(proofs)
			n, err := compactScratchReceiptScript.Run(ctx, client, []string{key, "lottery_finance_rounds"}, original, `{"deliveryConfirmed":true}`, int64(time.Hour/time.Millisecond), string(evidence)).Int()
			if err != nil || n != 0 || client.Get(ctx, key).Val() != before {
				t.Fatal("late cleanup overwrote newer evidence", n, err)
			}
			if mode == "changed-receipt" && client.PTTL(ctx, key).Val() > time.Minute {
				t.Fatal("newer receipt TTL changed")
			}
		})
	}
}
