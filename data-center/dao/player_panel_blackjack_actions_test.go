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

func TestBlackjackBehaviorPersistsIndependentFlagsAndStripsPrivateHistory(t *testing.T) {
	panelFixtureIdentity(t)
	db, script := gogameTestDB(t, gogameSQLStep{contains: "ON DUPLICATE KEY UPDATE", check: func(args []driver.NamedValue) {
		if len(args) != 12 || args[8].Value != false || args[9].Value != true || args[10].Value != true {
			t.Fatal("invalid blackjack column projection", args)
		}
		var saved playerPanelRecord
		if err := json.Unmarshal([]byte(args[11].Value.(string)), &saved); err != nil {
			t.Fatal(err)
		}
		if saved.BehaviorVersion != 2 || saved.BlackjackActions == nil || !saved.BlackjackActions.ManualBet || !saved.BlackjackActions.Doubled || !saved.BlackjackActions.Split || saved.BlackjackActions.InsuranceBuy || saved.Raised {
			t.Fatal("lost blackjack facts", saved)
		}
		if string(saved.Players) != "[]" || string(saved.Events) != "null" || string(saved.CommonCards) != "null" || strings.Contains(args[11].Value.(string), "wallet") {
			t.Fatal("private history persisted", saved)
		}
	}})
	_, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelWrite", `{"records":[{"id":"bj-v2","agentId":7,"userId":42,"currency":"CNY","roomType":31,"roomId":123,"inning":1,"endedAt":1800000000000,"net":20,"effectiveBet":200,"behaviorVersion":2,"countersKnown":true,"raised":false,"voluntary":true,"blackjackActions":{"manualBet":true,"doubled":true,"split":true,"insuranceBuy":false},"players":[{"uid":42,"wallet":999,"cards":[1,2]}],"events":[{"cards":[1,2]}],"commonCards":[3,4]}]}`)
	if err != nil || len(script.steps) != 0 {
		t.Fatal(err, script.steps)
	}
}

func TestBlackjackBehaviorRejectsConflictingFlagsAndKeepsUnsupportedRecordsUnknown(t *testing.T) {
	for _, patch := range []string{`"countersKnown":false,"raised":false,"voluntary":true,"blackjackActions":{"manualBet":true}`, `"countersKnown":true,"raised":true,"voluntary":true,"blackjackActions":{"manualBet":true}`, `"countersKnown":true,"raised":false,"voluntary":true`, `"countersKnown":true,"raised":false,"voluntary":false,"blackjackActions":{"manualBet":true}`} {
		panelFixtureIdentity(t)
		db, script := gogameTestDB(t)
		raw := `{"records":[{"id":"invalid-bj","agentId":7,"userId":42,"currency":"CNY","roomType":31,"roomId":123,"inning":1,"endedAt":1800000000000,"behaviorVersion":2,` + patch + `}]}`
		if _, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelWrite", raw); status.Code(err) != codes.InvalidArgument || len(script.steps) != 0 {
			t.Fatal("conflicting blackjack facts accepted", err)
		}
	}
	for _, room := range []int{24, 25, 32} {
		panelFixtureIdentity(t)
		db, _ := gogameTestDB(t, gogameSQLStep{contains: "ON DUPLICATE KEY UPDATE", check: func(args []driver.NamedValue) {
			var saved playerPanelRecord
			if json.Unmarshal([]byte(args[11].Value.(string)), &saved) != nil || saved.BlackjackActions != nil || saved.BehaviorVersion != 0 || saved.CountersKnown {
				t.Fatal("blackjack facts leaked to another game", saved)
			}
		}})
		r := playerPanelRecord{ID: "other", AgentID: 7, UserID: 42, Currency: "CNY", RoomType: room, RoomID: 123, Inning: 1, EndedAt: 1800000000000, BehaviorVersion: 2, CountersKnown: true, Voluntary: true}
		raw, _ := json.Marshal(map[string]any{"records": []playerPanelRecord{r}})
		if _, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelWrite", string(raw)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBlackjackActionTotalsFilterActualGameRoomAndKnownActionFlags(t *testing.T) {
	panelFixtureIdentity(t)
	db, script := gogameTestDB(t, gogameSQLStep{contains: "COUNT(*)", columns: []string{"total", "effective_bet_hands", "net_win_hands", "known", "voluntary", "raised", "table_hands", "table_known", "table_voluntary", "doubled", "split", "insurance_buy"}, values: [][]driver.Value{{int64(4), int64(4), int64(2), int64(4), int64(2), int64(0), int64(2), int64(2), int64(1), int64(1), int64(2), int64(0)}}, check: func(args []driver.NamedValue) {
		if len(args) != 8 || args[0].Value != int64(123) || args[1].Value != int64(123) || args[2].Value != int64(123) || args[3].Value != int64(7) || args[4].Value != int64(42) || args[5].Value != "CNY" || args[6].Value != int64(31) {
			t.Fatal("blackjack query scope lost", args)
		}
	}})
	value, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelRead", `{"agentId":7,"userId":42,"targetId":42,"currency":"CNY","roomId":123,"roomType":31}`)
	if err != nil || len(script.steps) != 0 {
		t.Fatal(err, script.steps)
	}
	stats := value.(map[string]any)["statistics"].(map[string]any)
	if stats["behaviorHands"] != int64(4) || stats["voluntaryHands"] != int64(2) || stats["raisedHands"] != int64(0) || stats["doubleHands"] != int64(1) || stats["splitHands"] != int64(2) || stats["insuranceBuyHands"] != int64(0) {
		t.Fatal(stats)
	}
	for _, name := range []string{"blackjackActions.doubled", "blackjackActions.split", "blackjackActions.insuranceBuy"} {
		if !strings.Contains(script.log[0], name) {
			t.Fatal("action source missing", script.log)
		}
	}
}

func TestBlackjackActionTotalsRejectImpossibleCounters(t *testing.T) {
	for _, data := range []struct{ known, raised, doubled, split, insurance int64 }{
		{4, 1, 0, 0, 0}, {4, 0, 5, 0, 0}, {4, 0, 0, -1, 0}, {4, 0, 0, 0, 5}, {2, 0, 3, 0, 0},
	} {
		panelFixtureIdentity(t)
		db, _ := gogameTestDB(t, gogameSQLStep{contains: "COUNT(*)", columns: []string{"total", "known", "raised", "voluntary", "doubled", "split", "insurance_buy"}, values: [][]driver.Value{{int64(4), data.known, data.raised, int64(2), data.doubled, data.split, data.insurance}}})
		if _, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelRead", `{"agentId":7,"userId":42,"targetId":42,"currency":"CNY","roomId":123,"roomType":31}`); status.Code(err) != codes.Unavailable {
			t.Fatal("invalid blackjack counters accepted", data, err)
		}
	}
}
