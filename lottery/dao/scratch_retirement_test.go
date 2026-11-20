package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func retiredScratchRound(state string, at int64) map[string]any {
	return map[string]any{"agent": 7, "gameId": 91061, "state": state, "updatedAt": at,
		"settlement": map[string]any{"completedAt": at},
		"scratch":    map[string]any{"roundId": "original", "serialFull": "ticket", "userId": 42, "currency": "CNY", "mirrored": true, "soldMirrored": true, "claimed": true, "refunded": true, "kind": "ggl1", "winAmount": 200}}
}

func TestScratchRefundCleanupUsesOriginalDeadlineAndDoesNotRenewOrChangeFinancialState(t *testing.T) {
	rd, client := retentionRedis(t)
	ctx := context.Background()
	scope := ScratchScope{Agent: 7, GameID: 91061, UserID: 42, Currency: "CNY"}
	pending, sold := "gogame:scratch:pending:"+scope.Key(), "gogame:scratch:sold:"+scope.Market()+":ticket"
	raw, _ := json.Marshal(retiredScratchRound("VOIDED", time.Now().Unix()-600))
	client.HSet(ctx, "lottery_finance_rounds", "original", raw)
	client.SAdd(ctx, pending, "original", "another-active-ticket")
	client.Set(ctx, sold, "original", 0)
	if err := rd.DropScratchPending(scope, "original", "ticket"); err != nil {
		t.Fatal(err)
	}
	if client.SIsMember(ctx, pending, "original").Val() || !client.SIsMember(ctx, pending, "another-active-ticket").Val() || client.Get(ctx, sold).Val() != "original" {
		t.Fatal("cleanup changed another liability or erased sold protection")
	}
	ttl := client.PTTL(ctx, sold).Val()
	if ttl <= 0 || ttl > 50*time.Minute {
		t.Fatal("expiry was restarted from the cleanup time", ttl)
	}
	client.PExpire(ctx, sold, time.Minute)
	if err := rd.DropScratchPending(scope, "original", "ticket"); err != nil || client.PTTL(ctx, sold).Val() > time.Minute {
		t.Fatal("repeat refund extended shorter remaining expiry", err)
	}
	if client.HGet(ctx, "lottery_finance_rounds", "original").Val() != string(raw) {
		t.Fatal("refund cleanup rewrote the financial snapshot")
	}
}

func TestScratchRefundCleanupRejectsUnconfirmedForeignAndChangedOwnership(t *testing.T) {
	cases := []struct {
		name string
		edit func(map[string]any)
	}{
		{"active", func(r map[string]any) { r["state"] = "RESERVED" }},
		{"settled", func(r map[string]any) { r["state"] = "SETTLED" }},
		{"not-mirrored", func(r map[string]any) { r["scratch"].(map[string]any)["mirrored"] = false }},
		{"sold-not-mirrored", func(r map[string]any) { r["scratch"].(map[string]any)["soldMirrored"] = false }},
		{"legacy-sold-unknown", func(r map[string]any) { delete(r["scratch"].(map[string]any), "soldMirrored") }},
		{"not-refunded", func(r map[string]any) { r["scratch"].(map[string]any)["refunded"] = false }},
		{"agent", func(r map[string]any) { r["agent"] = 8 }},
		{"game", func(r map[string]any) { r["gameId"] = 91063 }},
		{"user", func(r map[string]any) { r["scratch"].(map[string]any)["userId"] = 43 }},
		{"currency", func(r map[string]any) { r["scratch"].(map[string]any)["currency"] = "USD" }},
		{"serial", func(r map[string]any) { r["scratch"].(map[string]any)["serialFull"] = "other" }},
		{"round", func(r map[string]any) { r["scratch"].(map[string]any)["roundId"] = "other" }},
		{"missing-time", func(r map[string]any) { delete(r, "settlement"); r["updatedAt"] = 0 }},
		{"future-time", func(r map[string]any) { r["settlement"].(map[string]any)["completedAt"] = time.Now().Unix() + 100 }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			rd, client := retentionRedis(t)
			ctx := context.Background()
			scope := ScratchScope{7, 91061, 42, "CNY"}
			round := retiredScratchRound("VOIDED", time.Now().Unix()-3600)
			test.edit(round)
			raw, _ := json.Marshal(round)
			pending, sold := "gogame:scratch:pending:"+scope.Key(), "gogame:scratch:sold:"+scope.Market()+":ticket"
			client.HSet(ctx, "lottery_finance_rounds", "original", raw)
			client.SAdd(ctx, pending, "original")
			client.Set(ctx, sold, "original", 0)
			if err := rd.DropScratchPending(scope, "original", "ticket"); err == nil {
				t.Fatal("invalid cleanup accepted")
			}
			if !client.SIsMember(ctx, pending, "original").Val() || client.PTTL(ctx, sold).Val() != -1 || client.HGet(ctx, "lottery_finance_rounds", "original").Val() != string(raw) {
				t.Fatal("failed cleanup partially changed original state")
			}
		})
	}
}

