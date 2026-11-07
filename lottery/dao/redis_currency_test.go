package dao

import (
	"errors"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func currencyTestRedis(t *testing.T) (*RedisDao, *miniredis.Miniredis) {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { client.Close() })
	return &RedisDao{cli: client}, server
}

func TestUpdatePlayerCurrency(t *testing.T) {
	for _, tc := range []struct {
		name    string
		current int64
		delta   int64
		want    int64
		wantErr error
	}{
		{name: "credit", current: 100, delta: 50, want: 150},
		{name: "debit", current: 100, delta: -40, want: 60},
		{name: "exact balance", current: 100, delta: -100, want: 0},
		{name: "insufficient", current: 100, delta: -101, wantErr: ErrInsufficientPlayerCurrency},
		{name: "zero delta", current: 100, want: 100},
		{name: "exact large balance", current: 1 << 53, delta: 1, want: 1<<53 + 1},
		{name: "large debit", current: 1<<53 + 1, delta: -(1<<53 + 1), want: 0},
		{name: "large insufficient", current: 1 << 53, delta: -(1<<53 + 1), wantErr: ErrInsufficientPlayerCurrency},
		{name: "maximum balance", current: math.MaxInt64 - 1, delta: 1, want: math.MaxInt64},
		{name: "minimum delta", current: math.MaxInt64, delta: math.MinInt64, wantErr: ErrInsufficientPlayerCurrency},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dao, server := currencyTestRedis(t)
			server.HSet("player_7", "id", "7", "currency", strconv.FormatInt(tc.current, 10), "exp", "10")
			server.SetTTL("player_7", time.Minute)
			got, err := dao.UpdatePlayerCurrency(7, tc.delta, 3, 0, 0)
			if got != tc.want || !errors.Is(err, tc.wantErr) {
				t.Fatalf("got (%d, %v), want (%d, %v)", got, err, tc.want, tc.wantErr)
			}
			if tc.wantErr != nil {
				if server.HGet("player_7", "currency") != strconv.FormatInt(tc.current, 10) ||
					server.HGet("player_7", "exp") != "10" || server.TTL("player_7") != time.Minute ||
					server.Exists("dirty_list_imp") {
					t.Fatal("rejected debit changed player data or dirty marker")
				}
				return
			}
			if server.HGet("player_7", "currency") != strconv.FormatInt(tc.want, 10) ||
				server.HGet("player_7", "exp") != "13" || server.TTL("player_7") != 20*time.Minute {
				t.Fatal("successful update did not preserve exact balance, experience or expiry")
			}
			if marked, err := server.SIsMember("dirty_list_imp", "7"); err != nil || !marked {
				t.Fatalf("missing dirty marker: %v", err)
			}
		})
	}
}

func TestUpdatePlayerCurrencyMissingCache(t *testing.T) {
	for _, fields := range [][]string{nil, {"currency", "100"}, {"id", "7"}} {
		dao, server := currencyTestRedis(t)
		if len(fields) > 0 {
			server.HSet("player_7", fields...)
		}
		if _, err := dao.UpdatePlayerCurrency(7, -1, 0, 0, 0); !errors.Is(err, redis.Nil) {
			t.Fatalf("incomplete player %v: want redis.Nil, got %v", fields, err)
		}
		if server.Exists("dirty_list_imp") {
			t.Fatal("missing player marked dirty")
		}
	}
}

func TestUpdatePlayerCurrencyConcurrentDebits(t *testing.T) {
	dao, server := currencyTestRedis(t)
	server.HSet("player_7", "id", "7", "currency", "100")
	var successes, insufficient atomic.Int64
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			balance, err := dao.UpdatePlayerCurrency(7, -10, 1, 0, 0)
			switch {
			case err == nil:
				successes.Add(1)
				if balance < 0 {
					t.Errorf("negative balance: %d", balance)
				}
			case errors.Is(err, ErrInsufficientPlayerCurrency):
				insufficient.Add(1)
			default:
				t.Errorf("unexpected update error: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
	if successes.Load() != 10 || insufficient.Load() != 54 || server.HGet("player_7", "currency") != "0" ||
		server.HGet("player_7", "exp") != "10" {
		t.Fatalf("successes=%d, insufficient=%d, currency=%s, exp=%s",
			successes.Load(), insufficient.Load(), server.HGet("player_7", "currency"), server.HGet("player_7", "exp"))
	}
}

func TestUpdatePlayerCurrencyDirtyMarkerFailure(t *testing.T) {
	dao, server := currencyTestRedis(t)
	server.HSet("player_7", "id", "7", "currency", "100")
	server.Set("dirty_list_imp", "wrong type")
	balance, err := dao.UpdatePlayerCurrency(7, -10, 0, 0, 0)
	if err != nil || balance != 90 || server.HGet("player_7", "currency") != "90" {
		t.Fatalf("committed debit reported as failed: balance=%d, err=%v", balance, err)
	}
}
