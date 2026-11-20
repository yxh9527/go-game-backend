package unified

import (
	"app/entity"
	"bufio"
	"context"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/olivere/elastic/v7"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"lottery/dao"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestFinalReportsAndBillsFailOnceLogAndNeverRetryAfterHandlerReplacement(t *testing.T) {
	for _, mode := range []string{financeModeNormal, financeModeVoidRefund, financeModeSeatNet, financeModeBankerNet} {
		t.Run(mode, func(t *testing.T) {
			logs, entries := observer.New(zap.ErrorLevel)
			restore := zap.ReplaceGlobals(zap.New(logs))
			defer restore()
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
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(503)
				fmt.Fprint(w, `{"error":"offline"}`)
			}))
			t.Cleanup(server.Close)
			es, err := elastic.NewClient(elastic.SetURL(server.URL), elastic.SetSniff(false), elastic.SetHealthcheck(false))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(es.Stop)
			svc := &LotteryService{rds: rd, es: dao.NewESDao(es), roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}}
			if err := svc.saveRoundLocked(round); err != nil {
				t.Fatal(err)
			}
			client.HSet(context.Background(), "player_42", "currency", 9999, "cash_revision", 8)
			if n, err := svc.projectSettlementReports(1); err != nil || n != 1 {
				t.Fatal(n, err)
			}
			if calls != 2 || entries.Len() != 2 {
				t.Fatal("expected one bill and one record attempt with logs", calls, entries.Len())
			}
			if !round.Settlement.ReportsAttempted || round.Settlement.ReportsIndexed || !round.Settlement.ReportWriteFailed || !round.Settlement.BillWriteFailed {
				t.Fatal("attempt was confused with ES success", round.Settlement)
			}
			for i := 0; i < 3; i++ {
				if _, err := svc.projectSettlementReports(1); err != nil {
					t.Fatal(err)
				}
			}
			loaded := &LotteryService{rds: rd, es: dao.NewESDao(es), roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}}
			loaded.loadRoundStates()
			if _, err := loaded.projectSettlementReports(1); err != nil || calls != 2 {
				t.Fatal("failed report retried", err, calls)
			}
			if client.HGet(context.Background(), "player_42", "currency").Val() != "9999" || client.HGet(context.Background(), "player_42", "cash_revision").Val() != "8" {
				t.Fatal("ES failure affected money")
			}
		})
	}
}

func TestFailedDebitAndRefundOutputsAreAttemptedButNotStoredAndNeverRequeued(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		t.Run(fmt.Sprint(cancel), func(t *testing.T) {
			mr := miniredis.RunT(t)
			client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
			t.Cleanup(func() { client.Close() })
			rd := dao.NewFinanceRedis(client)
			round := debitOutputFixture()
			var claim *financeCancelRequest
			if cancel {
				round, claim = completedCancelOutput()
				for _, bet := range round.Bets {
					bet.BillStored = true
				}
			}
			svc := &LotteryService{rds: rd, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}}
			if err := svc.saveRoundLocked(round); err != nil {
				t.Fatal(err)
			}
			calls := 0
			store := func([]*entity.CacheBillsReq) error { calls++; return fmt.Errorf("ES unavailable") }
			if n, err := svc.projectBetBills(store); err != nil || n != 1 {
				t.Fatal(n, err)
			}
			if cancel {
				if !claim.BillAttempted || claim.BillStored {
					t.Fatal(claim)
				}
			} else {
				bet := round.Bets["original-debit:rev:0:first"]
				if !bet.BillAttempted || bet.BillStored {
					t.Fatal(bet)
				}
			}
			if _, err := svc.projectBetBills(store); err != nil || calls != 1 {
				t.Fatal("failed bill resubmitted", err, calls)
			}
			loaded := &LotteryService{rds: rd, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}}
			loaded.loadRoundStates()
			if _, err := loaded.projectBetBills(store); err != nil || calls != 1 {
				t.Fatal("failed bill retried after replacement", err, calls)
			}
		})
	}
}

func TestPartialESFailureDoesNotBlockLaterBatchesOrMarkActualIndexSuccess(t *testing.T) {
	round := reportRound(145)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		count := 0
		scan := bufio.NewScanner(r.Body)
		for scan.Scan() {
			scan.Scan()
			count++
		}
		fmt.Fprint(w, `{"items":[`)
		for i := 0; i < count; i++ {
			if i > 0 {
				fmt.Fprint(w, ",")
			}
			status := 201
			if calls == 1 {
				status = 429
			}
			fmt.Fprintf(w, `{"index":{"status":%d}}`, status)
		}
		fmt.Fprint(w, `]}`)
	}))
	t.Cleanup(server.Close)
	es, err := elastic.NewClient(elastic.SetURL(server.URL), elastic.SetSniff(false), elastic.SetHealthcheck(false))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(es.Stop)
	svc := &LotteryService{es: dao.NewESDao(es), roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{round.RoundID: round}}
	if _, err := svc.projectSettlementReports(1); err != nil {
		t.Fatal(err)
	}
	if round.Settlement.ReportOffset != 100 || round.Settlement.ReportsAttempted {
		t.Fatal(round.Settlement)
	}
	if _, err := svc.projectSettlementReports(1); err != nil {
		t.Fatal(err)
	}
	if calls != 5 || !round.Settlement.ReportsAttempted || round.Settlement.ReportsIndexed {
		t.Fatal(calls, round.Settlement)
	}
	if _, err := svc.projectSettlementReports(1); err != nil || calls != 5 {
		t.Fatal("old failed batch retried", err, calls)
	}
}
