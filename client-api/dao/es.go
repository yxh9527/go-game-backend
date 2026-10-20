package dao

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"app/esindex"
	"client-api/config"

	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
)

type ESDao struct {
	es *elastic.Client
}

var esIns *ESDao

type RecordItem struct {
	AgentId        uint32  `json:"agentId"`
	UserId         uint32  `json:"userId"`
	Bet            float64 `json:"bet"`
	Currency       string  `json:"currency"`
	CurrencySymbol string  `json:"currencySymbol"`
	BaseBet        float64 `json:"base_bet"`
	Win            float64 `json:"win"`
	Rtp            float64 `json:"rtp"`
	PlayedDate     int64   `json:"playedDate"`
	RoundID        string  `json:"roundID"`
	Init           string  `json:"init"`
	Log            string  `json:"log"`
	Symbol         string  `json:"symbol"`
	Balance        float64 `json:"balance"`
	BalanceCash    float64 `json:"balance_cash"`
	BalanceBonus   float64 `json:"balance_bonus"`
	Hash           string  `json:"hash"`
	GameId         uint32  `json:"gameId"`
	GameName       string  `json:"gameName"`
}

type RecordListQuery struct {
	UserId   uint32
	Symbol   string
	Currency string
	GameId   uint32
	StartMs  int64
	EndMs    int64
	Page     int
	Size     int
}

func InitES(c *config.RunConfig) error {
	if c == nil || len(c.Elastic.Host) == 0 {
		return errors.New("elastic config missing")
	}
	client, err := elastic.NewClient(
		elastic.SetURL(c.Elastic.Host...),
		elastic.SetBasicAuth(c.Elastic.UserName, c.Elastic.Password),
		elastic.SetSniff(false),
	)
	if err != nil {
		zap.L().Error("create es client failed", zap.Error(err))
		return err
	}
	esIns = &ESDao{es: client}
	return nil
}

func ES() *ESDao {
	return esIns
}

