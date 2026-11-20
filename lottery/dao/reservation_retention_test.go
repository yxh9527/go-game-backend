package dao

import (
	"context"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"testing"
	"time"
)

func retentionRedis(t *testing.T) (*RedisDao, *redis.Client) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	return NewFinanceRedis(client), client
}
func releasedLedger(round string, at int64) string {
	return EncodeReservationLedger(ReservationLedgerReleased, "50", "settle:original", round, 7, "pool_1", "SETTLE", at)
}

func TestReleasedLedgerCleanupHonorsOneHourAndPreservesActiveUnknownAndExistingRounds(t *testing.T) {
	rd, client := retentionRedis(t)
	ctx := context.Background()
	now := time.Now().Unix()
	values := map[string]string{
		"old": releasedLedger("gone", now-3600), "fresh": releasedLedger("fresh", now-3599), "live-round": releasedLedger("still-present", now-3600),
		"active":  EncodeReservationLedger(ReservationLedgerActive, "50", "", "active-round", 7, "pool_1", "", now-99999),
		"unknown": "BROKEN|original", "bad-time": releasedLedger("gone", 0), "future": releasedLedger("gone", now+1),
	}
	for id, raw := range values {
		client.HSet(ctx, reservationLedgerRedisKey, id, raw)
	}
	client.HSet(ctx, "lottery_finance_rounds", "still-present", `{"state":"RESERVED"}`)
	client.ZAdd(ctx, "agent_reserved_data", redis.Z{Member: "7_pool_1", Score: 500})
	client.HSet(ctx, "player_42", "currency", 10000, "cash_revision", 8)
	_, count, err := rd.RemoveReleasedReservationLedgers(0, now)
	if err != nil || count != 1 {
		t.Fatal(count, err)
	}
	if client.HExists(ctx, reservationLedgerRedisKey, "old").Val() {
		t.Fatal("old released receipt remains")
	}
	for id, raw := range values {
		if id != "old" && client.HGet(ctx, reservationLedgerRedisKey, id).Val() != raw {
			t.Fatal("cleanup changed pending or unknown original", id)
		}
	}
	if client.ZScore(ctx, "agent_reserved_data", "7_pool_1").Val() != 500 || client.HGet(ctx, "player_42", "currency").Val() != "10000" || client.HGet(ctx, "player_42", "cash_revision").Val() != "8" {
		t.Fatal("cleanup changed funds or pool")
	}
	client.HDel(ctx, "lottery_finance_rounds", "still-present")
	_, count, err = rd.RemoveReleasedReservationLedgers(0, now)
	if err != nil || count != 1 {
		t.Fatal("receipt not removed after original snapshot left", count, err)
	}
}

func TestReleasedLedgerCleanupIsBoundedAndMakesProgressAcrossOldRecords(t *testing.T) {
	rd, client := retentionRedis(t)
	ctx := context.Background()
	now := time.Now().Unix()
	for i := 0; i < 205; i++ {
		client.HSet(ctx, reservationLedgerRedisKey, fmt.Sprint(i), releasedLedger(fmt.Sprint(i), now-3600))
	}
	var cursor uint64
	total := 0
	for pass := 0; pass < 20 && total < 205; pass++ {
		next, count, err := rd.RemoveReleasedReservationLedgers(cursor, now)
		if err != nil || count > 100 {
			t.Fatal("cleanup exceeded deletion budget", count, err)
		}
		cursor = next
		total += count
	}
	if total != 205 || client.HLen(ctx, reservationLedgerRedisKey).Val() != 0 {
		t.Fatal("old receipts accumulated", total)
	}
}

