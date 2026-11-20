package dao

import (
	"app/entity"
	"app/esindex"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"micro_service/services"
	"strconv"

	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
)

type ESDao struct {
	es *elastic.Client
}

func NewESDao(client *elastic.Client) *ESDao {
	return &ESDao{es: client}
}

func (esDao *ESDao) BulkRecordsSave(data []*entity.CacheRecordsReq) error {
	bulkService := esDao.es.Bulk()
	records := make([]elastic.BulkableRequest, 0)
	for _, req := range data {
		hashStr := fmt.Sprintf("%d|%d|%s", req.AgentId, req.UserId, req.RoundID)
		req.Hash = fmt.Sprintf("%x", md5.Sum([]byte(hashStr)))
		records = append(records, elastic.NewBulkIndexRequest().Index(esindex.Settlement()).Id(req.Hash).Doc(req))
	}
	bulkService.Add(records...)
	_, err := bulkService.Do(context.Background())
	if err != nil {
		zap.L().Error("BulkRecordsSave,批量插入数据失败", zap.Any("err", err), zap.Any("data", data))
	}
	return nil
}

func (esDao *ESDao) BulkBillsSave(data []*entity.CacheBillsReq) error {
	bulkService := esDao.es.Bulk()
	records := make([]elastic.BulkableRequest, 0)
	for _, req := range data {
		records = append(records, elastic.NewBulkIndexRequest().Index(esindex.FlowingWater()).Doc(req))
	}
	bulkService.Add(records...)
	_, err := bulkService.Do(context.Background())
	if err != nil {
		zap.L().Error("BulkBillsSave,批量插入数据失败", zap.Any("err", err), zap.Any("data", data))
	}
	return nil
}

func (esDao *ESDao) GetRecords(userId int64, symbol, hash, currency string) []*services.RecordItem {
	querys := make([]elastic.Query, 0)
	if hash != "" {
		querys = append(querys, elastic.NewTermQuery("hash", hash))
	} else {
		querys = append(querys, elastic.NewTermQuery("userId", userId))
		querys = append(querys, elastic.NewTermQuery("symbol", symbol))
		querys = append(querys, elastic.NewTermQuery("complete", true))
	}
	if currency != "" {
		querys = append(querys, elastic.NewMatchPhraseQuery("currency", currency))
	}
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	var sorce []string = nil
	if hash == "" {
		sorce = []string{"userId", "agentId", "bet", "currency", "currencySymbol", "base_bet", "win", "rtp", "playedDate", "roundID", "symbol", "hash", "balance", "balance_cash", "balance_bonus", "isTourist"}
	} else {
		sorce = []string{"init", "log"}
	}
	includeFields := elastic.NewFetchSourceContext(true).Include(sorce...)
	resp, _ := esDao.es.Search().Index(esindex.Settlement()).FetchSourceContext(includeFields).
		Query(boolQuery).
		Pretty(true).
		Size(100).
		Sort("playedDate", false).
		Do(context.Background())
	records := make([]*services.RecordItem, 0)
	for _, v := range resp.Hits.Hits {
		b, _ := v.Source.MarshalJSON()
		r := &services.RecordItem{}
		json.Unmarshal(b, r)
		records = append(records, r)
	}
	return records
}

func (esDao *ESDao) GetRecordsByRoundId(roundId string) []*services.RecordItem {
	querys := make([]elastic.Query, 0)
	rid, _ := strconv.ParseInt(roundId, 10, 64)
	querys = append(querys, elastic.NewTermQuery("roundID", rid))
	// querys = append(querys, elastic.NewTermQuery("symbol", symbol))
	sorce := []string{"currency", "currencySymbol", "symbol", "log"}
	includeFields := elastic.NewFetchSourceContext(true).Include(sorce...)
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	resp, _ := esDao.es.Search().Index(esindex.Settlement()).FetchSourceContext(includeFields).
		Query(boolQuery).
		Pretty(true).
		Size(100).
		Sort("playedDate", false).
		Do(context.Background())
	records := make([]*services.RecordItem, 0)
	for _, v := range resp.Hits.Hits {
		b, _ := v.Source.MarshalJSON()
		r := &services.RecordItem{}
		json.Unmarshal(b, r)
		records = append(records, r)
	}
	return records
}

func (esDao *ESDao) GetRecordsReplayDataByRoundId(roundId string) []*services.RecordItem {
	querys := make([]elastic.Query, 0)
	rid, _ := strconv.ParseInt(roundId, 10, 64)
	querys = append(querys, elastic.NewTermQuery("roundID", rid))
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	resp, err := esDao.es.Search().Index(esindex.Settlement()).
		Query(boolQuery).
		Pretty(true).
		Size(100).
		Sort("playedDate", false).
		Do(context.Background())
	if err != nil {
		return nil
	}
	records := make([]*services.RecordItem, 0)
	for _, v := range resp.Hits.Hits {
		b, _ := v.Source.MarshalJSON()
		r := &services.RecordItem{}
		json.Unmarshal(b, r)
		records = append(records, r)
	}
	return records
}

func (esDao *ESDao) GetRecordsReplayDataByToken(token string) []*services.RecordItem {
	querys := make([]elastic.Query, 0)
	querys = append(querys, elastic.NewTermQuery("hash", token))
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	resp, err := esDao.es.Search().Index(esindex.Settlement()).
		Query(boolQuery).
		Pretty(true).
		Size(100).
		Sort("playedDate", false).
		Do(context.Background())
	records := make([]*services.RecordItem, 0)
	if err != nil {
		return nil
	}
	for _, v := range resp.Hits.Hits {
		b, _ := v.Source.MarshalJSON()
		r := &services.RecordItem{}
		json.Unmarshal(b, r)
		records = append(records, r)
	}
	return records
}

func (esDao *ESDao) GetRtpGreaterThan10(symbol string, userId int64) []*services.RecordItem {
	querys := make([]elastic.Query, 0)
	sorce := []string{"userId", "agentId", "bet", "currency", "currencySymbol", "base_bet", "win", "rtp", "playedDate", "roundID", "symbol", "hash", "balance", "balance_cash", "balance_bonus"}
	includeFields := elastic.NewFetchSourceContext(true).Include(sorce...)
	querys = append(querys, elastic.NewTermQuery("userId", userId))
	querys = append(querys, elastic.NewRangeQuery("rtp").Gte(10))
	querys = append(querys, elastic.NewTermQuery("symbol", symbol))
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	resp, err := esDao.es.Search().Index(esindex.Settlement()).FetchSourceContext(includeFields).
		Query(boolQuery).
		Pretty(true).
		Size(100).
		Sort("playedDate", false).
		Do(context.Background())
	if err != nil {
		return nil
	}
	records := make([]*services.RecordItem, 0)
	for _, v := range resp.Hits.Hits {
		b, _ := v.Source.MarshalJSON()
		r := &services.RecordItem{}
		json.Unmarshal(b, r)
		records = append(records, r)
	}
	return records
}
