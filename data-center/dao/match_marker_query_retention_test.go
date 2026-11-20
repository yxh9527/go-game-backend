package dao

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestMatchMarkerRetentionUsesOriginalCashTimeAfterLateSQLConfirmation(t *testing.T) {
	for _, operation := range []string{"charge", "refund", "award"} {
		for _, age := range []time.Duration{30 * time.Minute, 2 * time.Hour} {
			t.Run(operation+"/"+age.String(), func(t *testing.T) {
				client, p := matchPaymentRedis(t)
				ctx := context.Background()
				input := MatchPaymentInput{Operation: operation, Plan: p, IncludeService: operation == "refund"}
				receipt := map[string]any{"key": p.Key, "currency": p.Currency, "amount": 110}
				raw, _ := json.Marshal(map[string]any{"fingerprint": paymentHash(input), "receipt": receipt, "createdAt": time.Now().Add(-age).Unix()})
				key := paymentMarker(p, operation)
				client.Set(ctx, key, raw, 0)
				if err := retainConfirmedMatchMarker(ctx, client, input, receipt); err != nil {
					t.Fatal(err)
				}
				if age >= time.Hour {
					if client.Exists(ctx, key).Val() != 0 {
						t.Fatal("late SQL confirmation extended an already expired cash receipt")
					}
				} else if ttl := client.PTTL(ctx, key).Val(); ttl <= 0 || ttl > time.Hour-age {
					t.Fatal("cash deadline was reset at confirmation", ttl)
				}
			})
		}
	}
}

func TestConfirmedLegacyMatchMarkerWithoutOriginalTimeIsRemovedOnlyAfterSQLProof(t *testing.T) {
	client, p := matchPaymentRedis(t)
	ctx := context.Background()
	input := MatchPaymentInput{Operation: "charge", Plan: p}
	receipt := map[string]any{"key": p.Key, "paidGold": 100, "paidService": 10, "currency": "CNY"}
	raw, _ := json.Marshal(map[string]any{"fingerprint": paymentHash(input), "receipt": receipt})
	key := paymentMarker(p, "charge")
	client.Set(ctx, key, raw, 0)
	if err := retainConfirmedMatchMarker(ctx, client, input, receipt); err != nil || client.Exists(ctx, key).Val() != 0 {
		t.Fatal("confirmed legacy receipt received a new lifetime", err)
	}
	for _, created := range []int64{0, -1, time.Now().Add(time.Hour).Unix()} {
		raw, _ = json.Marshal(map[string]any{"fingerprint": paymentHash(input), "receipt": receipt, "createdAt": created})
		client.Set(ctx, key, raw, 0)
		if err := retainConfirmedMatchMarker(ctx, client, input, receipt); err == nil || client.PTTL(ctx, key).Val() != -1 {
			t.Fatal("invalid original cash time was accepted", created, err)
		}
	}
}

