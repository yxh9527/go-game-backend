package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func cashFixture(t *testing.T) (*redis.Client, *miniredis.Miniredis, CashAccount) {
	t.Helper()
	s := miniredis.RunT(t)
	c := redis.NewClient(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() { c.Close() })
	a := CashAccount{UserID: 42, AgentID: 7, Currency: "CNY"}
	s.HSet("player_42", "id", "42", "agent_id", "7", "currency_type", "CNY", "currency", "1000")
	return c, s, a
}
func cashRun(t *testing.T, c redis.UniversalClient, a CashAccount, id string, steps ...CashStep) json.RawMessage {
	t.Helper()
	for i := range steps {
		steps[i].Account = 1
		if steps[i].ID == "" {
			steps[i].ID = fmt.Sprintf("%s:%d", id, i)
		}
	}
	snapshot, e := CashRead(context.Background(), c, a)
	if e != nil {
		t.Fatal(e)
	}
	var version struct{ Revision int64 }
	_ = json.Unmarshal(snapshot, &version)
	a.Revision = version.Revision
	out, err := CashApply(context.Background(), c, CashRequest{ID: id, Accounts: []CashAccount{a}, Steps: steps})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
func TestCashCarryAndSynchronousFunds(t *testing.T) {
	c, s, a := cashFixture(t)
	reserve := CashStep{Kind: "reserve", Token: "lease", TableKey: "7:holdem:1", Amount: 600}
	cashRun(t, c, a, "reserve", reserve)
	if s.HGet("player_42", "currency") != "1000" {
		t.Fatal("carry debited wallet")
	}
	ck := cashKeys(a)[1]
	if s.TTL(ck) != 5*time.Minute {
		t.Fatal("missing five minute TTL")
	}
	cashRun(t, c, a, "bet", CashStep{Kind: "carryDelta", Token: "lease", TableKey: reserve.TableKey, Revision: 1, Amount: -100})
	if s.HGet("player_42", "currency") != "900" {
		t.Fatal("wager not reflected in wallet")
	}
	cashRun(t, c, a, "pay", CashStep{Kind: "carryDelta", Token: "lease", TableKey: reserve.TableKey, Revision: 2, Amount: 200})
	cashRun(t, c, a, "release", CashStep{Kind: "release", Token: "lease", TableKey: reserve.TableKey, Revision: 3, Amount: 700})
	if s.Exists(ck) || s.HGet("player_42", "currency") != "1100" {
		t.Fatal("release credited wallet or left carry")
	}
}
func TestCashRevisionAndBatchRollback(t *testing.T) {
	c, s, a := cashFixture(t)
	ctx := context.Background()
	request := CashRequest{ID: "transaction", Accounts: []CashAccount{a}, Steps: []CashStep{{ID: "step", Account: 1, Kind: "debit", Amount: 300}}}
	if _, err := CashApply(ctx, c, request); err != nil {
		t.Fatal(err)
	}
	if _, err := CashApply(ctx, c, request); status.Code(err) != codes.FailedPrecondition {
		t.Fatal("stale version accepted", err)
	}
	if s.HGet("player_42", "currency") != "700" {
		t.Fatal("duplicate changed balance")
	}
	a.Revision = 1
	bad := CashRequest{ID: "atomic-reject", Accounts: []CashAccount{a}, Steps: []CashStep{{ID: "first", Account: 1, Kind: "credit", Amount: 10}, {ID: "second", Account: 1, Kind: "debit", Amount: 800}}}
	if _, err := CashApply(ctx, c, bad); status.Code(err) != codes.FailedPrecondition {
		t.Fatal(err)
	}
	if s.HGet("player_42", "currency") != "700" || s.HGet("player_42", "cash_revision") != "1" {
		t.Fatal("partial transaction committed")
	}
	for _, key := range s.Keys() {
		if strings.Contains(key, "ledger") || strings.Contains(key, "pending") {
			t.Fatal("unexpected financial-history record", key)
		}
	}
}
func TestCashConcurrentFreeBalanceDebits(t *testing.T) {
	c, s, a := cashFixture(t)
	cashRun(t, c, a, "carry", CashStep{Kind: "reserve", Token: "lease", TableKey: "7:holdem:1", Amount: 800})
	var successes atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("bet-%d", i)
			for attempt := 0; attempt < 200; attempt++ {
				raw, err := CashRead(context.Background(), c, a)
				if err != nil {
					t.Error(err)
					return
				}
				var snapshot struct{ Revision int64 }
				_ = json.Unmarshal(raw, &snapshot)
				account := a
				account.Revision = snapshot.Revision
				_, err = CashApply(context.Background(), c, CashRequest{ID: id, Accounts: []CashAccount{account}, Steps: []CashStep{{ID: id, Account: 1, Kind: "debit", Amount: 20}}})
				if err == nil {
					successes.Add(1)
					return
				}
				if strings.Contains(err.Error(), "insufficient") {
					return
				}
				if status.Code(err) != codes.FailedPrecondition {
					t.Error(err)
					return
				}
			}
			t.Error("cash conflict retry exhausted")
		}(i)
	}
	wg.Wait()
	if successes.Load() != 10 || s.HGet("player_42", "currency") != "800" {
		t.Fatalf("success=%d balance=%s", successes.Load(), s.HGet("player_42", "currency"))
	}
}
func TestCashLeaseRenewExpiryAndFencing(t *testing.T) {
	c, s, a := cashFixture(t)
	ctx := context.Background()
	table := "7:holdem:1"
	cashRun(t, c, a, "carry", CashStep{Kind: "reserve", Token: "old", TableKey: table, Amount: 600})
	s.FastForward(4 * time.Minute)
	cashRun(t, c, a, "renew", CashStep{Kind: "renew", Token: "old", TableKey: table, Revision: 1})
	if s.TTL(cashKeys(a)[1]) != 5*time.Minute {
		t.Fatal("renewal did not set five minutes")
	}
	s.FastForward(5 * time.Minute)
	cashRun(t, c, a, "new-carry", CashStep{Kind: "reserve", Token: "new", TableKey: "7:zjh:2", Amount: 500})
	cashRun(t, c, a, "late-release", CashStep{Kind: "release", Token: "old", TableKey: table, Revision: 1, Amount: 600})
	stale := CashRequest{ID: "late-pay", Accounts: []CashAccount{a}, Steps: []CashStep{{ID: "stale", Account: 1, Kind: "carryDelta", Token: "old", TableKey: table, Revision: 1, Amount: 100}}}
	if _, err := CashApply(ctx, c, stale); status.Code(err) != codes.FailedPrecondition {
		t.Fatal("old owner modified money", err)
	}
	if s.HGet("player_42", "currency") != "1000" {
		t.Fatal("lease expiry changed money")
	}
	raw, err := CashRead(ctx, c, a)
	if err != nil {
		t.Fatal(err)
	}
	var state struct{ Carry struct{ Token string } }
	_ = json.Unmarshal(raw, &state)
	if state.Carry.Token != "new" {
		t.Fatal("old release deleted new lease")
	}
}
func TestCashIsolationAndValidation(t *testing.T) {
	c, s, a := cashFixture(t)
	ctx := context.Background()
	for _, wrong := range []CashAccount{{UserID: 42, AgentID: 8, Currency: "CNY"}, {UserID: 42, AgentID: 7, Currency: "USD"}} {
		if _, err := CashRead(ctx, c, wrong); status.Code(err) != codes.PermissionDenied {
			t.Fatal("account not isolated", err)
		}
	}
	s.Set("dirty_list_imp", "bad type")
	_, err := CashApply(ctx, c, CashRequest{ID: "wrongtype", Accounts: []CashAccount{a}, Steps: []CashStep{{ID: "step", Account: 1, Kind: "debit", Amount: 1}}})
	if err == nil || s.HGet("player_42", "currency") != "1000" {
		t.Fatal("Lua error partially changed funds", err)
	}
}
