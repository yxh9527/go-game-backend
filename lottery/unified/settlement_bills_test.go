package unified

import (
	"app/entity"
	"bufio"
	"fmt"
	"github.com/olivere/elastic/v7"
	"github.com/shopspring/decimal"
	"lottery/dao"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func billRound(count int) *financeRound {
	round := reportRound(count)
	round.Settlement.Mode = financeModeSeatNet
	for _, report := range round.Settlement.Reports {
		round.Settlement.Bills = append(round.Settlement.Bills, settlementBill(round, report.Symbol, report.UserId, report.Currency, decimal.NewFromInt(1), "seat_net"))
	}
	return round
}

func billES(t *testing.T) (*dao.ESDao, *bool) {
	t.Helper()
	outage := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if outage {
			w.WriteHeader(503)
			fmt.Fprint(w, `{"error":"offline"}`)
			return
		}
		count := 0
		scan := bufio.NewScanner(r.Body)
		for scan.Scan() {
			if !scan.Scan() {
				t.Error("missing bulk document")
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
	client, err := elastic.NewClient(elastic.SetURL(server.URL), elastic.SetSniff(false), elastic.SetHealthcheck(false), elastic.SetMaxRetries(0))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Stop)
	return dao.NewESDao(client), &outage
}

func TestNetBillComesFromOriginalSnapshotAndConfirmedBalance(t *testing.T) {
	for _, mode := range []string{financeModeSeatNet, financeModeBankerNet} {
		round := billRound(1)
		round.Settlement.Mode = mode
		source := round.Settlement.Bills[0]
		id, created := source.FlowingWaterOn, source.CreateTime
		rows, err := settlementBillBatch(round)
		if err != nil || len(rows) != 1 || rows[0].CurrentScore != 12.34 || rows[0].FlowingWaterOn != id || rows[0].CreateTime != created || source.CurrentScore != 0 {
			t.Fatal(rows, err)
		}
		for _, mutate := range []func(){
			func() { source.AgentId++ }, func() { source.Currency = "USD" }, func() { source.UserId++ }, func() { source.GameId++ }, func() { source.FlowingWaterOn = "" },
		} {
			original := *source
			mutate()
			if _, err := settlementBillBatch(round); err == nil {
				t.Fatal("invalid original bill accepted", source)
			}
			*source = original
		}
		round.Settlement.Status = financeSettlementStatusClaimed
		if rows, err := settlementBillBatch(round); err != nil || len(rows) != 0 {
			t.Fatal("unconfirmed bill was emitted", rows, err)
		}
	}
}

func TestNetBillPersistenceIsBoundedAndPendingBillsDoNotExtendRetention(t *testing.T) {
	round := billRound(145)
	es, _ := billES(t)
	service := &LotteryService{es: es, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{round.RoundID: round}}
	counts := []int{}
	store := func(rows []*entity.CacheBillsReq) error { counts = append(counts, len(rows)); return nil }
	if _, err := service.projectSettlementReportsWithBills(1, store); err != nil {
		t.Fatal(err)
	}
	if round.Settlement.BillOffset != 100 || round.Settlement.ReportsIndexed {
		t.Fatal(round.Settlement)
	}
	if !service.isRoundExpired(round, time.Now().Unix()+financeRoundRetainSeconds+100) {
		t.Fatal("pending bills extended the one-hour snapshot retention")
	}
	if _, err := service.projectSettlementReportsWithBills(1, store); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(counts) != "[100 45]" || round.Settlement.BillOffset != 145 || !round.Settlement.ReportsIndexed {
		t.Fatal(counts, round.Settlement)
	}
	legacy := reportRound(1)
	if rows, err := settlementBillBatch(legacy); err != nil || len(rows) != 0 {
		t.Fatal("legacy snapshot invented bills", rows, err)
	}
}
