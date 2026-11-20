package dao

import (
	"app/entity"
	"app/esindex"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"sync"
	"time"
)

const financialReportPrefix = "final-record:"

type finalFinancialRecord struct {
	Kind    string                  `json:"kind"`
	ID      string                  `json:"id"`
	Version int64                   `json:"version"`
	Record  *entity.CacheRecordsReq `json:"record,omitempty"`
	Bill    *entity.CacheBillsReq   `json:"bill,omitempty"`
}

func financialRecord(record *entity.CacheRecordsReq) (finalFinancialRecord, error) {
	if record == nil || record.UserId == 0 || record.Currency == "" || record.RoundID == "" {
		return finalFinancialRecord{}, fmt.Errorf("financial record identity missing")
	}
	copy := *record
	id := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%d|%d|%s", copy.AgentId, copy.UserId, copy.RoundID))))
	copy.Hash = id
	version := copy.RowVersion
	if version <= 0 {
		return finalFinancialRecord{}, fmt.Errorf("financial record version missing")
	}
	return finalFinancialRecord{Kind: "record", ID: id, Version: version, Record: &copy}, nil
}
func financialBill(bill *entity.CacheBillsReq) (finalFinancialRecord, error) {
	if bill == nil || bill.UserId == 0 || bill.Currency == "" || bill.FlowingWaterOn == "" {
		return finalFinancialRecord{}, fmt.Errorf("financial bill identity missing")
	}
	copy := *bill
	id := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%d|%d|%s", copy.AgentId, copy.UserId, copy.FlowingWaterOn))))
	return finalFinancialRecord{Kind: "bill", ID: id, Version: 1, Bill: &copy}, nil
}
func (row finalFinancialRecord) owner() (uint32, string) {
	if row.Kind == "record" && row.Record != nil {
		return row.Record.UserId, row.Record.Currency
	}
	if row.Kind == "bill" && row.Bill != nil {
		return row.Bill.UserId, row.Bill.Currency
	}
	return 0, ""
}

// Store the original output document in the existing storage table before
// returning from SaveRecord/SaveBill. Never mutate a wallet or fabricate a round.
func (dd *DBDao) StoreFinancialRecords(ctx context.Context, records []*entity.CacheRecordsReq, bills []*entity.CacheBillsReq) (err error) {
	defer func() {
		if err != nil {
			ResetFinalRecordStorageReadiness()
		}
	}()
	rows := make([]finalFinancialRecord, 0, len(records)+len(bills))
	for _, record := range records {
		row, err := financialRecord(record)
		if err != nil {
			return err
		}
		rows = append(rows, row)
	}
	for _, bill := range bills {
		row, err := financialBill(bill)
		if err != nil {
			return err
		}
		rows = append(rows, row)
	}
	return dd.storeFinalFinancialRows(ctx, rows)
}
func (dd *DBDao) storeFinalFinancialRows(ctx context.Context, rows []finalFinancialRecord) error {
	if dd == nil || dd.player == nil {
		return fmt.Errorf("durable final record storage unavailable")
	}
	if len(rows) == 0 || len(rows) > 100 {
		return fmt.Errorf("invalid final record batch")
	}
	return dd.player.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			user, currency := row.owner()
			if user == 0 || currency == "" || row.ID == "" || row.Version < 1 {
				return fmt.Errorf("invalid final record")
			}
			raw, err := json.Marshal(row)
			if err != nil {
				return err
			}
			if len(raw) > 2*1024*1024 {
				return fmt.Errorf("final financial record exceeds storage limit")
			}
			// Preserve different snapshots of an evolving legacy record. ES external
			// versions prevent an older accepted snapshot from overwriting a later one.
			key := financialReportPrefix + fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", row.Kind, row.ID, row.Version))))
			_, err = tx.Statement.ConnPool.ExecContext(ctx, "INSERT INTO fg_game_storage (user_id,game_id,currency_symbol,storage_key,payload,expire_at,updated_at) VALUES (?,0,?,?,?,0,?) ON DUPLICATE KEY UPDATE storage_key=VALUES(storage_key)", user, currency, key, string(raw), time.Now().Unix())
			if err != nil {
				return err
			}
			var original struct{ Payload string }
			if err = tx.Raw("SELECT payload FROM fg_game_storage WHERE user_id=? AND game_id=0 AND currency_symbol=? AND storage_key=? FOR UPDATE", user, currency, key).Scan(&original).Error; err != nil {
				return err
			}
			if original.Payload != string(raw) {
				return fmt.Errorf("financial document version conflicts with its original stored payload")
			}
		}
		return nil
	})
}

