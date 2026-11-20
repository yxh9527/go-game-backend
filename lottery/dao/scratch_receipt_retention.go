package dao

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"strconv"
	"strings"
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
			if n == 0 {
				// An unconfirmed purchase keeps its original deadline and all
				// rights. Only remove repeated private round copies, using the
				// same identity checks and compare-and-swap as request recovery.
				if scope, ok := scannedScratchReceiptScope(key); ok {
					if slim, compactErr := rd.compactPendingScratchReceiptKey(ctx, scope, key, raw); compactErr == nil && slim != raw {
						changed++
					}
				}
			}
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

func scannedScratchReceiptScope(key string) (ScratchScope, bool) {
	const prefix = "lottery:settle_wallet:scratch-buy:"
	if !strings.HasPrefix(key, prefix) {
		return ScratchScope{}, false
	}
	parts := strings.Split(strings.TrimPrefix(key, prefix), ":")
	if len(parts) != 5 || len(parts[4]) != 64 {
		return ScratchScope{}, false
	}
	for _, c := range parts[4] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return ScratchScope{}, false
		}
	}
	ids := [3]uint32{}
	for i := range ids {
		value, err := strconv.ParseUint(parts[i], 10, 32)
		if err != nil {
			return ScratchScope{}, false
		}
		ids[i] = uint32(value)
	}
	scope := ScratchScope{Agent: ids[0], GameID: ids[1], UserID: ids[2], Currency: parts[3]}
	if scope.GameID == 0 || scope.UserID == 0 || len(scope.Currency) == 0 || len(scope.Currency) > 12 || scope.Key()+":"+parts[4] != strings.TrimPrefix(key, prefix) {
		return ScratchScope{}, false
	}
	for _, c := range scope.Currency {
		if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return ScratchScope{}, false
		}
	}
	return scope, true
}

var shortenDeliveredReceiptScript = redis.NewScript(`
if redis.call('GET',KEYS[1])~=ARGV[1]then return 0 end
local ttl=redis.call('PTTL',KEYS[1]);local remaining=tonumber(ARGV[2])
if remaining<=0 then redis.call('DEL',KEYS[1]);return 1 end
if ttl==-1 or ttl>remaining then redis.call('PEXPIRE',KEYS[1],remaining);return 1 end
return 0
`)
