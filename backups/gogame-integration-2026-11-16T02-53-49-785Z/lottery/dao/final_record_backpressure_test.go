package dao

import (
	"app/entity"
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func resetFinancialMemory(t *testing.T) {
	financialMemory.Lock()
	before := financialMemory.rows
	at, ready := financialMemory.at, financialMemory.ready
	financialMemory.rows = map[string]finalFinancialRecord{}
	financialMemory.at = time.Time{}
	financialMemory.ready = false
	financialMemory.Unlock()
	t.Cleanup(func() {
		financialMemory.Lock()
		financialMemory.rows = before
		financialMemory.at = at
		financialMemory.ready = ready
		financialMemory.Unlock()
	})
}

func TestFailedBillRetentionDoesNotBlockOrLoseOriginalAndGatesNewWagers(t *testing.T) {
	resetFinancialMemory(t)
	bill := &entity.CacheBillsReq{AgentId: 7, UserId: 42, Currency: "CNY", RoundID: "actual", FlowingWaterOn: "bill-one", Bet: -1, CurrentScore: 10}
	if err := RetainFinancialBills([]*entity.CacheBillsReq{bill}); err != nil {
		t.Fatal(err)
	}
	bill.CurrentScore = 999
	if len(financialMemory.rows) != 1 {
		t.Fatal(financialMemory.rows)
	}
	blocked, script := finalReportDB(t)
	if err := blocked.FinalRecordStorageReady(context.Background()); err == nil || len(script.log) != 0 {
		t.Fatal("pending original output did not block new admission", err, script.log)
	}
	failed, steps := finalReportDB(t, finalSQLStep{contains: "INSERT INTO fg_game_storage", err: errors.New("SQL down")})
	if err := failed.FlushFinancialRecordMemory(context.Background()); err == nil || len(financialMemory.rows) != 1 || steps.log[len(steps.log)-1] != "rollback" {
		t.Fatal(err, financialMemory.rows)
	}
	var payload string
	for _, row := range financialMemory.rows {
		raw, _ := json.Marshal(row)
		payload = string(raw)
		if row.Bill.CurrentScore != 10 {
			t.Fatal("retained pointer was changed")
		}
	}
	restored, steps := finalReportDB(t, finalSQLStep{contains: "INSERT INTO fg_game_storage"}, finalSQLStep{contains: "FOR UPDATE", columns: []string{"payload"}, values: [][]driver.Value{{payload}}})
	if err := restored.FlushFinancialRecordMemory(context.Background()); err != nil || len(financialMemory.rows) != 0 || len(steps.steps) != 0 {
		t.Fatal(err, financialMemory.rows)
	}
	ready, steps := finalReportDB(t, finalSQLStep{contains: "SELECT storage_key FROM fg_game_storage LIMIT 1", columns: []string{"storage_key"}})
	if err := ready.FinalRecordStorageReady(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := ready.FinalRecordStorageReady(context.Background()); err != nil || len(steps.log) != 1 {
		t.Fatal("healthy preflight not reused", err, steps.log)
	}
}

func TestRecordFailureRetainsEveryVersionRatherThanReplacingWithNewest(t *testing.T) {
	resetFinancialMemory(t)
	record := &entity.CacheRecordsReq{AgentId: 7, UserId: 42, Currency: "CNY", RoundID: "original", RowVersion: 10, Log: "one"}
	if err := RetainFinancialRecords([]*entity.CacheRecordsReq{record}, nil); err != nil {
		t.Fatal(err)
	}
	record.RowVersion = 20
	record.Log = "two"
	if err := RetainFinancialRecords([]*entity.CacheRecordsReq{record}, nil); err != nil {
		t.Fatal(err)
	}
	if len(financialMemory.rows) != 2 {
		t.Fatal("original record versions overwritten")
	}
	if err := RetainFinancialRecords([]*entity.CacheRecordsReq{record}, nil); err != nil || len(financialMemory.rows) != 2 {
		t.Fatal("retry duplicated a version", err)
	}
	record.Log = "conflict"
	if err := RetainFinancialRecords([]*entity.CacheRecordsReq{record}, nil); err == nil {
		t.Fatal("same version accepted conflicting content")
	}
	if err := (&DBDao{}).FinalRecordStorageReady(context.Background()); err == nil {
		t.Fatal("missing DB allowed wager")
	}
}

func TestReadinessFailureIsExplicitAndItsCacheCanBeInvalidated(t *testing.T) {
	resetFinancialMemory(t)
	db, steps := finalReportDB(t, finalSQLStep{contains: "SELECT storage_key", err: errors.New("table unavailable")})
	err := db.FinalRecordStorageReady(context.Background())
	if err == nil || !strings.Contains(err.Error(), "preflight") {
		t.Fatal(err)
	}
	if err = db.FinalRecordStorageReady(context.Background()); err == nil || len(steps.log) != 1 {
		t.Fatal("failed preflight not bounded", err)
	}
	ResetFinalRecordStorageReadiness()
	if !financialMemory.at.IsZero() || financialMemory.ready {
		t.Fatal("stale readiness cache retained")
	}
}
