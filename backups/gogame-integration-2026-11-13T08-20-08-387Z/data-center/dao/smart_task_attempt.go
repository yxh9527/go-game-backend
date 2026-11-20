package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"time"
)

func validSmartAttempt(input smartTaskInput) bool {
	attempt := input.Attempt
	round, ok := attempt["roundId"].(string)
	if !ok || !cashID.MatchString(round) {
		return false
	}
	id, ok := attempt["attemptId"].(string)
	if !ok || !cashID.MatchString(id) {
		return false
	}
	return gogameNumber(attempt["agent"]) == input.AgentID && gogameNumber(attempt["gameId"]) > 0 && gogameNumber(attempt["level"]) >= 0 && gogameNumber(attempt["inning"]) > 0
}

func (dd *DBDao) beginSmartTaskAttempt(ctx context.Context, input smartTaskInput) (any, error) {
	if input.AgentID <= 0 || input.UserID <= 0 || input.RoomType != 58 && input.RoomType != 59 || input.TaskID == "" || input.InstanceID == "" || input.ExpectedVersion < 1 || !validSmartAttempt(input) {
		return nil, status.Error(codes.InvalidArgument, "invalid task attempt")
	}
	var task map[string]any
	err := dd.manager.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rows, e := gogameRows(ctx, tx, "SELECT payload FROM gg_smart_task WHERE id=? AND agent_id=? AND user_id=? FOR UPDATE", input.TaskID, input.AgentID, input.UserID)
		if e != nil {
			return e
		}
		if len(rows) != 1 {
			return status.Error(codes.NotFound, "task not found")
		}
		if e = DecodeGogameJSON(rows[0]["payload"].(string), &task); e != nil {
			return e
		}
		if task["taskId"] != input.TaskID || gogameNumber(task["agentId"]) != input.AgentID || gogameNumber(task["userId"]) != input.UserID || gogameNumber(task["roomType"]) != input.RoomType {
			return paymentInvalid("task execution identity mismatch")
		}
		if input.Attempt["areaId"] != nil && gogameNumber(input.Attempt["areaId"]) != gogameNumber(task["areaId"]) {
			return paymentInvalid("task execution betting area mismatch")
		}
		if task["instanceId"] != input.InstanceID || gogameNumber(task["executionProtocol"]) != 2 || task["deleted"] == true || task["status"] != "running" || task["stopping"] == true || paymentMap(task["control"])["stopRequested"] == true {
			return paymentInvalid("task does not allow another wager")
		}
		if existing := paymentMap(task["activeAttempt"]); len(existing) > 0 {
			if paymentHash(existing) == paymentHash(input.Attempt) {
				return nil
			}
			return paymentInvalid("previous wager responsibility remains open")
		}
		if gogameNumber(task["version"]) != input.ExpectedVersion || gogameNumber(task["pendingInning"]) > 0 || gogameNumber(task["completedCount"]) >= gogameNumber(task["stopCountLimit"]) {
			return paymentInvalid("task version or count changed")
		}
		if gogameNumber(input.Attempt["inning"]) <= gogameNumber(task["lastSettledInning"]) {
			return paymentInvalid("task inning already settled")
		}
		task["activeAttempt"] = input.Attempt
		task["version"] = input.ExpectedVersion + 1
		raw, e := json.Marshal(task)
		if e != nil {
			return e
		}
		_, e = tx.Statement.ConnPool.ExecContext(ctx, "UPDATE gg_smart_task SET version=?,payload=? WHERE id=?", input.ExpectedVersion+1, string(raw), input.TaskID)
		return e
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"task": task}, nil
}