// Ping 启动自检。
func (d *ESDao) Ping() error {
	if d == nil || d.es == nil {
		return errors.New("es not initialized")
	}
	_, err := d.es.CatHealth().Do(context.Background())
	if err != nil {
		return err
	}
	return nil
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

// ListRecords 直连 ES 旧注单索引，不经过 data-center。
func (d *ESDao) ListRecords(q RecordListQuery) ([]*RecordItem, int64, error) {
	if d == nil || d.es == nil {
		return nil, 0, errors.New("es not initialized")
	}
	if q.UserId == 0 {
		return []*RecordItem{}, 0, nil
	}
	page, size := normalizePageSize(q.Page, q.Size)
	querys := make([]elastic.Query, 0, 8)
	querys = append(querys, elastic.NewTermQuery("userId", q.UserId))
	querys = append(querys, elastic.NewTermQuery("complete", true))
	if q.GameId > 0 && strings.TrimSpace(q.Symbol) != "" {
		querys = append(querys, elastic.NewBoolQuery().Should(
			elastic.NewTermQuery("gameId", q.GameId),
			elastic.NewTermQuery("symbol", q.Symbol),
		).MinimumNumberShouldMatch(1))
	} else if strings.TrimSpace(q.Symbol) != "" {
		querys = append(querys, elastic.NewTermQuery("symbol", q.Symbol))
	} else if q.GameId > 0 {
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
		"gameId", "gameName", "init", "log",
	}
	from := (page - 1) * size
	resp, err := d.es.Search().
		Index(esindex.Settlement()).
		FetchSourceContext(elastic.NewFetchSourceContext(true).Include(source...)).
		Query(boolQuery).
		From(from).
		Size(size).
		Sort("playedDate", false).
		TrackTotalHits(true).
		Do(context.Background())
	if err != nil {
		zap.L().Error("ListRecords failed", zap.Error(err))
		return nil, 0, err
	}
	total := int64(0)
	if resp.Hits != nil && resp.Hits.TotalHits != nil {
		total = resp.Hits.TotalHits.Value
	}
	out := make([]*RecordItem, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		b, _ := hit.Source.MarshalJSON()
		item := &RecordItem{}
		_ = json.Unmarshal(b, item)
		out = append(out, item)
	}
	return out, total, nil
}

// GetRecordDetail 必须匹配 recordId + userId + gameId。
func (d *ESDao) GetRecordDetail(recordId string, userId, gameId uint32, symbol string) (*RecordItem, bool, error) {
	if d == nil || d.es == nil {
		return nil, false, errors.New("es not initialized")
	}
	recordId = strings.TrimSpace(recordId)
	if recordId == "" || userId == 0 || gameId == 0 {
		return nil, false, nil
	}
	idQueries := []elastic.Query{
		elastic.NewTermQuery("hash", recordId),
		elastic.NewTermQuery("roundID", recordId),
	}
	for _, idQuery := range idQueries {
		should := []elastic.Query{elastic.NewTermQuery("gameId", gameId)}
		if strings.TrimSpace(symbol) != "" {
			should = append(should, elastic.NewTermQuery("symbol", symbol))
		}
		boolQuery := elastic.NewBoolQuery().Must(
			idQuery,
			elastic.NewTermQuery("userId", userId),
			elastic.NewBoolQuery().Should(should...).MinimumNumberShouldMatch(1),
		)
		resp, err := d.es.Search().
			Index(esindex.Settlement()).
			Query(boolQuery).
			Size(1).
			Sort("playedDate", false).
			Do(context.Background())
		if err != nil {
			zap.L().Error("GetRecordDetail failed", zap.Error(err), zap.String("recordId", recordId))
			return nil, false, err
		}
		if resp == nil || resp.Hits == nil || len(resp.Hits.Hits) == 0 {
			continue
		}
		b, _ := resp.Hits.Hits[0].Source.MarshalJSON()
		item := &RecordItem{}
		if err := json.Unmarshal(b, item); err != nil {
			return nil, false, err
		}
		return item, true, nil
	}
	return nil, false, nil
}

// BillItem 对应 ES 流水索引 fg_gp_flowing_water（lottery SaveBill）。
// CreateTime 为 Unix 秒（lottery 写入 now.Unix()），不是毫秒。
//
// 新约定（见 lottery/doc/流水字段变更清单.md）：
//   bet   = 下注金额（负数；无下注为 0）
//   award = 到账金额（>=0；无到账为 0）——返奖/结算/回退都写在 award，没有 win 字段
// 旧数据可能没有 award：此时 bet 仍是账变 delta，读侧会归一化。
type BillItem struct {
	UserId         uint32  `json:"userId"`
	AgentId        uint32  `json:"agentId"`
	GameId         uint32  `json:"gameId"`
	Symbol         string  `json:"symbol"`
	Bet            float64 `json:"bet"`
	Award          float64 `json:"award"`
	CurrentScore   float64 `json:"currentScore"`
	Currency       string  `json:"currency"`
	CurrencySymbol string  `json:"currencySymbol"`
	CreateTime     int64   `json:"createTime"`
	RoundID        string  `json:"roundId"`
	FlowingWaterOn string  `json:"flowingWaterOn"`
	Desc           string  `json:"desc"`
	GameName       string  `json:"gameName"`
}

// normalizeBillAmounts 按新字段约定输出；兼容旧文档（无 award）。
// 归一化后：bet<=0 表示下注，award>=0 表示到账；净变动 = bet + award。
func normalizeBillAmounts(raw []byte, item *BillItem) {
	if item == nil {
		return
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return
	}
	if _, hasAward := probe["award"]; hasAward {
		// 新数据：纠正符号约定
		if item.Award < 0 {
			item.Award = 0
		}
		if item.Bet > 0 {
			// 防御：新字段下 bet 不应为正
			item.Bet = 0
		}
		return
	}
	// 旧数据：bet 是账变 delta
	if item.Bet > 0 {
		item.Award = item.Bet
		item.Bet = 0
	} else {
		item.Award = 0
		// item.Bet 保持负数或 0
	}
}

// normalizeFlowCreateTime 把查询时间统一成 Unix 秒，兼容调用方传毫秒。
func normalizeFlowCreateTime(msOrSec int64) int64 {
	if msOrSec <= 0 {
		return 0
	}
	// 毫秒时间戳约 1e12+，秒约 1e9。
	if msOrSec > 1_000_000_000_000 {
		return msOrSec / 1000
	}
	return msOrSec
}

type BillListQuery struct {
	UserId   uint32
	Symbol   string
	Currency string
	GameId   uint32
	StartMs  int64
	EndMs    int64
	Page     int
	Size     int
	// OnlySettlement 为 true 时只查 desc=返奖 的流水。
	OnlySettlement bool
}

// ListBills 直连 ES 流水索引，不经 data-center / lottery 中转。
func (d *ESDao) ListBills(q BillListQuery) ([]*BillItem, int64, error) {
	if d == nil || d.es == nil {
		return nil, 0, errors.New("es not initialized")
	}
	if q.UserId == 0 {
		return []*BillItem{}, 0, nil
	}
	page, size := normalizePageSize(q.Page, q.Size)
	querys := make([]elastic.Query, 0, 8)
	querys = append(querys, elastic.NewTermQuery("userId", q.UserId))
	if q.GameId > 0 && strings.TrimSpace(q.Symbol) != "" {
		querys = append(querys, elastic.NewBoolQuery().Should(
			elastic.NewTermQuery("gameId", q.GameId),
			elastic.NewTermQuery("symbol", q.Symbol),
		).MinimumNumberShouldMatch(1))
	} else if strings.TrimSpace(q.Symbol) != "" {
		querys = append(querys, elastic.NewTermQuery("symbol", q.Symbol))
	} else if q.GameId > 0 {
		querys = append(querys, elastic.NewTermQuery("gameId", q.GameId))
	}
	if strings.TrimSpace(q.Currency) != "" {
		querys = append(querys, elastic.NewMatchPhraseQuery("currency", q.Currency))
	}
	if q.OnlySettlement {
		// 玩家流水页只要 desc=返奖（兼容 text / keyword 映射）。
		querys = append(querys, elastic.NewBoolQuery().Should(
			elastic.NewTermQuery("desc.keyword", "返奖"),
			elastic.NewTermQuery("desc", "返奖"),
			elastic.NewMatchPhraseQuery("desc", "返奖"),
		).MinimumNumberShouldMatch(1))
	}
	startSec := normalizeFlowCreateTime(q.StartMs)
	endSec := normalizeFlowCreateTime(q.EndMs)
	if startSec > 0 || endSec > 0 {
		rq := elastic.NewRangeQuery("createTime")
		if startSec > 0 {
			rq = rq.Gte(startSec)
		}
		if endSec > 0 {
			rq = rq.Lte(endSec)
		}
		querys = append(querys, rq)
	}
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	from := (page - 1) * size
	resp, err := d.es.Search().
		Index(esindex.FlowingWater()).
		Query(boolQuery).
		From(from).
		Size(size).
		Sort("createTime", false).
		TrackTotalHits(true).
		Do(context.Background())
	if err != nil {
		zap.L().Error("ListBills failed", zap.Error(err))
		return nil, 0, err
	}
	total := int64(0)
	if resp.Hits != nil && resp.Hits.TotalHits != nil {
		total = resp.Hits.TotalHits.Value
	}
	out := make([]*BillItem, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		b, _ := hit.Source.MarshalJSON()
		item := &BillItem{}
		_ = json.Unmarshal(b, item)
		normalizeBillAmounts(b, item)
		if q.OnlySettlement && strings.TrimSpace(item.Desc) != "返奖" {
			continue
		}
		out = append(out, item)
	}
	if q.OnlySettlement && int64(len(out)) < total {
		// ES 文本分词可能多召回；以精确 desc 过滤后的本页条数为准时，总数用 count 重算。
		countQ := elastic.NewBoolQuery().Must(querys...)
		if cnt, cErr := d.es.Count().Index(esindex.FlowingWater()).Query(countQ).Do(context.Background()); cErr == nil {
			total = cnt
		}
	}
	return out, total, nil
}