func TestMatchPaymentQueryRetiresCommittedCashWithoutChangingBalanceOrRevision(t *testing.T) {
	for _, operation := range []string{"charge", "refund", "award"} {
		t.Run(operation, func(t *testing.T) {
			client, p := matchPaymentRedis(t)
			ctx := context.Background()
			charge := map[string]any{"key": p.Key, "paidGold": 100, "paidService": 10, "currency": "CNY"}
			receipt := charge
			state := int64(1)
			detail := &matchPaymentDetail{Fingerprint: paymentHash(p), Status: "charged", TicketIDs: []int64{}, Receipt: charge}
			document := matchPaymentDocument(p, detail, "registered")
			if operation == "refund" {
				receipt = map[string]any{"key": p.Key, "currency": "CNY", "refunded": 110, "restoredTickets": 0, "uncharged": false}
				detail.Status = "refunded"
				detail.Refund = receipt
				document = matchPaymentDocument(p, detail, "refunded")
				state = 4
				input := MatchPaymentInput{Operation: "charge", Plan: p}
				raw, _ := json.Marshal(map[string]any{"fingerprint": paymentHash(input), "receipt": charge, "createdAt": time.Now().Add(-2 * time.Hour).Unix()})
				client.Set(ctx, paymentMarker(p, "charge"), raw, 0)
			} else if operation == "award" {
				p.Key, p.Gold, p.Service, p.Total, p.Kind, p.Amount = "match:7:9:award:42:0", 0, 0, 0, "money", 50
				receipt = map[string]any{"key": p.Key, "currency": "CNY", "kind": "money", "amount": 50, "ticketId": 0}
				document = map[string]any{"registrations": map[string]any{"42": map[string]any{"status": "winner", "rank": 1, "agentId": 7}}, "outbox": []any{}, "prizes": []any{map[string]any{"plan": p, "status": "paid", "authorityReceipt": receipt}}}
				state = 3
			}
			input := MatchPaymentInput{Operation: operation, Plan: p, IncludeService: operation == "refund"}
			raw, _ := json.Marshal(map[string]any{"fingerprint": paymentHash(input), "receipt": receipt, "createdAt": time.Now().Add(-30 * time.Minute).Unix()})
			key := paymentMarker(p, operation)
			client.Set(ctx, key, raw, 0)
			balance, revision := client.HGet(ctx, "player_42", "currency").Val(), client.HGet(ctx, "player_42", "cash_revision").Val()
			db, _ := gogameTestDB(t, matchPaymentSQL(p, document, state, nil, nil)...)
			value, err := (&DBDao{manager: db}).matchPayment(ctx, client, MatchPaymentInput{Operation: "query", Plan: p})
			if err != nil || value == nil {
				t.Fatal("confirmed query failed", value, err)
			}
			if ttl := client.PTTL(ctx, key).Val(); ttl <= 0 || ttl > 30*time.Minute {
				t.Fatal("query left a committed receipt permanent or renewed its time", ttl)
			}
			if operation == "refund" && client.Exists(ctx, paymentMarker(p, "charge")).Val() != 0 {
				t.Fatal("query left the already expired original charge marker")
			}
			if client.HGet(ctx, "player_42", "currency").Val() != balance || client.HGet(ctx, "player_42", "cash_revision").Val() != revision {
				t.Fatal("query replayed money")
			}
		})
	}
}

func TestMatchPaymentQueryDoesNotRetireUnconfirmedConflictingOrFailedSQLReceipts(t *testing.T) {
	for _, kind := range []string{"unconfirmed", "different-receipt", "SQL-failure", "Redis-failure"} {
		t.Run(kind, func(t *testing.T) {
			client, p := matchPaymentRedis(t)
			ctx := context.Background()
			receipt := map[string]any{"key": p.Key, "paidGold": 100, "paidService": 10, "currency": "CNY"}
			detail := &matchPaymentDetail{Fingerprint: paymentHash(p), Status: "charged", TicketIDs: []int64{}, Receipt: receipt}
			if kind == "unconfirmed" {
				detail.Status, detail.Receipt = "prepared", nil
			}
			input := MatchPaymentInput{Operation: "charge", Plan: p}
			original := receipt
			if kind == "different-receipt" {
				original = map[string]any{"key": p.Key, "paidGold": 999, "paidService": 10, "currency": "CNY"}
			}
			raw, _ := json.Marshal(map[string]any{"fingerprint": paymentHash(input), "receipt": original, "createdAt": time.Now().Add(-2 * time.Hour).Unix()})
			key := paymentMarker(p, "charge")
			client.Set(ctx, key, raw, 0)
			var failure error
			if kind == "SQL-failure" {
				failure = errors.New("SQL did not commit")
			}
			if kind == "Redis-failure" {
				client.Close()
			}
			db, _ := gogameTestDB(t, matchPaymentSQL(p, matchPaymentDocument(p, detail, "registered"), 1, nil, failure)...)
			_, err := (&DBDao{manager: db}).matchPayment(ctx, client, MatchPaymentInput{Operation: "query", Plan: p})
			if kind == "SQL-failure" && err == nil || kind != "SQL-failure" && err != nil {
				t.Fatal("query result changed by retention", kind, err)
			}
			if kind != "Redis-failure" && (client.PTTL(ctx, key).Val() != -1 || client.Get(ctx, key).Val() != string(raw)) {
				t.Fatal("query deleted an unconfirmed or different receipt", kind)
			}
		})
	}
}

