package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

var compactScratchReceiptScript = redis.NewScript(`
if redis.call('GET',KEYS[1])~=ARGV[1] then return 0 end
local ttl=redis.call('PTTL',KEYS[1])
if ttl~=-1 and ttl<=0 then return 0 end
local limit=tonumber(ARGV[3])
if limit<=0 then redis.call('DEL',KEYS[1]);return 1 end
if ttl==-1 or ttl>limit then ttl=limit end
redis.call('SET',KEYS[1],ARGV[2])
redis.call('PEXPIRE',KEYS[1],ttl)
return 1
`)

// Only after the original entire paid ticket batch has been durably mirrored.
// Keep the same key, result, fingerprint and remaining TTL, dropping duplicate
// recovery snapshots that are no longer needed for purchase delivery.
func (rd *RedisDao) CompactDeliveredScratchReceipt(parent context.Context, scope ScratchScope, requestID, original string) error {
	var source struct {
		Fingerprint string          `json:"fingerprint"`
		Result      json.RawMessage `json:"result"`
		Rounds      []string        `json:"rounds"`
		Confirmed   bool            `json:"deliveryConfirmed"`
		CreatedAt   int64           `json:"createdAt"`
	}
	if json.Unmarshal([]byte(original), &source) != nil || source.Fingerprint == "" || len(source.Result) == 0 || string(source.Result) == "null" || requestID == "" {
		return fmt.Errorf("invalid delivered scratch receipt")
	}
	if !source.Confirmed && len(source.Rounds) == 0 {
		return fmt.Errorf("scratch receipt has no paid batch")
	}
	ctx, cancel := context.WithTimeout(parent, time.Second)
	defer cancel()
	now := time.Now().Unix()
	if source.CreatedAt <= 0 {
		ttl, err := rd.cli.PTTL(ctx, scratchBuyMarker(scope, requestID)).Result()
		if err != nil {
			return err
		}
		if ttl <= 0 && ttl != -1 {
			return fmt.Errorf("scratch purchase receipt expired")
		}
		window := 7 * 24 * time.Hour
		if source.Confirmed && ttl > 0 && ttl <= time.Hour {
			window = time.Hour
		}
		source.CreatedAt = now
		if ttl > 0 && ttl < window {
			source.CreatedAt = now - int64((window-ttl)/time.Second)
		}
	}
	if source.CreatedAt > now {
		return fmt.Errorf("scratch receipt original time invalid")
	}
	compact, err := json.Marshal(struct {
		Fingerprint string          `json:"fingerprint"`
		Result      json.RawMessage `json:"result"`
		Confirmed   bool            `json:"deliveryConfirmed"`
		CreatedAt   int64           `json:"createdAt"`
	}{source.Fingerprint, source.Result, true, source.CreatedAt})
	if err != nil {
		return err
	}
	remaining := (source.CreatedAt + 3600 - now) * 1000
	n, err := compactScratchReceiptScript.Run(ctx, rd.cli, []string{scratchBuyMarker(scope, requestID)}, original, string(compact), remaining).Int()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("scratch purchase receipt changed before compaction")
	}
	return nil
}
