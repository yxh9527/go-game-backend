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

func TestMahjongDiscardsPersistOnlyFactsForAllFamilies(t *testing.T) {
	for _, room := range []int{26, 28, 30} {
		panelFixtureIdentity(t)
		db, script := gogameTestDB(t, gogameSQLStep{contains: "ON DUPLICATE KEY UPDATE", check: func(args []driver.NamedValue) {
			var record playerPanelRecord
			if len(args) != 12 || args[8].Value != false || args[9].Value != false || args[10].Value != true || json.Unmarshal([]byte(args[11].Value.(string)), &record) != nil {
				t.Fatal(args)
			}
			if record.BehaviorVersion != 5 || record.MahjongDiscards == nil || !record.MahjongDiscards.ClientDiscard || !record.MahjongDiscards.ServerDiscard || record.BlackjackActions != nil || record.SeatChoices != nil || record.ArrangementChoices != nil || string(record.Players) != "[]" || string(record.Events) != "null" || strings.Contains(args[11].Value.(string), "wallet") {
				t.Fatal("invalid projection or private data", record)
			}
		}})
		raw := fmt.Sprintf(`{"records":[{"id":"mahjong","agentId":7,"userId":42,"currency":"CNY","roomType":%d,"roomId":123,"inning":1,"endedAt":1800000000000,"behaviorVersion":5,"countersKnown":true,"mahjongDiscards":{"clientDiscard":true,"serverDiscard":true},"blackjackActions":{},"seatChoices":{},"arrangementChoices":{},"players":[{"wallet":999,"cards":[1,2]}],"events":[{"cards":[1,2]}]}]}`, room)
		if _, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelWrite", raw); err != nil || len(script.steps) != 0 {
			t.Fatal(err, script.steps)
		}
	}
}

func TestMahjongDiscardsRejectPokerFlagsUnknownEvidenceAndMissingFacts(t *testing.T) {
	for _, room := range []int{26, 28, 30} {
		for _, row := range []struct {
			known, voluntary, raised bool
			facts                    string
		}{
			{false, false, false, `{}`}, {true, true, false, `{}`}, {true, false, true, `{}`}, {true, false, false, `null`},
		} {
			panelFixtureIdentity(t)
			db, script := gogameTestDB(t)
			raw := fmt.Sprintf(`{"records":[{"id":"bad-mahjong","agentId":7,"userId":42,"currency":"CNY","roomType":%d,"roomId":123,"inning":1,"endedAt":1800000000000,"behaviorVersion":5,"countersKnown":%t,"voluntary":%t,"raised":%t,"mahjongDiscards":%s}]}`, room, row.known, row.voluntary, row.raised, row.facts)
			if _, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelWrite", raw); status.Code(err) != codes.InvalidArgument || len(script.steps) != 0 {
				t.Fatal("invalid facts accepted", room, row, err)
			}
		}
	}
}

func TestMahjongDiscardTotalsKeepIndependentSourcesAndExactAccountGameScope(t *testing.T) {
	for _, room := range []int{26, 28, 30} {
		panelFixtureIdentity(t)
		db, script := gogameTestDB(t, gogameSQLStep{contains: "COUNT(*)", columns: []string{"total", "known", "table_hands", "table_known", "client_discard", "server_discard"}, values: [][]driver.Value{{int64(4), int64(4), int64(2), int64(2), int64(3), int64(4)}}, check: func(args []driver.NamedValue) {
			if len(args) != 8 || args[0].Value != int64(123) || args[3].Value != int64(7) || args[4].Value != int64(42) || args[5].Value != "CNY" || args[6].Value != int64(room) {
				t.Fatal("scope lost", args)
			}
		}})
		value, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelRead", fmt.Sprintf(`{"agentId":7,"userId":42,"targetId":42,"currency":"CNY","roomId":123,"roomType":%d}`, room))
		if err != nil || len(script.steps) != 0 {
			t.Fatal(err, script.steps)
		}
		stats := value.(map[string]any)["statistics"].(map[string]any)
		if stats["clientDiscardHands"] != int64(3) || stats["serverDiscardHands"] != int64(4) || stats["voluntaryHands"] != int64(0) {
			t.Fatal(stats)
		}
		for _, field := range []string{"mahjongDiscards.clientDiscard", "mahjongDiscards.serverDiscard", "room_type=?", "ended_at<=?"} {
			if !strings.Contains(script.log[0], field) {
				t.Fatal("missing source or scope", script.log)
			}
		}
	}
}

func TestMahjongDiscardTotalsRejectInvalidRangesAndPokerActions(t *testing.T) {
	for _, room := range []int{26, 28, 30} {
		for _, row := range []struct{ client, server, voluntary, raised int64 }{
			{-1, 0, 0, 0}, {5, 0, 0, 0}, {1, -1, 0, 0}, {1, 5, 0, 0}, {1, 0, 1, 0}, {1, 0, 1, 1},
		} {
			panelFixtureIdentity(t)
			db, _ := gogameTestDB(t, gogameSQLStep{contains: "COUNT(*)", columns: []string{"total", "known", "client_discard", "server_discard", "voluntary", "raised"}, values: [][]driver.Value{{int64(4), int64(4), row.client, row.server, row.voluntary, row.raised}}})
			if _, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelRead", fmt.Sprintf(`{"agentId":7,"userId":42,"targetId":42,"currency":"CNY","roomId":123,"roomType":%d}`, room)); status.Code(err) != codes.Unavailable {
				t.Fatal("invalid totals accepted", room, row, err)
			}
		}
	}
}
