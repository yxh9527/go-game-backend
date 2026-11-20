package unified

import (
	"context"
	"go.uber.org/zap"
	"time"
)

func (d *LotteryService) retainCompletedFinanceReceipts(ctx context.Context, round *financeRound) {
	if d.rds == nil || round == nil {
		return
	}
	completed := round.Settlement != nil && round.Settlement.Status == financeSettlementStatusCompleted
	if !completed {
		for _, claim := range round.CancelRequests {
			if claim != nil && claim.Status == financeCancelStatusCompleted && claim.WalletRefunded {
				completed = true
				break
			}
		}
	}
	if !completed {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if _, err := d.rds.RetainCompletedFinanceReceipts(ctx, round.RoundID, time.Now().Unix()); err != nil {
		// Retention maintenance cannot turn an already completed payment into a
		// failed payment response and trigger another monetary operation.
		zap.L().Error("completed finance receipt retention failed", zap.String("roundId", round.RoundID), zap.Error(err))
	}
}
