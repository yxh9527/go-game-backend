package dao

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"testing"
)

func TestZJHBehaviorFactsPersistVersionedFlagsWithoutPrivateHistory(t *testing.T) {
	panelFixtureIdentity(t)
	db, script := gogameTestDB(t, gogameSQLStep{contains: "ON DUPLICATE KEY UPDATE", check: func(args []driver.NamedValue) {
		if len(args) != 12 {
			t.Fatal(args)
		}
		if args[8].Value != true || args[9].Value != true || args[10].Value != true {
			t.Fatal("confirmed facts lost", args)
		}
		var record playerPanelRecord
		if err := json.Unmarshal([]byte(args[11].Value.(string)), &record); err != nil {
			t.Fatal(err)
		}
		if record.BehaviorVersion != 1 || !record.Raised || !record.Voluntary || !record.CountersKnown || string(record.Players) != "[]" || string(record.Events) != "null" || string(record.CommonCards) != "null" {
			t.Fatal("private history or lost facts", record)
		}
	}})
	input := playerPanelRecord{ID: "zjh-action", AgentID: 7, UserID: 42, Currency: "CNY", RoomType: 25, RoomID: 123, Inning: 1, EndedAt: 1800000000000, Net: 10, EffectiveBet: 20, BehaviorVersion: 1, Raised: true, Voluntary: true, CountersKnown: true,
		Players: json.RawMessage(`[{"uid":42,"handCards":[1,2,3],"wallet":999}]`), Events: json.RawMessage(`[{"cards":[1,2,3]}]`), CommonCards: json.RawMessage(`[4,5]`)}
	raw, _ := json.Marshal(map[string]any{"records": []playerPanelRecord{input}})
	if _, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelWrite", string(raw)); err != nil || len(script.steps) != 0 {
		t.Fatal(err, script.steps)
	}
}

func TestZJHBehaviorWriteRejectsInvalidKnownFlagsAndLeavesUnsupportedGamesUnknown(t *testing.T) {
	for _, invalid := range []playerPanelRecord{{Raised: true, Voluntary: false, CountersKnown: true}, {CountersKnown: false}} {
		panelFixtureIdentity(t)
		db, script := gogameTestDB(t)
		invalid.ID = "invalid"
		invalid.AgentID = 7
		invalid.UserID = 42
		invalid.Currency = "CNY"
		invalid.RoomType = 25
		invalid.RoomID = 123
		invalid.Inning = 1
		invalid.EndedAt = 1800000000000
		invalid.BehaviorVersion = 1
		raw, _ := json.Marshal(map[string]any{"records": []playerPanelRecord{invalid}})
		if _, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelWrite", string(raw)); status.Code(err) != codes.InvalidArgument || len(script.steps) != 0 {
			t.Fatal("invalid flags accepted", err)
		}
	}
	for _, roomType := range []int{24, 25, 32} {
		panelFixtureIdentity(t)
		db, _ := gogameTestDB(t, gogameSQLStep{contains: "ON DUPLICATE KEY UPDATE", check: func(args []driver.NamedValue) {
			var saved playerPanelRecord
			if json.Unmarshal([]byte(args[11].Value.(string)), &saved) != nil || saved.BehaviorVersion != 0 || saved.CountersKnown || saved.Raised || saved.Voluntary {
				t.Fatal("unsupported evidence retained", saved)
			}
		}})
		r := playerPanelRecord{ID: "legacy", AgentID: 7, UserID: 42, Currency: "CNY", RoomType: roomType, RoomID: 123, Inning: 1, EndedAt: 1800000000000, BehaviorVersion: 2, CountersKnown: true, Raised: true, Voluntary: true}
		raw, _ := json.Marshal(map[string]any{"records": []playerPanelRecord{r}})
		if _, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelWrite", string(raw)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestZJHBehaviorTotalsShareOneScopedQueryAndNeverReadActionDetails(t *testing.T) {
	panelFixtureIdentity(t)
	db, script := gogameTestDB(t, gogameSQLStep{contains: "COUNT(*)", columns: []string{"total", "effective_bet_hands", "net_win_hands", "known", "voluntary", "raised", "table_hands", "table_known", "table_voluntary"}, values: [][]driver.Value{{int64(10), int64(10), int64(5), int64(9), int64(6), int64(2), int64(3), int64(3), int64(2)}}, check: func(args []driver.NamedValue) {
		if len(args) != 8 || args[0].Value != int64(123) || args[1].Value != int64(123) || args[2].Value != int64(123) || args[3].Value != int64(7) || args[4].Value != int64(42) || args[5].Value != "CNY" || args[6].Value != int64(25) {
			t.Fatal("query scope lost", args)
		}
	}})
	value, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelRead", `{"agentId":7,"userId":42,"targetId":42,"currency":"CNY","roomId":123,"roomType":25}`)
	if err != nil || len(script.steps) != 0 {
		t.Fatal(err, script.steps)
	}
	stats := value.(map[string]any)["statistics"].(map[string]any)
	if stats["behaviorHands"] != int64(9) || stats["voluntaryHands"] != int64(6) || stats["raisedHands"] != int64(2) || stats["tableBehaviorHands"] != int64(3) || stats["tableVoluntaryHands"] != int64(2) {
		t.Fatal(stats)
	}
	for _, part := range []string{"counters_known", "voluntary=1", "raised=1", "ended_at<=?", "room_type=?"} {
		if !strings.Contains(script.log[0], part) {
			t.Fatal("missing action evidence scope", script.log)
		}
	}
	if strings.Contains(script.log[0], "$.events") || strings.Contains(script.log[0], "$.players") {
		t.Fatal("statistics read raw action history", script.log)
	}
}

func TestZJHBehaviorTotalsRejectCorruptRelationshipsAndAllowGenuineEmptyData(t *testing.T) {
	valid := nativePanelTotals{Known: 10, Voluntary: 5, Raised: 3, TableHands: 4, TableKnown: 4, TableVoluntary: 2}
	if err := zjhPanelActions(nativePanelTotals{}, 0, map[string]any{}); err != nil {
		t.Fatal(err)
	}
	for _, edit := range []func(*nativePanelTotals){
		func(s *nativePanelTotals) { s.Known = -1 }, func(s *nativePanelTotals) { s.Known = 11 }, func(s *nativePanelTotals) { s.Voluntary = 11 }, func(s *nativePanelTotals) { s.Raised = 6 },
		func(s *nativePanelTotals) { s.TableHands = 11 }, func(s *nativePanelTotals) { s.TableKnown = 5 }, func(s *nativePanelTotals) { s.TableVoluntary = 5 },
	} {
		s := valid
		edit(&s)
		if err := zjhPanelActions(s, 10, map[string]any{}); status.Code(err) != codes.Unavailable {
			t.Fatal("corrupt behavior count accepted", s, err)
		}
	}
	if err := zjhPanelActions(valid, 10, map[string]any{}); err != nil {
		t.Fatal(err)
	}
}
