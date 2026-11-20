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

func TestSeatChoiceFactsPersistWithoutPrivateHistoryOrPokerRaiseFlags(t *testing.T) {
	for _, room := range []int{24, 32} {
		panelFixtureIdentity(t)
		db, script := gogameTestDB(t, gogameSQLStep{contains: "ON DUPLICATE KEY UPDATE", check: func(args []driver.NamedValue) {
			var record playerPanelRecord
			if len(args) != 12 || args[8].Value != false || args[9].Value != true || args[10].Value != true || json.Unmarshal([]byte(args[11].Value.(string)), &record) != nil {
				t.Fatal("invalid seat choice projection", args)
			}
			if record.BehaviorVersion != 3 || record.SeatChoices == nil || !record.SeatChoices.ManualBet || !record.SeatChoices.AboveMinimumBet || record.BlackjackActions != nil || string(record.Players) != "[]" || string(record.Events) != "null" {
				t.Fatal(record)
			}
			if record.SeatChoices.ManualGrab != (room == 24) || record.SeatChoices.PositiveGrab != (room == 24) || strings.Contains(args[11].Value.(string), "wallet") {
				t.Fatal("wrong game behavior or private data", record)
			}
		}})
		raw := fmt.Sprintf(`{"records":[{"id":"seat","agentId":7,"userId":42,"currency":"CNY","roomType":%d,"roomId":123,"inning":1,"endedAt":1800000000000,"behaviorVersion":3,"countersKnown":true,"voluntary":true,"raised":false,"seatChoices":{"betOpportunity":true,"manualBet":true,"aboveMinimumBet":true,"manualGrab":%t,"positiveGrab":%t},"players":[{"wallet":999,"cards":[1,2]}],"events":[{"cards":[1,2]}]}]}`, room, room == 24, room == 24)
		if _, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelWrite", raw); err != nil || len(script.steps) != 0 {
			t.Fatal(err, script.steps)
		}
	}
}

func TestSeatChoiceWriteRejectsInvalidOpportunityManualAndGrabRelationships(t *testing.T) {
	for _, row := range []struct {
		room                     int
		facts                    string
		voluntary, raised, known bool
	}{
		{24, `{"betOpportunity":false,"manualBet":true}`, true, false, true},
		{24, `{"betOpportunity":true,"manualBet":false,"aboveMinimumBet":true}`, false, false, true},
		{24, `{"betOpportunity":true,"manualBet":false,"manualGrab":false,"positiveGrab":true}`, false, false, true},
		{32, `{"betOpportunity":false}`, false, false, true},
		{32, `{"betOpportunity":true,"manualGrab":true}`, false, false, true},
		{24, `{"betOpportunity":true,"manualBet":true}`, false, false, true},
		{24, `{"betOpportunity":true,"manualBet":true}`, true, true, true},
		{24, `{"betOpportunity":true,"manualBet":true}`, true, false, false},
		{24, `null`, false, false, true},
	} {
		panelFixtureIdentity(t)
		db, script := gogameTestDB(t)
		raw := fmt.Sprintf(`{"records":[{"id":"bad-seat","agentId":7,"userId":42,"currency":"CNY","roomType":%d,"roomId":123,"inning":1,"endedAt":1800000000000,"behaviorVersion":3,"countersKnown":%t,"raised":%t,"voluntary":%t,"seatChoices":%s}]}`, row.room, row.known, row.raised, row.voluntary, row.facts)
		if _, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelWrite", raw); status.Code(err) != codes.InvalidArgument || len(script.steps) != 0 {
			t.Fatal("invalid seat facts accepted", row, err)
		}
	}
}

func TestSeatChoiceTotalsUseExactGameAndActualBetOpportunityDenominator(t *testing.T) {
	for _, room := range []int{24, 32} {
		panelFixtureIdentity(t)
		opportunities, grab, positive := int64(3), int64(2), int64(1)
		if room == 32 {
			opportunities, grab, positive = 4, 0, 0
		}
		db, script := gogameTestDB(t, gogameSQLStep{contains: "COUNT(*)", columns: []string{"total", "known", "voluntary", "raised", "table_hands", "table_known", "table_voluntary", "bet_opportunities", "above_minimum_bet", "manual_grab", "positive_grab"}, values: [][]driver.Value{{int64(4), int64(4), int64(2), int64(0), int64(2), int64(2), int64(1), opportunities, int64(1), grab, positive}}, check: func(args []driver.NamedValue) {
			if len(args) != 8 || args[0].Value != int64(123) || args[3].Value != int64(7) || args[4].Value != int64(42) || args[5].Value != "CNY" || args[6].Value != int64(room) {
				t.Fatal("scope lost", args)
			}
		}})
		value, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelRead", fmt.Sprintf(`{"agentId":7,"userId":42,"targetId":42,"currency":"CNY","roomId":123,"roomType":%d}`, room))
		if err != nil || len(script.steps) != 0 {
			t.Fatal(err, script.steps)
		}
		stats := value.(map[string]any)["statistics"].(map[string]any)
		if stats["betOpportunityHands"] != opportunities || stats["manualGrabHands"] != grab || stats["positiveGrabHands"] != positive || stats["aboveMinimumBetHands"] != int64(1) || stats["voluntaryHands"] != int64(2) {
			t.Fatal(stats)
		}
		for _, field := range []string{"seatChoices.betOpportunity", "seatChoices.aboveMinimumBet", "seatChoices.manualGrab", "seatChoices.positiveGrab"} {
			if !strings.Contains(script.log[0], field) {
				t.Fatal("missing factual source", script.log)
			}
		}
	}
}

func TestSeatChoiceTotalsRejectInvalidDenominatorsAndGameSpecificGrabCounts(t *testing.T) {
	for _, row := range []struct {
		room                                 int
		opportunities, above, grab, positive int64
	}{
		{24, 1, 0, 0, 0}, {24, 5, 0, 0, 0}, {24, 3, 3, 0, 0}, {24, 3, 0, 5, 0}, {24, 3, 0, 1, 2}, {32, 3, 0, 0, 0}, {32, 4, 0, 1, 0},
	} {
		panelFixtureIdentity(t)
		db, _ := gogameTestDB(t, gogameSQLStep{contains: "COUNT(*)", columns: []string{"total", "known", "voluntary", "bet_opportunities", "above_minimum_bet", "manual_grab", "positive_grab"}, values: [][]driver.Value{{int64(4), int64(4), int64(2), row.opportunities, row.above, row.grab, row.positive}}})
		if _, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelRead", fmt.Sprintf(`{"agentId":7,"userId":42,"targetId":42,"currency":"CNY","roomId":123,"roomType":%d}`, row.room)); status.Code(err) != codes.Unavailable {
			t.Fatal("invalid seat counts accepted", row, err)
		}
	}
}