func TestReleasedLedgerCleanupRechecksOriginalAndRoundAtomically(t *testing.T) {
	rd, client := retentionRedis(t)
	ctx := context.Background()
	now := time.Now().Unix()
	original := releasedLedger("round", now-3600)
	client.HSet(ctx, reservationLedgerRedisKey, "changed", EncodeReservationLedger(ReservationLedgerActive, "50", "", "round", 7, "pool_1", "", now))
	n, err := removeReleasedReservationsScript.Run(ctx, rd.cli, []string{reservationLedgerRedisKey, "lottery_finance_rounds"}, "changed", original, "round").Int()
	if err != nil || n != 0 {
		t.Fatal("changed original was removed", n, err)
	}
	client.HSet(ctx, reservationLedgerRedisKey, "changed", original)
	client.HSet(ctx, "lottery_finance_rounds", "round", `{"state":"RESERVED"}`)
	n, err = removeReleasedReservationsScript.Run(ctx, rd.cli, []string{reservationLedgerRedisKey, "lottery_finance_rounds"}, "changed", original, "round").Int()
	if err != nil || n != 0 {
		t.Fatal("newly present round was ignored", n, err)
	}
	client.Del(ctx, "lottery_finance_rounds")
	client.Set(ctx, "lottery_finance_rounds", "bad type", 0)
	if _, _, err = rd.RemoveReleasedReservationLedgers(0, now); err == nil {
		t.Fatal("invalid round key ignored")
	}
	if client.HGet(ctx, reservationLedgerRedisKey, "changed").Val() != original {
		t.Fatal("failed cleanup partially deleted receipt")
	}
}

func TestLateLegacyReleaseAfterCleanupCannotConsumeAnotherRoundsReservation(t *testing.T) {
	rd, client := retentionRedis(t)
	ctx := context.Background()
	now := time.Now().Unix()
	amount := decimal.NewFromInt(50)
	client.HSet(ctx, reservationLedgerRedisKey, "old-res", EncodeReservationLedger(ReservationLedgerReleased, "50", "settle:old", "old-round", 7, "pool_1", "SETTLE", now-3600))
	if result, err := rd.TryReserveAgentPoolWithLedger(7, "pool_1", decimal.NewFromInt(1000), amount, "other-res", "other-round"); err != nil || !result.Ok {
		t.Fatal(result, err)
	}
	if _, count, err := rd.RemoveReleasedReservationLedgers(0, now); err != nil || count != 1 {
		t.Fatal(count, err)
	}
	for i := 0; i < 2; i++ {
		result, err := rd.ReleaseAgentReservedWithLedger(7, "pool_1", amount, "old-res", "old-round", "settle:old", "SETTLE", true)
		if err != nil || result.Ok {
			t.Fatal("missing old evidence allowed another pool release", result, err)
		}
	}
	if !rd.GetAgentReserved(7, "pool_1").Equal(amount) || client.HExists(ctx, reservationLedgerRedisKey, "old-res").Val() {
		t.Fatal("late release consumed another round or recreated old receipt")
	}
	if err := rd.SeedReservationLedgerFromRound(7, "pool_1", amount, "old-res", "old-round"); err == nil {
		t.Fatal("stale adjustment recreated a cleaned ledger")
	}
}

