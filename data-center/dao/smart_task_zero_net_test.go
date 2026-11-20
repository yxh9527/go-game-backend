package dao

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func zeroNetTaskDocument() (map[string]any, map[string]any, map[string]any) {
	task, document := smartDocumentFixture()
	document["win"] = document["bet"]
	var envelope map[string]any
	_ = DecodeGogameJSON(fmt.Sprint(document["log"]), &envelope)
	paymentMap(envelope["completedDisplayReport"])["net"] = int64(0)
	envelope["smartTaskSettlement"] = map[string]any{"version": int64(1), "state": "SETTLED", "mode": "NORMAL", "roundId": "gg-original", "settlementId": "original-settlement", "completedAt": time.Now().Unix() - 7200,
		"taskId": task["taskId"], "attemptId": "attempt-one", "inning": int64(1)}
	return task, document, envelope
}

func TestZeroNetTaskNeedsExactNormalSettlementProofInsteadOfInferringRefundOrDraw(t *testing.T) {
	for _, variant := range []string{"normal", "missing", "null", "refund", "voided", "wrong-round", "wrong-task", "wrong-attempt", "wrong-inning", "unknown-version", "missing-id", "missing-time", "future-time"} {
		t.Run(variant, func(t *testing.T) {
			task, document, envelope := zeroNetTaskDocument()
			proof := paymentMap(envelope["smartTaskSettlement"])
			switch variant {
			case "missing":
				delete(envelope, "smartTaskSettlement")
			case "null":
				envelope["smartTaskSettlement"] = nil
			case "refund":
				proof["mode"] = "VOID_REFUND"
			case "voided":
				proof["state"] = "VOIDED"
			case "wrong-round":
				proof["roundId"] = "another"
			case "wrong-task":
				proof["taskId"] = "another"
			case "wrong-attempt":
				proof["attemptId"] = "another"
			case "wrong-inning":
				proof["inning"] = int64(2)
			case "unknown-version":
				proof["version"] = int64(2)
			case "missing-id":
				delete(proof, "settlementId")
			case "missing-time":
				delete(proof, "completedAt")
			case "future-time":
				proof["completedAt"] = time.Now().Unix() + 10
			}
			raw, _ := json.Marshal(envelope)
			document["log"] = string(raw)
			net, err := smartTaskDocumentNet(task, document)
			if variant == "normal" {
				if err != nil || net != 0 {
					t.Fatal(net, err)
				}
			} else if err == nil {
				t.Fatal("ambiguous or mismatched zero result accepted", net)
			}
		})
	}
}

func TestPresentContradictoryProofCannotBeIgnoredOnNonzeroTaskResult(t *testing.T) {
	task, document, envelope := zeroNetTaskDocument()
	document["win"] = "2.00"
	paymentMap(envelope["completedDisplayReport"])["net"] = int64(100)
	paymentMap(envelope["smartTaskSettlement"])["mode"] = "VOID_REFUND"
	raw, _ := json.Marshal(envelope)
	document["log"] = string(raw)
	if _, err := smartTaskDocumentNet(task, document); err == nil {
		t.Fatal("contradictory financial proof ignored")
	}
}

func TestConfirmedZeroNetESResultStopsExactlyOnceAndCountsStakeWithoutWinLossOrMoneyMutation(t *testing.T) {
	task, document, envelope := zeroNetTaskDocument()
	raw, _ := json.Marshal(envelope)
	document["log"] = string(raw)
	task["pendingInning"] = int64(1)
	task["winCount"], task["loseCount"] = int64(2), int64(3)
	calls := 0
	dd, es, _ := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		id := strings.TrimPrefix(r.URL.Path, "/go_gp_settlement/_doc/")
		json.NewEncoder(w).Encode(map[string]any{"_id": id, "found": true, "_source": document})
	})
	client := RedisIns().Client
	client.HSet(context.Background(), "player_42", "currency", 10000, "cash_revision", 7)
	raw, _ = json.Marshal(task)
	var savedJSON string
	db, script := gogameTestDB(t, gogameSQLStep{contains: "FOR UPDATE", columns: []string{"payload"}, values: [][]driver.Value{{string(raw)}}}, gogameSQLStep{contains: "UPDATE gg_smart_task SET active_key=NULL", check: func(args []driver.NamedValue) {
		savedJSON = args[1].Value.(string)
		var saved map[string]any
		if err := DecodeGogameJSON(savedJSON, &saved); err != nil {
			t.Fatal(err)
		}
		if saved["status"] != "stopped" || saved["stopReason"] != "CONFIRMED_FINAL_RESULT" || gogameNumber(saved["completedCount"]) != 1 || gogameNumber(saved["totalBetAmount"]) != 100 || gogameNumber(saved["profit"]) != 0 || gogameNumber(saved["winCount"]) != 2 || gogameNumber(saved["loseCount"]) != 3 || saved["activeAttempt"] != nil || saved["pendingInning"] != nil {
			t.Fatal(saved)
		}
	}})
	dd.manager = db
	input := smartTaskInput{AgentID: 7, UserID: 42, RoomType: 58, TaskID: "7:42:fixed-one"}
	if _, err := dd.resolveSmartTask(context.Background(), input, es); err != nil || calls != 1 || len(script.steps) != 0 {
		t.Fatal(err, calls, script.log)
	}
	db, script = gogameTestDB(t, gogameSQLStep{contains: "FOR UPDATE", columns: []string{"payload"}, values: [][]driver.Value{{savedJSON}}})
	dd.manager = db
	if _, err := dd.resolveSmartTask(context.Background(), input, es); err != nil || calls != 1 || len(script.steps) != 0 {
		t.Fatal("repeat queried or counted again", err, calls, script.log)
	}
	if client.HGet(context.Background(), "player_42", "currency").Val() != "10000" || client.HGet(context.Background(), "player_42", "cash_revision").Val() != "7" || client.HExists(context.Background(), "lottery_finance_rounds", "gg-original").Val() {
		t.Fatal("zero-net resolution mutated funds or recreated snapshot")
	}
}
