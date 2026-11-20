package rpc

import (
	"context"
	"go.uber.org/zap"
	"time"
)

func (d *LotteryService) recordStorageAllowsWager() bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := d.db.FinalRecordStorageReady(ctx); err != nil {
		zap.L().Error("new wager rejected before money mutation: final record storage unavailable", zap.Error(err))
		return false
	}
	return true
}

func (d *LotteryService) flushLegacyRecordCache(at int64) {
	if d.recordsCache == nil {
		return
	}
	d.recordsCache.lock.Lock()
	defer d.recordsCache.lock.Unlock()
	count := 0
	for key, item := range d.recordsCache.records {
		if count >= 128 {
			return
		}
		count++
		select {
		case d.RecordChan <- item.record:
			if at > item.TimeOut || item.record.Complete {
				delete(d.recordsCache.records, key)
			}
		default:
			// Leave the original cache entry in place; no blocking send, deletion,
			// replacement with a newer pointer, or extra transfer goroutine.
			return
		}
	}
}
