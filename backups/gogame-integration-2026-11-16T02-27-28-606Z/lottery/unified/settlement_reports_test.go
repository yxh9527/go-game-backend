package unified

import (
	"app/entity"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/olivere/elastic/v7"
	"github.com/redis/go-redis/v9"
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

func TestFinalSnapshotSurvivesProcessReplacementAndESOutageWithoutReplayingMoney(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()
	rd := dao.NewFinanceRedis(client)
	outage := true
	var batches [][]map[string]any
	var ids [][]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scan := bufio.NewScanner(r.Body)
		rows := []map[string]any{}
		batchIDs := []string{}
		for scan.Scan() {
			var head map[string]map[string]any
			_ = json.Unmarshal(scan.Bytes(), &head)
			batchIDs = append(batchIDs, head["index"]["_id"].(string))
			if !scan.Scan() {
				t.Error("missing bulk record")
				return
			}
			var row map[string]any
			_ = json.Unmarshal(scan.Bytes(), &row)
			rows = append(rows, row)
		}
		batches = append(batches, rows)
		ids = append(ids, batchIDs)
		w.Header().Set("Content-Type", "application/json")
		if outage {
			w.WriteHeader(503)
			fmt.Fprint(w, `{"error":"offline"}`)
			return
		}
		fmt.Fprint(w, `{"errors":false,"items":[`)
		for i := range rows {
			if i > 0 {
				fmt.Fprint(w, ",")
			}
			fmt.Fprint(w, `{"index":{"status":201}}`)
		}
		fmt.Fprint(w, `]}`)
	}))
	defer server.Close()
	es, err := elastic.NewClient(elastic.SetURL(server.URL), elastic.SetSniff(false), elastic.SetHealthcheck(false), elastic.SetMaxRetries(0))
	if err != nil {
		t.Fatal(err)
	}
	defer es.Stop()
	old := &LotteryService{rds: rd, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}}
	round := reportRound(1)
	if err = old.saveRoundLocked(round); err != nil {
		t.Fatal(err)
	}
	client.HSet(context.Background(), "player_42", "currency", 1234, "cash_revision", 9)
	next := &LotteryService{rds: rd, es: dao.NewESDao(es), roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}}
	next.loadRoundStates()
	if len(next.rounds) != 1 {
		t.Fatal("final report payload was not loaded")
	}
	if _, err = next.projectSettlementReports(1); err == nil {
		t.Fatal("ES failure marked as indexed")
	}
	raw, _ := rd.HGet(financeRoundHash, round.RoundID)
	var stored financeRound
	_ = json.Unmarshal([]byte(raw), &stored)
	if stored.Settlement.ReportsIndexed {
		t.Fatal("report lost on ES outage")
	}
	outage = false
	if n, e := next.projectSettlementReports(1); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	if len(ids) != 2 || ids[0][0] != ids[1][0] {
		t.Fatal("retry created another document identity", ids)
	}
	if batches[1][0]["balance"] != 12.34 || batches[1][0]["playedDate"] != float64(123456789) || batches[1][0]["log"] != `{"result":"actual"}` {
		t.Fatal(batches)
	}
	if client.HGet(context.Background(), "player_42", "currency").Val() != "1234" || client.HGet(context.Background(), "player_42", "cash_revision").Val() != "9" {
		t.Fatal("report projection touched money")
	}
	raw, _ = rd.HGet(financeRoundHash, round.RoundID)
	_ = json.Unmarshal([]byte(raw), &stored)
	if !stored.Settlement.ReportsIndexed {
		t.Fatal("ES success not saved")
	}
	if _, e := next.projectSettlementReports(1); e != nil || len(ids) != 2 {
		t.Fatal("completed projection repeated", e)
	}
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
