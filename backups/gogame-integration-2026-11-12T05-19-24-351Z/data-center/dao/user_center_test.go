package dao

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/olivere/elastic/v7"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func centerFixture(t *testing.T, work func(http.ResponseWriter, *http.Request)) (*DBDao, *ESDao, UserCenterQuery) {
	t.Helper()
	client, _, a := cashFixture(t)
	before := redisDao
	redisDao = &RedisDao{Client: client}
	t.Cleanup(func() { redisDao = before })
	server := httptest.NewServer(http.HandlerFunc(work))
	t.Cleanup(server.Close)
	es, err := elastic.NewClient(elastic.SetURL(server.URL), elastic.SetSniff(false), elastic.SetHealthcheck(false))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(es.Stop)
	now := time.Now().UnixMilli()
	return &DBDao{}, NewESDao(es), UserCenterQuery{UserID: a.UserID, AgentID: a.AgentID, Currency: a.Currency, Kind: "battles", RoomType: 19, StartTime: now - 86400000, EndTime: now, Page: 1, Size: 10, SortDesc: true}
}
func TestUserCenterQueryIsScopedAndGroupsActualRooms(t *testing.T) {
	calls := 0
	now := time.Now().UnixMilli()
	dd, es, q := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, _ := io.ReadAll(r.Body)
		text := string(body)
		for _, needle := range []string{`"userId":42`, `"agentId":7`, `"currency":{"query":"CNY"}`, `"roomType":19`, `"privateRoom":false`} {
			if !strings.Contains(text, needle) {
				t.Errorf("missing scope %s in %s", needle, text)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"hits":{"total":{"value":2,"relation":"eq"},"hits":[]},"aggregations":{"totalScore":{"value":30},"roomCount":{"value":1},"rooms":{"buckets":[{"key":700000,"doc_count":2,"score":{"value":30},"bet":{"value":300},"started":{"value":%d},"sample":{"hits":{"total":{"value":1},"hits":[{"_source":{"id":"record","userId":42,"agentId":7,"currency":"CNY","gameId":"holdem","roomType":19,"roomId":700000,"inning":2,"endedAt":%d,"net":20,"effectiveBet":200,"players":[],"detail":{}}}]}}}]}}}`, now-1000, now)
	})
	result, err := dd.UserCenter(context.Background(), es, q)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(map[string]any)
	if out["total"].(int64) != 1 || out["totalScore"].(float64) != 30 {
		t.Fatal(out)
	}
	rows := out["records"].([]UserCenterRecord)
	if len(rows) != 1 || rows[0].Aggregate["totalHands"].(int64) != 2 {
		t.Fatal(rows)
	}
	q.AgentID = 8
	if _, err = dd.UserCenter(context.Background(), es, q); status.Code(err) != codes.PermissionDenied {
		t.Fatal("agent mismatch not rejected", err)
	}
	if calls != 1 {
		t.Fatal("mismatched scope queried ES")
	}
}
func TestUserCenterStorageErrorsAreNotEmptySuccess(t *testing.T) {
	dd, es, q := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `{"error":{"type":"unavailable","reason":"offline"},"status":500}`)
	})
	if _, err := dd.UserCenter(context.Background(), es, q); err == nil {
		t.Fatal("ES outage masked as empty list")
	}
	q.Page = 0
	if _, err := dd.UserCenter(context.Background(), es, q); status.Code(err) != codes.InvalidArgument {
		t.Fatal(err)
	}
}
func TestUserCenterWritesOnlyCompletedDisplayRecords(t *testing.T) {
	calls := 0
	dd, es, _ := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, _ := io.ReadAll(r.Body)
		text := string(body)
		if !strings.Contains(text, `"create"`) || strings.Contains(text, "cash-ledger") {
			t.Error("unexpected record write", text)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"errors":true,"items":[{"create":{"status":409}}]}`)
	})
	balance := int64(1000)
	record := UserCenterRecord{ID: "7:holdem:1:one:42", UserID: 42, AgentID: 7, Currency: "CNY", GameID: "holdem", RoomType: 19, RoomID: 700000, Inning: 1, EndedAt: time.Now().UnixMilli(), EffectiveBet: 100, Net: 50, Balance: &balance, Players: json.RawMessage(`[]`), Detail: json.RawMessage(`{}`)}
	if _, err := dd.WriteUserCenter(context.Background(), es, []UserCenterRecord{record}); err != nil {
		t.Fatal("existing record should stay original", err)
	}
	if calls != 1 {
		t.Fatal(calls)
	}
	record.Net = cashMax + 1
	if _, err := dd.WriteUserCenter(context.Background(), es, []UserCenterRecord{record}); status.Code(err) != codes.InvalidArgument {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("invalid display record was written")
	}
}
func TestUserCenterFundsMergeExistingSourcesWithCorrectUnits(t *testing.T) {
	now := time.Now().UnixMilli()
	dd, es, q := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "gogame_records") {
			fmt.Fprintf(w, `{"hits":{"total":{"value":1},"hits":[{"_source":{"id":"game","userId":42,"agentId":7,"currency":"CNY","roomType":19,"roomId":700000,"inning":1,"endedAt":%d,"net":50,"balance":1050,"effectiveBet":100,"detail":{},"players":[]}}]}}`, now-3000)
			return
		}
		fmt.Fprintf(w, `{"hits":{"total":{"value":1},"hits":[{"_id":"old-bill","_source":{"agentId":7,"userId":42,"gameId":99,"currency":"CNY","createTime":%d,"desc":"rebate","bet":0,"award":1.5,"currentScore":12.34}}]}}`, (now-2000)/1000)
	})
	database, script := gogameTestDB(t, gogameSQLStep{contains: "gp_user_score_log", columns: []string{"count"}, values: [][]driver.Value{{int64(1)}}},
		gogameSQLStep{contains: "gp_user_score_log", columns: []string{"id", "log_on", "log_type", "score", "user_score", "created"}, values: [][]driver.Value{{int64(1), "score-log", int64(1), "2.50", "14.84", now / 1000}}})
	dd.manager = database
	q.Kind = "funds"
	q.RoomType = 0
	q.Size = 2
	value, err := dd.UserCenter(context.Background(), es, q)
	if err != nil {
		t.Fatal(err)
	}
	out := value.(map[string]any)
	logs := out["logs"].([]map[string]any)
	if out["total"].(int64) != 3 || len(logs) != 2 || logs[0]["point"].(int64) != 250 || logs[0]["lastPoint"].(int64) != 1484 || logs[1]["point"].(int64) != 150 || logs[1]["lastPoint"].(int64) != 1234 {
		t.Fatal(out)
	}
	if len(script.steps) != 0 {
		t.Fatal("transfer source was not queried")
	}
}