func TestScratchClaimRetirementPreservesOriginalPrizeAndDoesNotExtendExpiryOnRepeat(t *testing.T) {
	rd, client := retentionRedis(t)
	ctx := context.Background()
	scope := ScratchScope{7, 91061, 42, "CNY"}
	pending, sold := "gogame:scratch:pending:"+scope.Key(), "gogame:scratch:sold:"+scope.Market()+":ticket"
	totals := fmt.Sprintf("gogame:scratch:accumulated:%d:%d:%s", scope.Agent, scope.UserID, scope.Currency)
	round := retiredScratchRound("SETTLED", time.Now().Unix()-600)
	raw, _ := json.Marshal(round)
	client.HSet(ctx, "lottery_finance_rounds", "original", raw)
	client.Set(ctx, sold, "original", 0)
	client.SAdd(ctx, pending, "original")
	for _, wrong := range []struct {
		kind  string
		prize int64
	}{{"GGL2", 200}, {"GGL1", 999}} {
		if err := rd.FinishScratch(scope, "original", "ticket", wrong.kind, wrong.prize); err == nil || !client.SIsMember(ctx, pending, "original").Val() || client.HLen(ctx, totals).Val() != 0 {
			t.Fatal("mismatched claim mutated the accumulated prize", wrong, err)
		}
	}
	if err := rd.FinishScratch(scope, "original", "ticket", "GGL1", 200); err != nil {
		t.Fatal(err)
	}
	if client.HGet(ctx, totals, "GGL1").Val() != "200" || client.PTTL(ctx, sold).Val() > 50*time.Minute || client.PTTL(ctx, sold).Val() <= 0 {
		t.Fatal("claim changed original prize or expiry")
	}
	client.PExpire(ctx, sold, time.Minute)
	if err := rd.FinishScratch(scope, "original", "ticket", "GGL1", 200); err != nil || client.HGet(ctx, totals, "GGL1").Val() != "200" || client.PTTL(ctx, sold).Val() > time.Minute {
		t.Fatal("repeated claim counted prize or renewed expiry", err)
	}
}

func TestScratchRetirementDoesNotRecreateMissingMappingOrOverwriteAnotherRound(t *testing.T) {
	for _, state := range []string{"SETTLED", "VOIDED"} {
		for _, mapping := range []string{"", "another-round"} {
			t.Run(state+"/"+mapping, func(t *testing.T) {
				rd, client := retentionRedis(t)
				ctx := context.Background()
				scope := ScratchScope{7, 91061, 42, "CNY"}
				raw, _ := json.Marshal(retiredScratchRound(state, time.Now().Unix()-60))
				client.HSet(ctx, "lottery_finance_rounds", "original", raw)
				pending, sold := "gogame:scratch:pending:"+scope.Key(), "gogame:scratch:sold:"+scope.Market()+":ticket"
				client.SAdd(ctx, pending, "original")
				if mapping != "" {
					client.Set(ctx, sold, mapping, 0)
				}
				var err error
				if state == "SETTLED" {
					err = rd.FinishScratch(scope, "original", "ticket", "GGL1", 200)
				} else {
					err = rd.DropScratchPending(scope, "original", "ticket")
				}
				if mapping == "" && (err != nil || client.Exists(ctx, sold).Val() != 0) {
					t.Fatal("missing sold cache was recreated", err)
				}
				if mapping != "" && (err == nil || client.Get(ctx, sold).Val() != mapping || client.PTTL(ctx, sold).Val() != -1 || !client.SIsMember(ctx, pending, "original").Val()) {
					t.Fatal("old cleanup changed a newer mapping", err)
				}
			})
		}
	}
}

func TestScratchRetirementDeletesAlreadyOverdueMappingAndRejectsMissingSnapshot(t *testing.T) {
	rd, client := retentionRedis(t)
	ctx := context.Background()
	scope := ScratchScope{7, 91061, 42, "CNY"}
	sold := "gogame:scratch:sold:" + scope.Market() + ":ticket"
	raw, _ := json.Marshal(retiredScratchRound("VOIDED", time.Now().Unix()-3601))
	client.HSet(ctx, "lottery_finance_rounds", "original", raw)
	client.Set(ctx, sold, "original", 0)
	if err := rd.DropScratchPending(scope, "original", "ticket"); err != nil || client.Exists(ctx, sold).Val() != 0 {
		t.Fatal("overdue mapping retained", err)
	}
	client.HDel(ctx, "lottery_finance_rounds", "original")
	client.Set(ctx, sold, "original", 0)
	if err := rd.DropScratchPending(scope, "original", "ticket"); err == nil || client.PTTL(ctx, sold).Val() != -1 {
		t.Fatal("cleanup guessed an unknown completion time", err)
	}
}
