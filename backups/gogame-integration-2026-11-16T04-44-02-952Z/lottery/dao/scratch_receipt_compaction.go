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
redis.call('SET',KEYS[1],ARGV[2])
if ttl~=-1 then redis.call('PEXPIRE',KEYS[1],ttl) end
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
	}
	if json.Unmarshal([]byte(original), &source) != nil || source.Fingerprint == "" || len(source.Result) == 0 || string(source.Result) == "null" || requestID == "" {
		return fmt.Errorf("invalid delivered scratch receipt")
	}
	if source.Confirmed {
		return nil
	}
	if len(source.Rounds) == 0 {
		return fmt.Errorf("scratch receipt has no paid batch")
	}
	compact, err := json.Marshal(struct {
		Fingerprint string          `json:"fingerprint"`
		Result      json.RawMessage `json:"result"`
		Confirmed   bool            `json:"deliveryConfirmed"`
	}{source.Fingerprint, source.Result, true})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, time.Second)
	defer cancel()
	n, err := compactScratchReceiptScript.Run(ctx, rd.cli, []string{scratchBuyMarker(scope, requestID)}, original, string(compact)).Int()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("scratch purchase receipt changed before compaction")
	}
	return nil
}