// This reads the already existing final financial result. It never submits a
// bet, refund, settlement, balance mutation or reconstructed game outcome.
func smartAttemptFinalNet(task, round map[string]any) (int64, error) {
	attempt := paymentMap(task["activeAttempt"])
	if len(attempt) == 0 {
		return 0, paymentInvalid("task has no authoritative attempt identity")
	}
	if round["roundId"] != attempt["roundId"] || gogameNumber(round["agent"]) != gogameNumber(task["agentId"]) || gogameNumber(round["gameId"]) != gogameNumber(attempt["gameId"]) || gogameNumber(round["level"]) != gogameNumber(attempt["level"]) {
		return 0, paymentInvalid("task financial identity mismatch")
	}
	settlement := paymentMap(round["settlement"])
	if round["state"] != "SETTLED" || settlement["status"] != "COMPLETED" {
		return 0, paymentInvalid("original wager is not finally settled")
	}
	reports, _ := settlement["reports"].([]any)
	for _, row := range reports {
		report := paymentMap(row)
		if gogameNumber(report["userId"]) != gogameNumber(task["userId"]) || report["currency"] != task["currency"] {
			continue
		}
		if gogameNumber(report["agentId"]) != gogameNumber(task["agentId"]) || report["complete"] != true {
			return 0, paymentInvalid("financial report is not a confirmed task result")
		}
		var envelope map[string]any
		if DecodeGogameJSON(fmt.Sprint(report["log"]), &envelope) != nil {
			continue
		}
		display := paymentMap(envelope["completedDisplayReport"])
		reference := paymentMap(display["smartTask"])
		if reference["taskId"] != task["taskId"] || reference["attemptId"] != attempt["attemptId"] || gogameNumber(reference["inning"]) != gogameNumber(attempt["inning"]) {
			continue
		}
		if gogameNumber(display["userId"]) != gogameNumber(task["userId"]) || gogameNumber(display["agentId"]) != gogameNumber(task["agentId"]) || gogameNumber(display["roomId"]) != gogameNumber(task["roomId"]) || gogameNumber(display["roomType"]) != gogameNumber(task["roomType"]) {
			return 0, paymentInvalid("settled display task identity mismatch")
		}
		bet, e := decimal.NewFromString(fmt.Sprint(report["bet"]))
		if e != nil {
			return 0, paymentInvalid("invalid task stake")
		}
		win, e := decimal.NewFromString(fmt.Sprint(report["win"]))
		if e != nil {
			return 0, paymentInvalid("invalid task payout")
		}
		stake, net := bet.Shift(2), win.Sub(bet).Shift(2)
		if !stake.IsInteger() || !stake.Equal(decimal.NewFromInt(gogameNumber(task["betAmount"]))) || !net.IsInteger() || net.Abs().GreaterThan(decimal.NewFromInt(cashMax)) || !net.Equal(decimal.NewFromInt(gogameNumber(display["net"]))) {
			return 0, paymentInvalid("final task money does not match")
		}
		results, _ := settlement["results"].([]any)
		confirmed := false
		for _, item := range results {
			value := paymentMap(item)
			balance, parseErr := decimal.NewFromString(fmt.Sprint(value["currency"]))
			if gogameNumber(value["userId"]) == gogameNumber(task["userId"]) && value["code"] != nil && gogameNumber(value["code"]) == 0 && parseErr == nil && !balance.IsNegative() && balance.Shift(2).IsInteger() {
				confirmed = true
				break
			}
		}
		if !confirmed {
			return 0, paymentInvalid("task settlement account is not confirmed")
		}
		return net.IntPart(), nil
	}
	return 0, paymentInvalid("original task final report is not available")
}

