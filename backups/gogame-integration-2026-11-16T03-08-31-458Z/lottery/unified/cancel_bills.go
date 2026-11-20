package unified

import (
	"app/entity"
	"crypto/sha256"
	"fmt"
	"github.com/shopspring/decimal"
)

func originalCancelBillID(round *financeRound, claim *financeCancelRequest) string {
	identity := fmt.Sprintf("cancel|%d|%s|%s|%d|%s", round.Agent, round.RoundID, claim.RequestID, claim.UserID, claim.CurrencyType)
	return fmt.Sprintf("L%x", sha256.Sum256([]byte(identity)))
}

func originalCancelBill(round *financeRound, claim *financeCancelRequest) (*entity.CacheBillsReq, error) {
	if claim == nil || claim.BillID == "" || claim.BillStored {
		return nil, nil
	}
	if claim.Status != financeCancelStatusCompleted || !claim.Success || !claim.WalletRefunded {
		return nil, nil
	}
	if claim.UserID == 0 || round.GameID == 0 || round.Symbol == "" || claim.BillCreatedAt <= 0 || claim.CompletedAt <= 0 || claim.BillID != originalCancelBillID(round, claim) {
		return nil, fmt.Errorf("original cancellation bill identity is not confirmed")
	}
	amount, err := parseMoney(claim.RefundAmount)
	if err != nil || !amount.IsPositive() {
		return nil, fmt.Errorf("original cancellation refund amount invalid")
	}
	balance, err := parseMoney(claim.Currency)
	if err != nil {
		return nil, fmt.Errorf("original cancellation balance is not confirmed")
	}
	total := decimal.Zero
	seen := map[string]bool{}
	for _, id := range claim.CanceledBetIDs {
		bet := round.Bets[id]
		if seen[id] || bet == nil || bet.BetID != id || bet.UserID != claim.UserID || bet.CurrencyType != claim.CurrencyType || bet.Status != financeBetStatusCanceled || bet.Accepted || bet.Code != 0 {
			return nil, fmt.Errorf("original cancellation debit reference mismatch")
		}
		stake, err := parseMoney(bet.Amount)
		if err != nil {
			return nil, err
		}
		total = total.Add(stake)
		seen[id] = true
	}
	if len(seen) == 0 || !total.Equal(amount) {
		return nil, fmt.Errorf("original cancellation does not cover the refunded stake")
	}
	return &entity.CacheBillsReq{AgentId: round.Agent, UserId: claim.UserID, GameId: round.GameID, Symbol: round.Symbol, Currency: claim.CurrencyType,
		Bet: amount.InexactFloat64(), CurrentScore: balance.InexactFloat64(), CreateTime: claim.BillCreatedAt, RoundID: round.RoundID, FlowingWaterOn: claim.BillID, Desc: "bet_cancel"}, nil
}
