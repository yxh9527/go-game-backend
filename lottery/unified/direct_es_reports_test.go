package unified

import (
	"app/entity"
	"testing"
)

func TestBillAndRecordStoreMethodsRequireESAndUseNoMySQL(t *testing.T) {
	service := &LotteryService{}
	if err := service.storeSettlementBillBatch([]*entity.CacheBillsReq{{FlowingWaterOn: "original"}}); err == nil {
		t.Fatal("missing bill ES ignored")
	}
	if err := service.storeSettlementReportBatch([]*entity.CacheRecordsReq{{RoundID: "original"}}); err == nil {
		t.Fatal("missing record ES ignored")
	}
	if err := service.storeSettlementBillBatch(nil); err != nil {
		t.Fatal(err)
	}
	if err := service.storeSettlementReportBatch(nil); err != nil {
		t.Fatal(err)
	}
}
