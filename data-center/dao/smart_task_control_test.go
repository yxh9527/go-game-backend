package dao

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
)

func taskControlFixture() map[string]any {
	return map[string]any{"taskId": "7:42:fixed-one", "agentId": int64(7), "userId": int64(42), "roomId": int64(58010000), "roomType": int64(58), "version": int64(2), "instanceId": "owner-one", "status": "running", "currency": "CNY", "areaId": int64(1), "betAmount": int64(100), "stopCountLimit": int64(10), "takeProfit": int64(0), "stopLoss": int64(0), "strategy": "FIXED"}
}

func TestCrossInstanceTaskControlIsScopedDurableAndIdempotent(t *testing.T) {
	for _, kind := range []string{"request-stop", "repeat-stop", "delete-running", "delete-pending", "delete-stopped", "foreign-user"} {
		t.Run(kind, func(t *testing.T) {
			task := taskControlFixture()
			action := "stop"
			want := codes.OK
			switch kind {
			case "repeat-stop":
				task["control"] = map[string]any{"revision": int64(1), "stopRequested": true}
			case "delete-running":
				action = "delete"
				want = codes.FailedPrecondition
			case "delete-pending":
				action = "delete"
				task["status"] = "stopped"
				task["pendingInning"] = int64(1)
				want = codes.FailedPrecondition
			case "delete-stopped":
				action = "delete"
				task["status"] = "stopped"
			case "foreign-user":
				want = codes.NotFound
			}
			raw, _ := json.Marshal(task)
			query := gogameSQLStep{contains: "AND agent_id=? AND user_id=? AND room_type=? FOR UPDATE", columns: []string{"payload"}, values: [][]driver.Value{{string(raw)}}, check: func(args []driver.NamedValue) {
				if args[1].Value != int64(7) || args[2].Value != int64(42) || args[3].Value != int64(58) {
					t.Fatal(args)
				}
			}}
			if kind == "foreign-user" {
				query.values = nil
			}
			steps := []gogameSQLStep{query}
			if kind == "request-stop" || kind == "delete-stopped" {
				steps = append(steps, gogameSQLStep{contains: "UPDATE gg_smart_task SET version=?,payload=? WHERE id=?", check: func(args []driver.NamedValue) {
					var saved map[string]any
					if err := DecodeGogameJSON(args[1].Value.(string), &saved); err != nil {
						t.Fatal(err)
					}
					if kind == "request-stop" {
						if paymentMap(saved["control"])["stopRequested"] != true || gogameNumber(saved["version"]) != 2 {
							t.Fatal(saved)
						}
					} else if saved["deleted"] != true || gogameNumber(saved["version"]) != 3 {
						t.Fatal(saved)
					}
				}})
			}
			db, script := gogameTestDB(t, steps...)
			result, err := (&DBDao{manager: db}).controlSmartTask(context.Background(), smartTaskInput{AgentID: 7, UserID: 42, RoomType: 58, TaskID: "7:42:fixed-one", Action: action})
			if status.Code(err) != want {
				t.Fatal(err)
			}
			end := "rollback"
			if want == codes.OK {
				end = "commit"
				if result.(map[string]any)["task"] == nil {
					t.Fatal("no authoritative task returned")
				}
			}
			if len(script.steps) != 0 || script.log[len(script.log)-1] != end {
				t.Fatal(script.log)
			}
		})
	}
}

func TestOwnerSnapshotPreservesRemoteStopAndCannotEraseIt(t *testing.T) {
	original := taskControlFixture()
	original["control"] = map[string]any{"stopRequested": true, "revision": int64(4)}
	raw, _ := json.Marshal(original)
	snapshot := taskControlFixture()
	snapshot["version"] = int64(3)
	snapshot["pendingInning"] = int64(1)
	db, script := gogameTestDB(t, gogameSQLStep{contains: "SELECT version,payload FROM gg_smart_task", columns: []string{"version", "payload"}, values: [][]driver.Value{{int64(2), string(raw)}}},
		gogameSQLStep{contains: "UPDATE gg_smart_task SET active_key=?,version=?,payload=?", check: func(args []driver.NamedValue) {
			var saved map[string]any
			if err := DecodeGogameJSON(args[2].Value.(string), &saved); err != nil {
				t.Fatal(err)
			}
			if paymentMap(saved["control"])["stopRequested"] != true || gogameNumber(saved["pendingInning"]) != 1 {
				t.Fatal("owner snapshot erased the remote command or wager", saved)
			}
		}})
	request, _ := json.Marshal(map[string]any{"task": snapshot})
	result, err := (&DBDao{manager: db}).gogameSmartTask(context.Background(), "smartTaskSave", string(request))
	if err != nil {
		t.Fatal(err)
	}
	if paymentMap(result.(map[string]any)["task"].(map[string]any)["control"])["stopRequested"] != true {
		t.Fatal(result)
	}
	if len(script.steps) != 0 || script.log[len(script.log)-1] != "commit" {
		t.Fatal(script.log)
	}
}

func TestControlBatchesRejectRepeatedAndUnboundedIds(t *testing.T) {
	service := &DBDao{}
	for _, ids := range [][]string{{}, {"one", "one"}, make([]string, 101)} {
		if _, err := service.readSmartTaskControls(context.Background(), smartTaskInput{InstanceID: "owner", TaskIDs: ids}); status.Code(err) != codes.InvalidArgument {
			t.Fatal(err)
		}
	}
}

func TestControlBatchReturnsOnlyTheActualOwnerInstance(t *testing.T) {
	owner, other := taskControlFixture(), taskControlFixture()
	other["taskId"] = "7:43:fixed-two"
	other["instanceId"] = "owner-two"
	first, _ := json.Marshal(owner)
	second, _ := json.Marshal(other)
	db, script := gogameTestDB(t, gogameSQLStep{contains: "gg_smart_task", columns: []string{"id", "payload"}, values: [][]driver.Value{{owner["taskId"], string(first)}, {other["taskId"], string(second)}}})
	response, err := (&DBDao{manager: db}).readSmartTaskControls(context.Background(), smartTaskInput{InstanceID: "owner-one", TaskIDs: []string{"7:42:fixed-one", "7:43:fixed-two"}})
	if err != nil {
		t.Fatal(err)
	}
	rows := response.([]map[string]any)
	if len(rows) != 1 || rows[0]["taskId"] != "7:42:fixed-one" || len(script.steps) != 0 {
		t.Fatal(rows, script.log)
	}
}
