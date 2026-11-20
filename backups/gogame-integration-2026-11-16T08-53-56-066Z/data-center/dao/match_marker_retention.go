package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"time"
)

var shortenConfirmedMatchMarkerScript = redis.NewScript(`
if redis.call('GET',KEYS[1])~=ARGV[1] then return 0 end
local ttl=redis.call('PTTL',KEYS[1])
if ttl==-1 or ttl>3600000 then redis.call('PEXPIRE',KEYS[1],3600000) end
return 1
`)

// Call only after the original SQL transaction has confirmed its receipt.
// Missing markers stay missing; retries never extend a shorter lifetime.
func retainConfirmedMatchMarker(ctx context.Context, client redis.UniversalClient, input MatchPaymentInput, receipt map[string]any) error {
	if client == nil || receipt == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if input.Operation != "charge" && input.Operation != "refund" && input.Operation != "award" {
		return fmt.Errorf("invalid confirmed payment operation")
	}
	key := paymentMarker(input.Plan, input.Operation)
	raw, err := client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}
	var original struct {
		Fingerprint string         `json:"fingerprint"`
		Receipt     map[string]any `json:"receipt"`
	}
	if json.Unmarshal([]byte(raw), &original) != nil || original.Fingerprint != paymentHash(input) || original.Receipt == nil || paymentHash(original.Receipt) != paymentHash(receipt) {
		return fmt.Errorf("confirmed SQL receipt does not match original Redis marker")
	}
	changed, err := shortenConfirmedMatchMarkerScript.Run(ctx, client, []string{key}, raw).Int()
	if err != nil {
		return err
	}
	if changed != 1 {
		return fmt.Errorf("payment marker changed during retention confirmation")
	}
	return nil
}

func finishConfirmedMatchMarker(ctx context.Context, client redis.UniversalClient, input MatchPaymentInput, receipt map[string]any) {
	if err := retainConfirmedMatchMarker(ctx, client, input, receipt); err != nil {
		// The SQL receipt is already committed. A retention failure cannot be
		// reported as failed payment and induce another financial operation.
		zap.L().Error("confirmed match payment marker retention pending", zap.Uint32("agentId", input.Plan.AgentID), zap.Int64("matchId", input.Plan.MatchID), zap.String("operation", input.Operation), zap.Error(err))
	}
}
