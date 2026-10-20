package dao

import (
	"app/config"
	"app/esindex"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// 玩家信息
type EsUserInfo struct {
	UserId                      uint32 //玩家id
	ProfLoss, EffBet, Pump, Bet float64
}

type EsInfo struct {
	es *elastic.Client
}

type QznnSettlementDetail struct {
	Details []struct {
		SettlementItems struct {
			Qznn struct {
				PlayerId uint32 `json:"player_id"`
				IsBanker bool   `json:"is_banker"`
			} `json:"qznn"`
		} `json:"SettlementDetail"`
	} `json:"details"`
}

type tracelog struct{}

// 实现输出
func (tracelog) Printf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

func initES(c *config.RunConfig) (*elastic.Client, error) {
	strs := []string{}
	for _, v := range c.Elastic.Host {
		strs = append(strs, v)
	}
	if client, err := elastic.NewClient(
		elastic.SetURL(strs...),
		elastic.SetBasicAuth(c.Elastic.UserName, c.Elastic.Password),
		elastic.SetSniff(false),
		elastic.SetErrorLog(log.New(os.Stderr, "ELASTIC ", log.LstdFlags)),
		elastic.SetInfoLog(log.New(os.Stdout, "", log.LstdFlags)),
		elastic.SetTraceLog(new(tracelog))); err == nil {
		return client, nil
	} else {
		zap.L().Error("创建es客户端失败", zap.Error(err), zap.Any("strs", c), zap.Any("strs", strs))
		return nil, err
	}
}

func NewEsInfo(esConfig *config.RunConfig) *EsInfo {
	es := &EsInfo{}
	if client, err := initES(esConfig); err == nil {
		es.es = client
		return es
	} else {
		zap.L().Error("创建es链接失败", zap.Error(err))
	}
	return nil
}

// 获取所有注单数据
func (esi *EsInfo) GetAllSettlementData(bTime, eTime int64) []*config.DataItem {
	startTm := time.Now().UnixNano()
	settlementData := make([]*config.DataItem, 0, 128)
	query := elastic.NewBoolQuery()
	includesFields := []string{"userId", "gameId", "agentId", "bet", "win", "playedDate"}
	source := elastic.NewFetchSourceContext(true).Include(includesFields...)
	query1 := query.Must(
		elastic.NewRangeQuery("isTourist").Lte(0),
		elastic.NewRangeQuery("playedDate").Gte(bTime*1000).Lt(eTime*1000),
	)
	valueDecode := func(jds []*elastic.SearchHit) {
		for _, v := range jds {
			data := &config.DataItem{}
			if er := json.Unmarshal(v.Source, data); er != nil {
				zap.L().Error("反序列化注单信息失败", zap.Error(er))
				return
			}
			settlementData = append(settlementData, data)
		}
	}
	if res, err := esi.es.Scroll(esindex.Settlement()).Scroll("30s").
		Size(10000).
		Query(query1).
		FetchSourceContext(source).
		Do(context.Background()); err == nil {
		valueDecode(res.Hits.Hits)
		if len(res.Hits.Hits) >= 10000 {
			scoreId := res.ScrollId
			for {
				if scoreId == "" {
					endTm := time.Now().UnixNano()
					zap.L().Debug("GetAllSettlementData", zap.Any("耗时", endTm-startTm))
					return settlementData
				}
				if result, e := esi.es.Scroll(esindex.Settlement()).Scroll("30s").
					Pretty(true).
					ScrollId(scoreId).
					Size(10000).
					FetchSourceContext(source).
					Do(context.Background()); e == nil {
					valueDecode(result.Hits.Hits)
					if len(result.Hits.Hits) < 10000 {
						break
					}
					scoreId = result.ScrollId
				} else {
					endTm := time.Now().UnixNano()
					zap.L().Debug("GetAllSettlementData", zap.Any("耗时", endTm-startTm))
					return settlementData
				}
			}
		}
	} else {
		if err.Error() == "EOF" {
			return settlementData
		}
	}

	endTm := time.Now().UnixNano()
	zap.L().Debug("GetAllSettlementData", zap.Any("耗时", endTm-startTm))

	return settlementData
}

// 获取指定代理的活跃人数
func (esi *EsInfo) GetRegisterActive(agentId, bTime, eTime int64) map[uint32]bool {
	userIds := make(map[uint32]bool)
	query := elastic.NewBoolQuery()
	includesFields := []string{"userId", "gameId", "agentId", "bet", "win", "playedDate"}
	source := elastic.NewFetchSourceContext(true).Include(includesFields...)
	query1 := query.Must(elastic.NewRangeQuery("playedDate").Lt(eTime * 1000))
	query2 := query.Must(elastic.NewRangeQuery("playedDate").Gte(bTime * 1000))
	query3 := query.Must(elastic.NewTermQuery("agentId", agentId))

	valueDecode := func(jds []*elastic.SearchHit) {
		for _, v := range jds {
			data := &config.DataItem{}
			if er := json.Unmarshal(v.Source, data); er != nil {
				zap.L().Error("反序列化注单信息失败", zap.Error(er))
				return
			}
			userIds[uint32(data.UserId)] = true
		}
	}

	if res, err := esi.es.Scroll(esindex.Settlement()).Scroll("30s").
		Size(1000).
		Query(query1).
		Query(query2).
		Query(query3).
		FetchSourceContext(source).
		Do(context.Background()); err == nil {
		valueDecode(res.Hits.Hits)
		if len(res.Hits.Hits) >= 1000 {
			scoreId := res.ScrollId
			for {
				if scoreId == "" {
					return userIds
				}
				if result, e := esi.es.Scroll(esindex.Settlement()).Scroll("30s").
					ScrollId(scoreId).
					Size(1000).
					Query(query1).
					Query(query2).
					Query(query3).
					FetchSourceContext(source).
					Do(context.Background()); e == nil {
					valueDecode(result.Hits.Hits)
					if len(result.Hits.Hits) < 1000 {
						break
					}
					scoreId = result.ScrollId
				} else {
					return userIds
				}
			}
		}
	} else {
		if err.Error() == "EOF" {
			return userIds
		}
	}
	return userIds
}

// 获取指定代理和游戏的活跃人数
func (esi *EsInfo) GetAgentRegisterActiveByGameId(agentId, gameId, bTime, eTime int64) map[uint32]bool {
	userIds := make(map[uint32]bool)
	query := elastic.NewBoolQuery()
	source := elastic.NewFetchSourceContext(true).Exclude("init", "log")
	query1 := query.Must(elastic.NewRangeQuery("playedDate").Lt(eTime * 1000))
	query2 := query.Must(elastic.NewRangeQuery("playedDate").Gte(bTime * 1000))
	query3 := query.Must(elastic.NewTermQuery("gameId", gameId))
	query4 := query.Must(elastic.NewTermQuery("agentId", agentId))

	valueDecode := func(jds []*elastic.SearchHit) {
		for _, v := range jds {
			data := &config.DataItem{}
			if er := json.Unmarshal(v.Source, data); er != nil {
				zap.L().Error("反序列化注单信息失败", zap.Error(er))
				return
			}
			userIds[uint32(data.UserId)] = true
		}
	}

	if res, err := esi.es.Scroll(esindex.Settlement()).Scroll("30s").
		Size(1000).
		Query(query1).
		Query(query2).
		Query(query3).
		Query(query4).
		FetchSourceContext(source).
		Do(context.Background()); err == nil {
		valueDecode(res.Hits.Hits)
		if len(res.Hits.Hits) >= 1000 {
			scoreId := res.ScrollId
			for {
				if scoreId == "" {
					return userIds
				}
				if result, e := esi.es.Scroll(esindex.Settlement()).Scroll("30s").
					ScrollId(scoreId).
					Size(1000).
					Query(query1).
					Query(query2).
					Query(query3).
					Query(query4).
					FetchSourceContext(source).
					Do(context.Background()); e == nil {
					valueDecode(result.Hits.Hits)
					if len(result.Hits.Hits) < 1000 {
						break
					}
					scoreId = result.ScrollId
				} else {
					return userIds
				}
			}
		}
	} else {
		if err.Error() == "EOF" {
			return userIds
		}
	}
	return userIds
}

// 获取所有代理指定游戏的活跃人数
func (esi *EsInfo) GetRegisterActiveByGameId(gameId, bTime, eTime int64) map[uint32]bool {
	userIds := make(map[uint32]bool)
	query := elastic.NewBoolQuery()
	source := elastic.NewFetchSourceContext(true).Exclude("init", "log")
	query1 := query.Must(elastic.NewRangeQuery("playedDate").Lt(eTime * 1000))
	query2 := query.Must(elastic.NewRangeQuery("playedDate").Gte(bTime * 1000))
	query3 := query.Must(elastic.NewTermQuery("gameId", gameId))

	valueDecode := func(jds []*elastic.SearchHit) {
		for _, v := range jds {
			data := &config.DataItem{}
			if er := json.Unmarshal(v.Source, data); er != nil {
				zap.L().Error("反序列化注单信息失败", zap.Error(er))
				return
			}
			userIds[uint32(data.UserId)] = true
		}
	}

	if res, err := esi.es.Scroll(esindex.Settlement()).Scroll("30s").
		Size(1000).
		Query(query1).
		Query(query2).
		Query(query3).
		FetchSourceContext(source).
		Do(context.Background()); err == nil {
		valueDecode(res.Hits.Hits)
		if len(res.Hits.Hits) >= 1000 {
			scoreId := res.ScrollId
			for {
				if scoreId == "" {
					return userIds
				}
				if result, e := esi.es.Scroll(esindex.Settlement()).Scroll("30s").
					ScrollId(scoreId).
					Size(1000).
					Query(query1).
					Query(query2).
					Query(query3).
					FetchSourceContext(source).
					Do(context.Background()); e == nil {
					valueDecode(result.Hits.Hits)
					if len(result.Hits.Hits) < 1000 {
						break
					}
					scoreId = result.ScrollId
				} else {
					return userIds
				}
			}
		}
	} else {
		if err.Error() == "EOF" {
			return userIds
		}
	}
	return userIds
}

// 获取所有代理和所有游戏的活跃人数
func (esi *EsInfo) GetAllRegisterActive(bTime, eTime int64) map[uint32]bool {
	userIds := make(map[uint32]bool)
	query := elastic.NewBoolQuery()
	source := elastic.NewFetchSourceContext(true).Exclude("init", "log")
	query1 := query.Must(elastic.NewRangeQuery("playedDate").Lt(eTime * 1000))
	query2 := query.Must(elastic.NewRangeQuery("playedDate").Gte(bTime * 1000))

	valueDecode := func(jds []*elastic.SearchHit) {
		for _, v := range jds {
			data := &config.DataItem{}
			if er := json.Unmarshal(v.Source, data); er != nil {
				zap.L().Error("反序列化注单信息失败", zap.Error(er))
				return
			}
			userIds[uint32(data.UserId)] = true
		}
	}

	if res, err := esi.es.Scroll(esindex.Settlement()).Scroll("30s").
		Size(1000).
		Query(query1).
		Query(query2).
		FetchSourceContext(source).
		Do(context.Background()); err == nil {
		valueDecode(res.Hits.Hits)
		if len(res.Hits.Hits) >= 1000 {
			scoreId := res.ScrollId
			for {
				if scoreId == "" {
					return userIds
				}
				if result, e := esi.es.Scroll(esindex.Settlement()).Scroll("30s").
					ScrollId(scoreId).
					Size(1000).
					Query(query1).
					Query(query2).
					FetchSourceContext(source).
					Do(context.Background()); e == nil {
					valueDecode(result.Hits.Hits)
					if len(result.Hits.Hits) < 1000 {
						break
					}
					scoreId = result.ScrollId
				} else {
					return userIds
				}
			}
		}
	} else {
		if err.Error() == "EOF" {
			return userIds
		}
	}
	return userIds
}

// 获取所有的活跃玩家包含老用户
func (esi *EsInfo) GetAllRegisterActiveByGameId(bTime, eTime, gameId int64) int64 {
	source := elastic.NewFetchSourceContext(true).Exclude("init", "log")
	if gameId != 0 {
		query := elastic.NewBoolQuery()
		query1 := query.Must(elastic.NewTermQuery("gameId", gameId))
		query3 := query.Must(elastic.NewRangeQuery("playedDate").Lt(eTime * 1000))
		query4 := query.Must(elastic.NewRangeQuery("playedDate").Gte(bTime * 1000))
		queryAggs := elastic.NewCardinalityAggregation().Field("userId")
		if res, e := esi.es.Search().Size(0).Index().Index(esindex.Settlement()).
			Query(query1).
			Query(query3).
			Query(query4).
			FetchSourceContext(source).
			Aggregation("count", queryAggs).
			Do(context.Background()); e == nil {
			c, b := res.Aggregations.Cardinality("count")
			if b {
				return int64(*c.Value)
			}
		}
	} else {
		query := elastic.NewBoolQuery()
		query1 := query.Must(elastic.NewRangeQuery("playedDate").Lt(eTime * 1000))
		query2 := query.Must(elastic.NewRangeQuery("playedDate").Gte(bTime * 1000))
		queryAggs := elastic.NewCardinalityAggregation().Field("userId")
		zap.L().Debug("查询所有活跃", zap.Any("bTimg", bTime), zap.Any("eTime", eTime))
		if res, e := esi.es.Search().Size(0).Index().Index(esindex.Settlement()).
			Query(query1).
			Query(query2).
			Aggregation("count", queryAggs).
			Do(context.Background()); e == nil {
			c, b := res.Aggregations.Cardinality("count")
			if b {
				return int64(*c.Value)
			}
		} else {
			zap.L().Error("error", zap.Any("msg", e))
		}
	}
	return 0
}

// 获取代理指定游戏的数据统计
func (esi *EsInfo) GetAgentEffProByGameId(agentId, gameId, bTime, eTime int64) (decimal.Decimal, decimal.Decimal) {
	source := elastic.NewFetchSourceContext(true).Exclude("init", "log")
	query := elastic.NewBoolQuery()
	query1 := query.Must(elastic.NewTermQuery("agentId", agentId))
	query2 := query.Must(elastic.NewTermQuery("gameId", gameId))
	query3 := query.Must(elastic.NewRangeQuery("playedDate").Lt(eTime * 1000))
	query4 := query.Must(elastic.NewRangeQuery("playedDate").Gte(bTime * 1000))
	query5 := query.Must(elastic.NewRangeQuery("isTourist").Lte(0))

	aggsEff := elastic.NewSumAggregation().Field("bet")
	aggsPro := elastic.NewSumAggregation().Field("win")
	if res, e := esi.es.Search().Index().Size(0).Index(esindex.Settlement()).
		Query(query1).
		Query(query2).
		Query(query3).
		Query(query4).
		Query(query5).
		FetchSourceContext(source).
		Aggregation("eff", aggsEff).
		Aggregation("pro", aggsPro).
		Do(context.Background()); e == nil {
		eff, _ := res.Aggregations.Sum("eff")
		pro, _ := res.Aggregations.Sum("pro")
		return decimal.NewFromFloat(*eff.Value), decimal.NewFromFloat(*pro.Value)
	}
	return decimal.Zero, decimal.Zero
}

// 获取所有代理指定游戏的数据统计
func (esi *EsInfo) GetEffProByGameId(gameId, bTime, eTime int64) (decimal.Decimal, decimal.Decimal) {
	source := elastic.NewFetchSourceContext(true).Exclude("init", "log")
	query := elastic.NewBoolQuery()
	query2 := query.Must(elastic.NewTermQuery("gameId", gameId))
	query3 := query.Must(elastic.NewRangeQuery("playedDate").Lt(eTime * 1000))
	query4 := query.Must(elastic.NewRangeQuery("playedDate").Gte(bTime * 1000))
	query5 := query.Must(elastic.NewRangeQuery("isTourist").Lte(0))
	aggsEff := elastic.NewSumAggregation().Field("bet")
	aggsPro := elastic.NewSumAggregation().Field("win")
	if res, e := esi.es.Search().Index().Size(0).Index(esindex.Settlement()).
		Query(query2).
		Query(query3).
		Query(query4).
		Query(query5).
		FetchSourceContext(source).
		Aggregation("eff", aggsEff).
		Aggregation("pro", aggsPro).
		Do(context.Background()); e == nil {
		eff, _ := res.Aggregations.Sum("eff")
		pro, _ := res.Aggregations.Sum("pro")
		return decimal.NewFromFloat(*eff.Value), decimal.NewFromFloat(*pro.Value)
	}
	return decimal.Zero, decimal.Zero
}

// 获取所有代理所有游戏的数据统计
func (esi *EsInfo) GetAllEffPro(bTime, eTime int64) (decimal.Decimal, decimal.Decimal) {
	source := elastic.NewFetchSourceContext(true).Exclude("init", "log")
	query := elastic.NewBoolQuery()
	query3 := query.Must(elastic.NewRangeQuery("playedDate").Lt(eTime * 1000))
	query4 := query.Must(elastic.NewRangeQuery("playedDate").Gte(bTime * 1000))
	query5 := query.Must(elastic.NewRangeQuery("isTourist").Lte(0))
	aggsEff := elastic.NewSumAggregation().Field("bet")
	aggsPro := elastic.NewSumAggregation().Field("win")
	if res, e := esi.es.Search().Index().Size(0).Index(esindex.Settlement()).
		Query(query3).
		Query(query4).
		Query(query5).
		FetchSourceContext(source).
		Aggregation("eff", aggsEff).
		Aggregation("pro", aggsPro).
		Do(context.Background()); e == nil {
		eff, _ := res.Aggregations.Sum("eff")
		pro, _ := res.Aggregations.Sum("pro")
		return decimal.NewFromFloat(*eff.Value), decimal.NewFromFloat(*pro.Value)
	}
	return decimal.Zero, decimal.Zero
}

// 获取代理指定游戏注单数
func (esi *EsInfo) GetAgentBetsCountByGameId(agentId, gameId, bTime, eTime int64) int64 {
	query := elastic.NewBoolQuery()
	query1 := query.Must(elastic.NewTermQuery("agentId", agentId))
	query2 := query.Must(elastic.NewTermQuery("gameId", gameId))
	query3 := query.Must(elastic.NewRangeQuery("playedDate").Lt(eTime * 1000))
	query4 := query.Must(elastic.NewRangeQuery("playedDate").Gte(bTime * 1000))
	query5 := query.Must(elastic.NewRangeQuery("isTourist").Lte(0))
	if res, e := esi.es.Count().Index().Index(esindex.Settlement()).
		Query(query1).
		Query(query2).
		Query(query3).
		Query(query4).
		Query(query5).
		Do(context.Background()); e == nil {
		return res
	}
	return 0
}

// 获取所有代理指定游戏注单数
func (esi *EsInfo) GetBetsCountByGameId(gameId, bTime, eTime int64) int64 {
	query := elastic.NewBoolQuery()
	query2 := query.Must(elastic.NewTermQuery("gameId", gameId))
	query3 := query.Must(elastic.NewRangeQuery("playedDate").Lt(eTime * 1000))
	query4 := query.Must(elastic.NewRangeQuery("playedDate").Gte(bTime * 1000))
	query5 := query.Must(elastic.NewRangeQuery("isTourist").Lte(0))
	if res, e := esi.es.Count().Index().Index(esindex.Settlement()).
		Query(query2).
		Query(query3).
		Query(query4).
		Query(query5).
		Do(context.Background()); e == nil {
		return res
	}
	return 0
}

// 获取所有代理所有游戏注单数
func (esi *EsInfo) GetAllBetsCount(bTime, eTime int64) int64 {
	query := elastic.NewBoolQuery()
	query3 := query.Must(elastic.NewRangeQuery("playedDate").Lt(eTime * 1000))
	query4 := query.Must(elastic.NewRangeQuery("playedDate").Gte(bTime * 1000))
	query5 := query.Must(elastic.NewRangeQuery("isTourist").Lte(0))
	if res, e := esi.es.Count().Index().Index(esindex.Settlement()).
		Query(query3).
		Query(query4).
		Query(query5).
		Do(context.Background()); e == nil {
		return res
	}
	return 0
}

// 获取代理指定游戏活跃玩家的次留点数
func (esi *EsInfo) GetAgentKeepAliveByGameId(agentId, gameId, bTime, eTime int64) (uint32, uint32) {
	var keepAlive uint32 = 0
	var notKeepAlive uint32 = 0
	yesterdayPlayers := esi.GetAgentRegisterActiveByGameId(agentId, gameId, bTime, eTime)
	eTime = bTime
	bTime = bTime - 24*60*60
	theDayBeforeYesterdayPlayers := esi.GetAgentRegisterActiveByGameId(agentId, gameId, bTime, eTime)

	zap.L().Debug("次留点数相关",
		zap.Any("agentId", agentId),
		zap.Any("gameId", gameId),
		zap.Any("yesterdayPlayers", yesterdayPlayers),
		zap.Any("theDayBeforeYesterdayPlayers", theDayBeforeYesterdayPlayers))

	// 前天活跃并且昨天活跃，则次留点数加1， 前天活跃且昨天没活跃，则未次留点加1
	for user, _ := range theDayBeforeYesterdayPlayers {
		if _, ok := yesterdayPlayers[user]; ok {
			keepAlive += 1
		} else {
			notKeepAlive += 1
		}
	}
	return keepAlive, notKeepAlive
}

// 获取所有代理指定游戏活跃玩家的次留点数
func (esi *EsInfo) GetKeepAliveByGameId(gameId, bTime, eTime int64) (uint32, uint32) {
	var keepAlive uint32 = 0
	var notKeepAlive uint32 = 0
	yesterdayPlayers := esi.GetRegisterActiveByGameId(gameId, bTime, eTime)
	eTime = bTime
	bTime = bTime - 24*60*60
	theDayBeforeYesterdayPlayers := esi.GetRegisterActiveByGameId(gameId, bTime, eTime)

	zap.L().Debug("次留点数相关",
		zap.Any("gameId", gameId),
		zap.Any("yesterdayPlayers", yesterdayPlayers),
		zap.Any("theDayBeforeYesterdayPlayers", theDayBeforeYesterdayPlayers))

	// 前天活跃并且昨天活跃，则次留点数加1， 前天活跃且昨天没活跃，则未次留点加1
	for user := range theDayBeforeYesterdayPlayers {
		if _, ok := yesterdayPlayers[user]; ok {
			keepAlive += 1
		} else {
			notKeepAlive += 1
		}
	}
	return keepAlive, notKeepAlive
}

// 获取所有代理所有游戏活跃玩家的次留点数
func (esi *EsInfo) GetAllKeepAlive(bTime, eTime int64) (uint32, uint32) {
	var keepAlive uint32 = 0
	var notKeepAlive uint32 = 0
	yesterdayPlayers := esi.GetAllRegisterActive(bTime, eTime)
	eTime = bTime
	bTime = bTime - 24*60*60
	theDayBeforeYesterdayPlayers := esi.GetAllRegisterActive(bTime, eTime)

	zap.L().Debug("次留点数相关(所有游戏，所有代理)",
		zap.Any("yesterdayPlayers", yesterdayPlayers),
		zap.Any("theDayBeforeYesterdayPlayers", theDayBeforeYesterdayPlayers))

	// 前天活跃并且昨天活跃，则次留点数加1， 前天活跃且昨天没活跃，则未次留点加1
	for user := range theDayBeforeYesterdayPlayers {
		if _, ok := yesterdayPlayers[user]; ok {
			keepAlive += 1
		} else {
			notKeepAlive += 1
		}
	}
	return keepAlive, notKeepAlive
}

// 获取所有代理游戏玩家盈亏排行
func (esi *EsInfo) GetProRank(bTime, eTime uint32, isAsc bool) []*config.RankResult {
	var result []*config.RankResult
	source := elastic.NewFetchSourceContext(true).Exclude("init", "log")
	query := elastic.NewBoolQuery()
	query2 := query.Must(elastic.NewRangeQuery("playedDate").Lt(eTime * 1000))
	query3 := query.Must(elastic.NewRangeQuery("playedDate").Gte(bTime * 1000))
	query5 := query.Must(elastic.NewRangeQuery("isTourist").Lte(0))
	aggsEff := elastic.NewSumAggregation().Field("bet")
	aggsPro := elastic.NewSumAggregation().Field("win")
	termAggs := elastic.NewTermsAggregation().Field("userId").Size(20).
		OrderByAggregation("pro", isAsc).
		SubAggregation("eff", aggsEff).
		SubAggregation("pro", aggsPro)
	if res, e := esi.es.Search().Index().Size(0).Index(esindex.Settlement()).
		Query(query2).
		Query(query3).
		Query(query5).
		FetchSourceContext(source).
		Aggregation("user_id", termAggs).
		Do(context.Background()); e == nil {
		if data, ok := res.Aggregations.Terms("user_id"); ok {
			i := len(data.Buckets)
			for j := 0; j < i; j++ {
				rItem := &config.RankResult{}
				item := data.Buckets[j]
				eff, effOk := item.Sum("eff")
				pro, proOk := item.Sum("pro")
				tmpId, _ := strconv.Atoi(item.KeyNumber.String())
				rItem.UserId = uint32(tmpId)
				if effOk {
					rItem.Eff = decimal.NewFromFloat(*eff.Value)
				}
				if proOk {
					rItem.Pro = decimal.NewFromFloat(*pro.Value)
				}
				result = append(result, rItem)
			}
		}
	}
	return result
}

// 获取代理指定游戏玩家盈亏排行
func (esi *EsInfo) GetAgentProRank(agentId, bTime, eTime uint32, isAsc bool) []*config.RankResult {
	var result []*config.RankResult
	source := elastic.NewFetchSourceContext(true).Exclude("init", "log")
	query := elastic.NewBoolQuery()
	query1 := query.Must(elastic.NewTermQuery("agentId", agentId))
	query2 := query.Must(elastic.NewRangeQuery("playedDate").Lt(eTime * 1000))
	query3 := query.Must(elastic.NewRangeQuery("playedDate").Gte(bTime * 1000))
	query5 := query.Must(elastic.NewRangeQuery("isTourist").Lte(0))
	aggsEff := elastic.NewSumAggregation().Field("bet")
	aggsPro := elastic.NewSumAggregation().Field("win")
	termAggs := elastic.NewTermsAggregation().Field("userId").Size(20).
		OrderByAggregation("pro", isAsc).
		SubAggregation("eff", aggsEff).
		SubAggregation("pro", aggsPro)
	if res, e := esi.es.Search().Index().Size(0).Index(esindex.Settlement()).
		Query(query1).
		Query(query2).
		Query(query3).
		Query(query5).
		FetchSourceContext(source).
		Aggregation("user_id", termAggs).
		Do(context.Background()); e == nil {
		if data, ok := res.Aggregations.Terms("user_id"); ok {
			i := len(data.Buckets)
			for j := 0; j < i; j++ {
				rItem := &config.RankResult{}
				item := data.Buckets[j]
				rItem.AgentId = agentId
				eff, effOk := item.Sum("eff")
				pro, proOk := item.Sum("pro")
				tmpId, _ := strconv.Atoi(item.KeyNumber.String())
				rItem.UserId = uint32(tmpId)
				if effOk {
					rItem.Eff = decimal.NewFromFloat(*eff.Value)
				}
				if proOk {
					rItem.Pro = decimal.NewFromFloat(*pro.Value)
				}
				result = append(result, rItem)
			}
		}
	}
	return result
}
