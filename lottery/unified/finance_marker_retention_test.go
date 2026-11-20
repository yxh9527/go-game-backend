package unified

import (
	"context"
	"micro_service/services"
	"testing"
	"time"
)

func TestCompletedCancelRetiresCashMarkerAndLateRetryReturnsOriginalWithoutAnotherRefund(t *testing.T) {
	f := newScratchFixture(t)
	ctx := context.Background()
	bet := &services.BetRequest{RequestId: "original-bet", RoundId: "receipt-round", Agent: 7, GameId: 91061, Level: 1, Items: []*services.BetItem{{BetId: "receipt-round:rev:0:one", UserId: 42, CurrencyType: "CNY", Amount: "1.00", AreaId: "1"}}}
	response, err := f.svc.Bet(ctx, bet)
	if err != nil || !response.Items[0].Accepted {
		t.Fatal(response, err)
	}
	cancel := &services.CancelBetRequest{RequestId: "receipt-cancel", RoundId: bet.RoundId, Agent: 7, GameId: 91061, Level: 1, UserId: 42, CurrencyType: "CNY", ExpectedBetDigest: response.BetDigest}
	refunded, err := f.svc.CancelBet(ctx, cancel)
	if err != nil || !refunded.Success || f.balance(42) != 100000 {
		t.Fatal(refunded, err)
	}
	key := cancelWalletMarkerKey(cancel.RequestId)
	if ttl := f.client.PTTL(ctx, key).Val(); ttl > time.Hour || ttl < 59*time.Minute {
		t.Fatal("completed cancel kept seven-day marker", ttl)
	}
	f.client.PExpire(ctx, key, time.Millisecond)
	f.client.Del(ctx, key) // The original completed request remains authoritative.
	refunded, err = f.svc.CancelBet(ctx, cancel)
	if err != nil || !refunded.Success || f.balance(42) != 100000 || f.client.Exists(ctx, key).Val() != 0 {
		t.Fatal("late request refunded again after marker cleanup", refunded, err)
	}
}
