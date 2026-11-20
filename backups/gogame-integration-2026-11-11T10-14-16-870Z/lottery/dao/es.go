package dao

import (
	"app/config"
	"app/entity"
	"app/esindex"
	"context"
	"crypto/md5"
	"fmt"

	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
)

func InitES(c *config.RunConfig) (*elastic.Client, error) {
	strs := []string{}
	strs = append(strs, c.Elastic.Host...)
	if client, err := elastic.NewClient(elastic.SetURL(strs...), elastic.SetBasicAuth(c.Elastic.UserName, c.Elastic.Password), elastic.SetSniff(false)); err == nil {
		return client, nil
	} else {
		zap.L().Error("创建es客户端失败", zap.Error(err), zap.Any("strs", c.Elastic), zap.Any("strs", strs))
		return nil, err
	}
}

type ESDao struct {
	Client *elastic.Client
}

func NewESDao(client *elastic.Client) *ESDao {
	return &ESDao{Client: client}
}

type PoolRecordLog struct {
	AgentId    uint32  `json:"agentId"`
	Symbol     string  `json:"symbol"`
	AreaId     uint32  `json:"areaId"`
	PoolValue  float64 `json:"poolValue"`
	Normal     float64 `json:"normal"`
	NormalRate float64 `json:"normalRate"`
	Min        float64 `json:"min"`
	MinRate    float64 `json:"minRate"`
	Max        float64 `json:"max"`
	MaxRate    float64 `json:"maxRate"`
	Ctl        float64 `json:"ctl"`
	Revenue    float64 `json:"revenue"`
	CreateTime int64   `json:"createTime"`
}

func (e *ESDao) Add(log *PoolRecordLog) {
	if _, err := e.Client.Index().Index(esindex.PoolRecordLog()).BodyJson(log).Do(context.Background()); err != nil {
		zap.L().Error("数据入库失败", zap.Error(err))
		return
	}
}

func (esDao *ESDao) BulkRecordsSave(data []*entity.CacheRecordsReq) error {
	bulkService := esDao.Client.Bulk()
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
	bulkService := esDao.Client.Bulk()
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

// func (esDao *ESDao) GetGameState(req *services.SlotsGetGameStateReq) *services.SlotsSaveGameStateReq {
// 	querys := make([]elastic.Query, 0)
// 	querys = append(querys, elastic.NewTermQuery("userId", req.UserId))
// 	querys = append(querys, elastic.NewTermQuery("symbol", req.Symbol))
// 	querys = append(querys, elastic.NewMatchPhraseQuery("currency", req.Currency))
// 	// zap.L().Debug("获取游戏状态", zap.Any("req", req))
// 	boolQuery := elastic.NewBoolQuery().Must(querys...)
// 	resp, err := esDao.Client.Search().Index(esindex.GameStates()).
// 		Query(boolQuery).
// 		Pretty(true).
// 		Do(context.Background())
// 	if err != nil {
// 		zap.L().Error("获取游戏状态失败", zap.Any("err", err))
// 		return nil
// 	}
// 	records := make([]*services.SlotsSaveGameStateReq, 0, 8)
// 	for _, v := range resp.Hits.Hits {
// 		b, _ := v.Source.MarshalJSON()
// 		r := &services.SlotsSaveGameStateReq{}
// 		json.Unmarshal(b, r)
// 		records = append(records, r)
// 	}
// 	if len(records) > 0 {
// 		return records[0]
// 	}
// 	return nil
// }
