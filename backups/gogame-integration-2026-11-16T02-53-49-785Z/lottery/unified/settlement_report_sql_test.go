package unified

import (
	"app/entity"
	"bufio"
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
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

func sqlReportRound(mode string, count int) *financeRound {
	round := reportRound(count)
	round.Settlement.Mode = mode
	if mode == financeModeVoidRefund {
		round.State = string(financeStateVoided)
	}
	for i, report := range round.Settlement.Reports {
		report.RowVersion = int64(100 + i)
	}
	return round
}

func canonicalSQLReport(t *testing.T, report *entity.CacheRecordsReq) string {
	t.Helper()
	copy := *report
	copy.Hash = fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%d|%d|%s", copy.AgentId, copy.UserId, copy.RoundID))))
	raw, err := json.Marshal(copy)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestFinalReportSQLAcceptancePrecedesESAndRetriesOriginalDocumentAfterReplacement(t *testing.T) {
	for _, mode := range []string{financeModeNormal, financeModeVoidRefund, financeModeSeatNet, financeModeBankerNet} {
		t.Run(mode, func(t *testing.T) {
			mr := miniredis.RunT(t)
			client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
			t.Cleanup(func() { client.Close() })
			rd := dao.NewFinanceRedis(client)
			round := sqlReportRound(mode, 1)
			old := &LotteryService{rds: rd, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}}
			if err := old.saveRoundLocked(round); err != nil {
				t.Fatal(err)
			}
			client.HSet(context.Background(), "player_42", "currency", 9999, "cash_revision", 8)
			sqlDown, esDown := true, true
			events := []string{}
			stored := map[string]string{}
			store := func(rows []*entity.CacheRecordsReq) error {
				events = append(events, "SQL")
				if sqlDown {
					return errors.New("SQL unavailable")
				}
				for _, row := range rows {
					if row.Balance != 12.34 || row.PlayedDate != 123456789 || row.Log != `{"result":"actual"}` || row.RowVersion != 100 {
						t.Fatal("original final report changed", row)
					}
					key := fmt.Sprintf("%d:%s", row.UserId, row.RoundID)
					raw := canonicalSQLReport(t, row)
					if previous, ok := stored[key]; ok && previous != raw {
						t.Fatal("retry replaced original accepted final report")
					}
					stored[key] = raw
				}
				return nil
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				events = append(events, "ES")
				if len(stored) != 1 {
					t.Error("ES started before SQL accepted original report")
				}
				w.Header().Set("Content-Type", "application/json")
				if esDown {
					w.WriteHeader(503)
					fmt.Fprint(w, `{"error":"offline"}`)
					return
				}
				scan := bufio.NewScanner(r.Body)
				count := 0
				for scan.Scan() {
					if !scan.Scan() {
						t.Error("missing document")
						return
					}
					count++
				}
				fmt.Fprint(w, `{"items":[`)
				for i := 0; i < count; i++ {
					if i > 0 {
						fmt.Fprint(w, ",")
					}
					fmt.Fprint(w, `{"index":{"status":201}}`)
				}
				fmt.Fprint(w, `]}`)
			}))
			t.Cleanup(server.Close)
			es, err := elastic.NewClient(elastic.SetURL(server.URL), elastic.SetSniff(false), elastic.SetHealthcheck(false), elastic.SetMaxRetries(0))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(es.Stop)
			svc := &LotteryService{rds: rd, es: dao.NewESDao(es), roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}}
			svc.loadRoundStates()
			if _, err := svc.projectSettlementOutputs(1, nil, store); err == nil {
				t.Fatal("SQL outage reported success")
			}
			if fmt.Sprint(events) != "[SQL]" || svc.rounds[round.RoundID].Settlement.ReportOffset != 0 || svc.rounds[round.RoundID].Settlement.ReportsIndexed {
				t.Fatal("SQL failure advanced ES or progress", events)
			}
			sqlDown = false
			if _, err := svc.projectSettlementOutputs(1, nil, store); err == nil {
				t.Fatal("ES outage reported success")
			}
			if fmt.Sprint(events) != "[SQL SQL ES]" || svc.rounds[round.RoundID].Settlement.ReportOffset != 0 {
				t.Fatal(events)
			}
			// SQL has accepted the original output. Replacing the handler and
			// changing the current wallet must not change that accepted document.
			replacement := &LotteryService{rds: rd, es: dao.NewESDao(es), roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}}
			replacement.loadRoundStates()
			esDown = false
			if n, err := replacement.projectSettlementOutputs(1, nil, store); err != nil || n != 1 {
				t.Fatal(n, err)
			}
			final := replacement.rounds[round.RoundID].Settlement
			if !final.ReportsIndexed || final.ReportOffset != 1 || len(stored) != 1 {
				t.Fatal(final, stored)
			}
			before := len(events)
			if n, err := replacement.projectSettlementOutputs(1, nil, store); err != nil || n != 0 || len(events) != before {
				t.Fatal("completed document repeated", n, err, events)
			}
			if client.HGet(context.Background(), "player_42", "currency").Val() != "9999" || client.HGet(context.Background(), "player_42", "cash_revision").Val() != "8" {
				t.Fatal("final document acceptance replayed money")
			}
		})
	}
}

