package dao

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestLegacyScratchRetirementRequiresOriginalClaimedSQLTicket(t *testing.T) {
	now := time.Now().Unix()
	for _, test := range []struct {
		name    string
		edit    func(map[string]any)
		allowed bool
	}{
		{"old-claimed", func(map[string]any) {}, true},
		{"not-claimed", func(v map[string]any) { v["claimed"] = false }, false},
		{"within-hour", func(v map[string]any) { v["claimedAt"] = (now - 3599) * 1000 }, false},
		{"unknown-time", func(v map[string]any) { delete(v, "claimedAt") }, false},
		{"future", func(v map[string]any) { v["claimedAt"] = (now + 1) * 1000 }, false},
		{"other-user", func(v map[string]any) { v["userId"] = 43 }, false},
		{"other-currency", func(v map[string]any) { v["currency"] = "USD" }, false},
		{"other-round", func(v map[string]any) { v["roundId"] = "other" }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			ticket := map[string]any{"claimed": true, "claimedAt": (now - 3600) * 1000, "userId": 42, "currency": "CNY", "serialFull": "ticket", "roundId": "original"}
			test.edit(ticket)
			raw, _ := json.Marshal(ticket)
			dd, script := soldTestDB(t, soldSQLStep{contains: "storage_key", columns: []string{"payload", "expire_at"}, values: [][]driver.Value{{soldOwnerRaw(42), int64(0)}}}, soldSQLStep{contains: "storage_key", columns: []string{"payload", "expire_at"}, values: [][]driver.Value{{string(raw), int64(0)}}})
			proof, err := dd.ScratchClaimedOwner(context.Background(), ScratchScope{Agent: 7, GameID: 91061}, "ticket", "original", now)
			if err != nil || (proof != nil) != test.allowed || len(script.steps) != 0 {
				t.Fatal("invalid retirement proof", proof, err)
			}
		})
	}
	dd, _ := soldTestDB(t, soldSQLStep{contains: "storage_key", err: errors.New("database unavailable")})
	if _, err := dd.ScratchClaimedOwner(context.Background(), ScratchScope{Agent: 7, GameID: 91061}, "ticket", "original", now); err == nil {
		t.Fatal("SQL error treated as confirmed")
	}
}

func TestLegacyScratchRetirementOnlyRemovesProvenOriginalMappingAndPendingReference(t *testing.T) {
	for _, mode := range []string{"confirmed", "no-proof", "new-snapshot", "changed-mapping", "existing-snapshot", "SQL-error"} {
		t.Run(mode, func(t *testing.T) {
			rd, client := retentionRedis(t)
			ctx := context.Background()
			key := "gogame:scratch:sold:7:91061:ticket"
			pending := "gogame:scratch:pending:7:91061:42:CNY"
			client.Set(ctx, key, "original", 0)
			client.SAdd(ctx, pending, "original", "other")
			if mode == "existing-snapshot" {
				client.HSet(ctx, "lottery_finance_rounds", "original", "unconfirmed financial state")
			}
			calls := 0
			_, n, err := rd.RemoveProvenLegacyScratchSold(0, func(_ context.Context, scope ScratchScope, serial, id string, now int64) (*ScratchRetiredOwner, error) {
				calls++
				if scope.Agent != 7 || scope.GameID != 91061 || serial != "ticket" || id != "original" {
					t.Fatal("scope changed")
				}
				if mode == "no-proof" {
					return nil, nil
				}
				if mode == "SQL-error" {
					return nil, errors.New("SQL unavailable")
				}
				if mode == "new-snapshot" {
					client.HSet(ctx, "lottery_finance_rounds", "original", "new live state")
				}
				if mode == "changed-mapping" {
					client.Set(ctx, key, "another-round", 0)
				}
				return &ScratchRetiredOwner{UserID: 42, Currency: "CNY"}, nil
			})
			if mode == "SQL-error" {
				if err == nil {
					t.Fatal("maintenance swallowed SQL failure")
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if mode == "confirmed" {
				if n != 1 || client.Exists(ctx, key).Val() != 0 || client.SIsMember(ctx, pending, "original").Val() {
					t.Fatal("confirmed mapping remains")
				}
			} else if n != 0 || client.Exists(ctx, key).Val() != 1 || !client.SIsMember(ctx, pending, "original").Val() {
				t.Fatal("unconfirmed or concurrent rights removed")
			}
			if !client.SIsMember(ctx, pending, "other").Val() {
				t.Fatal("other pending rights removed")
			}
			if mode == "existing-snapshot" && calls != 0 {
				t.Fatal("live state queried for retirement")
			}
		})
	}
}
