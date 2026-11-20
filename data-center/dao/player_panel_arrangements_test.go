package dao

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"testing"
)

func TestArrangementFactsPersistWithoutCardsOrPokerFlags(t *testing.T) {
	panelFixtureIdentity(t)
	db, script := gogameTestDB(t, gogameSQLStep{contains: "ON DUPLICATE KEY UPDATE", check: func(args []driver.NamedValue) {
		var record playerPanelRecord
		if len(args) != 12 || args[8].Value != false || args[9].Value != false || args[10].Value != true || json.Unmarshal([]byte(args[11].Value.(string)), &record) != nil {
			t.Fatal("invalid arrangement projection", args)
		}
		if record.BehaviorVersion != 4 || record.ArrangementChoices == nil || !record.ArrangementChoices.ClientSubmitted || !record.ArrangementChoices.Extended || record.BlackjackActions != nil || record.SeatChoices != nil || string(record.Players) != "[]" || string(record.Events) != "null" || strings.Contains(args[11].Value.(string), "wallet") {
			t.Fatal("wrong source or leaked private history", record)
		}
	}})
	raw := `{"records":[{"id":"arranged","agentId":7,"userId":42,"currency":"CNY","roomType":27,"roomId":123,"inning":1,"endedAt":1800000000000,"behaviorVersion":4,"countersKnown":true,"arrangementChoices":{"clientSubmitted":true,"extended":true},"blackjackActions":{},"seatChoices":{},"players":[{"wallet":999,"cards":[1,2]}],"events":[{"cards":[1,2]}]}]}`
	if _, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelWrite", raw); err != nil || len(script.steps) != 0 {
		t.Fatal(err, script.steps)
	}
}

func TestArrangementWriteRejectsPokerFlagsAndUnknownEvidence(t *testing.T) {
	for _, row := range []struct {
		known, voluntary, raised bool
		facts                    string
	}{
		{false, false, false, `{"clientSubmitted":true}`}, {true, true, false, `{}`}, {true, false, true, `{}`}, {true, false, false, `null`},
	} {
		panelFixtureIdentity(t)
		db, script := gogameTestDB(t)
		raw := fmt.Sprintf(`{"records":[{"id":"bad-arrangement","agentId":7,"userId":42,"currency":"CNY","roomType":27,"roomId":123,"inning":1,"endedAt":1800000000000,"behaviorVersion":4,"countersKnown":%t,"voluntary":%t,"raised":%t,"arrangementChoices":%s}]}`, row.known, row.voluntary, row.raised, row.facts)
		if _, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelWrite", raw); status.Code(err) != codes.InvalidArgument || len(script.steps) != 0 {
			t.Fatal("invalid arrangement facts accepted", row, err)
		}
	}
}

func TestArrangementTotalsReadFactsWithExactGameAndAccountScope(t *testing.T) {
	panelFixtureIdentity(t)
	db, script := gogameTestDB(t, gogameSQLStep{contains: "COUNT(*)", columns: []string{"total", "known", "table_hands", "table_known", "client_arrangement", "extended_arrangement"}, values: [][]driver.Value{{int64(4), int64(4), int64(2), int64(2), int64(3), int64(1)}}, check: func(args []driver.NamedValue) {
		if len(args) != 8 || args[0].Value != int64(123) || args[3].Value != int64(7) || args[4].Value != int64(42) || args[5].Value != "CNY" || args[6].Value != int64(27) {
			t.Fatal("scope lost", args)
		}
	}})
	value, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelRead", `{"agentId":7,"userId":42,"targetId":42,"currency":"CNY","roomId":123,"roomType":27}`)
	if err != nil || len(script.steps) != 0 {
		t.Fatal(err, script.steps)
	}
	stats := value.(map[string]any)["statistics"].(map[string]any)
	if stats["clientArrangementHands"] != int64(3) || stats["extendedArrangementHands"] != int64(1) || stats["behaviorHands"] != int64(4) || stats["voluntaryHands"] != int64(0) {
		t.Fatal(stats)
	}
	for _, field := range []string{"arrangementChoices.clientSubmitted", "arrangementChoices.extended", "room_type=?", "ended_at<=?"} {
		if !strings.Contains(script.log[0], field) {
			t.Fatal("missing factual source or scope", script.log)
		}
	}
}

func TestArrangementTotalsRejectCountsBeyondKnownAndPokerFlags(t *testing.T) {
	for _, row := range []struct{ client, extended, voluntary, raised int64 }{
		{-1, 0, 0, 0}, {5, 0, 0, 0}, {1, -1, 0, 0}, {1, 5, 0, 0}, {1, 0, 1, 0}, {1, 0, 1, 1},
	} {
		panelFixtureIdentity(t)
		db, _ := gogameTestDB(t, gogameSQLStep{contains: "COUNT(*)", columns: []string{"total", "known", "client_arrangement", "extended_arrangement", "voluntary", "raised"}, values: [][]driver.Value{{int64(4), int64(4), row.client, row.extended, row.voluntary, row.raised}}})
		if _, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelRead", `{"agentId":7,"userId":42,"targetId":42,"currency":"CNY","roomId":123,"roomType":27}`); status.Code(err) != codes.Unavailable {
			t.Fatal("invalid arrangement totals accepted", row, err)
		}
	}
}