func (es *ESDao) indexFinalFinancialRows(ctx context.Context, rows []finalFinancialRecord) error {
	if es == nil || es.Client == nil {
		return fmt.Errorf("financial ES unavailable")
	}
	bulk := es.Client.Bulk()
	for _, row := range rows {
		if row.Kind == "record" && row.Record != nil {
			bulk.Add(elastic.NewBulkIndexRequest().Index(esindex.Settlement()).Id(row.ID).Version(row.Version).VersionType("external_gte").Doc(row.Record))
		} else if row.Kind == "bill" && row.Bill != nil {
			bulk.Add(elastic.NewBulkIndexRequest().Index(esindex.FlowingWater()).Id(row.ID).OpType("create").Doc(row.Bill))
		} else {
			return fmt.Errorf("invalid stored final financial record")
		}
	}
	response, err := bulk.Do(ctx)
	if err != nil {
		return err
	}
	if response == nil || len(response.Items) != len(rows) {
		return fmt.Errorf("final financial record bulk response incomplete")
	}
	for _, item := range response.Items {
		if len(item) != 1 {
			return fmt.Errorf("invalid financial record bulk item")
		}
		for _, result := range item {
			if result == nil || result.Status < 200 || result.Status >= 300 && result.Status != 409 {
				return fmt.Errorf("final financial record ES projection pending")
			}
		}
	}
	return nil
}

// Projection scans only original report documents, not unfinished financial
// transactions. Replacing the service cannot replay a bet, refund, or payout.
func (dd *DBDao) ProjectFinalFinancialRecords(ctx context.Context, es *ESDao) (int, error) {
	if dd == nil || dd.player == nil {
		return 0, fmt.Errorf("final record storage unavailable")
	}
	var stored []GameStorageRecord
	if err := dd.player.WithContext(ctx).Where("game_id=0 AND expire_at=0 AND storage_key LIKE ?", financialReportPrefix+"%").Order("id").Limit(40).Find(&stored).Error; err != nil {
		return 0, err
	}
	if len(stored) == 0 {
		return 0, nil
	}
	rows := make([]finalFinancialRecord, 0, len(stored))
	ids := make([]uint64, 0, len(stored))
	for _, item := range stored {
		var row finalFinancialRecord
		if json.Unmarshal([]byte(item.Payload), &row) != nil {
			return 0, fmt.Errorf("invalid durable financial report")
		}
		user, currency := row.owner()
		if user != item.UserId || currency != item.CurrencySymbol {
			return 0, fmt.Errorf("durable financial report owner mismatch")
		}
		var validated finalFinancialRecord
		var err error
		if row.Kind == "record" {
			validated, err = financialRecord(row.Record)
		} else if row.Kind == "bill" {
			validated, err = financialBill(row.Bill)
		} else {
			return 0, fmt.Errorf("invalid financial report kind")
		}
		if err != nil || validated.ID != row.ID || validated.Version != row.Version {
			return 0, fmt.Errorf("durable financial record identity mismatch")
		}
		rows = append(rows, row)
		ids = append(ids, item.ID)
	}
	if err := es.indexFinalFinancialRows(ctx, rows); err != nil {
		return 0, err
	}
	if err := dd.player.WithContext(ctx).Model(&GameStorageRecord{}).Where("id IN ? AND game_id=0 AND expire_at=0", ids).Update("expire_at", time.Now().Add(31*24*time.Hour).Unix()).Error; err != nil {
		return 0, err
	}
	return len(rows), nil
}
func (dd *DBDao) expireFinalFinancialRecords(ctx context.Context) error {
	if dd == nil || dd.player == nil {
		return fmt.Errorf("final record storage unavailable")
	}
	_, err := dd.player.Statement.ConnPool.ExecContext(ctx, "DELETE FROM fg_game_storage WHERE game_id=0 AND storage_key LIKE ? AND expire_at>0 AND expire_at<? LIMIT 100", financialReportPrefix+"%", time.Now().Unix())
	return err
}

var finalRecordProjection sync.Once

func StartFinalFinancialRecordProjection(db *DBDao, es *ESDao) {
	finalRecordProjection.Do(func() {
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			passes := 0
			for range ticker.C {
				ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
				err := db.FlushFinancialRecordMemory(ctx)
				if err == nil {
					_, err = db.ProjectFinalFinancialRecords(ctx, es)
				}
				passes++
				if err == nil && passes%60 == 0 {
					err = db.expireFinalFinancialRecords(ctx)
				}
				cancel()
				if err != nil {
					zap.L().Error("durable final financial record projection pending", zap.Error(err))
				}
			}
		}()
	})
}
