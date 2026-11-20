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

func settlementBillBatch(round *financeRound) ([]*entity.CacheBillsReq, error) {
	settlement := round.Settlement
	if settlement == nil || settlement.Status != financeSettlementStatusCompleted || len(settlement.Bills) == 0 || settlement.ReportsIndexed {
		return nil, nil
	}
	if round.State != string(financeStateSettled) || (settlement.Mode != financeModeSeatNet && settlement.Mode != financeModeBankerNet) {
		return nil, fmt.Errorf("final bills require confirmed net settlement")
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
		copy := *source
		copy.CurrentScore = report.Balance
		rows = append(rows, &copy)
		seen[source.UserId] = true
	}
	if len(rows) != len(owners) {
		return nil, fmt.Errorf("final bills do not cover confirmed accounts")
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
