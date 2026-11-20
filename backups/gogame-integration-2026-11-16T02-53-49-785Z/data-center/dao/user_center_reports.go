package dao

import (
	"app/esindex"
	"app/tables/gogame"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"math"
	"time"
)

// This is an already-settled display record in the existing game-storage table,
// never a financial operation, balance journal, or unfinished round.
type centerReport struct {
	Index  string           `json:"index"`
	Record UserCenterRecord `json:"record"`
}

const centerReportPrefix = "uc-report:"

func reportTable() string { return (gogame.ScratchTicketStorage{}).TableName() }

func validateCenterReportBatch(records []UserCenterRecord, index string) error {
	if index != centerIndex() && index != esindex.Name("gogame_replays") {
		return status.Error(codes.InvalidArgument, "invalid report destination")
	}
	if len(records) < 1 || len(records) > 100 {
		return status.Error(codes.InvalidArgument, "invalid report batch")
	}
	now := time.Now().UnixMilli()
	for _, r := range records {
		if !cashID.MatchString(r.ID) || r.UserID == 0 || r.AgentID == 0 || !cashCurrency.MatchString(r.Currency) || r.RoomType <= 0 || r.RoomID <= 0 || r.RoomID > cashMax || r.Inning <= 0 || r.Inning > cashMax || r.EndedAt < now-31*86400000 || r.EndedAt > now+300000 || r.EffectiveBet < 0 || r.EffectiveBet > cashMax || math.Abs(float64(r.Net)) > float64(cashMax) || r.Balance != nil && (*r.Balance < 0 || *r.Balance > cashMax) || len(r.Detail) > 256000 || !json.Valid(r.Detail) || len(r.Players) > 0 && !json.Valid(r.Players) {
			return status.Error(codes.InvalidArgument, "invalid settled report")
		}
		if index == esindex.Name("gogame_replays") && r.RoomType != 19 && r.RoomType != 101 {
			return status.Error(codes.InvalidArgument, "invalid replay game")
		}
	}
	return nil
}

func (dd *DBDao) storeCenterReports(ctx context.Context, es *ESDao, records []UserCenterRecord, index string) (any, error) {
	if err := validateCenterReportBatch(records, index); err != nil {
		return nil, err
	}
	verified := map[string]bool{}
	for _, r := range records {
		key := fmt.Sprintf("%d:%d:%s", r.AgentID, r.UserID, r.Currency)
		if !verified[key] {
			if err := dd.centerIdentity(ctx, r.UserID, r.AgentID, r.Currency); err != nil {
				return nil, err
			}
			verified[key] = true
		}
	}
	if dd.db == nil {
		return nil, status.Error(codes.Unavailable, "durable report storage unavailable")
	}
	keys, err := dd.storeCenterReportsSQL(ctx, records, index)
	if err != nil {
		return nil, err
	}
	indexed := false
	projection, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if _, err = dd.indexCenterRecords(projection, es, records, index); err == nil {
		indexed = dd.markCenterReportsIndexed(ctx, keys) == nil
	}
	return map[string]any{"stored": len(records), "durable": true, "indexed": indexed}, nil
}

func (dd *DBDao) storeCenterReportsSQL(ctx context.Context, records []UserCenterRecord, index string) (keys []string, err error) {
	err = dd.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, r := range records {
			r.Aggregate = nil
			payload, e := json.Marshal(centerReport{Index: index, Record: r})
			if e != nil {
				return e
			}
			key := centerReportPrefix + fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%d:%s:%s", index, r.AgentID, r.UserID, r.Currency, r.ID))))
			// Preserve the first confirmed record. An acknowledgement retry may not
			// silently replace a result with changed cards, amounts or identity.
			_, e = tx.Statement.ConnPool.ExecContext(ctx, "INSERT INTO "+reportTable()+" (user_id,game_id,currency_symbol,storage_key,payload,expire_at,updated_at) VALUES (?,0,?,?,?,0,?) ON DUPLICATE KEY UPDATE storage_key=VALUES(storage_key)", r.UserID, r.Currency, key, string(payload), time.Now().Unix())
			if e != nil {
				return e
			}
			rows, e := gogameRows(ctx, tx, "SELECT payload FROM "+reportTable()+" WHERE user_id=? AND game_id=0 AND currency_symbol=? AND storage_key=? FOR UPDATE", r.UserID, r.Currency, key)
			if e != nil {
				return e
			}
			if len(rows) != 1 || rows[0]["payload"] != string(payload) {
				return status.Error(codes.FailedPrecondition, "settled report conflicts with original stored record")
			}
			keys = append(keys, key)
		}
		return nil
	})
	return
}

func (dd *DBDao) markCenterReportsIndexed(ctx context.Context, keys []string) error {
	for _, key := range keys {
		if _, err := dd.db.Statement.ConnPool.ExecContext(ctx, "UPDATE "+reportTable()+" SET expire_at=? WHERE game_id=0 AND storage_key=? AND expire_at=0", time.Now().Add(31*24*time.Hour).Unix(), key); err != nil {
			return err
		}
	}
	return nil
}

// Retry projection of durable final reports. Multiple data-center processes can
// safely repeat ES create operations; there are no wallet/settlement calls here.
func (dd *DBDao) FlushCenterReports(ctx context.Context, es *ESDao) (int, error) {
	if dd.db == nil {
		return 0, status.Error(codes.Unavailable, "report storage unavailable")
	}
	rows, err := gogameRows(ctx, dd.db, "SELECT storage_key,payload FROM "+reportTable()+" WHERE game_id=0 AND expire_at=0 AND storage_key LIKE ? ORDER BY id LIMIT 100", centerReportPrefix+"%")
	if err != nil {
		return 0, err
	}
	count := 0
	for _, row := range rows {
		var report centerReport
		if err = json.Unmarshal([]byte(row["payload"].(string)), &report); err != nil {
			return count, err
		}
		if report.Index != centerIndex() && report.Index != esindex.Name("gogame_replays") {
			return count, status.Error(codes.FailedPrecondition, "invalid stored report destination")
		}
		if _, err = dd.indexCenterRecords(ctx, es, []UserCenterRecord{report.Record}, report.Index); err != nil {
			return count, err
		}
		if err = dd.markCenterReportsIndexed(ctx, []string{row["storage_key"].(string)}); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// Expiry applies only after ES has confirmed the record, never to an unindexed
// report. Cleanup is bounded and cannot touch existing game recovery records.
func (dd *DBDao) ExpireIndexedCenterReports(ctx context.Context) error {
	if dd.db == nil {
		return status.Error(codes.Unavailable, "report storage unavailable")
	}
	_, err := dd.db.Statement.ConnPool.ExecContext(ctx, "DELETE FROM "+reportTable()+" WHERE game_id=0 AND storage_key LIKE ? AND expire_at>0 AND expire_at<? LIMIT 100", centerReportPrefix+"%", time.Now().Unix())
	return err
}
