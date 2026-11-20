package dao

import (
	"context"
	"crypto/md5"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"strings"
	"testing"
	"time"
)

func smartDocumentFixture() (map[string]any, map[string]any) {
	task, round := attemptFixture()
	document := paymentMap(paymentMap(round["settlement"])["reports"].([]any)[0])
	document["roundID"], document["gameId"], document["level"], document["balance"] = "gg-original", int64(92058), int64(1), "101.00"
	var envelope map[string]any
	_ = DecodeGogameJSON(fmt.Sprint(document["log"]), &envelope)
	display := paymentMap(envelope["completedDisplayReport"])
	display["currency"], display["inning"], display["effectiveBet"] = "CNY", int64(1), int64(100)
	raw, _ := json.Marshal(envelope)
	document["log"] = string(raw)
	return task, document
}

func TestSmartTaskRetiredSnapshotIsNotReadFromOldArchiveOrUsedBeyondOneHour(t *testing.T) {
	for _, variant := range []string{"archive-only", "expired-settled", "expired-voided", "expired-reservation", "legacy-expired", "fresh", "old-active", "unknown-time"} {
		t.Run(variant, func(t *testing.T) {
			client, _, _ := cashFixture(t)
			previous := redisDao
			redisDao = &RedisDao{Client: client}
			t.Cleanup(func() { redisDao = previous })
			task, round := attemptFixture()
			at := time.Now().Unix()
			switch variant {
			case "expired-settled", "expired-voided":
				paymentMap(round["settlement"])["completedAt"] = at - 3600
				if variant == "expired-voided" {
					round["state"] = "VOIDED"
				}
			case "expired-reservation":
				round["state"] = "EXPIRED"
				round["reservation"] = map[string]any{"expiresAt": at - 3600}
			case "legacy-expired":
				delete(paymentMap(round["settlement"]), "completedAt")
				round["updatedAt"] = at - 3600
			case "fresh":
				paymentMap(round["settlement"])["completedAt"] = at - 3598
			case "old-active":
				round["state"] = "BETTING"
				round["updatedAt"] = at - 99999
			case "unknown-time":
				delete(paymentMap(round["settlement"]), "completedAt")
			}
			raw, _ := json.Marshal(round)
			client.HSet(context.Background(), "lottery_finance_rounds_archive", "gg-original", raw)
			if variant != "archive-only" {
				client.HSet(context.Background(), "lottery_finance_rounds", "gg-original", raw)
			}
			result, err := smartTaskCurrentSnapshot(context.Background(), task)
			if variant == "unknown-time" {
				if err == nil {
					t.Fatal("unconfirmed time accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if (variant == "fresh" || variant == "old-active") != (result != nil) {
				t.Fatal("wrong financial retention behavior", variant, result)
			}
			if client.HGet(context.Background(), "lottery_finance_rounds_archive", "gg-original").Val() != string(raw) {
				t.Fatal("read operation changed archive")
			}
		})
	}
}

func TestSmartTaskDocumentRequiresExactOriginalExecutionAndConfirmedMoney(t *testing.T) {
	for _, variant := range []string{"win", "loss", "incomplete", "round", "agent", "user", "game", "level", "currency", "stake", "balance", "fraction", "zero-net", "missing-log", "task", "attempt", "inning", "room", "room-type", "display-currency", "display-net", "display-stake"} {
		t.Run(variant, func(t *testing.T) {
			task, document := smartDocumentFixture()
			var envelope map[string]any
			_ = DecodeGogameJSON(fmt.Sprint(document["log"]), &envelope)
			display := paymentMap(envelope["completedDisplayReport"])
			reference := paymentMap(display["smartTask"])
			switch variant {
			case "loss":
				document["win"] = "0.00"
				display["net"] = int64(-100)
			case "incomplete":
				document["complete"] = false
			case "round":
				document["roundID"] = "another"
			case "agent":
				document["agentId"] = int64(8)
			case "user":
				document["userId"] = int64(43)
			case "game":
				document["gameId"] = int64(92059)
			case "level":
				document["level"] = int64(2)
			case "currency":
				document["currency"] = "USD"
			case "stake":
				document["bet"] = "2.00"
			case "balance":
				delete(document, "balance")
			case "fraction":
				document["win"] = "2.001"
			case "zero-net":
				document["win"] = "1.00"
				display["net"] = int64(0)
			case "task":
				reference["taskId"] = "another"
			case "attempt":
				reference["attemptId"] = "another"
			case "inning":
				display["inning"] = int64(2)
			case "room":
				display["roomId"] = int64(58010001)
			case "room-type":
				display["roomType"] = int64(59)
			case "display-currency":
				display["currency"] = "USD"
			case "display-net":
				display["net"] = int64(101)
			case "display-stake":
				display["effectiveBet"] = int64(200)
			}
			raw, _ := json.Marshal(envelope)
			document["log"] = string(raw)
			if variant == "missing-log" {
				delete(document, "log")
			}
			net, err := smartTaskDocumentNet(task, document)
			if variant == "win" || variant == "loss" {
				want := int64(100)
				if variant == "loss" {
					want = -100
				}
				if err != nil || net != want {
					t.Fatal(net, err)
				}
			} else if err == nil {
				t.Fatal("unverified historical result accepted", variant, net)
			}
		})
	}
}

func TestSmartTaskResolvesFromExactSettlementAfterRedisSnapshotRemovalWithoutChangingWallet(t *testing.T) {
	for _, expired := range []bool{false, true} {
		t.Run(fmt.Sprint(expired), func(t *testing.T) {
			task, document := smartDocumentFixture()
			task["pendingInning"] = int64(1)
			id := fmt.Sprintf("%x", md5.Sum([]byte("7|42|gg-original")))
			calls := 0
			dd, es, _ := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "GET" || r.URL.Path != "/go_gp_settlement/_doc/"+id {
					t.Error("not an exact original settlement read", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{"_id": id, "found": true, "_source": document})
			})
			client := RedisIns().Client
			if expired {
				_, round := attemptFixture()
				paymentMap(round["settlement"])["completedAt"] = time.Now().Unix() - 3600
				raw, _ := json.Marshal(round)
				client.HSet(context.Background(), "lottery_finance_rounds", "gg-original", raw)
			}
			client.HSet(context.Background(), "player_42", "currency", 10000, "cash_revision", 7)
			raw, _ := json.Marshal(task)
			db, script := gogameTestDB(t, gogameSQLStep{contains: "FOR UPDATE", columns: []string{"payload"}, values: [][]driver.Value{{string(raw)}}}, gogameSQLStep{contains: "UPDATE gg_smart_task SET active_key=NULL", check: func(args []driver.NamedValue) {
				var saved map[string]any
				if err := DecodeGogameJSON(args[1].Value.(string), &saved); err != nil {
					t.Fatal(err)
				}
				if saved["status"] != "stopped" || gogameNumber(saved["profit"]) != 100 || gogameNumber(saved["completedCount"]) != 1 || saved["activeAttempt"] != nil || saved["pendingInning"] != nil {
					t.Fatal(saved)
				}
			}})
			dd.manager = db
			payload, _ := json.Marshal(smartTaskInput{AgentID: 7, UserID: 42, RoomType: 58, TaskID: "7:42:fixed-one"})
			if _, err := dd.ResolveSmartTask(context.Background(), es, string(payload)); err != nil || calls != 1 || len(script.steps) != 0 {
				t.Fatal(err, calls, script.log)
			}
			if client.HGet(context.Background(), "player_42", "currency").Val() != "10000" || client.HGet(context.Background(), "player_42", "cash_revision").Val() != "7" {
				t.Fatal("result resolution changed money")
			}
			if !expired && client.HExists(context.Background(), "lottery_finance_rounds", "gg-original").Val() {
				t.Fatal("retired snapshot was recreated")
			}
		})
	}
}

func TestSmartTaskMissingInvalidOrFailedESResultKeepsOriginalResponsibility(t *testing.T) {
	for _, variant := range []string{"missing", "failed", "wrong-document", "refund-ambiguous"} {
		t.Run(variant, func(t *testing.T) {
			task, document := smartDocumentFixture()
			dd, es, _ := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				id := strings.TrimPrefix(r.URL.Path, "/go_gp_settlement/_doc/")
				if variant == "missing" {
					w.WriteHeader(404)
					fmt.Fprint(w, `{"found":false}`)
					return
				}
				if variant == "failed" {
					w.WriteHeader(503)
					fmt.Fprint(w, `{"error":"unavailable"}`)
					return
				}
				if variant == "wrong-document" {
					document["userId"] = int64(43)
				}
				if variant == "refund-ambiguous" {
					document["win"] = "1.00"
				}
				json.NewEncoder(w).Encode(map[string]any{"_id": id, "found": true, "_source": document})
			})
			raw, _ := json.Marshal(task)
			db, script := gogameTestDB(t, gogameSQLStep{contains: "FOR UPDATE", columns: []string{"payload"}, values: [][]driver.Value{{string(raw)}}})
			dd.manager = db
			_, err := dd.resolveSmartTask(context.Background(), smartTaskInput{AgentID: 7, UserID: 42, RoomType: 58, TaskID: "7:42:fixed-one"}, es)
			if err == nil || script.log[len(script.log)-1] != "rollback" || len(script.steps) != 0 {
				t.Fatal(err, script.log)
			}
		})
	}
}

func TestSmartTaskRedisFailureCannotFallBackToArchiveOrES(t *testing.T) {
	client, _, _ := cashFixture(t)
	previous := redisDao
	redisDao = &RedisDao{Client: client}
	t.Cleanup(func() { redisDao = previous })
	client.Set(context.Background(), "lottery_finance_rounds", "wrong-type", 0)
	task, round := attemptFixture()
	raw, _ := json.Marshal(round)
	client.HSet(context.Background(), "lottery_finance_rounds_archive", "gg-original", raw)
	if _, err := smartTaskCurrentSnapshot(context.Background(), task); status.Code(err) != codes.Unavailable {
		t.Fatal(err)
	}
	if client.Get(context.Background(), "lottery_finance_rounds").Val() != "wrong-type" {
		t.Fatal("query changed original key")
	}
	if _, err := client.HGet(context.Background(), "lottery_finance_rounds_archive", "gg-original").Result(); err == redis.Nil {
		t.Fatal("query deleted original archive")
	}
}
