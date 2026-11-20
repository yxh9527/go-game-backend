package unified

import (
	"app/entity"
	"bufio"
	"fmt"
	"github.com/olivere/elastic/v7"
	"lottery/dao"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func reportRound(count int) *financeRound {
	round := &financeRound{RoundID: "final-report-round", Agent: 7, GameID: 92019, State: string(financeStateSettled), UpdatedAt: time.Now().Unix(), Settlement: &financeSettlement{SettlementID: "final-settlement", Status: financeSettlementStatusCompleted, CompletedAt: time.Now().Unix(), Mode: financeModeNormal}}
	for i := 0; i < count; i++ {
		uid := uint32(42 + i)
		round.Settlement.Reports = append(round.Settlement.Reports, &entity.CacheRecordsReq{AgentId: 7, UserId: uid, GameId: 92019, RoundID: round.RoundID, Complete: true, Currency: "CNY", Symbol: "test", Bet: 1, Win: 2, PlayedDate: 123456789, Log: `{"result":"actual"}`})
		round.Settlement.Results = append(round.Settlement.Results, &financeSettlementResult{UserID: uid, Currency: "12.34", Code: 0})
	}
	return round
}

func TestPendingReportsDoNotExtendOneHourRetentionAndIncompleteResultsCannotBeProjected(t *testing.T) {
	service := &LotteryService{}
	round := reportRound(1)
	now := time.Now().Unix() + financeRoundRetainSeconds + 100
	if !service.isRoundExpired(round, now) {
		t.Fatal("pending report extended the one-hour retention")
	}
	round.Settlement.ReportsIndexed = true
	if !service.isRoundExpired(round, now) {
		t.Fatal("indexed final round retained forever")
	}
	round.Settlement.ReportsIndexed = false
	round.Settlement.Status = financeSettlementStatusClaimed
	if rows, err := settlementReportBatch(round); err != nil || len(rows) != 0 {
		t.Fatal("unconfirmed settlement projected", rows, err)
	}
	round.Settlement.Status = financeSettlementStatusCompleted
	round.Settlement.Results[0].Currency = ""
	if _, err := settlementReportBatch(round); err == nil {
		t.Fatal("missing authoritative final balance accepted")
	}
	round.Settlement.Results[0].Currency = "1.00"
	round.Settlement.Reports[0].AgentId = 8
	if _, err := settlementReportBatch(round); err == nil {
		t.Fatal("cross-agent report accepted")
	}
}

func TestReportProjectionBoundsEachPassAndPersistsProgress(t *testing.T) {
	batchSizes := []int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := 0
		scanner := bufio.NewScanner(r.Body)
		for scanner.Scan() {
			if !scanner.Scan() {
				t.Error("missing document")
				break
			}
			count++
		}
		batchSizes = append(batchSizes, count)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"errors":false,"items":[`)
		for i := 0; i < count; i++ {
			if i > 0 {
				fmt.Fprint(w, ",")
			}
			fmt.Fprint(w, `{"index":{"status":200}}`)
		}
		fmt.Fprint(w, `]}`)
	}))
	defer server.Close()
	es, err := elastic.NewClient(elastic.SetURL(server.URL), elastic.SetSniff(false), elastic.SetHealthcheck(false))
	if err != nil {
		t.Fatal(err)
	}
	defer es.Stop()
	round := reportRound(145)
	service := &LotteryService{es: dao.NewESDao(es), roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{round.RoundID: round}}
	if _, err = service.projectSettlementReports(1); err != nil {
		t.Fatal(err)
	}
	if round.Settlement.ReportOffset != 100 || round.Settlement.ReportsIndexed {
		t.Fatal("pass was not bounded", round.Settlement.ReportOffset)
	}
	if _, err = service.projectSettlementReports(1); err != nil {
		t.Fatal(err)
	}
	if round.Settlement.ReportOffset != 145 || !round.Settlement.ReportsIndexed {
		t.Fatal(round.Settlement)
	}
	if fmt.Sprint(batchSizes) != "[40 40 20 40 5]" {
		t.Fatal(batchSizes)
	}
}

func TestESOutageBackpressureRetainsOriginalReportsAndClearsAfterProjection(t *testing.T) {
	round := reportRound(1)
	round.Settlement.Reports = make([]*entity.CacheRecordsReq, 50000)
	for i := range round.Settlement.Reports {
		round.Settlement.Reports[i] = &entity.CacheRecordsReq{Log: "original"}
	}
	service := &LotteryService{rounds: map[string]*financeRound{"one": round}}
	if !service.reportBacklogFullLocked(100) {
		t.Fatal("new rounds not blocked on full final report backlog")
	}
	if len(round.Settlement.Reports) != 50000 || round.Settlement.Reports[0].Log != "original" {
		t.Fatal("reports discarded by backpressure")
	}
	round.Settlement.ReportsIndexed = true
	if service.reportBacklogFullLocked(101) {
		t.Fatal("backpressure did not clear after confirmation")
	}
}
