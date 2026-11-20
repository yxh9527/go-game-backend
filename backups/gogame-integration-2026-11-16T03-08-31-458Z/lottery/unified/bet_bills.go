package unified

import (
	"app/entity"
	"context"
	"crypto/sha256"
	"fmt"
	"time"
)

func originalBetBillID(round *financeRound, bet *financeBetSnapshot) string {
	identity := fmt.Sprintf("bet|%d|%s|%s|%d|%s", round.Agent, round.RoundID, bet.BetID, bet.UserID, bet.CurrencyType)
	return fmt.Sprintf("L%x", sha256.Sum256([]byte(identity)))
}

func originalBetBill(round *financeRound, bet *financeBetSnapshot) (*entity.CacheBillsReq, error) {
	if bet == nil || bet.BillID == "" || bet.BillStored {
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
// the existing reporting timer and SQL store, without replaying a wager.
func (d *LotteryService) projectBetBills(store func([]*entity.CacheBillsReq) error) (int, error) {
	if d.roundLock == nil || d.rds == nil {
		return 0, nil
	}
	type pending struct {
		round         *financeRound
		betID, billID string
		cancel        bool
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
					if bet != nil && bet.BillID != "" && bet.BetID != id {
						d.roundLock.Unlock()
						return 0, fmt.Errorf("original debit bet key mismatch")
					}
					row, err := originalBetBill(round, bet)
					if err != nil {
						d.roundLock.Unlock()
						return 0, err
					}
					if row != nil {
						if !selected {
							rounds++
							selected = true
						}
						items = append(items, pending{round, id, row.FlowingWaterOn, false})
						rows = append(rows, row)
					}
					if len(rows) >= 100 || scanned >= 1000 {
						break scan
					}
				}
			} else {
				for id, claim := range round.CancelRequests {
					scanned++
					if claim != nil && claim.BillID != "" && claim.RequestID != id {
						d.roundLock.Unlock()
						return 0, fmt.Errorf("original cancellation request key mismatch")
					}
					row, err := originalCancelBill(round, claim)
					if err != nil {
						d.roundLock.Unlock()
						return 0, err
					}
					if row != nil {
						if !selected {
							rounds++
							selected = true
						}
						items = append(items, pending{round, id, row.FlowingWaterOn, true})
						rows = append(rows, row)
					}
					if len(rows) >= 100 || scanned >= 1000 {
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
	if len(rows) == 0 {
		return 0, nil
	}
	if err := store(rows); err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	count := 0
	for _, item := range items {
		var confirmed bool
		var err error
		if item.cancel {
			confirmed, err = d.rds.ConfirmFinanceCancelBill(ctx, item.round.RoundID, item.betID, item.billID)
		} else {
			confirmed, err = d.rds.ConfirmFinanceBetBill(ctx, item.round.RoundID, item.betID, item.billID)
		}
		if err != nil {
			return count, err
		}
		if !confirmed {
			continue
		}
		d.roundLock.Lock()
		if current := d.rounds[item.round.RoundID]; current != nil {
			if item.cancel {
				if claim := current.CancelRequests[item.betID]; claim != nil && claim.BillID == item.billID {
					claim.BillStored = true
				}
			} else if bet := current.Bets[item.betID]; bet != nil && bet.BillID == item.billID {
				bet.BillStored = true
			}
		}
		d.roundLock.Unlock()
		count++
	}
	return count, nil
}
