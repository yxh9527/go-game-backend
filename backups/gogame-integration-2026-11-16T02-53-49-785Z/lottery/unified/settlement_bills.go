package unified

import (
	"app/entity"
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/shopspring/decimal"
	"math"
	"time"
)

// This is the original final bill, carried by the existing terminal snapshot.
// Its identity and creation time are fixed before the atomic monetary commit.
func settlementBill(round *financeRound, symbol string, userID uint32, currency string, delta decimal.Decimal, description string) *entity.CacheBillsReq {
	identity := fmt.Sprintf("%d|%s|%s|%d|%s", round.Agent, round.RoundID, round.Settlement.SettlementID, userID, currency)
	return &entity.CacheBillsReq{AgentId: round.Agent, UserId: userID, GameId: round.GameID,
		Symbol: symbol, Currency: currency, Bet: delta.InexactFloat64(), CreateTime: round.Settlement.CompletedAt,
		RoundID: round.RoundID, FlowingWaterOn: fmt.Sprintf("L%x", sha256.Sum256([]byte(identity))), Desc: description}
}

func normalSettlementBills(round *financeRound, settlement *financeSettlement) []*entity.CacheBillsReq {
	copy := *round
	copy.Settlement = settlement
	created := time.Now().Unix()
	rows := make([]*entity.CacheBillsReq, 0, len(settlement.Reports))
	for _, report := range settlement.Reports {
		if report == nil || report.Win <= 0 {
			continue
		}
		description := "settlement"
		if settlement.Mode == financeModeVoidRefund {
			description = "void_refund"
		}
		bill := settlementBill(&copy, report.Symbol, report.UserId, report.Currency, decimal.NewFromFloat(report.Win), description)
		bill.CreateTime = created
		rows = append(rows, bill)
	}
	return rows
}

func settlementBillBatch(round *financeRound) ([]*entity.CacheBillsReq, error) {
	settlement := round.Settlement
	if settlement == nil || settlement.Status != financeSettlementStatusCompleted || len(settlement.Bills) == 0 || settlement.ReportsIndexed {
		return nil, nil
	}
	netMode := settlement.Mode == financeModeSeatNet || settlement.Mode == financeModeBankerNet
	normalMode := settlement.Mode == financeModeNormal || settlement.Mode == financeModeVoidRefund
	expectedState := string(financeStateSettled)
	if settlement.Mode == financeModeVoidRefund {
		expectedState = string(financeStateVoided)
	}
	if !netMode && !normalMode || round.State != expectedState {
		return nil, fmt.Errorf("final bills require a matching confirmed settlement")
	}
	reports, err := settlementReportBatch(round)
	if err != nil {
		return nil, err
	}
	owners := map[uint32]*entity.CacheRecordsReq{}
	for _, report := range reports {
		owners[report.UserId] = report
	}
	rows := make([]*entity.CacheBillsReq, 0, len(settlement.Bills))
	seen := map[uint32]bool{}
	for _, source := range settlement.Bills {
		if source == nil || source.AgentId != round.Agent || source.RoundID != round.RoundID || source.GameId != round.GameID || source.FlowingWaterOn == "" || source.CreateTime <= 0 || seen[source.UserId] || math.IsNaN(source.Bet) || math.IsInf(source.Bet, 0) {
			return nil, fmt.Errorf("invalid original settlement bill")
		}
		report := owners[source.UserId]
		if report == nil || report.Currency != source.Currency || report.Symbol != source.Symbol {
			return nil, fmt.Errorf("bill has no matching confirmed account")
		}
		if normalMode {
			if math.IsNaN(report.Win) || math.IsInf(report.Win, 0) {
				return nil, fmt.Errorf("invalid original payout amount")
			}
			amount, err := parseMoney(decimal.NewFromFloat(report.Win).String())
			description := "settlement"
			if settlement.Mode == financeModeVoidRefund {
				description = "void_refund"
			}
			if err != nil || !amount.IsPositive() || !decimal.NewFromFloat(source.Bet).Equal(amount) || source.Desc != description {
				return nil, fmt.Errorf("payout bill differs from original confirmed amount")
			}
		}
		copy := *source
		copy.CurrentScore = report.Balance
		rows = append(rows, &copy)
		seen[source.UserId] = true
	}
	for user, report := range owners {
		if (netMode || report.Win > 0) && !seen[user] {
			return nil, fmt.Errorf("final bills do not cover confirmed payable accounts")
		}
	}
	return rows, nil
}

func (d *LotteryService) storeSettlementBillBatch(rows []*entity.CacheBillsReq) error {
	if len(rows) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return d.db.StoreFinancialRecords(ctx, nil, rows)
}
