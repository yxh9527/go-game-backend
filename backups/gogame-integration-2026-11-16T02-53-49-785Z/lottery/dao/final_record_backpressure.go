package dao

import (
	"app/entity"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"
)

// This is only an in-process fallback for original output documents while SQL
// is unavailable. It is explicitly not a durable receipt or wallet journal.
// New wagering is gated until all retained documents reach SQL; already accepted
// financial obligations continue and may add their original reports here.
var financialMemory = struct {
	sync.Mutex
	rows  map[string]finalFinancialRecord
	at    time.Time
	ready bool
}{rows: make(map[string]finalFinancialRecord)}

func RetainFinancialBills(bills []*entity.CacheBillsReq) error {
	return RetainFinancialRecords(nil, bills)
}
func RetainFinancialRecords(records []*entity.CacheRecordsReq, bills []*entity.CacheBillsReq) error {
	rows := make([]finalFinancialRecord, 0, len(bills))
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
	financialMemory.Lock()
	defer financialMemory.Unlock()
	for _, row := range rows {
		key := fmt.Sprintf("%s:%s:%d", row.Kind, row.ID, row.Version)
		if old, exists := financialMemory.rows[key]; exists {
			a, _ := json.Marshal(old)
			b, _ := json.Marshal(row)
			if string(a) != string(b) {
				return fmt.Errorf("original retained financial document conflicts")
			}
		} else {
			financialMemory.rows[key] = row
		}
	}
	financialMemory.ready = false
	financialMemory.at = time.Time{}
	return nil
}

func (dd *DBDao) FlushFinancialRecordMemory(ctx context.Context) error {
	financialMemory.Lock()
	keys := make([]string, 0, 40)
	for key := range financialMemory.rows {
		keys = append(keys, key)
		if len(keys) >= 40 {
			break
		}
	}
	sort.Strings(keys)
	rows := make([]finalFinancialRecord, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, financialMemory.rows[key])
	}
	financialMemory.Unlock()
	if len(rows) == 0 {
		return nil
	}
	if err := dd.storeFinalFinancialRows(ctx, rows); err != nil {
		return err
	}
	financialMemory.Lock()
	defer financialMemory.Unlock()
	for i, key := range keys {
		current, exists := financialMemory.rows[key]
		if !exists {
			continue
		}
		a, _ := json.Marshal(current)
		b, _ := json.Marshal(rows[i])
		if string(a) == string(b) {
			delete(financialMemory.rows, key)
		}
	}
	return nil
}

// A read-only preflight, before any new debit or pool change. SQL failure after
// this probe still uses the original retained-output path; this is backpressure,
// not a cross-system financial transaction or a zero-loss guarantee.
func (dd *DBDao) FinalRecordStorageReady(ctx context.Context) error {
	if dd == nil || dd.player == nil {
		return fmt.Errorf("final record database unavailable")
	}
	financialMemory.Lock()
	if len(financialMemory.rows) > 0 {
		financialMemory.Unlock()
		return fmt.Errorf("previous financial output still awaits durable storage")
	}
	at := time.Now()
	if !financialMemory.at.IsZero() && at.Sub(financialMemory.at) < time.Second {
		ready := financialMemory.ready
		financialMemory.Unlock()
		if !ready {
			return fmt.Errorf("final record SQL preflight unavailable")
		}
		return nil
	}
	financialMemory.Unlock()
	db, err := dd.player.DB()
	if err == nil {
		err = db.PingContext(ctx)
	}
	// Probe the existing output table as well: PING alone cannot detect a missing
	// schema or inaccessible table. This does not migrate or write anything.
	if err == nil {
		var found string
		err = dd.player.WithContext(ctx).Raw("SELECT storage_key FROM fg_game_storage LIMIT 1").Scan(&found).Error
	}
	financialMemory.Lock()
	financialMemory.at = at
	financialMemory.ready = err == nil && len(financialMemory.rows) == 0
	ready := financialMemory.ready
	financialMemory.Unlock()
	if !ready {
		return fmt.Errorf("final record SQL preflight unavailable")
	}
	return nil
}

func ResetFinalRecordStorageReadiness() {
	financialMemory.Lock()
	financialMemory.at = time.Time{}
	financialMemory.ready = false
	financialMemory.Unlock()
}
