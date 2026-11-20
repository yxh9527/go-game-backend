package unified

import (
	"app/entity"
	"context"
	"crypto/sha256"
	"fmt"
	"go.uber.org/zap"
	"time"
)

func originalBetBillID(round *financeRound, bet *financeBetSnapshot) string {
	identity := fmt.Sprintf("bet|%d|%s|%s|%d|%s", round.Agent, round.RoundID, bet.BetID, bet.UserID, bet.CurrencyType)
	return fmt.Sprintf("L%x", sha256.Sum256([]byte(identity)))
}

func originalBetBill(round *financeRound, bet *financeBetSnapshot) (*entity.CacheBillsReq, error) {
	if bet == nil || bet.BillID == "" || bet.BillStored || bet.BillAttempted {
		return nil, nil
	}
	wasAccepted := bet.Accepted || bet.Status == financeBetStatusCanceled
	if !wasAccepted || bet.Code != 0 || bet.UserID == 0 || round.GameID == 0 || round.Symbol == "" || bet.BillCreatedAt <= 0 || bet.BillID != originalBetBillID(round, bet) {
		return nil, fmt.Errorf("original debit bill identity is not confirmed")
	}
	amount, err := parseMoney(bet.Amount)
	if err != nil || !amount.IsPositive() {
		return nil, fmt.Errorf("original debit amount is not confirmed")
	}
	balance, err := parseMoney(bet.Currency)
	if err != nil {
		return nil, fmt.Errorf("original debit balance is not confirmed")
	}
	return &entity.CacheBillsReq{AgentId: round.Agent, UserId: bet.UserID, GameId: round.GameID, Symbol: round.Symbol,
		Currency: bet.CurrencyType, Bet: amount.Neg().InexactFloat64(), CurrentScore: balance.InexactFloat64(),
		CreateTime: bet.BillCreatedAt, RoundID: round.RoundID, FlowingWaterOn: bet.BillID, Desc: "bet"}, nil
}

// Deliver only already-accepted debit outputs from the original round. Reuse
// the existing reporting timer and ES writer, without replaying a wager.
func (d *LotteryService) projectBetBills(store func([]*entity.CacheBillsReq) error) (int, error) {
	if d.roundLock == nil || d.rds == nil {
		return 0, nil
	}
	type pending struct {
		round         *financeRound
		betID, billID string
		cancel        bool
		invalid       bool
	}
	items := make([]pending, 0, 100)
	rows := make([]*entity.CacheBillsReq, 0, 100)
	d.roundLock.Lock()
	d.billScanCancelsFirst = !d.billScanCancelsFirst
	cancelFirst := d.billScanCancelsFirst
	scanned, rounds, visited := 0, 0, 0
	now := time.Now().Unix()
scan:
	for _, round := range d.rounds {
		visited++
		if visited > 1000 {
			break
		}
		if d.isRoundExpired(round, now) {
			continue
		}
		selected := false
		for phase := 0; phase < 2; phase++ {
			if (phase == 0) != cancelFirst {
				for id, bet := range round.Bets {
					scanned++
					row, err := originalBetBill(round, bet)
					if bet != nil && bet.BillID != "" && !bet.BillStored && !bet.BillAttempted && bet.BetID != id {
						err = fmt.Errorf("original debit bet key mismatch")
					}
					if err != nil {
						zap.L().Error("invalid debit bill skipped; no retry", zap.String("roundId", round.RoundID), zap.String("betId", id), zap.String("billId", bet.BillID), zap.Error(err))
						bet.BillAttempted = true
						if !selected {
							rounds++
							selected = true
						}
						items = append(items, pending{round, id, bet.BillID, false, true})
						row = nil
					}
					if row != nil {
						if !selected {
							rounds++
							selected = true
						}
						items = append(items, pending{round, id, row.FlowingWaterOn, false, false})
						rows = append(rows, row)
					}
					if len(items) >= 100 || scanned >= 1000 {
						break scan
					}
				}
			} else {
				for id, claim := range round.CancelRequests {
					scanned++
					row, err := originalCancelBill(round, claim)
					if claim != nil && claim.BillID != "" && !claim.BillStored && !claim.BillAttempted && claim.Status == financeCancelStatusCompleted && claim.RequestID != id {
						err = fmt.Errorf("original cancellation request key mismatch")
					}
					if err != nil {
						zap.L().Error("invalid cancellation bill skipped; no retry", zap.String("roundId", round.RoundID), zap.String("requestId", id), zap.String("billId", claim.BillID), zap.Error(err))
						claim.BillAttempted = true
						if !selected {
							rounds++
							selected = true
						}
						items = append(items, pending{round, id, claim.BillID, true, true})
						row = nil
					}
					if row != nil {
						if !selected {
							rounds++
							selected = true
						}
						items = append(items, pending{round, id, row.FlowingWaterOn, true, false})
						rows = append(rows, row)
					}
					if len(items) >= 100 || scanned >= 1000 {
						break scan
					}
				}
			}
		}
		if rounds >= 4 {
			break
		}
	}
	d.roundLock.Unlock()
	if len(items) == 0 {
		return 0, nil
	}
	var writeErr error
	if len(rows) > 0 {
		writeErr = store(rows)
	}
	if writeErr != nil {
		zap.L().Error("ES bill write failed; no retry", zap.Error(writeErr), zap.Int("count", len(rows)))
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	count := 0
	for _, item := range items {
		success := !item.invalid && writeErr == nil
		// Mark the resident attempt before saving output metadata. A failure
		// to persist this marker must not resubmit the ES output in this process.
		d.roundLock.Lock()
		if current := d.rounds[item.round.RoundID]; current != nil {
			if item.cancel {
				if claim := current.CancelRequests[item.betID]; claim != nil && claim.BillID == item.billID {
					claim.BillAttempted = true
					claim.BillStored = success
				}
			} else if bet := current.Bets[item.betID]; bet != nil && bet.BillID == item.billID {
				bet.BillAttempted = true
				bet.BillStored = success
			}
		}
		d.roundLock.Unlock()
		var confirmed bool
		var err error
		if item.cancel {
			confirmed, err = d.rds.ConfirmFinanceCancelBill(ctx, item.round.RoundID, item.betID, item.billID, success)
		} else {
			confirmed, err = d.rds.ConfirmFinanceBetBill(ctx, item.round.RoundID, item.betID, item.billID, success)
		}
		if err != nil {
			zap.L().Error("bill attempt marker not saved", zap.Error(err))
			continue
		}
		if !confirmed {
			continue
		}
		d.roundLock.Lock()
		if current := d.rounds[item.round.RoundID]; current != nil {
			if item.cancel {
				if claim := current.CancelRequests[item.betID]; claim != nil && claim.BillID == item.billID {
					claim.BillStored = success
				}
			} else if bet := current.Bets[item.betID]; bet != nil && bet.BillID == item.billID {
				bet.BillStored = success
			}
		}
		d.roundLock.Unlock()
		count++
	}
	return count, nil
}
