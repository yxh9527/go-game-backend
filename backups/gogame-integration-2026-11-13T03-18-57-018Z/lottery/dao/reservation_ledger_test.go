package dao

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

func TestEncodeParseReservationLedger(t *testing.T) {
	raw := EncodeReservationLedger(
		ReservationLedgerActive, "12.5", "", "r1", 9, "sym_1", "", 100,
	)
	if ParseReservationLedgerStatus(raw) != ReservationLedgerActive {
		t.Fatalf("status=%q raw=%q", ParseReservationLedgerStatus(raw), raw)
	}
	released := EncodeReservationLedger(
		ReservationLedgerReleased, "12.5", "settle:s1", "r1", 9, "sym_1", "SETTLE", 200,
	)
	if ParseReservationLedgerStatus(released) != ReservationLedgerReleased {
		t.Fatalf("released status=%q", ParseReservationLedgerStatus(released))
	}
}

func newTestRedis(t *testing.T) (*miniredis.Miniredis, *RedisDao) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	cli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = cli.Close() })
	rd := NewRedisDaoWithClient(cli)
	SetRedisDaoForTest(rd)
	return mr, rd
}

func TestReservationLedgerReleaseIdempotentAndIsolated(t *testing.T) {
	_, rd := newTestRedis(t)
	base := decimal.RequireFromString("1000")
	amt := decimal.RequireFromString("100")

	a, err := rd.TryReserveAgentPoolWithLedger(1, "game_1", base, amt, "res-A", "round-A")
	if err != nil || !a.Ok {
		t.Fatalf("reserve A: ok=%v err=%v", a, err)
	}
	b, err := rd.TryReserveAgentPoolWithLedger(1, "game_1", base, amt, "res-B", "round-B")
	if err != nil || !b.Ok {
		t.Fatalf("reserve B: ok=%v err=%v", b, err)
	}
	q := rd.GetAgentReserved(1, "game_1")
	if !q.Equal(decimal.RequireFromString("200")) {
		t.Fatalf("Q after two reserves=%s", q)
	}

	r1, err := rd.ReleaseAgentReservedWithLedger(
		1, "game_1", amt, "res-A", "round-A", "settle:sA", "SETTLE", false,
	)
	if err != nil || !r1.Ok || r1.AlreadyReleased {
		t.Fatalf("first release A: %+v err=%v", r1, err)
	}
	r2, err := rd.ReleaseAgentReservedWithLedger(
		1, "game_1", amt, "res-A", "round-A", "settle:sA", "SETTLE", false,
	)
	if err != nil || !r2.Ok || !r2.AlreadyReleased {
		t.Fatalf("second release A must be idempotent: %+v err=%v", r2, err)
	}
	q = rd.GetAgentReserved(1, "game_1")
	if !q.Equal(decimal.RequireFromString("100")) {
		t.Fatalf("Q after double-release A should keep B: %s", q)
	}

	// 旧路径无 ledger 的双释会吞 B；ledger 下第二次不得再扣。
	raw, _ := rd.GetReservationLedgerRaw("res-A")
	if ParseReservationLedgerStatus(raw) != ReservationLedgerReleased {
		t.Fatalf("ledger A status=%q", ParseReservationLedgerStatus(raw))
	}
}

func TestSettleWalletMarkerIdempotent(t *testing.T) {
	mr, rd := newTestRedis(t)
	mr.HSet("player_7", "id", "7")
	mr.HSet("player_7", "currency", "10000") // 100.00 元 = 10000 分

	first, err := rd.CreditPlayerCurrencyWithSettleMarker(7, 500, "settle-1")
	if err != nil || first.AlreadyDone {
		t.Fatalf("first credit: %+v err=%v", first, err)
	}
	if first.NewCurrency != 10500 {
		t.Fatalf("balance=%d", first.NewCurrency)
	}
	second, err := rd.CreditPlayerCurrencyWithSettleMarker(7, 500, "settle-1")
	if err != nil || !second.AlreadyDone {
		t.Fatalf("second credit must skip: %+v err=%v", second, err)
	}
	cur, err := rd.GetPlayerCurrency(7)
	if err != nil || cur != 10500 {
		t.Fatalf("final balance=%d err=%v", cur, err)
	}
	_ = context.Background()
}

func TestReleaseMigrateLegacyWithoutLedger(t *testing.T) {
	_, rd := newTestRedis(t)
	base := decimal.RequireFromString("1000")
	amt := decimal.RequireFromString("50")
	// 旧路径只占 Q、无 ledger。
	ok, err := rd.TryReserveAgentPool(1, "g_1", base, amt)
	if err != nil || !ok {
		t.Fatalf("legacy reserve: %v %v", ok, err)
	}
	rel, err := rd.ReleaseAgentReservedWithLedger(
		1, "g_1", amt, "legacy-res", "round-L", "expire:x", "EXPIRE", true,
	)
	if err != nil || !rel.Ok {
		t.Fatalf("migrate release: %+v err=%v", rel, err)
	}
	rel2, err := rd.ReleaseAgentReservedWithLedger(
		1, "g_1", amt, "legacy-res", "round-L", "expire:x", "EXPIRE", true,
	)
	if err != nil || !rel2.Ok || !rel2.AlreadyReleased {
		t.Fatalf("second migrate release: %+v err=%v", rel2, err)
	}
	if !rd.GetAgentReserved(1, "g_1").IsZero() {
		t.Fatalf("Q should be zero, got %s", rd.GetAgentReserved(1, "g_1"))
	}
}
