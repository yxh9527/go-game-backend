package unified

import (
	"app/entity"
	"fmt"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"time"
)

// Backpressure applies only to a new round. Previously accepted rounds retain
// their ability to complete or refund while ES is unavailable.
func (d *LotteryService) reportBacklogFullLocked(at int64) bool {
	if d.reportAdmissionAt == at {
		return d.reportAdmissionBlocked
	}
	d.reportAdmissionAt = at
	count, bytes := 0, 0
	for _, round := range d.rounds {
		for _, bet := range round.Bets {
			if bet != nil && bet.BillID != "" && !bet.BillStored {
				count++
				bytes += 512
			}
		}
		for _, claim := range round.CancelRequests {
			if claim != nil && claim.BillID != "" && !claim.BillStored {
				count++
				bytes += 512
			}
		}
		if count >= 50000 || bytes >= 128*1024*1024 {
			d.reportAdmissionBlocked = true
			return true
		}
		if round.Settlement == nil || round.Settlement.ReportsIndexed {
			continue
		}
		for _, report := range round.Settlement.Reports {
			if report != nil {
				count++
				bytes += len(report.Log) + len(report.Init) + 512
			}
		}
		bytes += len(round.Settlement.Bills) * 512
		if count >= 50000 || bytes >= 128*1024*1024 {
			d.reportAdmissionBlocked = true
			return true
		}
	}
	d.reportAdmissionBlocked = false
	return false
}

func settlementReportBatch(round *financeRound) ([]*entity.CacheRecordsReq, error) {
	settlement := round.Settlement
	if settlement == nil || settlement.Status != financeSettlementStatusCompleted || settlement.ReportsIndexed || len(settlement.Reports) == 0 {
		return nil, nil
	}
	if round.State != string(financeStateSettled) && round.State != string(financeStateVoided) {
		return nil, fmt.Errorf("final reports require a terminal round")
	}
	balances := map[uint32]decimal.Decimal{}
	for _, result := range settlement.Results {
		if result == nil || result.Code != 0 {
			return nil, fmt.Errorf("final report result is not confirmed")
		}
		balance, err := parseMoney(result.Currency)
		if err != nil {
			return nil, fmt.Errorf("invalid confirmed report balance")
		}
		if _, exists := balances[result.UserID]; exists {
			return nil, fmt.Errorf("duplicate final report result")
		}
		balances[result.UserID] = balance
	}
	reports := make([]*entity.CacheRecordsReq, 0, len(settlement.Reports))
	seen := map[uint32]bool{}
	for _, source := range settlement.Reports {
		if source == nil || source.RoundID != round.RoundID || source.AgentId != round.Agent || !source.Complete || seen[source.UserId] {
			return nil, fmt.Errorf("invalid final reporting snapshot")
		}
		balance, exists := balances[source.UserId]
		if !exists {
			return nil, fmt.Errorf("final report has no confirmed result")
		}
		row := *source
		row.Balance = balance.InexactFloat64()
		row.BalanceCash = row.Balance
		if round.Scratch != nil {
			if err := attachScratchDisplayReport(round, &row); err != nil {
				return nil, err
			}
		}
		if err := attachSmartTaskSettlementProof(round, &row); err != nil {
			return nil, err
		}
		reports = append(reports, &row)
		seen[source.UserId] = true
	}
	if len(reports) != len(balances) {
		return nil, fmt.Errorf("final reports do not cover every confirmed account")
	}
	return reports, nil
}

// Use only the already existing final round snapshots. No wallet, balance
// mutation, refund, or settlement is replayed by this ES projection.
func (d *LotteryService) projectSettlementReports(limit int) (int, error) {
	return d.projectSettlementReportsWithBills(limit, d.storeSettlementBillBatch)
}

func (d *LotteryService) projectSettlementReportsWithBills(limit int, storeBills func([]*entity.CacheBillsReq) error) (int, error) {
	if d.es == nil || d.roundLock == nil || limit < 1 {
		return 0, nil
	}
	d.roundLock.RLock()
	ids := make([]string, 0, limit)
	for id, round := range d.rounds {
		if round.Settlement != nil && len(round.Settlement.Reports) > 0 && !round.Settlement.ReportsIndexed && round.Settlement.Status == financeSettlementStatusCompleted {
			ids = append(ids, id)
			if len(ids) >= limit {
				break
			}
		}
	}
	d.roundLock.RUnlock()
	count := 0
	for _, id := range ids {
		d.roundLock.RLock()
		round := d.rounds[id]
		if round == nil {
			d.roundLock.RUnlock()
			continue
		}
		settlement := round.Settlement
		offset := settlement.ReportOffset
		billOffset := settlement.BillOffset
		reports, err := settlementReportBatch(round)
		var bills []*entity.CacheBillsReq
		if err == nil {
			bills, err = settlementBillBatch(round)
		}
		d.roundLock.RUnlock()
		if err != nil {
			return count, err
		}
		if len(reports) == 0 {
			continue
		}
		if offset < 0 || offset > len(reports) {
			return count, fmt.Errorf("invalid report projection position")
		}
		if billOffset < 0 || billOffset > len(bills) {
			return count, fmt.Errorf("invalid bill persistence position")
		}
		billStop := billOffset + 100
		if billStop > len(bills) {
			billStop = len(bills)
		}
		if billStop > billOffset {
			if err = storeBills(bills[billOffset:billStop]); err != nil {
				return count, err
			}
		}
		stop := offset + 100
		if stop > len(reports) {
			stop = len(reports)
		}
		for start := offset; start < stop; start += 40 {
			end := start + 40
			if end > stop {
				end = stop
			}
			if err = d.es.BulkRecordsSave(reports[start:end]); err != nil {
				return count, err
			}
			if err = d.es.BulkSettledDisplayReports(reports[start:end]); err != nil {
				return count, err
			}
		}
		d.roundLock.Lock()
		if current := d.rounds[id]; current == round && current.Settlement == settlement && !settlement.ReportsIndexed {
			// Persistence acknowledgement is part of projection completion. A lost
			// acknowledgement repeats fixed-ID ES writes, never monetary operations.
			settlement.ReportsIndexed = stop == len(reports) && billStop == len(bills)
			settlement.ReportOffset = stop
			settlement.BillOffset = billStop
			if err = d.saveRoundLocked(current); err != nil {
				settlement.ReportsIndexed = false
				settlement.ReportOffset = offset
				settlement.BillOffset = billOffset
				d.roundLock.Unlock()
				return count, err
			}
			count++
		}
		d.roundLock.Unlock()
	}
	return count, nil
}

func (d *LotteryService) startSettlementReportProjection() {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			for batch := 0; batch < 4; batch++ {
				count, err := d.projectBetBills(d.storeSettlementBillBatch)
				if err != nil {
					zap.L().Error("original bet bill persistence pending", zap.Error(err))
					break
				}
				if count < 100 {
					break
				}
			}
			if _, err := d.projectSettlementReports(4); err != nil {
				zap.L().Error("final settlement report ES projection pending", zap.Error(err))
			}
		}
	}()
}
