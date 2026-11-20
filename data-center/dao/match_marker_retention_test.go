package dao

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestConfirmedMatchMarkerBecomesOneHourWithoutRenewalOrRecreation(t *testing.T) {
	for _, operation := range []string{"charge", "refund", "award"} {
		t.Run(operation, func(t *testing.T) {
			client, plan := matchPaymentRedis(t)
			ctx := context.Background()
			input := MatchPaymentInput{Operation: operation, Plan: plan}
			receipt := map[string]any{"key": plan.Key, "currency": plan.Currency, "amount": 110}
			raw, _ := json.Marshal(map[string]any{"fingerprint": paymentHash(input), "receipt": receipt, "createdAt": time.Now().Unix()})
			key := paymentMarker(plan, operation)
			client.Set(ctx, key, string(raw), 0)
			if err := retainConfirmedMatchMarker(ctx, client, input, receipt); err != nil {
				t.Fatal(err)
			}
			if ttl := client.TTL(ctx, key).Val(); ttl <= 0 || ttl > time.Hour {
				t.Fatal("confirmed receipt not one hour", ttl)
			}
			client.PExpire(ctx, key, 30*time.Minute)
			if err := retainConfirmedMatchMarker(ctx, client, input, receipt); err != nil {
				t.Fatal(err)
			}
			if ttl := client.PTTL(ctx, key).Val(); ttl > 30*time.Minute || ttl <= 0 {
				t.Fatal("retry renewed original expiry", ttl)
			}
			client.Del(ctx, key)
			if err := retainConfirmedMatchMarker(ctx, client, input, receipt); err != nil {
				t.Fatal(err)
			}
			if client.Exists(ctx, key).Val() != 0 {
				t.Fatal("late confirmation recreated Redis receipt")
			}
		})
	}
}

func TestMatchMarkerShortensLegacySevenDaysButRejectsWrongOrChangedOriginal(t *testing.T) {
	client, p := matchPaymentRedis(t)
	ctx := context.Background()
	input := MatchPaymentInput{Operation: "charge", Plan: p}
	receipt := map[string]any{"key": p.Key, "paidGold": 100, "paidService": 10, "currency": "CNY"}
	raw, _ := json.Marshal(map[string]any{"fingerprint": paymentHash(input), "receipt": receipt})
	key := paymentMarker(p, "charge")
	client.Set(ctx, key, string(raw), 7*24*time.Hour)
	if err := retainConfirmedMatchMarker(ctx, client, input, receipt); err != nil || client.Exists(ctx, key).Val() != 0 {
		t.Fatal("confirmed legacy marker not removed", err)
	}
	for _, value := range []string{"invalid", `{"fingerprint":"wrong","receipt":{}}`} {
		client.Set(ctx, key, value, 0)
		if err := retainConfirmedMatchMarker(ctx, client, input, receipt); err == nil || client.TTL(ctx, key).Val() != -1 {
			t.Fatal("mismatched marker given an expiry", err)
		}
	}
	client.Set(ctx, key, string(raw), 0)
	if err := retainConfirmedMatchMarker(ctx, client, input, map[string]any{"paidGold": 999}); err == nil || client.TTL(ctx, key).Val() != -1 {
		t.Fatal("different SQL receipt shortened original", err)
	}
	client.Set(ctx, key, "concurrent change", 0)
	if n, err := shortenConfirmedMatchMarkerScript.Run(ctx, client, []string{key}, string(raw), 3600000).Int(); err != nil || n != 0 || client.TTL(ctx, key).Val() != -1 {
		t.Fatal("concurrent marker replacement was changed", n, err)
	}
}