func (dd *DBDao) resolveSmartTask(ctx context.Context, input smartTaskInput) (any, error) {
	if input.AgentID <= 0 || input.UserID <= 0 || input.RoomType != 58 && input.RoomType != 59 || input.TaskID == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid task resolution")
	}
	var task map[string]any
	err := dd.manager.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rows, e := gogameRows(ctx, tx, "SELECT payload FROM gg_smart_task WHERE id=? AND agent_id=? AND user_id=? AND room_type=? FOR UPDATE", input.TaskID, input.AgentID, input.UserID, input.RoomType)
		if e != nil {
			return e
		}
		if len(rows) != 1 {
			return status.Error(codes.NotFound, "task not found")
		}
		if e = DecodeGogameJSON(rows[0]["payload"].(string), &task); e != nil {
			return e
		}
		if task["taskId"] != input.TaskID || gogameNumber(task["agentId"]) != input.AgentID || gogameNumber(task["userId"]) != input.UserID || gogameNumber(task["roomType"]) != input.RoomType {
			return paymentInvalid("task resolution identity mismatch")
		}
		if task["status"] == "stopped" && gogameNumber(task["pendingInning"]) == 0 && len(paymentMap(task["activeAttempt"])) == 0 {
			return nil
		}
		if gogameNumber(task["executionProtocol"]) != 2 {
			return paymentInvalid("legacy task requires original owner confirmation")
		}
		attempt := paymentMap(task["activeAttempt"])
		if len(attempt) > 0 {
			if gogameNumber(task["lastSettledInning"]) >= gogameNumber(attempt["inning"]) || gogameNumber(task["completedCount"]) >= gogameNumber(task["stopCountLimit"]) {
				return paymentInvalid("task result has already been counted or exceeds its limit")
			}
			rd := RedisIns()
			if rd == nil || rd.Client == nil {
				return status.Error(codes.Unavailable, "financial snapshot query unavailable")
			}
			raw, e := rd.Client.HGet(ctx, "lottery_finance_rounds", fmt.Sprint(attempt["roundId"])).Result()
			if e != nil {
				raw, e = rd.Client.HGet(ctx, "lottery_finance_rounds_archive", fmt.Sprint(attempt["roundId"])).Result()
			}
			if e != nil {
				return paymentInvalid("original financial result is unavailable; responsibility retained")
			}
			var round map[string]any
			if e = DecodeGogameJSON(raw, &round); e != nil {
				return e
			}
			refunded, e := smartAttemptRefunded(task, round)
			if e != nil {
				return e
			}
			if refunded {
				task["lastRefundedInning"] = attempt["inning"]
				task["refundCount"] = gogameNumber(task["refundCount"]) + 1
				amount := gogameNumber(task["refundedAmount"]) + gogameNumber(task["betAmount"])
				if amount < 0 || amount > cashMax {
					return paymentInvalid("task refunded amount exceeds safe range")
				}
				task["refundedAmount"] = amount
				task["resolutionKind"] = "REFUNDED"
			} else {
				net, e := smartAttemptFinalNet(task, round)
				if e != nil {
					return e
				}
				profit := gogameNumber(task["profit"]) + net
				total := gogameNumber(task["totalBetAmount"]) + gogameNumber(task["betAmount"])
				if profit > cashMax || profit < -cashMax || total < 0 || total > cashMax {
					return paymentInvalid("task statistics exceed safe range")
				}
				task["profit"] = profit
				task["totalBetAmount"] = total
				task["completedCount"] = gogameNumber(task["completedCount"]) + 1
				if net > 0 {
					task["winCount"] = gogameNumber(task["winCount"]) + 1
				} else if net < 0 {
					task["loseCount"] = gogameNumber(task["loseCount"]) + 1
				}
				trend, _ := task["profitTrend"].([]any)
				task["profitTrend"] = append(trend, profit)
				task["lastSettledInning"] = attempt["inning"]
			}
		} else if gogameNumber(task["pendingInning"]) > 0 {
			return paymentInvalid("task pending wager has no trusted reference")
		}
		task["status"] = "stopped"
		task["stopping"] = true
		task["stopReason"] = "CONFIRMED_FINAL_RESULT"
		if task["resolutionKind"] == "REFUNDED" {
			task["stopReason"] = "CONFIRMED_ORIGINAL_REFUND"
		}
		task["resolvedAt"] = time.Now().UnixMilli()
		task["version"] = gogameNumber(task["version"]) + 1
		delete(task, "activeAttempt")
		delete(task, "pendingInning")
		raw, e := json.Marshal(task)
		if e != nil {
			return e
		}
		_, e = tx.Statement.ConnPool.ExecContext(ctx, "UPDATE gg_smart_task SET active_key=NULL,version=?,payload=? WHERE id=?", gogameNumber(task["version"]), string(raw), input.TaskID)
		return e
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"task": task}, nil
}
