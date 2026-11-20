package dao

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"time"
)

// Upgrade confirmed receipts and batches whose original authoritative snapshots
// prove every ticket was already mirrored. Never deliver or settle a ticket here.
func (rd *RedisDao) ShortenDeliveredScratchReceipts(cursor uint64) (uint64, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	keys, next, err := rd.cli.Scan(ctx, cursor, "lottery:settle_wallet:scratch-buy:*", 100).Result()
	if err != nil {
		return cursor, 0, err
	}
	if len(keys) > 100 {
		keys = keys[:100]
	}
	changed := 0
	for _, key := range keys {
		if err := ctx.Err(); err != nil {
			return next, changed, err
		}
		size, err := rd.cli.StrLen(ctx, key).Result()
		if err != nil {
			return next, changed, err
		}
		if size == 0 || size > 128*1024 {
			continue
		}
		raw, err := rd.cli.Get(ctx, key).Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return next, changed, err
		}
		var receipt struct {
			Confirmed bool  `json:"deliveryConfirmed"`
			CreatedAt int64 `json:"createdAt"`
		}
		if json.Unmarshal([]byte(raw), &receipt) != nil {
			continue
		}
		if !receipt.Confirmed {
			n, err := rd.compactMirroredScratchReceipt(ctx, key, raw, time.Now().Unix())
			if err != nil {
				return next, changed, err
			}
			changed += n
			continue
		}
		ttl, err := rd.cli.PTTL(ctx, key).Result()
		if err != nil {
			return next, changed, err
		}
		now := time.Now().Unix()
		created := receipt.CreatedAt
		if created <= 0 {
			if ttl > 0 && ttl <= time.Hour {
				continue
			}
			created = now
			if ttl > 0 && ttl < 7*24*time.Hour {
				created = now - int64((7*24*time.Hour-ttl)/time.Second)
			}
		}
		if created > now {
			continue
		}
		remaining := (created + 3600 - now) * 1000
		n, err := shortenDeliveredReceiptScript.Run(ctx, rd.cli, []string{key}, raw, remaining).Int()
		if err != nil {
			return next, changed, err
		}
		changed += n
	}
	return next, changed, nil
}

var shortenDeliveredReceiptScript = redis.NewScript(`
if redis.call('GET',KEYS[1])~=ARGV[1]then return 0 end
local ttl=redis.call('PTTL',KEYS[1]);local remaining=tonumber(ARGV[2])
if remaining<=0 then redis.call('DEL',KEYS[1]);return 1 end
if ttl==-1 or ttl>remaining then redis.call('PEXPIRE',KEYS[1],remaining);return 1 end
return 0
`)
