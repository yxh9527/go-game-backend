package dao

import (
	"app/entity"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func financialHistoryHit(inning int64, ended, net, bet int64) map[string]any {
	report := UserCenterRecord{ID: fmt.Sprintf("7:holdem:700000:%d:42", inning), UserID: 42, AgentID: 7, Currency: "CNY", GameID: "holdem", RoomType: 19, RoomID: 700000, Inning: inning, EndedAt: ended, Net: net, EffectiveBet: bet, Detail: json.RawMessage(`{"replay":{"firstGameList":[]}}`), Players: json.RawMessage(`[]`)}
	log, _ := json.Marshal(map[string]any{"completedDisplayReport": report})
	return map[string]any{"_source": entity.CacheRecordsReq{UserId: 42, AgentId: 7, Currency: "CNY", GameId: 91019, RoundID: fmt.Sprint(inning), Complete: true, Bet: float64(bet) / 100, Win: float64(bet+net) / 100, Balance: 10.50, Log: string(log), PlayedDate: ended}}
}

func TestCashHistoryUsesOriginalLogAndRejectsAlteredOwnershipAmountsOrInit(t *testing.T) {
	hit := financialHistoryHit(1, time.Now().UnixMilli(), 50, 100)
	source := hit["_source"].(entity.CacheRecordsReq)
	row, err := settlementHistoryRecord(&source)
	if err != nil || row == nil || row.Net != 50 || row.Balance == nil || *row.Balance != 1050 {
		t.Fatal(row, err)
	}
	for _, change := range []func(*entity.CacheRecordsReq){func(r *entity.CacheRecordsReq) { r.AgentId++ }, func(r *entity.CacheRecordsReq) { r.UserId++ }, func(r *entity.CacheRecordsReq) { r.Currency = "USD" }, func(r *entity.CacheRecordsReq) { r.Win = 9 }} {
		copy := source
		change(&copy)
		if _, err := settlementHistoryRecord(&copy); err == nil {
			t.Fatal("modified financial history accepted", copy)
		}
	}
	source.Init = source.Log
	source.Log = "{}"
	if row, err := settlementHistoryRecord(&source); err != nil || row != nil {
		t.Fatal("init used as substitute for missing original log", row, err)
	}
}

func TestOriginalCashHistoryPaginationGroupingAndRoomScopes(t *testing.T) {
	now := time.Now().UnixMilli()
	dd, es, q := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "go_gp_settlement") {
			t.Error("cash history read display index", r.URL.Path)
		}
		hits := []any{financialHistoryHit(1, now-100, 10, 100), financialHistoryHit(2, now-50, -20, 200), financialHistoryHit(3, now, 40, 300)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"hits": map[string]any{"hits": hits}})
	})
	q.EndTime = now
	q.Kind = "hands"
	q.Size = 1
	q.Page = 2
	q.SortDesc = true
	value, err := dd.UserCenter(context.Background(), es, q)
	if err != nil {
		t.Fatal(err)
	}
	out := value.(map[string]any)
	rows := out["records"].([]UserCenterRecord)
	if out["total"].(int64) != 3 || out["totalScore"].(float64) != 30 || len(rows) != 1 || rows[0].Inning != 2 {
		t.Fatal(out)
	}
	q.Kind = "round"
	q.Page = 1
	q.Inning = 1
	q.RoomID = 700000
	value, err = dd.UserCenter(context.Background(), es, q)
	if err != nil || len(value.(map[string]any)["records"].([]UserCenterRecord)) != 1 {
		t.Fatal(value, err)
	}
	q.RoomID = 700001
	value, err = dd.UserCenter(context.Background(), es, q)
	if err != nil || value.(map[string]any)["total"].(int64) != 0 {
		t.Fatal("other room returned", value, err)
	}
}

func TestCompetitionReplayKeepsNonCashIndexAndNeverReadsFinancialSnapshots(t *testing.T) {
	dd, es, q := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "go_gogame_replays") || strings.Contains(r.URL.Path, "gp_settlement") {
			t.Error("competition queried cash finance", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"hits":{"total":{"value":1},"hits":[{"_source":{"id":"match","userId":42,"agentId":7,"currency":"CNY","roomType":101,"roomId":81000001,"inning":3,"endedAt":%d,"detail":{"replay":{}}}}]}}`, time.Now().UnixMilli()-1000)
	})
	q.Kind = "replay"
	q.RoomType = 101
	q.RoomID = 81000001
	value, err := dd.UserCenter(context.Background(), es, q)
	if err != nil {
		t.Fatal(err)
	}
	rows := value.(map[string]any)["records"].([]UserCenterRecord)
	if len(rows) != 1 || rows[0].RoomType != 101 || rows[0].Balance != nil {
		t.Fatal("competition became cash history", value)
	}
}
