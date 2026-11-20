package dao

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// Writers and deletion lock the same template row, so a concurrent publish
// cannot commit a reference to an already deleted template.
func gogameWithBlinds(ctx context.Context, db *gorm.DB, table string, values map[string]any, work func(*gorm.DB) error) error {
	id, err := gogameBlindReference(table, values)
	if err != nil {
		return err
	}
	if id == 0 {
		return work(db)
	}
	locked := func(tx *gorm.DB) error {
		rows, e := gogameRows(ctx, tx, "SELECT id FROM gg_blinds WHERE id=? FOR UPDATE", id)
		if e != nil {
			return e
		}
		if len(rows) == 0 {
			return status.Error(codes.FailedPrecondition, "blind template no longer exists")
		}
		return work(tx)
	}
	if _, ok := db.Statement.ConnPool.(gorm.TxCommitter); ok {
		return locked(db)
	}
	return db.WithContext(ctx).Transaction(locked)
}

func gogameBlindReference(table string, values map[string]any) (int64, error) {
	value := values["blinds_id"]
	if table == "gg_match_schedule" {
		if values["payload"] == nil {
			return 0, nil
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(fmt.Sprint(values["payload"])), &payload); err != nil {
			return 0, status.Error(codes.InvalidArgument, "invalid schedule payload")
		}
		value = payload["blinds_id"]
	} else if table != "gg_match" {
		return 0, nil
	}
	if value == nil {
		return 0, nil
	}
	var id int64
	if err := json.Unmarshal([]byte(fmt.Sprint(value)), &id); err != nil || id < 0 {
		return 0, status.Error(codes.InvalidArgument, "invalid blind template id")
	}
	return id, nil
}

func (dd *DBDao) gogameDeleteBlinds(ctx context.Context, id int64) (saved map[string]any, err error) {
	if id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	err = dd.manager.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rows, e := gogameRows(ctx, tx, "SELECT * FROM gg_blinds WHERE id=? FOR UPDATE", id)
		if e != nil {
			return e
		}
		if len(rows) == 0 {
			return status.Error(codes.NotFound, "blind template not found")
		}
		// Locking reads see the latest committed references, including writers
		// that held this template lock immediately before this transaction.
		used, e := gogameRows(ctx, tx, "SELECT id FROM gg_match WHERE blinds_id=? LIMIT 1 FOR UPDATE", id)
		if e != nil {
			return e
		}
		if len(used) > 0 {
			return status.Error(codes.FailedPrecondition, "blind template is referenced by a match")
		}
		used, e = gogameRows(ctx, tx, "SELECT id FROM gg_match_schedule WHERE JSON_UNQUOTE(JSON_EXTRACT(payload,'$.blinds_id'))=? LIMIT 1 FOR UPDATE", fmt.Sprint(id))
		if e != nil {
			return e
		}
		if len(used) > 0 {
			return status.Error(codes.FailedPrecondition, "blind template is referenced by a schedule")
		}
		_, e = tx.Statement.ConnPool.ExecContext(ctx, "DELETE FROM gg_blinds WHERE id=?", id)
		saved = rows[0]
		return e
	})
	return
}
