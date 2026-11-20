package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// References retain the old rounds []string envelope for compatible readers.
// A reference cannot reconstruct a financial snapshot or regenerate a prize.
type scratchPaidReference struct {
	RoundID string `json:"roundId"`
	Agent   uint32 `json:"agent"`
	GameID  uint32 `json:"gameId"`
	Scratch *struct {
		Serial   string `json:"serialFull"`
		UserID   uint32 `json:"userId"`
		Currency string `json:"currency"`
	} `json:"scratch"`
}

var slimPendingScratchScript = redis.NewScript(`
if redis.call('GET',KEYS[1])~=ARGV[1] then return 0 end
local ttl=redis.call('PTTL',KEYS[1])
if ttl~=-1 and ttl<=0 then return 0 end
redis.call('SET',KEYS[1],ARGV[2])
if ttl>0 then redis.call('PEXPIRE',KEYS[1],ttl) end
return 1
`)

// Only remove duplicated private data. Unconfirmed paid rights retain their
// original expiry; one-hour confirmed-receipt cleanup is a separate operation.
func (rd *RedisDao) CompactPendingScratchReceipt(parent context.Context, scope ScratchScope, requestID, original string) (string, error) {
	if requestID == "" {
		return "", fmt.Errorf("missing pending scratch request")
	}
	return rd.compactPendingScratchReceiptKey(parent, scope, scratchBuyMarker(scope, requestID), original)
}

// The scanner already has the hashed Redis key; never hash its suffix again.
func (rd *RedisDao) compactPendingScratchReceiptKey(parent context.Context, scope ScratchScope, key, original string) (string, error) {
	var receipt struct {
		Fingerprint    string          `json:"fingerprint"`
		Result         json.RawMessage `json:"result"`
		Rounds         []string        `json:"rounds"`
		ReferencesOnly bool            `json:"referencesOnly,omitempty"`
		Confirmed      bool            `json:"deliveryConfirmed,omitempty"`
		CreatedAt      int64           `json:"createdAt"`
	}
	if len(original) > 8*1024*1024 || json.Unmarshal([]byte(original), &receipt) != nil || receipt.Fingerprint == "" || len(receipt.Result) == 0 || string(receipt.Result) == "null" || receipt.Confirmed || len(receipt.Rounds) == 0 || len(receipt.Rounds) > 100 {
		return "", fmt.Errorf("invalid pending scratch receipt")
	}
	seen := map[string]bool{}
	serials := map[string]bool{}
	for i, raw := range receipt.Rounds {
		var ref scratchPaidReference
		if json.Unmarshal([]byte(raw), &ref) != nil || ref.RoundID == "" || len(ref.RoundID) > 160 || ref.Agent != scope.Agent || ref.GameID != scope.GameID || ref.Scratch == nil || ref.Scratch.UserID != scope.UserID || ref.Scratch.Currency != scope.Currency || ref.Scratch.Serial == "" || len(ref.Scratch.Serial) > 80 || seen[ref.RoundID] || serials[ref.Scratch.Serial] {
			return "", fmt.Errorf("pending scratch reference identity mismatch")
		}
		seen[ref.RoundID] = true
		serials[ref.Scratch.Serial] = true
		encoded, err := json.Marshal(ref)
		if err != nil {
			return "", err
		}
		receipt.Rounds[i] = string(encoded)
	}
	// Already slim receipts still get the same identity checks above.
	if receipt.ReferencesOnly {
		return original, nil
	}
	receipt.ReferencesOnly = true
	slim, err := json.Marshal(receipt)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(parent, time.Second)
	defer cancel()
	n, err := slimPendingScratchScript.Run(ctx, rd.cli, []string{key}, original, string(slim)).Int()
	if err != nil {
		return "", err
	}
	if n != 1 {
		return "", fmt.Errorf("pending scratch receipt changed before compaction")
	}
	return string(slim), nil
}
