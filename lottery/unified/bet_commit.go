package unified

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"micro_service/services"
	"time"
)

func (d *LotteryService) commitAcceptedBet(round *financeRound, bet *financeBetSnapshot, amount int64) (*financeRound, bool, error) {
	if d.testCurrencyHook != nil {
		balance, code := d.testCurrencyHook(bet.UserID, -amount)
		if code != services.ErrorCode_OK {
			return nil, false, fmt.Errorf("FINANCE_INSUFFICIENT")
		}
		bet.Currency = decimalFromCent(balance).StringFixed(2)
		d.rounds[round.RoundID] = round
		return round, false, nil
	}
	if d.rds == nil {
		return nil, false, fmt.Errorf("finance storage unavailable")
	}
	round.UpdatedAt = time.Now().Unix()
	raw, err := jsoniter.MarshalToString(round)
	if err != nil {
		return nil, false, err
	}
	saved, already, err := d.rds.ApplyFinanceBet(round.RoundID, bet.BetID, raw, bet.UserID, round.Agent, bet.CurrencyType, amount)
	if err != nil {
		return nil, false, err
	}
	committed := &financeRound{}
	if err = jsoniter.UnmarshalFromString(saved, committed); err != nil {
		return nil, false, err
	}
	if committed.Bets[bet.BetID] == nil {
		return nil, false, fmt.Errorf("finance commit snapshot missing bet")
	}
	d.rounds[round.RoundID] = committed
	return committed, already, nil
}
