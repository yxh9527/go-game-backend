package dao

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// Only a never-registered draft may be physically removed. Runtime receipts
// remain attached to all other matches; archiving is a Node runtime operation.
func (dd *DBDao) gogameDeleteDraft(ctx context.Context, id int64) (saved map[string]any, err error) {
	if id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "match id required")
	}
	err = dd.manager.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rows, e := gogameRows(ctx, tx, "SELECT * FROM gg_match WHERE id=? FOR UPDATE", id)
		if e != nil {
			return e
		}
		if len(rows) == 0 {
			return status.Error(codes.NotFound, "match not found")
		}
		if gogameNumber(rows[0]["state"]) != 0 {
			return status.Error(codes.FailedPrecondition, "only unused drafts can be deleted")
		}
		docs, e := gogameRows(ctx, tx, "SELECT payload FROM gg_match_runtime WHERE match_id=?", id)
		if e != nil {
			return e
		}
		for _, doc := range docs {
			var state map[string]any
			if e = DecodeGogameJSON(doc["payload"].(string), &state); e != nil {
				return e
			}
			for _, key := range []string{"registrations", "prizes", "tables"} {
				switch value := state[key].(type) {
				case map[string]any:
					if len(value) > 0 {
						return status.Error(codes.FailedPrecondition, "draft has registration or financial history")
					}
				case []any:
					if len(value) > 0 {
						return status.Error(codes.FailedPrecondition, "draft has registration or financial history")
					}
				case nil:
				default:
					return status.Error(codes.FailedPrecondition, "invalid draft runtime")
				}
			}
		}
		for _, query := range []string{"SELECT match_id FROM gg_match_occurrence WHERE match_id=? LIMIT 1", "SELECT id FROM gg_ticket WHERE match_id=? LIMIT 1"} {
			used, e := gogameRows(ctx, tx, query, id)
			if e != nil {
				return e
			}
			if len(used) > 0 {
				return status.Error(codes.FailedPrecondition, "draft is referenced by a schedule or ticket")
			}
		}
		if _, e = tx.Statement.ConnPool.ExecContext(ctx, "DELETE FROM gg_match_runtime WHERE match_id=?", id); e != nil {
			return e
		}
		if _, e = tx.Statement.ConnPool.ExecContext(ctx, "DELETE FROM gg_match WHERE id=? AND state=0", id); e != nil {
			return e
		}
		saved = rows[0]
		return nil
	})
	return
}
