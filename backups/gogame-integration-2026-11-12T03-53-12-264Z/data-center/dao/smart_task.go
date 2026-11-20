package dao

import (
	"app/tables/gogame"
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type smartTaskInput struct {
	Task     map[string]any `json:"task"`
	AgentID  int64          `json:"agentId"`
	UserID   int64          `json:"userId"`
	RoomType int64          `json:"roomType"`
	Currency string         `json:"currency"`
}

func (dd *DBDao) gogameSmartTask(ctx context.Context, operation, payload string) (any, error) {
	var input smartTaskInput
	if err := DecodeGogameJSON(payload, &input); err != nil {
		return nil, err
	}
	if operation == "smartTaskList" {
		if input.AgentID <= 0 || input.UserID <= 0 || input.RoomType != 58 && input.RoomType != 59 {
			return nil, status.Error(codes.InvalidArgument, "invalid task owner")
		}
		var rows []gogame.SmartTask
		if err := dd.manager.WithContext(ctx).Where("agent_id=? AND user_id=? AND room_type=?", input.AgentID, input.UserID, input.RoomType).Limit(1001).Find(&rows).Error; err != nil {
			return nil, err
		}
		if len(rows) > 1000 {
			return nil, status.Error(codes.ResourceExhausted, "task history exceeds query limit")
		}
		tasks := []map[string]any{}
		for _, row := range rows {
			var task map[string]any
			if err := DecodeGogameJSON(row.Payload, &task); err != nil {
				return nil, err
			}
			if task["deleted"] != true {
				tasks = append(tasks, task)
			}
		}
		return tasks, nil
	}
	task := input.Task
	id, _ := task["taskId"].(string)
	agent, uid, room, roomType, version := gogameNumber(task["agentId"]), gogameNumber(task["userId"]), gogameNumber(task["roomId"]), gogameNumber(task["roomType"]), gogameNumber(task["version"])
	if len(id) < 1 || len(id) > 180 || agent <= 0 || uid <= 0 || room <= 0 || version < 1 || roomType != 58 && roomType != 59 {
		return nil, status.Error(codes.InvalidArgument, "invalid task identity")
	}
	var active any
	if task["status"] != "stopped" {
		active = fmt.Sprintf("%d:%d:%d", agent, uid, room)
	}
	raw, err := json.Marshal(task)
	if err != nil || len(raw) > 100000 {
		return nil, status.Error(codes.InvalidArgument, "task exceeds storage limit")
	}
	err = dd.manager.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rows, e := gogameRows(ctx, tx, "SELECT version,payload FROM gg_smart_task WHERE id=? FOR UPDATE", id)
		if e != nil {
			return e
		}
		if len(rows) == 0 {
			_, e = tx.Statement.ConnPool.ExecContext(ctx, "INSERT INTO gg_smart_task (id,agent_id,user_id,room_type,room_id,active_key,version,payload) VALUES (?,?,?,?,?,?,?,?)", id, agent, uid, roomType, room, active, version, string(raw))
			return e
		}
		var old map[string]any
		if e = DecodeGogameJSON(rows[0]["payload"].(string), &old); e != nil {
			return e
		}
		for _, key := range []string{"agentId", "userId", "roomId", "roomType", "instanceId", "areaId", "betAmount", "stopCountLimit", "takeProfit", "stopLoss", "currency", "strategy"} {
			if fmt.Sprint(old[key]) != fmt.Sprint(task[key]) {
				return paymentInvalid("smart task configuration conflict")
			}
		}
		previous := gogameNumber(rows[0]["version"])
		if version < previous {
			return nil
		}
		if version == previous {
			if string(raw) != rows[0]["payload"] {
				return paymentInvalid("smart task version conflict")
			}
			return nil
		}
		if old["status"] == "stopped" && task["status"] != "stopped" {
			return paymentInvalid("stopped task cannot resume")
		}
		_, e = tx.Statement.ConnPool.ExecContext(ctx, "UPDATE gg_smart_task SET active_key=?,version=?,payload=? WHERE id=?", active, version, string(raw), id)
		return e
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"stored": true}, nil
}
