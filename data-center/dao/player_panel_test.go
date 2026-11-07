package dao

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"testing"
	"time"
)

func panelFixtureIdentity(t *testing.T) {
	client, _, _ := cashFixture(t)
	before := redisDao
	redisDao = &RedisDao{Client: client}
	t.Cleanup(func() { redisDao = before })
}
func TestPlayerPanelChinaWindows(t *testing.T) {
	today, week, seven := panelDay(time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC))
	if today.Format("2006-01-02 15:04") != "2026-11-07 00:00" || week.Format("2006-01-02") != "2026-11-02" || seven.Format("2006-01-02") != "2026-11-01" {
		t.Fatal(today, week, seven)
	}
}
func TestPlayerPanelDailyTwentiethGiftCommitsUnderRowLock(t *testing.T) {
	panelFixtureIdentity(t)
	db, script := gogameTestDB(t,
		gogameSQLStep{contains: "INSERT INTO"},
		gogameSQLStep{contains: "FOR UPDATE", columns: []string{"agent_id", "player_id", "used"}, values: [][]driver.Value{{int64(7), int64(42), int64(19)}}},
		gogameSQLStep{contains: "UPDATE", check: func(args []driver.NamedValue) {
			if args[0].Value != int64(20) {
				t.Fatal(args)
			}
		}},
	)
	value, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelGift", `{"agentId":7,"userId":42,"targetId":43,"currency":"CNY","propId":2}`)
	if err != nil || value.(map[string]any)["remaining"] != 0 {
		t.Fatal(value, err)
	}
	if len(script.steps) != 0 || script.log[0] != "begin" || script.log[len(script.log)-1] != "commit" {
		t.Fatal(script.log)
	}
}
func TestPlayerPanelTwentyFirstGiftRollsBackWithoutSpendingWallet(t *testing.T) {
	panelFixtureIdentity(t)
	db, script := gogameTestDB(t, gogameSQLStep{contains: "INSERT INTO"}, gogameSQLStep{contains: "FOR UPDATE", columns: []string{"agent_id", "player_id", "used"}, values: [][]driver.Value{{int64(7), int64(42), int64(20)}}})
	_, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelGift", `{"agentId":7,"userId":42,"targetId":43,"currency":"CNY","propId":2}`)
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatal(err)
	}
	if len(script.steps) != 0 || script.log[len(script.log)-1] != "rollback" {
		t.Fatal(script.log)
	}
}
func TestPlayerPanelNotePartialUpdateRetainsNameAndIsAuthorScoped(t *testing.T) {
	panelFixtureIdentity(t)
	db, script := gogameTestDB(t,
		gogameSQLStep{contains: "INSERT INTO"},
		gogameSQLStep{contains: "FOR UPDATE", columns: []string{"agent_id", "player_id", "target_id", "remark_name", "remark_message", "mark"}, values: [][]driver.Value{{int64(7), int64(42), int64(43), "Tight", "old note", int64(2)}}, check: func(args []driver.NamedValue) {
			if args[0].Value != int64(7) || args[1].Value != int64(42) || args[2].Value != int64(43) {
				t.Fatal(args)
			}
		}},
		gogameSQLStep{contains: "UPDATE"},
	)
	value, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelNoteSet", `{"agentId":7,"userId":42,"targetId":43,"currency":"CNY","patch":{"mark":0}}`)
	if err != nil {
		t.Fatal(err)
	}
	out := value.(map[string]any)
	if out["remarkName"] != "Tight" || out["remarkMessage"] != "old note" || out["mark"] != 0 {
		t.Fatal(out)
	}
	if len(script.steps) != 0 {
		t.Fatal(script.steps)
	}
}
func TestPlayerPanelSettledHandsUseImmutableUniqueRowsAndExcludeRobots(t *testing.T) {
	panelFixtureIdentity(t)
	db, script := gogameTestDB(t, gogameSQLStep{contains: "ON DUPLICATE KEY UPDATE"})
	raw := `{"records":[{"id":"hand:1","agentId":7,"userId":42,"currency":"CNY","roomId":700000,"inning":1,"endedAt":1800000000000,"net":100,"players":[{"uid":42,"score":100}],"raised":true,"voluntary":true,"countersKnown":true}]}`
	if _, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelWrite", raw); err != nil {
		t.Fatal(err)
	}
	if len(script.steps) != 0 {
		t.Fatal(script.steps)
	}
	_, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelWrite", strings.Replace(raw, `"userId":42`, `"userId":90000001`, 1))
	if status.Code(err) != codes.InvalidArgument {
		t.Fatal(err)
	}
}
func TestPlayerPanelScopeRejectsWrongAgentBeforeQueryingStats(t *testing.T) {
	panelFixtureIdentity(t)
	db, script := gogameTestDB(t)
	_, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelRead", `{"agentId":8,"userId":42,"targetId":42,"currency":"CNY","roomId":700000}`)
	if status.Code(err) != codes.PermissionDenied || len(script.steps) != 0 {
		t.Fatal(err, script.steps)
	}
}
func TestPlayerPanelDTORejectsUnknownFields(t *testing.T) {
	var input playerPanelInput
	if err := DecodeGogameJSON(`{"userId":42,"wallet":10000}`, &input); err == nil {
		t.Fatal("wallet snapshot must not be accepted")
	}
	if err := json.Unmarshal([]byte(`{"patch":{"mark":0}}`), &input); err != nil || input.Patch.Mark == nil || *input.Patch.Mark != 0 {
		t.Fatal(input, err)
	}
}
func TestPlayerPanelReadsLifetimeTotalsWithoutReturningWallets(t *testing.T) {
	panelFixtureIdentity(t)
	db, script := gogameTestDB(t,
		gogameSQLStep{contains: "COUNT(*) AS total_hands", columns: []string{"total_hands", "known", "raised", "voluntary", "table_hands", "table_voluntary", "today_score", "week_score", "seven_hands"}, values: [][]driver.Value{{int64(5), int64(5), int64(2), int64(3), int64(3), int64(2), int64(100), int64(150), int64(0)}}},
		gogameSQLStep{contains: "gg_player_panel_hand", columns: []string{"payload"}},
	)
	value, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelRead", `{"agentId":7,"userId":42,"targetId":42,"currency":"CNY","roomId":700000}`)
	if err != nil {
		t.Fatal(err)
	}
	summary := value.(map[string]any)["statistics"].(map[string]any)
	if summary["totalHands"] != int64(5) || summary["todayScore"] != int64(100) || summary["flipManualBetPercent"] != float64(40) || summary["tableJoinPercent"] != float64(66.67) {
		t.Fatal(summary)
	}
	if len(script.steps) != 0 {
		t.Fatal(script.steps)
	}
}
func TestPlayerPanelRecordsKeepTotalAndScoreWhenOnlyOnePageIsReturned(t *testing.T) {
	panelFixtureIdentity(t)
	db, script := gogameTestDB(t,
		gogameSQLStep{contains: "COUNT(*) AS total", columns: []string{"total", "score"}, values: [][]driver.Value{{int64(1234), int64(5678)}}},
		gogameSQLStep{contains: "OFFSET", columns: []string{"payload"}, values: [][]driver.Value{{`{"userId":42,"roomId":700000,"inning":3,"net":100,"endedAt":1800000000000,"players":[{"uid":42}]}`}}},
	)
	value, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelRecords", `{"agentId":7,"userId":42,"targetId":42,"currency":"CNY","roomId":700000,"page":2,"size":1}`)
	if err != nil {
		t.Fatal(err)
	}
	out := value.(map[string]any)
	if out["total"] != int64(1234) || out["score"] != int64(5678) || len(out["records"].([]json.RawMessage)) != 1 || out["paged"] != true {
		t.Fatal(out)
	}
	if len(script.steps) != 0 {
		t.Fatal(script.steps)
	}
}
