package dao

import (
	"context"
	"database/sql/driver"
	"net/http"
	"testing"
	"time"
)

func TestTournamentHistoryIncludesReentryFeesAndRefunds(t *testing.T) {
	dd, es, q := centerFixture(t, func(http.ResponseWriter, *http.Request) { t.Error("tournament history must not query ES") })
	now := time.Now().UnixMilli()
	payload := `{"registrations":{"42":{"userId":42,"name":"me","rank":1,"status":"winner","receipt":{"paidGold":100,"paidService":10},"reentries":[{"receipt":{"paidGold":100,"paidService":10}},{"receipt":{"paidGold":100,"paidService":10},"refund":{"refunded":110}}]}},"prizes":[{"status":"paid","plan":{"userId":42,"kind":"money","amount":500,"currency":"CNY"},"receipt":{"amount":500,"currency":"CNY"}}]}`
	db, _ := gogameTestDB(t, gogameSQLStep{contains: "gg_match_runtime", columns: []string{"id", "agent_id", "currency", "start_time", "end_time", "state", "payload"}, values: [][]driver.Value{{int64(9), int64(7), "CNY", now - 10000, now, int64(3), payload}}})
	dd.manager = db
	q.Kind = "matches"
	q.RoomType = 101
	result, err := dd.UserCenter(context.Background(), es, q)
	if err != nil {
		t.Fatal(err)
	}
	if result.(map[string]any)["totalScore"].(int64) != 280 {
		t.Fatal("reentry payments or refunds missing from tournament history", result)
	}
}
