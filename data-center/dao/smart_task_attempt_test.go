package dao

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
	"time"
)

func attemptFixture() (map[string]any, map[string]any) {
	task := taskControlFixture()
	task["executionProtocol"] = int64(2)
	task["completedCount"] = int64(0)
	task["profit"] = int64(0)
	task["totalBetAmount"] = int64(0)
	task["activeAttempt"] = map[string]any{"roundId": "gg-original", "attemptId": "attempt-one", "inning": int64(1), "agent": int64(7), "gameId": int64(92058), "level": int64(1)}
	display := map[string]any{"userId": 42, "agentId": 7, "roomId": 58010000, "roomType": 58, "net": 100, "smartTask": map[string]any{"taskId": task["taskId"], "attemptId": "attempt-one", "inning": 1}}
	log, _ := json.Marshal(map[string]any{"completedDisplayReport": display})
	round := map[string]any{"roundId": "gg-original", "agent": int64(7), "gameId": int64(92058), "level": int64(1), "state": "SETTLED", "settlement": map[string]any{"status": "COMPLETED", "completedAt": time.Now().Unix(),
		"results": []any{map[string]any{"userId": int64(42), "code": int64(0), "currency": "101.00"}},
		"reports": []any{map[string]any{"userId": int64(42), "agentId": int64(7), "currency": "CNY", "complete": true, "bet": "1.00", "win": "2.00", "log": string(log)}}}}
	return task, round
}

func TestTaskFinalResultRequiresExactOriginalAttemptAndCompletedMoney(t *testing.T) {
	for _, kind := range []string{"confirmed", "not-settled", "different-round", "different-agent", "wrong-task", "wrong-net", "no-report"} {
		t.Run(kind, func(t *testing.T) {
			task, round := attemptFixture()
			settlement := paymentMap(round["settlement"])
			switch kind {
			case "not-settled":
				round["state"] = "RESERVED"
			case "different-round":
				round["roundId"] = "another"
			case "different-agent":
				round["agent"] = int64(8)
			case "wrong-task":
				paymentMap(task["activeAttempt"])["attemptId"] = "another"
			case "wrong-net":
				report := paymentMap(settlement["reports"].([]any)[0])
				report["win"] = "5.00"
			case "no-report":
				settlement["reports"] = []any{}
			}
			net, err := smartAttemptFinalNet(task, round)
			if kind == "confirmed" {
				if err != nil || net != 100 {
					t.Fatal(net, err)
				}
			} else if err == nil {
				t.Fatal("unverified financial result accepted")
			}
		})
	}
}

func TestBeginTaskAttemptCommitsReferenceBeforeWagerAndRepeatDoesNotStartAnother(t *testing.T) {
	for _, repeat := range []bool{false, true} {
		t.Run(string(rune('A'+boolInt(repeat))), func(t *testing.T) {
			task, _ := attemptFixture()
			attempt := paymentMap(task["activeAttempt"])
			if !repeat {
				delete(task, "activeAttempt")
			}
			raw, _ := json.Marshal(task)
			steps := []gogameSQLStep{{contains: "AND agent_id=? AND user_id=? FOR UPDATE", columns: []string{"payload"}, values: [][]driver.Value{{string(raw)}}}}
			if !repeat {
				steps = append(steps, gogameSQLStep{contains: "UPDATE gg_smart_task SET version=?,payload=?", check: func(args []driver.NamedValue) {
					var saved map[string]any
					if err := DecodeGogameJSON(args[1].Value.(string), &saved); err != nil {
						t.Fatal(err)
					}
					if paymentMap(saved["activeAttempt"])["roundId"] != "gg-original" || gogameNumber(saved["version"]) != 3 {
						t.Fatal(saved)
					}
				}})
			}
			db, script := gogameTestDB(t, steps...)
			_, err := (&DBDao{manager: db}).beginSmartTaskAttempt(context.Background(), smartTaskInput{AgentID: 7, UserID: 42, RoomType: 58, TaskID: "7:42:fixed-one", InstanceID: "owner-one", ExpectedVersion: 2, Attempt: attempt})
			if err != nil || len(script.steps) != 0 || script.log[len(script.log)-1] != "commit" {
				t.Fatal(err, script.log)
			}
		})
	}
}
func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func TestConfirmedTaskResolutionOnlyUpdatesStatisticsNeverWallet(t *testing.T) {
	client, _, _ := cashFixture(t)
	previous := redisDao
	redisDao = &RedisDao{Client: client}
	t.Cleanup(func() { redisDao = previous })
	task, round := attemptFixture()
	task["pendingInning"] = int64(1)
	raw, _ := json.Marshal(task)
	financial, _ := json.Marshal(round)
	client.HSet(context.Background(), "lottery_finance_rounds", "gg-original", string(financial))
	client.HSet(context.Background(), "player_42", "currency", 10000, "cash_revision", 7)
	db, script := gogameTestDB(t, gogameSQLStep{contains: "AND room_type=? FOR UPDATE", columns: []string{"payload"}, values: [][]driver.Value{{string(raw)}}},
		gogameSQLStep{contains: "UPDATE gg_smart_task SET active_key=NULL", check: func(args []driver.NamedValue) {
			var next map[string]any
			if err := DecodeGogameJSON(args[1].Value.(string), &next); err != nil {
				t.Fatal(err)
			}
			if next["status"] != "stopped" || gogameNumber(next["profit"]) != 100 || gogameNumber(next["completedCount"]) != 1 || next["activeAttempt"] != nil || next["pendingInning"] != nil {
				t.Fatal(next)
			}
		}})
	result, err := (&DBDao{manager: db}).resolveSmartTask(context.Background(), smartTaskInput{AgentID: 7, UserID: 42, RoomType: 58, TaskID: "7:42:fixed-one"})
	if err != nil || result == nil || len(script.steps) != 0 {
		t.Fatal(result, err, script.log)
	}
	if client.HGet(context.Background(), "player_42", "currency").Val() != "10000" || client.HGet(context.Background(), "player_42", "cash_revision").Val() != "7" {
		t.Fatal("task resolution touched funds")
	}
}