func TestProductionFinalReportPathCannotSkipMissingSQLStorage(t *testing.T) {
	round := sqlReportRound(financeModeNormal, 1)
	es, _ := billES(t)
	svc := &LotteryService{es: es, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{round.RoundID: round}}
	if n, err := svc.projectSettlementReports(1); err == nil || n != 0 || round.Settlement.ReportsIndexed || round.Settlement.ReportOffset != 0 {
		t.Fatal("production path bypassed missing original SQL acceptance", n, err)
	}
}

func TestFinalReportSQLBatchLimitAndExpiredOrClaimedResultsCannotBeSaved(t *testing.T) {
	round := sqlReportRound(financeModeNormal, 145)
	es, _ := billES(t)
	svc := &LotteryService{es: es, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{round.RoundID: round}}
	counts := []int{}
	store := func(rows []*entity.CacheRecordsReq) error { counts = append(counts, len(rows)); return nil }
	for i := 0; i < 2; i++ {
		if _, err := svc.projectSettlementOutputs(1, nil, store); err != nil {
			t.Fatal(err)
		}
	}
	if fmt.Sprint(counts) != "[100 45]" || !round.Settlement.ReportsIndexed {
		t.Fatal(counts, round.Settlement)
	}
	for _, variant := range []string{"expired", "claimed"} {
		round := sqlReportRound(financeModeNormal, 1)
		if variant == "expired" {
			round.Settlement.CompletedAt = time.Now().Unix() - 3600
		} else {
			round.Settlement.Status = financeSettlementStatusClaimed
		}
		svc.rounds = map[string]*financeRound{round.RoundID: round}
		if n, err := svc.projectSettlementOutputs(1, nil, func([]*entity.CacheRecordsReq) error {
			t.Fatal("ineligible final result submitted to SQL", variant)
			return nil
		}); err != nil || n != 0 {
			t.Fatal(n, err)
		}
	}
}

func TestFinalReportSQLFailurePreservesBothBillAndReportProgress(t *testing.T) {
	round := billRound(1)
	round.Settlement.Reports[0].RowVersion = 100
	es, _ := billES(t)
	svc := &LotteryService{es: es, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{round.RoundID: round}}
	events := []string{}
	if _, err := svc.projectSettlementOutputs(1, func(rows []*entity.CacheBillsReq) error { events = append(events, "bill"); return nil }, func([]*entity.CacheRecordsReq) error {
		events = append(events, "report")
		return errors.New("report SQL failed")
	}); err == nil {
		t.Fatal("failed report acceptance ignored")
	}
	if fmt.Sprint(events) != "[bill report]" || round.Settlement.ReportOffset != 0 || round.Settlement.BillOffset != 0 || round.Settlement.ReportsIndexed {
		t.Fatal("partial SQL acceptance advanced progress", events, round.Settlement)
	}
}
