package dao

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"testing"
	"time"
)

func refundedTaskFixture(voided bool) (map[string]any, map[string]any) {
	task, round := attemptFixture()
	attempt := paymentMap(task["activeAttempt"])
	attempt["areaId"] = int64(1)
	bet := map[string]any{"betId": "original-bet", "userId": int64(42), "currencyType": "CNY", "accepted": true, "code": int64(0), "areaId": "smart:attempt-one:1", "amount": "1.00", "status": "CANCELED"}
	round["bets"] = map[string]any{"original-bet": bet}
	round["state"] = "BETTING"
	delete(round, "settlement")
	round["cancelRequests"] = map[string]any{"cancel-one": map[string]any{"userId": int64(42), "currencyType": "CNY", "status": "COMPLETED", "success": true, "walletRefunded": true, "completedAt": int64(123), "currency": "100.00", "refundAmount": "1.00", "canceledBetIds": []any{"original-bet"}}}
	if voided {
		round["state"] = "VOIDED"
		bet["status"] = "ACTIVE"
		delete(round, "cancelRequests")
		round["settlement"] = map[string]any{"settlementId": "original-void", "mode": "VOID_REFUND", "status": "COMPLETED", "completedAt": time.Now().Unix(),
			"results": []any{map[string]any{"userId": int64(42), "code": int64(0), "currency": "100.00"}},
			"reports": []any{map[string]any{"agentId": int64(7), "userId": int64(42), "currency": "CNY", "roundID": "gg-original", "complete": true, "bet": "1.00", "win": "1.00"}}}
	}
	return task, round
}

func TestTaskRefundRequiresExactBetReferenceAndCompletedOriginalReturn(t *testing.T) {
	for _, voided := range []bool{false, true} {
		for _, failure := range []string{"", "no-bet", "different-attempt", "different-area", "different-player", "different-round", "not-completed", "partial", "missing-balance", "no-confirmation"} {
			task, round := refundedTaskFixture(voided)
			bet := paymentMap(paymentMap(round["bets"])["original-bet"])
			switch failure {
			case "no-bet":
				round["bets"] = map[string]any{}
			case "different-attempt":
				paymentMap(task["activeAttempt"])["attemptId"] = "another"
			case "different-area":
				bet["areaId"] = "1"
			case "different-player":
				bet["userId"] = int64(43)
			case "different-round":
				round["roundId"] = "another"
			case "not-completed":
				if voided {
					paymentMap(round["settlement"])["status"] = "CLAIMED"
				} else {
					paymentMap(paymentMap(round["cancelRequests"])["cancel-one"])["status"] = "CLAIMED"
				}
			case "partial":
				if voided {
					paymentMap(paymentMap(round["settlement"])["reports"].([]any)[0])["win"] = "0.50"
				} else {
					paymentMap(paymentMap(round["cancelRequests"])["cancel-one"])["refundAmount"] = "0.50"
				}
			case "missing-balance":
				if voided {
					delete(paymentMap(paymentMap(round["settlement"])["results"].([]any)[0]), "currency")
				} else {
					delete(paymentMap(paymentMap(round["cancelRequests"])["cancel-one"]), "currency")
				}
			case "no-confirmation":
				if voided {
					paymentMap(round["settlement"])["reports"] = []any{}
				} else {
					paymentMap(paymentMap(round["cancelRequests"])["cancel-one"])["walletRefunded"] = false
				}
			}
			confirmed, err := smartAttemptRefunded(task, round)
			if failure == "" {
				if err != nil || !confirmed {
					t.Fatal(voided, failure, confirmed, err)
				}
			} else if confirmed {
				t.Fatal("unconfirmed refund released task", voided, failure)
			}
		}
	}
}

func TestRefundResolutionStopsTaskWithoutAddingTurnoverOrTouchingWallet(t *testing.T) {
	for _, voided := range []bool{false, true} {
		client, _, _ := cashFixture(t)
		previous := redisDao
		redisDao = &RedisDao{Client: client}
		task, round := refundedTaskFixture(voided)
		task["pendingInning"] = int64(1)
		taskJSON, _ := json.Marshal(task)
		roundJSON, _ := json.Marshal(round)
		client.HSet(context.Background(), "lottery_finance_rounds", "gg-original", string(roundJSON))
		client.HSet(context.Background(), "player_42", "currency", 10000, "cash_revision", 7)
		db, script := gogameTestDB(t, gogameSQLStep{contains: "FOR UPDATE", columns: []string{"payload"}, values: [][]driver.Value{{string(taskJSON)}}}, gogameSQLStep{contains: "UPDATE gg_smart_task SET active_key=NULL", check: func(args []driver.NamedValue) {
			var saved map[string]any
			if err := DecodeGogameJSON(args[1].Value.(string), &saved); err != nil {
				t.Fatal(err)
			}
			if saved["status"] != "stopped" || saved["resolutionKind"] != "REFUNDED" || gogameNumber(saved["refundCount"]) != 1 || gogameNumber(saved["refundedAmount"]) != 100 || gogameNumber(saved["completedCount"]) != 0 || gogameNumber(saved["profit"]) != 0 || gogameNumber(saved["totalBetAmount"]) != 0 || saved["activeAttempt"] != nil || saved["pendingInning"] != nil {
				t.Fatal(saved)
			}
		}})
		result, err := (&DBDao{manager: db}).resolveSmartTask(context.Background(), smartTaskInput{AgentID: 7, UserID: 42, RoomType: 58, TaskID: "7:42:fixed-one"})
		redisDao = previous
		if err != nil || result == nil || len(script.steps) != 0 {
			t.Fatal(result, err, script.log)
		}
		if client.HGet(context.Background(), "player_42", "currency").Val() != "10000" || client.HGet(context.Background(), "player_42", "cash_revision").Val() != "7" {
			t.Fatal("refund task resolution mutated wallet")
		}
	}
}

func TestDuplicateCancelCoverageCannotProveARefundTwice(t *testing.T) {
	task, round := refundedTaskFixture(false)
	claims := paymentMap(round["cancelRequests"])
	claims["duplicated"] = claims["cancel-one"]
	if yes, err := smartAttemptRefunded(task, round); yes || err == nil {
		t.Fatal(yes, err)
	}
}

func TestRefundResolutionRetryPreservesRefundCountAndOriginalStatistics(t *testing.T) {
	task, _ := refundedTaskFixture(false)
	delete(task, "activeAttempt")
	task["status"] = "stopped"
	task["resolutionKind"] = "REFUNDED"
	task["refundCount"] = int64(1)
	task["refundedAmount"] = int64(100)
	raw, _ := json.Marshal(task)
	db, script := gogameTestDB(t, gogameSQLStep{contains: "FOR UPDATE", columns: []string{"payload"}, values: [][]driver.Value{{string(raw)}}})
	result, err := (&DBDao{manager: db}).resolveSmartTask(context.Background(), smartTaskInput{AgentID: 7, UserID: 42, RoomType: 58, TaskID: "7:42:fixed-one"})
	if err != nil {
		t.Fatal(err)
	}
	saved := paymentMap(result.(map[string]any)["task"])
	if gogameNumber(saved["refundCount"]) != 1 || gogameNumber(saved["refundedAmount"]) != 100 || gogameNumber(saved["completedCount"]) != 0 || len(script.steps) != 0 || script.log[len(script.log)-1] != "commit" {
		t.Fatal(saved, script.log)
	}
}
