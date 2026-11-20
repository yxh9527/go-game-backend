package rpc

// Reporting readiness depends on the configured ES destination and bounded
// output queues, never on a MySQL raw-document store.
func (d *LotteryService) recordStorageAllowsWager() bool {
	if d.es == nil || d.es.Client == nil {
		return false
	}
	if d.RecordChan != nil && len(d.RecordChan) >= cap(d.RecordChan) {
		return false
	}
	if d.BillChan != nil && len(d.BillChan) >= cap(d.BillChan) {
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
			delete(d.recordsCache.records, key)
		default:
			// Leave the original cache entry in place; no blocking send, deletion,
			// replacement with a newer pointer, or extra transfer goroutine.
			return
		}
	}
}