func TestLateMatchSQLConfirmationRetiresCashMarkerWithoutRepeatingDebit(t *testing.T) {
	client, p := matchPaymentRedis(t)
	ctx := context.Background()
	input := MatchPaymentInput{Operation: "charge", Plan: p}
	prepared := &matchPaymentDetail{Fingerprint: paymentHash(p), Status: "prepared", TicketIDs: []int64{}}
	pending := matchPaymentDocument(p, prepared, "pending")
	steps := append(matchPaymentSQL(p, matchPaymentDocument(p, nil, "pending"), 1, nil, nil), matchPaymentSQL(p, pending, 1, nil, errors.New("SQL unavailable"))...)
	db, _ := gogameTestDB(t, steps...)
	if _, err := (&DBDao{manager: db}).matchPayment(ctx, client, input); err == nil {
		t.Fatal("uncommitted cash acknowledged")
	}
	key := paymentMarker(p, "charge")
	if client.PTTL(ctx, key).Val() != -1 || client.HGet(ctx, "player_42", "currency").Val() != "890" {
		t.Fatal("pending financial evidence expired or original debit missing")
	}
	var marker map[string]any
	if err := json.Unmarshal([]byte(client.Get(ctx, key).Val()), &marker); err != nil || marker["createdAt"] == nil {
		t.Fatal("actual cash marker omitted original operation time", marker, err)
	}
	marker["createdAt"] = time.Now().Add(-2 * time.Hour).Unix()
	raw, _ := json.Marshal(marker)
	client.Set(ctx, key, raw, 0)
	revision := client.HGet(ctx, "player_42", "cash_revision").Val()
	var saved string
	steps = append(matchPaymentSQL(p, pending, 1, nil, nil), matchPaymentSQL(p, pending, 1, func(raw string) { saved = raw }, nil)...)
	db, _ = gogameTestDB(t, steps...)
	if _, err := (&DBDao{manager: db}).matchPayment(ctx, client, input); err != nil {
		t.Fatal(err)
	}
	if client.Exists(ctx, key).Val() != 0 || client.HGet(ctx, "player_42", "currency").Val() != "890" || client.HGet(ctx, "player_42", "cash_revision").Val() != revision {
		t.Fatal("late SQL confirmation renewed the marker or repeated its debit")
	}
	var committed map[string]any
	if err := DecodeGogameJSON(saved, &committed); err != nil {
		t.Fatal(err)
	}
	db, _ = gogameTestDB(t, matchPaymentSQL(p, committed, 1, nil, nil)...)
	if _, err := (&DBDao{manager: db}).matchPayment(ctx, client, MatchPaymentInput{Operation: "query", Plan: p}); err != nil || client.Exists(ctx, key).Val() != 0 {
		t.Fatal("status query recreated the expired marker", err)
	}
}

func TestPublicMatchQueryUsesOptionalRedisForConfirmedRetention(t *testing.T) {
	for _, configured := range []bool{false, true} {
		t.Run(map[bool]string{false: "without-Redis", true: "with-Redis"}[configured], func(t *testing.T) {
			client, p := matchPaymentRedis(t)
			ctx := context.Background()
			old := redisDao
			defer func() { redisDao = old }()
			redisDao = nil
			if configured {
				redisDao = &RedisDao{Client: client}
			}
			receipt := map[string]any{"key": p.Key, "paidGold": 100, "paidService": 10, "currency": "CNY"}
			input := MatchPaymentInput{Operation: "charge", Plan: p}
			raw, _ := json.Marshal(map[string]any{"fingerprint": paymentHash(input), "receipt": receipt, "createdAt": time.Now().Add(-30 * time.Minute).Unix()})
			key := paymentMarker(p, "charge")
			client.Set(ctx, key, raw, 0)
			detail := &matchPaymentDetail{Fingerprint: paymentHash(p), Status: "charged", TicketIDs: []int64{}, Receipt: receipt}
			db, _ := gogameTestDB(t, matchPaymentSQL(p, matchPaymentDocument(p, detail, "registered"), 1, nil, nil)...)
			query, _ := json.Marshal(MatchPaymentInput{Operation: "query", Plan: p})
			value, err := (&DBDao{manager: db}).gogameMatchPayment(ctx, string(query))
			if err != nil || value.(map[string]any)["status"] != "charged" {
				t.Fatal("SQL status query required Redis", value, err)
			}
			ttl := client.PTTL(ctx, key).Val()
			if configured && (ttl <= 0 || ttl > 30*time.Minute) || !configured && ttl != -1 {
				t.Fatal("optional Redis retention route was incorrect", configured, ttl)
			}
		})
	}
}
