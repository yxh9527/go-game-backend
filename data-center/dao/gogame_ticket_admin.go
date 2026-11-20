package dao

import (
	"context"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"strings"
	"time"
)

type ticketAdminInput struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"userId"`
	AgentID   int64  `json:"agentId"`
	Count     int64  `json:"count"`
	RequestID string `json:"requestId"`
}

// Issuance identity is stored on the existing holding rows. There is no separate
// payment journal; inventory and ownership commit in one existing SQL transaction.
func (dd *DBDao) gogameTicketAdmin(ctx context.Context, operation, payload string) (any, error) {
	var input ticketAdminInput
	if err := DecodeGogameJSON(payload, &input); err != nil {
		return nil, err
	}
	if input.ID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "ticket id required")
	}
	if operation == "deleteTicket" {
		var saved map[string]any
		err := dd.manager.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			rows, e := gogameRows(ctx, tx, "SELECT * FROM gg_ticket WHERE id=? FOR UPDATE", input.ID)
			if e != nil {
				return e
			}
			if len(rows) == 0 {
				return status.Error(codes.NotFound, "ticket not found")
			}
			if _, e = tx.Statement.ConnPool.ExecContext(ctx, "UPDATE gg_ticket SET disabled=1,update_time=? WHERE id=?", time.Now().Unix(), input.ID); e != nil {
				return e
			}
			saved = rows[0]
			saved["disabled"] = true
			return nil
		})
		return saved, err
	}
	if input.UserID <= 0 || input.AgentID <= 0 || input.Count <= 0 || input.Count > 1000 || !cashID.MatchString(input.RequestID) {
		return nil, status.Error(codes.InvalidArgument, "invalid ticket issuance")
	}
	table, err := dd.matchTicketTable(ctx)
	if err != nil {
		return nil, err
	}
	return dd.issueTicketsSQL(ctx, table, input)
}

func (dd *DBDao) issueTicketsSQL(ctx context.Context, table string, input ticketAdminInput) (result map[string]any, err error) {
	err = dd.manager.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ticket, e := gogameRows(ctx, tx, "SELECT remain,disabled FROM gg_ticket WHERE id=? FOR UPDATE", input.ID)
		if e != nil {
			return e
		}
		if len(ticket) == 0 {
			return status.Error(codes.NotFound, "ticket not found")
		}
		previous, e := gogameRows(ctx, tx, "SELECT id,ticket_id,owner,issue_agent FROM "+table+" WHERE issue_key=? ORDER BY issue_ordinal FOR UPDATE", input.RequestID)
		if e != nil {
			return e
		}
		if len(previous) > 0 {
			if int64(len(previous)) != input.Count {
				return paymentInvalid("issuance request conflict")
			}
			for _, row := range previous {
				if gogameNumber(row["ticket_id"]) != input.ID || gogameNumber(row["owner"]) != input.UserID || gogameNumber(row["issue_agent"]) != input.AgentID {
					return paymentInvalid("issuance request conflict")
				}
			}
			result = map[string]any{"requestId": input.RequestID, "issued": input.Count, "replayed": true}
			return nil
		}
		if ticket[0]["disabled"] == true || gogameNumber(ticket[0]["disabled"]) != 0 {
			return paymentInvalid("ticket type retired")
		}
		if gogameNumber(ticket[0]["remain"]) < input.Count {
			return paymentInvalid("ticket issuance inventory insufficient")
		}
		users := strings.TrimSuffix(table, "`gg_ticket_hold`") + "`user_info`"
		account, e := gogameRows(ctx, tx, "SELECT user_id FROM "+users+" WHERE user_id=? AND proxy_id=? FOR UPDATE", input.UserID, input.AgentID)
		if e != nil {
			return e
		}
		if len(account) != 1 {
			return status.Error(codes.PermissionDenied, "player does not belong to selected agent")
		}
		if _, e = tx.Statement.ConnPool.ExecContext(ctx, "UPDATE gg_ticket SET remain=remain-?,update_time=? WHERE id=?", input.Count, time.Now().Unix(), input.ID); e != nil {
			return e
		}
		for i := int64(1); i <= input.Count; i++ {
			if _, e = tx.Statement.ConnPool.ExecContext(ctx, "INSERT INTO "+table+" (ticket_id,owner,is_use,use_time,use_match_id,create_time,issue_key,issue_ordinal,issue_agent) VALUES (?,?,0,0,0,?,?,?,?)", input.ID, input.UserID, time.Now().Unix(), input.RequestID, i, input.AgentID); e != nil {
				return fmt.Errorf("ticket issuance rolled back: %w", e)
			}
		}
		result = map[string]any{"requestId": input.RequestID, "issued": input.Count, "replayed": false}
		return nil
	})
	return
}
