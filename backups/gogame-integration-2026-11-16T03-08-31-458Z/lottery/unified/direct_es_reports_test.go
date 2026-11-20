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
)

func TestProductionFinalReportsAndBillsWriteESWithNoMySQLDependencyAndRetryOriginalIDs(t *testing.T) {
	for _, mode := range []string{financeModeNormal, financeModeVoidRefund, financeModeSeatNet, financeModeBankerNet} {
		t.Run(mode, func(t *testing.T) {
			mr := miniredis.RunT(t)
			client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
			t.Cleanup(func() { client.Close() })
			rd := dao.NewFinanceRedis(client)
			round := billRound(1)
			if mode == financeModeNormal || mode == financeModeVoidRefund {
				round = normalBillRound(mode, 1)
				completeNormalBillRound(round)
			} else {
				round.Settlement.Mode = mode
			}
			outage := true
			ids := map[string][]string{}
			documents := map[string]string{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				scan := bufio.NewScanner(r.Body)
				count := 0
				for scan.Scan() {
					var head map[string]map[string]any
					if err := json.Unmarshal(scan.Bytes(), &head); err != nil {
						t.Error(err)
						return
					}
					op := head["index"]
					index, id := op["_index"].(string), op["_id"].(string)
					ids[index] = append(ids[index], id)
					if !scan.Scan() {
						t.Error("missing document")
						return
					}
					raw := string(scan.Bytes())
					if old, ok := documents[index+id]; ok && old != raw {
						t.Error("ES retry changed original output")
					}
					documents[index+id] = raw
					count++
				}
				if outage {
					w.WriteHeader(503)
					fmt.Fprint(w, `{"error":"offline"}`)
					return
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
			if err := svc.saveRoundLocked(round); err != nil {
				t.Fatal(err)
			}
			client.HSet(context.Background(), "player_42", "currency", 9999, "cash_revision", 8)
			if _, err := svc.projectSettlementReports(1); err == nil {
				t.Fatal("ES failure acknowledged")
			}
			if round.Settlement.ReportOffset != 0 || round.Settlement.BillOffset != 0 || round.Settlement.ReportsIndexed {
				t.Fatal("failed ES advanced original progress")
			}
			outage = false
			loaded := &LotteryService{rds: rd, es: dao.NewESDao(es), roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}}
			loaded.loadRoundStates()
			if n, err := loaded.projectSettlementReports(1); err != nil || n != 1 {
				t.Fatal(n, err)
			}
			if !loaded.rounds[round.RoundID].Settlement.ReportsIndexed {
				t.Fatal("ES success not confirmed")
			}
			if len(ids["go_gp_flowing_water"]) != 2 || ids["go_gp_flowing_water"][0] != ids["go_gp_flowing_water"][1] || len(ids["go_gp_settlement"]) != 1 {
				t.Fatal(ids)
			}
			if client.HGet(context.Background(), "player_42", "currency").Val() != "9999" || client.HGet(context.Background(), "player_42", "cash_revision").Val() != "8" {
				t.Fatal("output retry changed money")
			}
			before := len(ids["go_gp_settlement"])
			if _, err := loaded.projectSettlementReports(1); err != nil || len(ids["go_gp_settlement"]) != before {
				t.Fatal("confirmed output repeated", err)
			}
		})
	}
}

func TestFinalRecordESOutageAfterBillSuccessKeepsReportProgressAndOriginalSnapshot(t *testing.T) {
	round := reportRound(1)
	calls := 0
	fail := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if fail {
			fmt.Fprint(w, `{"errors":true,"items":[{"index":{"status":429}}]}`)
		} else {
			fmt.Fprint(w, `{"errors":false,"items":[{"index":{"status":201}}]}`)
		}
	}))
	t.Cleanup(server.Close)
	es, err := elastic.NewClient(elastic.SetURL(server.URL), elastic.SetSniff(false), elastic.SetHealthcheck(false))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(es.Stop)
	svc := &LotteryService{es: dao.NewESDao(es), roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{round.RoundID: round}}
	if _, err := svc.projectSettlementReports(1); err == nil || round.Settlement.ReportsIndexed || round.Settlement.ReportOffset != 0 {
		t.Fatal("partial ES success accepted", err)
	}
	fail = false
	if n, err := svc.projectSettlementReports(1); err != nil || n != 1 || calls != 2 {
		t.Fatal(n, err, calls)
	}
	if round.Settlement.Reports[0].Log != `{"result":"actual"}` || round.Settlement.Reports[0].Balance != 0 {
		t.Fatal("original report was rewritten")
	}
}

func TestBillAndRecordStoreMethodsRequireESAndUseNoMySQL(t *testing.T) {
	service := &LotteryService{}
	if err := service.storeSettlementBillBatch([]*entity.CacheBillsReq{{FlowingWaterOn: "original"}}); err == nil {
		t.Fatal("missing bill ES ignored")
	}
	if err := service.storeSettlementReportBatch([]*entity.CacheRecordsReq{{RoundID: "original"}}); err == nil {
		t.Fatal("missing record ES ignored")
	}
	if err := service.storeSettlementBillBatch(nil); err != nil {
		t.Fatal(err)
	}
	if err := service.storeSettlementReportBatch(nil); err != nil {
		t.Fatal(err)
	}
}
