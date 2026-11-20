package dao

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func noFundTransfers() []gogameSQLStep {
	return []gogameSQLStep{
		{contains: "gp_user_score_log", columns: []string{"count"}, values: [][]driver.Value{{int64(0)}}},
		{contains: "gp_user_score_log", columns: []string{"id", "log_on", "log_type", "score", "user_score", "created"}},
	}
}
func fundBillHit(id, flow, round string, game int, bet, award, balance float64, at int64) map[string]any {
	return map[string]any{"_id": id, "sort": []any{at, flow}, "_source": map[string]any{"userId": 42, "agentId": 7, "gameId": game, "currency": "CNY", "roundId": round, "flowingWaterOn": flow, "bet": bet, "award": award, "currentScore": balance, "createTime": at}}
}

func TestFundQueryDoesNotCountSettlementNetInAdditionToItsOriginalBills(t *testing.T) {
	now := time.Now().UnixMilli()
	at := now / 1000
	dd, es, q := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "go_gp_settlement") {
			json.NewEncoder(w).Encode(map[string]any{"hits": map[string]any{"hits": []any{financialHistoryHit(1, now-1000, 50, 100)}}})
			return
		}
		if !strings.Contains(r.URL.Path, "go_gp_flowing_water") {
			t.Error("wrong fund source", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"hits": map[string]any{"hits": []any{fundBillHit("paid", "B02", "1", 91019, 0, 1.5, 10.5, at), fundBillHit("wager", "B01", "1", 91019, -1, 0, 9, at-1)}}})
	})
	db, _ := gogameTestDB(t, noFundTransfers()...)
	dd.manager = db
	q.Kind = "funds"
	q.RoomType = 0
	q.Size = 10
	value, err := dd.UserCenter(context.Background(), es, q)
	if err != nil {
		t.Fatal(err)
	}
	out := value.(map[string]any)
	logs := out["logs"].([]map[string]any)
	if out["total"].(int64) != 2 || len(logs) != 2 || logs[0]["point"] != int64(150) || logs[1]["point"] != int64(-100) {
		t.Fatal("settlement net duplicated original cash movements", out)
	}
	if logs[0]["lastPoint"] != int64(1050) || logs[1]["lastPoint"] != int64(900) {
		t.Fatal("original bill balances changed", out)
	}
	for _, row := range logs {
		if row["orderKey"] != nil {
			t.Fatal("internal order field escaped", row)
		}
	}
}

func TestFundFallbackSurvivesMissingBillsAndOtherGameRoundNumberCollision(t *testing.T) {
	now := time.Now().UnixMilli()
	dd, es, q := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "go_gp_settlement") {
			json.NewEncoder(w).Encode(map[string]any{"hits": map[string]any{"hits": []any{financialHistoryHit(1, now-1000, 50, 100), financialHistoryHit(2, now-2000, -20, 100)}}})
			return
		}
		foreign := fundBillHit("foreign", "B04", "1", 91019, 0, 999, 999, now/1000)
		foreign["_source"].(map[string]any)["agentId"] = 8
		json.NewEncoder(w).Encode(map[string]any{"hits": map[string]any{"hits": []any{foreign, fundBillHit("other-game", "B03", "1", 99, 0, 1, 10, now/1000)}}})
	})
	db, _ := gogameTestDB(t, noFundTransfers()...)
	dd.manager = db
	q.Kind = "funds"
	q.RoomType = 0
	q.Size = 10
	value, err := dd.UserCenter(context.Background(), es, q)
	if err != nil {
		t.Fatal(err)
	}
	out := value.(map[string]any)
	logs := out["logs"].([]map[string]any)
	if out["total"].(int64) != 3 || len(logs) != 3 {
		t.Fatal("other game or foreign record suppressed authorized fallback", out)
	}
	for _, row := range logs {
		if row["point"] == int64(99900) {
			t.Fatal("foreign agent escaped", out)
		}
	}
}

