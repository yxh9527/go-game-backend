package unified

import (
	"context"
	"encoding/json"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"lottery/dao"
	"micro_service/services"
	"sync"
	"testing"
	"time"
)

func retentionService(t *testing.T) (*LotteryService, *redis.Client) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	return &LotteryService{rds: dao.NewFinanceRedis(client), roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}}, client
}

func TestStartupDeletesOldTerminalRowsAndLegacyArchiveButRetainsRecentAndActive(t *testing.T) {
	service, client := retentionService(t)
	ctx := context.Background()
	now := time.Now().Unix()
	for _, row := range []*financeRound{
		{RoundID: "settled", State: "SETTLED", Settlement: &financeSettlement{CompletedAt: now - 3600}},
		{RoundID: "voided", State: "VOIDED", Settlement: &financeSettlement{CompletedAt: now - 3600}},
		{RoundID: "expired", State: "EXPIRED", Reservation: &financeReservation{ExpiresAt: now - 3600}},
		{RoundID: "recent", State: "SETTLED", Settlement: &financeSettlement{CompletedAt: now - 60}},
		{RoundID: "active", State: "BETTING", UpdatedAt: now - 99999},
	} {
		raw, _ := json.Marshal(row)
		client.HSet(ctx, financeRoundHash, row.RoundID, string(raw))
	}
	client.HSet(ctx, "lottery_finance_rounds_archive", "obsolete", "legacy payload")
	service.loadRoundStates()
	if len(service.rounds) != 2 || service.rounds["recent"] == nil || service.rounds["active"] == nil || client.HLen(ctx, financeRoundHash).Val() != 2 || client.Exists(ctx, "lottery_finance_rounds_archive").Val() != 0 {
		t.Fatal("startup retained old snapshots or lost active ones", service.rounds)
	}
}

func TestStateQueryDoesNotReadOldArchiveAndExpiredRoundCannotBeResaved(t *testing.T) {
	service, client := retentionService(t)
	ctx := context.Background()
	now := time.Now().Unix()
	old := &financeRound{RoundID: "old", State: "SETTLED", Settlement: &financeSettlement{CompletedAt: now - 3600}}
	raw, _ := json.Marshal(old)
	client.HSet(ctx, financeRoundHash, old.RoundID, string(raw))
	service.rounds[old.RoundID] = old
	response, err := service.GetRoundFinanceState(ctx, &services.GetRoundFinanceStateReq{RoundId: old.RoundID})
	if err != nil || response.Code != services.ErrorCode_ROUND_NOT_FOUND || len(service.rounds) != 0 || client.HExists(ctx, financeRoundHash, old.RoundID).Val() {
		t.Fatal("expired query returned or retained original snapshot", response, err)
	}
	if err := service.saveRoundLocked(old); err != nil {
		t.Fatal(err)
	}
	if client.HExists(ctx, financeRoundHash, old.RoundID).Val() || len(service.rounds) != 0 {
		t.Fatal("late report retry recreated expired snapshot")
	}
	client.HSet(ctx, "lottery_finance_rounds_archive", old.RoundID, string(raw))
	response, err = service.GetRoundFinanceState(ctx, &services.GetRoundFinanceStateReq{RoundId: old.RoundID})
	if err != nil || response.Code != services.ErrorCode_ROUND_NOT_FOUND {
		t.Fatal("old Redis archive is still readable", response, err)
	}
}

func TestRecentSnapshotReplaysAndLegacySaveCannotRenewTerminalLifetime(t *testing.T) {
	service, client := retentionService(t)
	ctx := context.Background()
	ref := time.Now().Unix() - 100
	round := &financeRound{RoundID: "recent", State: "SETTLED", UpdatedAt: ref, Settlement: &financeSettlement{SettlementID: "original", Status: financeSettlementStatusCompleted}}
	if err := service.saveRoundLocked(round); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		response, err := service.GetRoundFinanceState(ctx, &services.GetRoundFinanceStateReq{RoundId: round.RoundID})
		if err != nil || response.Code != services.ErrorCode_OK || response.SettlementId != "original" {
			t.Fatal(response, err)
		}
		if err := service.saveRoundLocked(round); err != nil {
			t.Fatal(err)
		}
	}
	var stored financeRound
	if err := json.Unmarshal([]byte(client.HGet(ctx, financeRoundHash, round.RoundID).Val()), &stored); err != nil || stored.UpdatedAt != ref {
		t.Fatal("queries or report saves renewed original terminal time", stored, err)
	}
	if !service.isRoundExpired(round, ref+3600) {
		t.Fatal("legacy fallback lifetime was extended")
	}
}

func TestCleanupBatchBoundsDeletionAndPendingReportsCannotExtendExpiry(t *testing.T) {
	service, client := retentionService(t)
	ctx := context.Background()
	now := time.Now().Unix()
	for _, id := range []string{"one", "two", "three"} {
		round := reportRound(1)
		round.RoundID = id
		round.Settlement.CompletedAt = now - 3600
		raw, _ := json.Marshal(round)
		client.HSet(ctx, financeRoundHash, id, string(raw))
		service.rounds[id] = round
	}
	if n := service.purgeExpiredRoundsLocked(now, 2); n != 2 || len(service.rounds) != 1 || client.HLen(ctx, financeRoundHash).Val() != 1 {
		t.Fatal("cleanup ignored batch or pending reports extended expiry", n, len(service.rounds))
	}
	if n := service.purgeExpiredRoundsLocked(now, 2); n != 1 || len(service.rounds) != 0 || client.Exists(ctx, financeRoundHash).Val() != 0 {
		t.Fatal("expired snapshot remains", n, service.rounds)
	}
}