func TestUserCenterTournamentUsesRealRegistrationsAndPaidRewards(t *testing.T) {
	dd, es, q := centerFixture(t, func(w http.ResponseWriter, r *http.Request) { t.Error("tournament history must not query ES") })
	now := time.Now().UnixMilli()
	payload := `{"registrations":{"42":{"userId":42,"name":"me","rank":1,"status":"winner","receipt":{"paidGold":100,"paidService":10}},"43":{"userId":43,"name":"other","rank":2,"status":"eliminated"}},"prizes":[{"status":"paid","plan":{"userId":42,"kind":"money","amount":500,"currency":"CNY"},"receipt":{"amount":500,"currency":"CNY"}},{"status":"pending","plan":{"userId":42,"kind":"money","amount":999,"currency":"CNY"}}]}`
	database, _ := gogameTestDB(t, gogameSQLStep{contains: "gg_match_runtime", columns: []string{"id", "agent_id", "currency", "start_time", "end_time", "state", "championship_name", "payload", "current_player"},
		values: [][]driver.Value{{int64(9), int64(7), "CNY", now - 10000, now, int64(3), "event", payload, int64(2)}}})
	dd.manager = database
	q.Kind = "matches"
	q.RoomType = 101
	value, err := dd.UserCenter(context.Background(), es, q)
	if err != nil {
		t.Fatal(err)
	}
	out := value.(map[string]any)
	matches := out["matches"].([]map[string]any)
	if len(matches) != 1 || out["totalScore"].(int64) != 390 || matches[0]["rewardAmount"].(int64) != 500 || len(matches[0]["MTTUserRanks"].([]map[string]any)) != 2 {
		t.Fatal(out)
	}
}

func TestUserCenterTicketHistoryReadsExistingHoldingTimestamps(t *testing.T) {
	dd, es, q := centerFixture(t, func(w http.ResponseWriter, r *http.Request) { t.Error("ticket history must not query ES") })
	now := time.Now().Unix()
	database, _ := gogameTestDB(t, gogameSQLStep{contains: "gg_ticket_hold", columns: []string{"count"}, values: [][]driver.Value{{int64(2)}}},
		gogameSQLStep{contains: "gg_ticket_hold", columns: []string{"id", "created", "num", "source_id"}, values: [][]driver.Value{{"used:1", now, int64(-1), int64(1)}, {"issued:1", now - 100, int64(1), int64(0)}}})
	dd.db = database
	q.Kind = "tickets"
	q.RoomType = 0
	value, err := dd.UserCenter(context.Background(), es, q)
	if err != nil {
		t.Fatal(err)
	}
	out := value.(map[string]any)
	logs := out["logs"].([]map[string]any)
	if out["total"].(int64) != 2 || len(logs) != 2 || logs[0]["num"].(int) != -1 || logs[1]["num"].(int) != 1 {
		t.Fatal(out)
	}
	if _, invented := logs[0]["lastNum"]; invented {
		t.Fatal("unknown historical ticket balance was invented")
	}
}