func TestUnconfirmedTaskCannotBeResolvedOrLoseItsAttempt(t *testing.T) {
	client, _, _ := cashFixture(t)
	previous := redisDao
	redisDao = &RedisDao{Client: client}
	t.Cleanup(func() { redisDao = previous })
	task, _ := attemptFixture()
	raw, _ := json.Marshal(task)
	db, script := gogameTestDB(t, gogameSQLStep{contains: "FOR UPDATE", columns: []string{"payload"}, values: [][]driver.Value{{string(raw)}}})
	_, err := (&DBDao{manager: db}).resolveSmartTask(context.Background(), smartTaskInput{AgentID: 7, UserID: 42, RoomType: 58, TaskID: "7:42:fixed-one"})
	if status.Code(err) != codes.FailedPrecondition || len(script.steps) != 0 || script.log[len(script.log)-1] != "rollback" {
		t.Fatal(err, script.log)
	}
}

func TestResolvedTaskRetryDoesNotCountOrWriteAgain(t *testing.T) {
	task, _ := attemptFixture()
	delete(task, "activeAttempt")
	task["status"] = "stopped"
	task["completedCount"] = int64(1)
	task["profit"] = int64(100)
	raw, _ := json.Marshal(task)
	db, script := gogameTestDB(t, gogameSQLStep{contains: "FOR UPDATE", columns: []string{"payload"}, values: [][]driver.Value{{string(raw)}}})
	response, err := (&DBDao{manager: db}).resolveSmartTask(context.Background(), smartTaskInput{AgentID: 7, UserID: 42, RoomType: 58, TaskID: "7:42:fixed-one"})
	if err != nil || gogameNumber(response.(map[string]any)["task"].(map[string]any)["completedCount"]) != 1 || len(script.steps) != 0 || script.log[len(script.log)-1] != "commit" {
		t.Fatal(response, err, script.log)
	}
}

func TestStoppedOrStaleOwnerCannotBeginAnotherExecution(t *testing.T) {
	for _, reason := range []string{"stop-requested", "stopped", "stale-version", "different-owner"} {
		t.Run(reason, func(t *testing.T) {
			task, _ := attemptFixture()
			attempt := paymentMap(task["activeAttempt"])
			delete(task, "activeAttempt")
			switch reason {
			case "stop-requested":
				task["control"] = map[string]any{"stopRequested": true}
			case "stopped":
				task["status"] = "stopped"
			case "stale-version":
				task["version"] = int64(20)
			case "different-owner":
				task["instanceId"] = "other"
			}
			raw, _ := json.Marshal(task)
			db, script := gogameTestDB(t, gogameSQLStep{contains: "FOR UPDATE", columns: []string{"payload"}, values: [][]driver.Value{{string(raw)}}})
			_, err := (&DBDao{manager: db}).beginSmartTaskAttempt(context.Background(), smartTaskInput{AgentID: 7, UserID: 42, RoomType: 58, TaskID: "7:42:fixed-one", InstanceID: "owner-one", ExpectedVersion: 2, Attempt: attempt})
			if status.Code(err) != codes.FailedPrecondition || len(script.steps) != 0 || script.log[len(script.log)-1] != "rollback" {
				t.Fatal(err, script.log)
			}
		})
	}
}