func TestSQLFailureLeavesMatchMarkerUnboundedButConfirmedReceiptExpiresAndLateRetryUsesSQL(t *testing.T) {
	client, p := matchPaymentRedis(t)
	ctx := context.Background()
	input := MatchPaymentInput{Operation: "charge", Plan: p}
	prepared := &matchPaymentDetail{Fingerprint: paymentHash(p), Status: "prepared", TicketIDs: []int64{}}
	initial := matchPaymentDocument(p, nil, "pending")
	pending := matchPaymentDocument(p, prepared, "pending")
	steps := append(matchPaymentSQL(p, initial, 1, nil, nil), matchPaymentSQL(p, pending, 1, nil, errors.New("SQL failure"))...)
	db, _ := gogameTestDB(t, steps...)
	if _, err := (&DBDao{manager: db}).matchPayment(ctx, client, input); err == nil {
		t.Fatal("SQL failure acknowledged")
	}
	key := paymentMarker(p, "charge")
	if client.TTL(ctx, key).Val() != -1 || client.HGet(ctx, "player_42", "currency").Val() != "890" {
		t.Fatal("unconfirmed original payment was expired or changed")
	}
	var saved string
	steps = append(matchPaymentSQL(p, pending, 1, nil, nil), matchPaymentSQL(p, pending, 1, func(raw string) { saved = raw }, nil)...)
	db, _ = gogameTestDB(t, steps...)
	if _, err := (&DBDao{manager: db}).matchPayment(ctx, client, input); err != nil {
		t.Fatal(err)
	}
	if ttl := client.TTL(ctx, key).Val(); ttl <= 0 || ttl > time.Hour || client.HGet(ctx, "player_42", "currency").Val() != "890" {
		t.Fatal("SQL confirmation did not finalize marker")
	}
	var committed map[string]any
	if err := DecodeGogameJSON(saved, &committed); err != nil {
		t.Fatal(err)
	}
	client.Del(ctx, key)
	db, _ = gogameTestDB(t, append(matchPaymentSQL(p, committed, 1, nil, nil), matchPaymentSQL(p, committed, 1, nil, nil)...)...)
	value, err := (&DBDao{manager: db}).matchPayment(ctx, client, input)
	if err != nil || value.(map[string]any)["paidGold"] != float64(100) || client.HGet(ctx, "player_42", "currency").Val() != "890" || client.Exists(ctx, key).Val() != 0 {
		t.Fatal("late SQL replay repeated payment or recreated marker", value, err)
	}
}

func TestConfirmedRefundAndAwardReadSQLAfterTheirRedisMarkersAreRemoved(t *testing.T) {
	for _, operation := range []string{"refund", "award"} {
		t.Run(operation, func(t *testing.T) {
			client, p := matchPaymentRedis(t)
			ctx := context.Background()
			input := MatchPaymentInput{Operation: operation, Plan: p}
			var document map[string]any
			state := int64(4)
			if operation == "refund" {
				input.IncludeService = true
				detail := &matchPaymentDetail{Fingerprint: paymentHash(p), Status: "charged", TicketIDs: []int64{}, Receipt: map[string]any{"key": p.Key, "paidGold": 100, "paidService": 10, "currency": "CNY"}}
				document = matchPaymentDocument(p, detail, "refund_pending")
				client.HSet(ctx, "player_42", "currency", 890)
			} else {
				p.Key = "match:7:9:award:42:0"
				p.Gold = 0
				p.Service = 0
				p.Total = 0
				p.Kind = "money"
				p.Amount = 50
				input.Plan = p
				state = 3
				document = map[string]any{"registrations": map[string]any{"42": map[string]any{"status": "winner", "rank": 1, "agentId": 7}}, "outbox": []any{}, "prizes": []any{map[string]any{"plan": p, "status": "pending"}}}
			}
			var saved string
			db, _ := gogameTestDB(t, matchPaymentSQL(p, document, state, func(raw string) { saved = raw }, nil)...)
			first, err := (&DBDao{manager: db}).matchPayment(ctx, client, input)
			if err != nil {
				t.Fatal(err)
			}
			key := paymentMarker(p, operation)
			if ttl := client.TTL(ctx, key).Val(); ttl <= 0 || ttl > time.Hour {
				t.Fatal("confirmed operation retained a long marker", operation, ttl)
			}
			balance := client.HGet(ctx, "player_42", "currency").Val()
			revision := client.HGet(ctx, "player_42", "cash_revision").Val()
			var committed map[string]any
			if err = DecodeGogameJSON(saved, &committed); err != nil {
				t.Fatal(err)
			}
			client.Del(ctx, key)
			db, _ = gogameTestDB(t, matchPaymentSQL(p, committed, state, nil, nil)...)
			second, err := (&DBDao{manager: db}).matchPayment(ctx, client, input)
			if err != nil || paymentHash(first) != paymentHash(second) || client.Exists(ctx, key).Val() != 0 || client.HGet(ctx, "player_42", "currency").Val() != balance || client.HGet(ctx, "player_42", "cash_revision").Val() != revision {
				t.Fatal("late operation repeated money or recreated Redis marker", operation, first, second, err)
			}
		})
	}
}