func TestFundDeduplicationHappensBeforePagingAndExactTotalsMatchEachPage(t *testing.T) {
	for _, page := range []int{1, 2, 3} {
		t.Run(fmt.Sprint(page), func(t *testing.T) {
			now := time.Now().UnixMilli()
			dd, es, q := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				var hits []any
				if strings.Contains(r.URL.Path, "go_gp_settlement") {
					hits = []any{financialHistoryHit(1, now-1000, 50, 100), financialHistoryHit(2, now-3000, 20, 100), financialHistoryHit(3, now-5000, 30, 100)}
				} else {
					hits = []any{fundBillHit("paid", "B01", "1", 91019, 0, 1.5, 10.5, now/1000)}
				}
				json.NewEncoder(w).Encode(map[string]any{"hits": map[string]any{"hits": hits}})
			})
			db, _ := gogameTestDB(t, noFundTransfers()...)
			dd.manager = db
			q.Kind = "funds"
			q.RoomType = 0
			q.Page = page
			q.Size = 1
			value, err := dd.UserCenter(context.Background(), es, q)
			if err != nil {
				t.Fatal(err)
			}
			out := value.(map[string]any)
			logs := out["logs"].([]map[string]any)
			if out["total"].(int64) != 3 || len(logs) != 1 {
				t.Fatal("dedup page lost older fallback", out)
			}
			if page == 1 && logs[0]["id"] != "bill:paid" {
				t.Fatal(out)
			}
			if page > 1 && logs[0]["point"] != int64(page*10) {
				t.Fatal("wrong fallback after original bill dedup", out)
			}
		})
	}
}

func TestFundBillCoverageReadsPastTheVisiblePageAndUsesCursor(t *testing.T) {
	now := time.Now().UnixMilli()
	calls := 0
	dd, es, q := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "go_gp_settlement") {
			json.NewEncoder(w).Encode(map[string]any{"hits": map[string]any{"hits": []any{financialHistoryHit(1, now-1000, 50, 100)}}})
			return
		}
		var query map[string]any
		json.NewDecoder(r.Body).Decode(&query)
		if query["from"] != nil {
			t.Error("fund query used offset pagination")
		}
		if calls > 0 && query["search_after"] == nil {
			t.Error("fund query cursor missing")
		}
		start := calls * 100
		calls++
		end := start + 100
		if end > 101 {
			end = 101
		}
		hits := []any{}
		for i := start; i < end; i++ {
			round := fmt.Sprintf("other-%d", i)
			if i == 100 {
				round = "1"
			}
			hits = append(hits, fundBillHit(fmt.Sprint(i), fmt.Sprintf("B%03d", 101-i), round, 91019, 0, 1, 10, now/1000-int64(i)))
		}
		json.NewEncoder(w).Encode(map[string]any{"hits": map[string]any{"hits": hits}})
	})
	db, _ := gogameTestDB(t, noFundTransfers()...)
	dd.manager = db
	q.Kind = "funds"
	q.RoomType = 0
	q.Size = 1
	value, err := dd.UserCenter(context.Background(), es, q)
	if err != nil {
		t.Fatal(err)
	}
	out := value.(map[string]any)
	if calls != 2 || out["total"].(int64) != 101 || out["logs"].([]map[string]any)[0]["id"] != "bill:0" {
		t.Fatal("bill outside visible page caused duplicate settlement fallback", calls, out)
	}
}

func TestFundBillIdentityDedupAndStableSameSecondPagination(t *testing.T) {
	for _, page := range []int{1, 2} {
		t.Run(fmt.Sprint(page), func(t *testing.T) {
			now := time.Now().UnixMilli()
			dd, es, q := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				hits := []any{}
				if strings.Contains(r.URL.Path, "flowing_water") {
					hits = []any{fundBillHit("a", "B02", "1", 91019, 0, 2, 10, now/1000), fundBillHit("duplicate", "B02", "1", 91019, 0, 2, 10, now/1000), fundBillHit("z", "B01", "2", 91019, 0, 1, 9, now/1000)}
				}
				json.NewEncoder(w).Encode(map[string]any{"hits": map[string]any{"hits": hits}})
			})
			db, _ := gogameTestDB(t, noFundTransfers()...)
			dd.manager = db
			q.Kind = "funds"
			q.RoomType = 0
			q.Page = page
			q.Size = 1
			value, err := dd.UserCenter(context.Background(), es, q)
			if err != nil {
				t.Fatal(err)
			}
			out := value.(map[string]any)
			wanted := "bill:a"
			if page == 2 {
				wanted = "bill:z"
			}
			if out["total"].(int64) != 2 || out["logs"].([]map[string]any)[0]["id"] != wanted {
				t.Fatal("same-second sort changed between source and merged pagination", out)
			}
		})
	}
}

func TestFundBillSourceFailureCannotReturnPartialSummarySuccess(t *testing.T) {
	dd, es, q := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "gp_settlement") {
			json.NewEncoder(w).Encode(map[string]any{"hits": map[string]any{"hits": []any{financialHistoryHit(1, time.Now().UnixMilli()-1000, 50, 100)}}})
			return
		}
		w.WriteHeader(503)
		fmt.Fprint(w, `{"error":"bill storage unavailable"}`)
	})
	q.Kind = "funds"
	q.RoomType = 0
	if _, err := dd.UserCenter(context.Background(), es, q); err == nil {
		t.Fatal("missing original cash source hidden as partial successful funds")
	}
}
