package dao

import (
	"context"
	"encoding/json"
	"strings"

	"app/esindex"

	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"

	"micro_service/services"
)

type RecordListQuery struct {
	UserId   int64
	Symbol   string
	Hash     string
	Currency string
	GameId   uint32
	StartMs  int64
	EndMs    int64
	Page     int
	Size     int
}

func normalizePageSize(page, size int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}

// ListRecords 分页查询 ES 旧注单 pp_gp_settlement。
func (esDao *ESDao) ListRecords(q RecordListQuery) ([]*services.RecordItem, int64, error) {
	if strings.TrimSpace(q.Hash) != "" {
		items := esDao.GetRecords(q.UserId, q.Symbol, q.Hash, q.Currency)
		return items, int64(len(items)), nil
	}
	if q.UserId <= 0 {
		return []*services.RecordItem{}, 0, nil
	}

	page, size := normalizePageSize(q.Page, q.Size)
	querys := make([]elastic.Query, 0, 8)
	querys = append(querys, elastic.NewTermQuery("userId", q.UserId))
	querys = append(querys, elastic.NewTermQuery("complete", true))
	if strings.TrimSpace(q.Symbol) != "" {
		querys = append(querys, elastic.NewTermQuery("symbol", q.Symbol))
	}
	if q.GameId > 0 {
		querys = append(querys, elastic.NewTermQuery("gameId", q.GameId))
	}
	if strings.TrimSpace(q.Currency) != "" {
		querys = append(querys, elastic.NewMatchPhraseQuery("currency", q.Currency))
	}
	if q.StartMs > 0 || q.EndMs > 0 {
		rq := elastic.NewRangeQuery("playedDate")
		if q.StartMs > 0 {
			rq = rq.Gte(q.StartMs)
		}
		if q.EndMs > 0 {
			rq = rq.Lte(q.EndMs)
		}
		querys = append(querys, rq)
	}

	boolQuery := elastic.NewBoolQuery().Must(querys...)
	source := []string{
		"userId", "agentId", "bet", "currency", "currencySymbol", "base_bet", "win", "rtp",
		"playedDate", "roundID", "symbol", "hash", "balance", "balance_cash", "balance_bonus",
		"gameId", "gameName", "isTourist",
	}
	includeFields := elastic.NewFetchSourceContext(true).Include(source...)
	from := (page - 1) * size
	resp, err := esDao.es.Search().
		Index(esindex.Settlement()).
		FetchSourceContext(includeFields).
		Query(boolQuery).
		From(from).
		Size(size).
		Sort("playedDate", false).
		TrackTotalHits(true).
		Pretty(true).
		Do(context.Background())
	if err != nil {
		zap.L().Error("ListRecords es search failed", zap.Error(err), zap.Any("query", q))
		return nil, 0, err
	}

	total := int64(0)
	if resp.Hits != nil && resp.Hits.TotalHits != nil {
		total = resp.Hits.TotalHits.Value
	}
	records := make([]*services.RecordItem, 0, len(resp.Hits.Hits))
	for _, v := range resp.Hits.Hits {
		b, _ := v.Source.MarshalJSON()
		r := &services.RecordItem{}
		_ = json.Unmarshal(b, r)
		records = append(records, r)
	}
	return records, total, nil
}

// GetRecordDetailByKeys 按 recordId(hash/roundID) + userId + gameId 精确取详情。
// 不存在或不匹配返回 found=false，不算系统错误。
func (esDao *ESDao) GetRecordDetailByKeys(recordId string, userId, gameId uint32, symbol string) (*services.RecordItem, bool, error) {
	recordId = strings.TrimSpace(recordId)
	if recordId == "" || userId == 0 || gameId == 0 {
		return nil, false, nil
	}

	tryQueries := []elastic.Query{
		elastic.NewTermQuery("hash", recordId),
		elastic.NewTermQuery("roundID", recordId),
	}
	for _, idQuery := range tryQueries {
		querys := []elastic.Query{
			idQuery,
			elastic.NewTermQuery("userId", userId),
			elastic.NewTermQuery("gameId", gameId),
		}
		if strings.TrimSpace(symbol) != "" {
			// gameId 字段缺失的老数据可退回 symbol 匹配
			querys = []elastic.Query{
				idQuery,
				elastic.NewTermQuery("userId", userId),
				elastic.NewBoolQuery().Should(
					elastic.NewTermQuery("gameId", gameId),
					elastic.NewTermQuery("symbol", symbol),
				).MinimumNumberShouldMatch(1),
			}
		}
		boolQuery := elastic.NewBoolQuery().Must(querys...)
		resp, err := esDao.es.Search().
			Index(esindex.Settlement()).
			Query(boolQuery).
			Size(1).
			Sort("playedDate", false).
			Pretty(true).
			Do(context.Background())
		if err != nil {
			zap.L().Error("GetRecordDetailByKeys es search failed",
				zap.String("recordId", recordId),
				zap.Uint32("userId", userId),
				zap.Uint32("gameId", gameId),
				zap.Error(err))
			return nil, false, err
		}
		if resp == nil || resp.Hits == nil || len(resp.Hits.Hits) == 0 {
			continue
		}
		b, _ := resp.Hits.Hits[0].Source.MarshalJSON()
		r := &services.RecordItem{}
		if err := json.Unmarshal(b, r); err != nil {
			zap.L().Error("GetRecordDetailByKeys unmarshal failed", zap.Error(err))
			return nil, false, err
		}
		return r, true, nil
	}
	return nil, false, nil
}
