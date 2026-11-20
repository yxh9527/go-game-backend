package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/olivere/elastic/v7"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestFullCashReplayRoomReturnsOriginalRowsBeyondOnePage(t *testing.T) {
	now := time.Now().UnixMilli()
	calls := 0
	dd, es, q := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "go_gp_settlement") {
			t.Error("cash replay source changed", r.URL.Path)
		}
		var query map[string]any
		json.NewDecoder(r.Body).Decode(&query)
		if query["from"] != nil {
			t.Error("full replay used offsets")
		}
		if calls > 0 && query["search_after"] == nil {
			t.Error("second replay page has no cursor")
		}
		start := calls * 100
		calls++
		end := start + 100
		if end > 125 {
			end = 125
		}
		hits := []any{}
		for i := start; i < end; i++ {
			hit := financialHistoryHit(int64(i+1), now-int64(i+1000), 10, 100)
			hit["sort"] = []any{now - int64(i+1000), fmt.Sprintf("h-%03d", i)}
			hits = append(hits, hit)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"hits": map[string]any{"hits": hits}})
	})
	q.Kind = "replay"
	q.RoomID = 700000
	q.AllInnings = true
	q.Size = 10
	value, err := dd.UserCenter(context.Background(), es, q)
	if err != nil {
		t.Fatal(err)
	}
	out := value.(map[string]any)
	rows := out["records"].([]UserCenterRecord)
	if calls != 2 || len(rows) != 125 || out["total"].(int64) != 125 || out["totalScore"].(float64) != 1250 {
		t.Fatal("full room was truncated to the requested page size", calls, len(rows), out["total"])
	}
}

func TestFullCompetitionReplayUsesItsOwnCursorAndFiltersOtherAccounts(t *testing.T) {
	now := time.Now().UnixMilli()
	calls := 0
	dd, es, q := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "go_gogame_replays") || strings.Contains(r.URL.Path, "gp_settlement") {
			t.Error("competition consulted cash index", r.URL.Path)
		}
		var query map[string]any
		json.NewDecoder(r.Body).Decode(&query)
		raw, _ := json.Marshal(query)
		for _, scope := range []string{`"roomType":101`, `"userId":42`, `"agentId":7`, `"currency":{"query":"CNY"}`, `"endedAt"`, `"id.keyword"`} {
			if !strings.Contains(string(raw), scope) {
				t.Error("missing competition scope", scope, string(raw))
			}
		}
		if query["from"] != nil {
			t.Error("competition used deep offsets")
		}
		if calls > 0 && query["search_after"] == nil {
			t.Error("missing competition cursor")
		}
		start := calls * 100
		calls++
		end := start + 100
		if end > 125 {
			end = 125
		}
		hits := []any{}
		for i := start; i < end; i++ {
			row := map[string]any{"id": fmt.Sprintf("r-%03d", i), "userId": 42, "agentId": 7, "currency": "CNY", "roomType": 101, "roomId": 81000001, "inning": i + 1, "endedAt": now - int64(i+1000), "detail": map[string]any{"replay": map[string]any{"gameRoomId": 81000001, "currentInningNum": i + 1}}}
			if i == 123 {
				row["userId"] = 43
			}
			if i == 124 {
				row["roomId"] = 81000002
			}
			hits = append(hits, map[string]any{"_source": row, "sort": []any{now - int64(i+1000), fmt.Sprintf("r-%03d", i)}})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"hits": map[string]any{"hits": hits}})
	})
	q.Kind = "replay"
	q.RoomType = 101
	q.RoomID = 81000001
	q.AllInnings = true
	q.Size = 10
	value, err := dd.UserCenter(context.Background(), es, q)
	if err != nil {
		t.Fatal(err)
	}
	rows := value.(map[string]any)["records"].([]UserCenterRecord)
	if calls != 2 || len(rows) != 123 {
		t.Fatal("competition full-room read was truncated or leaked records", calls, len(rows))
	}
	for _, row := range rows {
		if row.Balance != nil || row.RoomType != 101 {
			t.Fatal("competition converted to cash result", row)
		}
	}
}

func TestFullReplayOptionRejectsUnboundedIdentityBeforeES(t *testing.T) {
	dd, es, q := centerFixture(t, func(http.ResponseWriter, *http.Request) { t.Fatal("invalid scope reached ES") })
	q.Kind = "replay"
	q.RoomType = 19
	q.RoomID = 700000
	q.AllInnings = true
	for _, change := range []func(*UserCenterQuery){func(q *UserCenterQuery) { q.Kind = "hands" }, func(q *UserCenterQuery) { q.RoomID = 0 }, func(q *UserCenterQuery) { q.RoomType = 25 }, func(q *UserCenterQuery) { q.Page = 2 }} {
		copy := q
		change(&copy)
		if _, err := dd.UserCenter(context.Background(), es, copy); err == nil {
			t.Fatal("unbounded full-room replay accepted", copy)
		}
	}
}

func TestCompetitionCursorRejectsRepeatedPageInsteadOfLooping(t *testing.T) {
	calls := 0
	_, es, _ := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		hits := make([]any, 100)
		for i := range hits {
			hits[i] = map[string]any{"_source": map[string]any{"id": "same", "roomType": 101}, "sort": []any{123, "same"}}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"hits": map[string]any{"hits": hits}})
	})
	if err := es.scanCompetitionReplayDocuments(context.Background(), elastic.NewMatchAllQuery(), func(*elastic.SearchHit) (bool, error) { return false, nil }); err == nil || calls > 2 {
		t.Fatal("competition cursor looped", calls, err)
	}
}
