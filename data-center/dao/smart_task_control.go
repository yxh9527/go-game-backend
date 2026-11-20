package dao

import (
	"app/tables/gogame"
	"context"
	"encoding/json"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"time"
)

func (dd *DBDao) controlSmartTask(ctx context.Context, input smartTaskInput) (any, error) {
	if input.AgentID <= 0 || input.UserID <= 0 || input.RoomType != 58 && input.RoomType != 59 || len(input.TaskID) < 1 || len(input.TaskID) > 180 || input.Action != "stop" && input.Action != "delete" {
		return nil, status.Error(codes.InvalidArgument, "invalid task control")
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
			return status.Error(codes.PermissionDenied, "task identity mismatch")
		}
		if task["deleted"] == true {
			if input.Action == "delete" {
				return nil
			}
			return status.Error(codes.NotFound, "task not found")
		}
		if input.Action == "delete" {
			if task["status"] != "stopped" || gogameNumber(task["pendingInning"]) > 0 || len(paymentMap(task["activeAttempt"])) > 0 {
				return paymentInvalid("wait for the owner to stop and settle accepted wagers")
			}
			task["deleted"] = true
			task["version"] = gogameNumber(task["version"]) + 1
		} else {
			if task["status"] == "stopped" {
				return nil
			}
			control := paymentMap(task["control"])
			if control["stopRequested"] == true {
				return nil
			}
			task["control"] = map[string]any{"stopRequested": true, "revision": gogameNumber(control["revision"]) + 1, "requestedAt": time.Now().UnixMilli()}
		}
		raw, e := json.Marshal(task)
		if e != nil {
			return e
		}
		_, e = tx.Statement.ConnPool.ExecContext(ctx, "UPDATE gg_smart_task SET version=?,payload=? WHERE id=?", gogameNumber(task["version"]), string(raw), input.TaskID)
		return e
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"task": task}, nil
}

func (dd *DBDao) readSmartTaskControls(ctx context.Context, input smartTaskInput) (any, error) {
	if input.InstanceID == "" || len(input.InstanceID) > 128 || len(input.TaskIDs) < 1 || len(input.TaskIDs) > 100 {
		return nil, status.Error(codes.InvalidArgument, "invalid task control batch")
	}
	seen := map[string]bool{}
	for _, id := range input.TaskIDs {
		if id == "" || len(id) > 180 || seen[id] {
			return nil, status.Error(codes.InvalidArgument, "invalid task ids")
		}
		seen[id] = true
	}
	var rows []gogame.SmartTask
	if err := dd.manager.WithContext(ctx).Where("id IN ?", input.TaskIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := []map[string]any{}
	for _, row := range rows {
		var task map[string]any
		if err := DecodeGogameJSON(row.Payload, &task); err != nil {
			return nil, err
		}
		if task["instanceId"] == input.InstanceID {
			result = append(result, task)
		}
	}
	return result, nil
}
