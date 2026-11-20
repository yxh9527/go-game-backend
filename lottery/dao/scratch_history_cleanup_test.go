package dao

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"testing"
	"time"
)

func TestScratchHistoryCleanupOnlyRemovesRetiredHistoryNamespaces(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()
	rd := NewFinanceRedis(client)
	ctx := context.Background()
	client.HSet(ctx, "gogame:scratch:history:7:42:room", "serial", "original")
	client.ZAdd(ctx, "gogame:scratch:history-order:7:42:room", redis.Z{Score: 1, Member: "serial"})
	client.Set(ctx, "scratch-history:v2:query", "cached request", 180*time.Second)
	client.Set(ctx, "gogame:scratch:history-other", "unrelated", 0)
	client.Set(ctx, "gogame:scratch:sold:serial", "round", 0)
	client.SAdd(ctx, "gogame:scratch:pending:7:42", "round")
	client.HSet(ctx, "lottery_finance_rounds", "round", "original paid state")
	cursor := uint64(0)
	for {
		next, err := rd.RemoveLegacyScratchHistory(cursor)
		if err != nil {
			t.Fatal(err)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	if client.Exists(ctx, "gogame:scratch:history:7:42:room", "gogame:scratch:history-order:7:42:room").Val() != 0 {
		t.Fatal("retired history keys remain")
	}
	for _, key := range []string{"scratch-history:v2:query", "gogame:scratch:history-other", "gogame:scratch:sold:serial", "gogame:scratch:pending:7:42", "lottery_finance_rounds"} {
		if client.Exists(ctx, key).Val() != 1 {
			t.Fatal("cleanup removed live data", key)
		}
	}
	if client.TTL(ctx, "scratch-history:v2:query").Val() != 180*time.Second {
		t.Fatal("cleanup extended query cache TTL")
	}
}
