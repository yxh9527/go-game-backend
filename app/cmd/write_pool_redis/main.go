package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"app/esindex"
	"app/tables"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

type poolItemJSON struct {
	Min        any `json:"min"`
	Normal     any `json:"normal"`
	Max        any `json:"max"`
	NormalRate any `json:"normalRate"`
	MinRate    any `json:"minRate"`
	MaxRate    any `json:"maxRate"`
	Control    any `json:"control"`
	Revenue    any `json:"revenue"`
	Base       any `json:"base"`
}

type poolConfigJSON struct {
	Symbol string                   `json:"symbol"`
	Name   string                   `json:"name"`
	NameZH string                   `json:"nameZH"`
	GameId int                      `json:"gameId"`
	Pool   map[string]*poolItemJSON `json:"pool"`
}

func main() {
	addr := "8.217.56.130:9531"
	pwd := "yxh2023!@#"
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		addr = v
	}
	if v := os.Getenv("REDIS_PWD"); v != "" {
		pwd = v
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     pwd,
		DB:           0,
		DialTimeout:  10 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		fmt.Fprintf(os.Stderr, "redis ping failed: %v\n", err)
		os.Exit(1)
	}

	seeds := tables.DefaultGameSeeds()
	pipe := rdb.Pipeline()
	n := 0
	for _, seed := range seeds {
		cfg := poolConfigJSON{
			Symbol: seed.Symbol,
			Name:   seed.Symbol,
			NameZH: seed.NameZH,
			GameId: seed.Number,
			Pool: map[string]*poolItemJSON{
				"1": {
					Min:        300,
					Normal:     900,
					Max:        3000,
					NormalRate: 0.75,
					MinRate:    0.65,
					MaxRate:    0.85,
					Control:    1,
					Revenue:    0.03,
					Base:       1,
				},
			},
		}
		raw, err := json.Marshal(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshal %s failed: %v\n", seed.Symbol, err)
			os.Exit(1)
		}
		key := esindex.ConfigKey("pool", seed.Symbol)
		pipe.Set(ctx, key, string(raw), 0)
		n++
		if n%50 == 0 {
			if _, err := pipe.Exec(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "pipeline exec failed: %v\n", err)
				os.Exit(1)
			}
			pipe = rdb.Pipeline()
		}
	}
	if n%50 != 0 {
		if _, err := pipe.Exec(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "pipeline exec failed: %v\n", err)
			os.Exit(1)
		}
	}

	// 抽查一条，确认可反序列化为 decimal 友好结构
	sampleKey := esindex.ConfigKey("pool", seeds[0].Symbol)
	val, err := rdb.Get(ctx, sampleKey).Result()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sample get failed: %v\n", err)
		os.Exit(1)
	}
	var check struct {
		Symbol string `json:"symbol"`
		GameId int    `json:"gameId"`
		Pool   map[string]struct {
			Min decimal.Decimal `json:"min"`
		} `json:"pool"`
	}
	if err := json.Unmarshal([]byte(val), &check); err != nil {
		fmt.Fprintf(os.Stderr, "sample unmarshal failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("ok: wrote %d keys to %s, prefix=%s, sample=%s gameId=%d\n",
		len(seeds), addr, esindex.PathRoot(), sampleKey, check.GameId)
}
