package dao

import (
	"context"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"testing"
)

func TestNativePlayerTotalsFilterExactGameAndAccountWithoutReadingDetails(t *testing.T) {
	for _, roomType := range []int{24, 25, 26, 27, 28, 30, 31, 32} {
		t.Run(fmt.Sprint(roomType), func(t *testing.T) {
			panelFixtureIdentity(t)
			db, script := gogameTestDB(t, gogameSQLStep{contains: "count(*)", columns: []string{"count"}, values: [][]driver.Value{{int64(25001)}}, check: func(args []driver.NamedValue) {
				if len(args) != 5 || args[0].Value != int64(7) || args[1].Value != int64(42) || args[2].Value != "CNY" || args[3].Value != int64(roomType) {
					t.Fatal("statistics scope changed", args)
				}
			}})
			value, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelRead", fmt.Sprintf(`{"agentId":7,"userId":42,"targetId":42,"currency":"CNY","roomId":123,"roomType":%d}`, roomType))
			if err != nil {
				t.Fatal(err)
			}
			result := value.(map[string]any)
			if result["statistics"].(map[string]any)["totalHands"] != int64(25001) || result["complete"] != false || len(result["records"].([]any)) != 0 || len(script.steps) != 0 {
				t.Fatal(result, script.steps)
			}
			if !strings.Contains(script.log[0], "room_type=?") || !strings.Contains(script.log[0], "ended_at<=?") {
				t.Fatal("missing game or future-time filter", script.log)
			}
		})
	}
}

func TestNativePlayerCountersStripUnneededPrivateDetailsAndSeparateSameIDsAcrossGames(t *testing.T) {
	panelFixtureIdentity(t)
	seen := map[string]bool{}
	db, script := gogameTestDB(t, gogameSQLStep{contains: "ON DUPLICATE KEY UPDATE", check: func(args []driver.NamedValue) {
		if len(args) != 24 {
			t.Fatal(args)
		}
		for offset, roomType := range []int{24, 25} {
			row := args[offset*12 : (offset+1)*12]
			id := row[0].Value.(string)
			if seen[id] || row[4].Value != int64(roomType) {
				t.Fatal("different games collided", row)
			}
			seen[id] = true
			var payload playerPanelRecord
			if json.Unmarshal([]byte(row[11].Value.(string)), &payload) != nil || string(payload.Players) != "[]" || string(payload.CommonCards) != "null" || string(payload.Events) != "null" || payload.CountersKnown || payload.Raised || payload.Voluntary {
				t.Fatal("unrelated private details or Holdem ratios persisted", row)
			}
		}
	}})
	records := []playerPanelRecord{}
	for _, roomType := range []int{24, 25} {
		records = append(records, playerPanelRecord{ID: "same-id", AgentID: 7, UserID: 42, Currency: "CNY", RoomType: roomType, RoomID: 123, Inning: 1, EndedAt: 1800000000000, Net: 20, EffectiveBet: 20,
			Players: json.RawMessage(`[{"uid":42,"handCards":[1,2],"wallet":9999}]`), CommonCards: json.RawMessage(`[3,4]`), Events: json.RawMessage(`[{"cards":[5,6]}]`), CountersKnown: true, Raised: true, Voluntary: true})
	}
	raw, _ := json.Marshal(map[string]any{"records": records})
	if _, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelWrite", string(raw)); err != nil || len(script.steps) != 0 {
		t.Fatal(err, script.steps)
	}
}

func TestLegacyHoldemStatisticsKeepTheirOriginalUniqueRecordIdentity(t *testing.T) {
	panelFixtureIdentity(t)
	db, _ := gogameTestDB(t, gogameSQLStep{contains: "ON DUPLICATE KEY UPDATE", check: func(args []driver.NamedValue) {
		want := fmt.Sprintf("%x", sha256.Sum256([]byte("7:42:CNY:old-hand")))
		if args[0].Value != want || args[4].Value != int64(19) {
			t.Fatal("legacy retry would insert a second hand", args)
		}
	}})
	_, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelWrite", `{"records":[{"id":"old-hand","agentId":7,"userId":42,"currency":"CNY","roomId":123,"inning":1,"endedAt":1800000000000,"players":[{"uid":42}]}]}`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNativeStatisticsRejectOtherGamesAndDoNotProvideAnAlternateDetailSource(t *testing.T) {
	panelFixtureIdentity(t)
	db, script := gogameTestDB(t)
	for _, roomType := range []int{-1, 58, 61, 101} {
		_, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelRead", fmt.Sprintf(`{"agentId":7,"userId":42,"targetId":42,"currency":"CNY","roomId":123,"roomType":%d}`, roomType))
		if status.Code(err) != codes.InvalidArgument {
			t.Fatal(roomType, err)
		}
	}
	_, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelRecords", `{"agentId":7,"userId":42,"targetId":42,"currency":"CNY","roomId":123,"roomType":25,"page":1,"size":10}`)
	if status.Code(err) != codes.InvalidArgument || len(script.log) != 0 {
		t.Fatal(err, script.log)
	}
}

func TestNativeStatisticsSourceFailureDoesNotReturnZero(t *testing.T) {
	panelFixtureIdentity(t)
	db, _ := gogameTestDB(t, gogameSQLStep{contains: "count(*)", err: errors.New("mysql unavailable")})
	if _, err := (&DBDao{db: db}).GogameCall(context.Background(), "playerPanelRead", `{"agentId":7,"userId":42,"targetId":42,"currency":"CNY","roomId":123,"roomType":25}`); err == nil {
		t.Fatal("failed query became an empty successful statistic")
	}
}
