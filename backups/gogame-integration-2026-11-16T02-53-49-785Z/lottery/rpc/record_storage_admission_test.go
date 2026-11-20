package rpc

import (
	"app/entity"
	"github.com/shopspring/decimal"
	"micro_service/services"
	"sync"
	"testing"
	"time"
)

func TestFullRecordChannelRetainsOriginalCacheAndDoesNotBlock(t *testing.T) {
	original := &entity.CacheRecordsReq{Hash: "one", Complete: true, Log: "original"}
	service := &LotteryService{RecordChan: make(chan *entity.CacheRecordsReq), recordsCache: &RecordCacheMgr{lock: &sync.RWMutex{}, records: map[string]*RecordItem{"one": {record: original, TimeOut: 1}}}}
	done := make(chan struct{})
	go func() { service.flushLegacyRecordCache(100); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("record transfer blocked on a full output channel")
	}
	if len(service.recordsCache.records) != 1 || service.recordsCache.records["one"].record != original {
		t.Fatal("cache dropped or changed its original record")
	}
	service.RecordChan = make(chan *entity.CacheRecordsReq, 1)
	service.flushLegacyRecordCache(100)
	if len(service.recordsCache.records) != 0 || <-service.RecordChan != original {
		t.Fatal("acknowledged channel transfer lost original record")
	}
}

func TestRecordCacheTransferLimitsTheWorkPerTick(t *testing.T) {
	service := &LotteryService{RecordChan: make(chan *entity.CacheRecordsReq, 1000), recordsCache: &RecordCacheMgr{lock: &sync.RWMutex{}, records: map[string]*RecordItem{}}}
	for i := 0; i < 500; i++ {
		key := string(rune(1000 + i))
		service.recordsCache.records[key] = &RecordItem{record: &entity.CacheRecordsReq{Complete: true}, TimeOut: 1}
	}
	service.flushLegacyRecordCache(100)
	if len(service.RecordChan) != 128 || len(service.recordsCache.records) != 372 {
		t.Fatal(len(service.RecordChan), len(service.recordsCache.records))
	}
}

func TestNewSlotsWagerCannotReachMoneyWhenRecordDatabaseIsUnavailable(t *testing.T) {
	service := &LotteryService{}
	currency, paid, code := service.SlotsBet(0, decimal.NewFromInt(1), &slotsLotteryReq{Bet: "1.00"})
	if currency != 0 || paid || code != services.ErrorCode_SYSTEM_ERROR {
		t.Fatal(currency, paid, code)
	}
	if value, paid := service.deductBet(7, 42, decimal.NewFromInt(1), "test", "round", "1.00", "CNY"); paid || !value.IsZero() {
		t.Fatal(value, paid)
	}
}
