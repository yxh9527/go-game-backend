package dao

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"testing"
)

func TestTerminalSnapshotDeletedAtOneHourWithoutTouchingFundsOrOtherRounds(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()
	rd := NewFinanceRedis(client)
	ctx := context.Background()
	client.HSet(ctx, "player_42", "currency", 12345, "cash_revision", 8)
	client.Set(ctx, "gogame:cash-carry:7:42:CNY", "original carry", 0)
	for _, state := range []string{"SETTLED", "VOIDED", "EXPIRED"} {
		var raw string
		if state == "EXPIRED" {
			raw = `{"state":"EXPIRED","updatedAt":9999,"reservation":{"expiresAt":1000,"releasePending":true}}`
		} else {
			raw = `{"state":"` + state + `","updatedAt":9999,"settlement":{"completedAt":1000,"reportsIndexed":false}}`
		}
		client.HSet(ctx, "lottery_finance_rounds", state, raw)
		client.SAdd(ctx, financeReconcileSetKey, state)
		if removed, err := rd.DeleteExpiredFinanceRound(ctx, state, 4599, 3600); err != nil || removed {
			t.Fatal("snapshot expired before an hour", state, removed, err)
		}
		if removed, err := rd.DeleteExpiredFinanceRound(ctx, state, 4600, 3600); err != nil || !removed {
			t.Fatal("terminal snapshot was not deleted at an hour", state, removed, err)
		}
		if client.HExists(ctx, "lottery_finance_rounds", state).Val() || client.SIsMember(ctx, financeReconcileSetKey, state).Val() {
			t.Fatal("snapshot or stale retry entry remains")
		}
	}
	for _, state := range []string{"BETTING", "RESERVED"} {
		client.HSet(ctx, "lottery_finance_rounds", state, `{"state":"`+state+`","updatedAt":1}`)
		if removed, err := rd.DeleteExpiredFinanceRound(ctx, state, 10000, 3600); err != nil || removed {
			t.Fatal("active financial round was removed", state, removed, err)
		}
	}
	if client.Exists(ctx, "lottery_finance_rounds_archive").Val() != 0 || client.HLen(ctx, "lottery_finance_rounds").Val() != 2 {
		t.Fatal("cleanup created an archive or removed unrelated rounds")
	}
	if client.HGet(ctx, "player_42", "currency").Val() != "12345" || client.HGet(ctx, "player_42", "cash_revision").Val() != "8" || client.Get(ctx, "gogame:cash-carry:7:42:CNY").Val() != "original carry" {
		t.Fatal("retention cleanup changed financial data")
	}
}

func TestRetentionChecksCurrentRedisStateAndErrorsDoNotPartiallyDelete(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()
	rd := NewFinanceRedis(client)
	ctx := context.Background()
	// A stale process must not remove an account's newly changed active state.
	client.HSet(ctx, "lottery_finance_rounds", "changed", `{"state":"BETTING","updatedAt":1000}`)
	if removed, err := rd.DeleteExpiredFinanceRound(ctx, "changed", 5000, 3600); err != nil || removed {
		t.Fatal(removed, err)
	}
	client.HSet(ctx, "lottery_finance_rounds", "changed", `{"state":"SETTLED","updatedAt":4999,"settlement":{"completedAt":4999}}`)
	if removed, err := rd.DeleteExpiredFinanceRound(ctx, "changed", 5000, 3600); err != nil || removed {
		t.Fatal("newer terminal snapshot deleted", removed, err)
	}
	client.HSet(ctx, "lottery_finance_rounds", "old", `{"state":"SETTLED","updatedAt":1}`)
	client.Set(ctx, financeReconcileSetKey, "invalid type", 0)
	if _, err := rd.DeleteExpiredFinanceRound(ctx, "old", 5000, 3600); err == nil {
		t.Fatal("invalid retry key ignored")
	}
	if !client.HExists(ctx, "lottery_finance_rounds", "old").Val() {
		t.Fatal("failed cleanup partially deleted snapshot")
	}
	client.Del(ctx, financeReconcileSetKey)
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := rd.DeleteExpiredFinanceRound(cancelled, "old", 5000, 3600); err == nil || !client.HExists(ctx, "lottery_finance_rounds", "old").Val() {
		t.Fatal("cancelled cleanup changed the original snapshot", err)
	}
	if removed, err := rd.DeleteExpiredFinanceRound(ctx, "old", 5000, 3600); err != nil || !removed {
		t.Fatal("legacy timestamp was not honored", removed, err)
	}
	if removed, err := rd.DeleteExpiredFinanceRound(ctx, "old", 5000, 3600); err != nil || !removed {
		t.Fatal("repeated cleanup is not idempotent", removed, err)
	}
}

func TestLegacyArchiveRemovedWithoutDeletingAnyOtherRedisKey(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()
	rd := NewFinanceRedis(client)
	ctx := context.Background()
	client.HSet(ctx, "lottery_finance_rounds_archive", "old", "old archived payload")
	client.HSet(ctx, "lottery_finance_rounds", "active", "active payload")
	if err := rd.RemoveLegacyFinanceArchive(); err != nil {
		t.Fatal(err)
	}
	if err := rd.RemoveLegacyFinanceArchive(); err != nil {
		t.Fatal(err)
	}
	if client.Exists(ctx, "lottery_finance_rounds_archive").Val() != 0 || client.HGet(ctx, "lottery_finance_rounds", "active").Val() != "active payload" {
		t.Fatal("legacy cleanup affected unrelated keys")
	}
}