func TestLegacyMigrationRequiresTheExactOriginalActiveReservation(t *testing.T) {
	rd, client := retentionRedis(t)
	ctx := context.Background()
	amount := decimal.NewFromInt(50)
	client.ZAdd(ctx, "agent_reserved_data", redis.Z{Member: "7_pool_1", Score: 100})
	for _, raw := range []string{
		`{"state":"SETTLED","agent":7,"poolSymbol":"pool_1","reservation":{"reservationId":"res","status":"RESERVED","totalPayoutCny":"50"}}`,
		`{"state":"RESERVED","agent":8,"poolSymbol":"pool_1","reservation":{"reservationId":"res","status":"RESERVED","totalPayoutCny":"50"}}`,
		`{"state":"RESERVED","agent":7,"poolSymbol":"pool_1","reservation":{"reservationId":"different","status":"RESERVED","totalPayoutCny":"50"}}`,
		`{"state":"RESERVED","agent":7,"poolSymbol":"pool_1","reservation":{"reservationId":"res","status":"RESERVED","totalPayoutCny":"60"}}`,
	} {
		client.HSet(ctx, "lottery_finance_rounds", "round", raw)
		result, err := rd.ReleaseAgentReservedWithLedger(7, "pool_1", amount, "res", "round", "settle:res", "SETTLE", true)
		if err != nil || result.Ok || client.HExists(ctx, reservationLedgerRedisKey, "res").Val() {
			t.Fatal("mismatched reservation migrated", raw, result, err)
		}
	}
	client.HSet(ctx, "lottery_finance_rounds", "round", `{"state":"RESERVED","agent":7,"poolSymbol":"pool_1","reservation":{"reservationId":"res","status":"RESERVED","totalPayoutCny":"50"}}`)
	if err := rd.SeedReservationLedgerFromRound(7, "pool_1", amount, "res", "round"); err != nil {
		t.Fatal(err)
	}
	if !rd.GetAgentReserved(7, "pool_1").Equal(decimal.NewFromInt(100)) {
		t.Fatal("migration changed actual pool occupancy")
	}
	result, err := rd.ReleaseAgentReservedWithLedger(7, "pool_1", amount, "res", "round", "settle:res", "SETTLE", true)
	if err != nil || !result.Ok || result.AlreadyReleased {
		t.Fatal(result, err)
	}
	result, err = rd.ReleaseAgentReservedWithLedger(7, "pool_1", amount, "res", "round", "settle:res", "SETTLE", true)
	if err != nil || !result.Ok || !result.AlreadyReleased || !rd.GetAgentReserved(7, "pool_1").Equal(amount) {
		t.Fatal("first release or repeat changed pool more than once", result, err)
	}
}

func TestReleasedReceiptRepeatDoesNotRenewLifetimeAndWrongScopeCannotReleasePool(t *testing.T) {
	rd, client := retentionRedis(t)
	ctx := context.Background()
	amount := decimal.NewFromInt(50)
	if result, err := rd.TryReserveAgentPoolWithLedger(7, "pool_1", decimal.NewFromInt(1000), amount, "res", "round"); err != nil || !result.Ok {
		t.Fatal(result, err)
	}
	for _, input := range []struct {
		agent       int64
		pool, round string
		amount      decimal.Decimal
	}{
		{8, "pool_1", "round", amount}, {7, "other_1", "round", amount}, {7, "pool_1", "other-round", amount}, {7, "pool_1", "round", decimal.NewFromInt(60)},
	} {
		result, err := rd.ReleaseAgentReservedWithLedger(input.agent, input.pool, input.amount, "res", input.round, "release", "SETTLE", false)
		if err != nil || result.Ok {
			t.Fatal("mismatched scope released an active receipt", input, result, err)
		}
	}
	if !rd.GetAgentReserved(7, "pool_1").Equal(amount) {
		t.Fatal("invalid release changed pool")
	}
	result, err := rd.ReleaseAgentReservedWithLedger(7, "pool_1", amount, "res", "round", "release", "SETTLE", false)
	if err != nil || !result.Ok {
		t.Fatal(result, err)
	}
	now := time.Now().Unix()
	original := releasedLedger("round", now-3600)
	client.HSet(ctx, reservationLedgerRedisKey, "res", original)
	result, err = rd.ReleaseAgentReservedWithLedger(7, "pool_1", amount, "res", "round", "release", "SETTLE", false)
	if err != nil || !result.Ok || !result.AlreadyReleased || client.HGet(ctx, reservationLedgerRedisKey, "res").Val() != original {
		t.Fatal("repeat renewed original released time", result, err)
	}
	if _, count, err := rd.RemoveReleasedReservationLedgers(0, now); err != nil || count != 1 {
		t.Fatal("repeat prevented expiry", count, err)
	}
}
