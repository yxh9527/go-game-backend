package common

import (
	"app/config"
	"app/entity/view"
	"app/esindex"
	"context"
	"crypto/md5"
	"fmt"
	"time"
	"web-api/dao"

	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
)

// 代理注单小时打点数据
func getSettlementStat(start, end time.Time) map[string]*view.DataAnalysisItem {
	//统计
	webIdAggs := elastic.NewTermsAggregation().Field("webId")
	aggs := elastic.NewTermsAggregation().Field("agentId").Size(10000)
	gameAggs := elastic.NewTermsAggregation().Field("gameId").Size(3000)
	gameAggs.SubAggregation("effectiveBetsTotal", elastic.NewSumAggregation().Field("exBet"))
	gameAggs.SubAggregation("profitLossTotal", elastic.NewSumAggregation().Field("exWin"))
	gameAggs.SubAggregation("userTotal", elastic.NewCardinalityAggregation().Field("userId"))
	gameAggs.SubAggregation("revenueTotal", elastic.NewSumAggregation().Field("exRevenue"))
	gameAggs.SubAggregation("pumpTotal", elastic.NewSumAggregation().Field("pump"))
	gameAggs.SubAggregation("chipsTotal", elastic.NewSumAggregation().Field("chips"))
	aggs.SubAggregation("gameId", gameAggs)
	webIdAggs.SubAggregation("agentId", aggs)
	query1 := elastic.NewRangeQuery("playedDate").Gte(start.UnixMilli()).Lt(end.UnixMilli())
	query2 := elastic.NewRangeQuery("isTourist").Lte(0)
	boolQuery := elastic.NewBoolQuery().Must(query1, query2)
	resp, err := dao.Es().Search().Index(esindex.Settlement()).Query(boolQuery).Aggregation("webId", webIdAggs).Size(0).Do(context.Background())
	if err != nil {
		zap.L().Error("获取注单打点数据异常", zap.Any("err", err))
		return nil
	}
	result := make(map[string]*view.DataAnalysisItem)
	webIdItems, ok := resp.Aggregations.Terms("webId")
	if ok {
		for _, w := range webIdItems.Buckets {
			webId := w.Key.(float64)
			items, ok := w.Aggregations.Terms("agentId")
			if ok {
				for _, item := range items.Buckets {
					agentId := item.Key.(float64)
					itemGames, ook := item.Aggregations.Terms("gameId")
					if !ook {
						continue
					}
					for _, id := range itemGames.Buckets {
						gameId := id.Key.(float64)
						effectivebets, _ := id.Sum("effectiveBetsTotal")
						profitLoss, _ := id.Sum("profitLossTotal")
						chips, _ := id.Sum("chipsTotal")
						pump, _ := id.Sum("pumpTotal")
						userCount, _ := id.Cardinality("userTotal")
						revenue, _ := id.Sum("revenueTotal")
						cfg := config.CfgIns.GetPoolCfgByGameId(int64(agentId), int64(gameId))
						tmp := &view.DataAnalysisItem{
							WebId:              int(webId),
							AgentId:            int(agentId),
							GameId:             int64(gameId),
							DocCount:           int(id.DocCount),
							ChipsTotal:         *chips.Value,
							ProfitLossTotal:    *profitLoss.Value,
							EffectiveBetsTotal: *effectivebets.Value,
							PumpTotal:          *pump.Value,
							UserTotal:          int(*userCount.Value),
							RevenueTotal:       *revenue.Value,
							GameName:           "",
						}
						if cfg != nil {
							if len(cfg.NameZH) > 0 {
								tmp.GameName = fmt.Sprintf("%s [%s]", cfg.Name, cfg.NameZH)
							} else {
								tmp.GameName = cfg.Name
							}
							tmp.Symbol = cfg.Symbol
						}
						result[fmt.Sprintf("%d-%d", int(agentId), int(gameId))] = tmp
					}
				}
			}
		}
	}
	return result
}

/*
"_source" : {
          "webId" : 23,
          "agentId" : 0,
          "gameId" : 3388,
          "symbol" : "vs50jfmulthold",
          "doc_count" : 898,
          "profitLossTotal" : 1002.4000000000001,
          "userTotal" : 1,
          "effectiveBetsTotal" : 1796,
          "chipsTotal" : 2496.8,
          "revenueTotal" : 74.56,
          "pumpTotal" : 0,
          "userBetsTotal" : 0,
          "date" : "2025-05-14 20:00:00",
          "startTime" : 1747220400,
          "endTime" : 1747224000,
          "createAt" : 1747224300,
          "gameName" : "Juicy Fruits Multihold [多汁水果多方持控]"
        }
*/

func getSettlementRangeData(start, end time.Time) map[string]*view.DataAnalysisItem {
	//统计
	webIdAggs := elastic.NewTermsAggregation().Field("webId")
	aggs := elastic.NewTermsAggregation().Field("agentId").Size(10000)
	gameAggs := elastic.NewTermsAggregation().Field("gameId").Size(3000)
	gameAggs.SubAggregation("effectiveBetsTotal", elastic.NewSumAggregation().Field("effectiveBetsTotal"))
	gameAggs.SubAggregation("profitLossTotal", elastic.NewSumAggregation().Field("profitLossTotal"))
	gameAggs.SubAggregation("userTotal", elastic.NewCardinalityAggregation().Field("userTotal"))
	gameAggs.SubAggregation("revenueTotal", elastic.NewSumAggregation().Field("revenueTotal"))
	gameAggs.SubAggregation("pumpTotal", elastic.NewSumAggregation().Field("pumpTotal"))
	gameAggs.SubAggregation("chipsTotal", elastic.NewSumAggregation().Field("chipsTotal"))
	gameAggs.SubAggregation("countTotal", elastic.NewSumAggregation().Field("doc_count"))
	aggs.SubAggregation("gameId", gameAggs)
	webIdAggs.SubAggregation("agentId", aggs)
	boolQuery := elastic.NewBoolQuery().Must(elastic.NewRangeQuery("startTime").Gte(start.Unix()), elastic.NewRangeQuery("endTime").Lte(end.Unix()))
	resp, err := dao.Es().Search().Index(esindex.DataAnalysisRange()).Query(boolQuery).Aggregation("webId", webIdAggs).Size(0).Do(context.Background())
	if err != nil {
		zap.L().Error("获取注单打点数据异常", zap.Any("err", err))
		return nil
	}
	result := make(map[string]*view.DataAnalysisItem)
	webIdItems, ok := resp.Aggregations.Terms("webId")
	if ok {
		for _, w := range webIdItems.Buckets {
			webId, _ := w.Key.(float64)
			items, ok := w.Aggregations.Terms("agentId")
			if ok {
				for _, item := range items.Buckets {
					agentId := item.Key.(float64)
					itemGames, ook := item.Aggregations.Terms("gameId")
					if !ook {
						continue
					}
					for _, id := range itemGames.Buckets {
						gameId := id.Key.(float64)
						effectivebets, _ := id.Sum("effectiveBetsTotal")
						profitLoss, _ := id.Sum("profitLossTotal")
						chips, _ := id.Sum("chipsTotal")
						pump, _ := id.Sum("pumpTotal")
						userCount, _ := id.Cardinality("userTotal")
						revenue, _ := id.Sum("revenueTotal")
						docCount, _ := id.Sum("countTotal")
						cfg := config.CfgIns.GetPoolCfgByGameId(int64(agentId), int64(gameId))
						tmp := &view.DataAnalysisItem{
							WebId:              int(webId),
							AgentId:            int(agentId),
							GameId:             int64(gameId),
							DocCount:           int(*docCount.Value),
							ChipsTotal:         *chips.Value,
							ProfitLossTotal:    *profitLoss.Value,
							EffectiveBetsTotal: *effectivebets.Value,
							PumpTotal:          *pump.Value,
							UserTotal:          int(*userCount.Value),
							RevenueTotal:       *revenue.Value,
							GameName:           "",
						}
						if cfg != nil {
							if len(cfg.NameZH) > 0 {
								tmp.GameName = fmt.Sprintf("%s [%s]", cfg.Name, cfg.NameZH)
							} else {
								tmp.GameName = cfg.Name
							}
							tmp.Symbol = cfg.Symbol
						}
						result[fmt.Sprintf("%d-%d", int(agentId), int(gameId))] = tmp
					}
				}
			}
		}
	}
	return result
}

// 保存数据
func DataAnalysisRangeSave(data map[string]*view.DataAnalysisItem) {
	bulkService := dao.Es().Bulk()
	records := make([]elastic.BulkableRequest, 0)
	for _, item := range data {
		records = append(records, elastic.NewBulkIndexRequest().Index(esindex.DataAnalysisRange()).Doc(item))
	}
	bulkService.Add(records...)
	_, err := bulkService.Do(context.Background())
	if err != nil {
		zap.L().Error("DataAnalysisRangeSave,批量插入数据失败", zap.Any("err", err), zap.Any("data", data))
	}
}

// 保存数据
func DataAnalysisSave(data map[string]*view.DataAnalysisItem) {
	bulkService := dao.Es().Bulk()
	records := make([]elastic.BulkableRequest, 0)
	for _, item := range data {
		records = append(records, elastic.NewBulkIndexRequest().Index(esindex.DataAnalysis()).Id(fmt.Sprintf("%x", md5.Sum([]byte(item.Date+fmt.Sprintf("%d", item.AgentId)+item.Symbol)))).Doc(item))
	}
	bulkService.Add(records...)
	_, err := bulkService.Do(context.Background())
	if err != nil {
		zap.L().Error("DataAnalysisRangeSave,批量插入数据失败", zap.Any("err", err), zap.Any("data", data))
	}
}

// 代理每小时打点数据
func AgentAggsByHour() {
	now := time.Now()
	//上一个小时的结束时间
	end := now.Add(-time.Duration(now.Second()+now.Minute()*60) * time.Second)
	//上一个小时的开始时间
	start := end.Add(-time.Hour)
	//统计区间内注单的统计数据
	settlementData := getSettlementStat(start, end)
	//合并保存数据
	for _, v := range settlementData {
		v.Date = now.Format("2006-01-02 15:00:00")
		v.StartTime = start.Unix()
		v.EndTime = end.Unix()
		v.CreateAt = end.Unix()
	}
	zap.L().Debug("=====>", zap.Any("start", start), zap.Any("end", end), zap.Any("settlementData", settlementData))
	if len(settlementData) > 0 {
		DataAnalysisRangeSave(settlementData)
	}
}

// 代理每天打点数据
func AgentAggsByDay() {
	now := time.Now()
	//上一个小时的结束时间
	end := now.Add(-time.Duration(now.Hour()*60*60+now.Second()+now.Minute()*60) * time.Second)
	//上一个小时的开始时间
	start := end.Add(-time.Duration(24*60*60) * time.Second)
	//统计区间内注单的统计数据
	settlementData := getSettlementRangeData(start, end)
	for _, v := range settlementData {
		v.Date = start.Format("2006-01-02")
		v.StartTime = start.Unix()
		v.EndTime = end.Unix()
		v.CreateAt = start.Unix()
	}
	zap.L().Debug("day aggs", zap.Any("start", start), zap.Any("end", end), zap.Any("settlementData", settlementData))
	if len(settlementData) > 0 {
		DataAnalysisSave(settlementData)
	}
}

func AgentAggsNow() {
	now := time.Now()
	start := now.Add(-time.Duration(now.Hour()*60*60+now.Second()+now.Minute()*60) * time.Second)
	end := time.Now()
	//统计区间内注单的统计数据
	settlementData := getSettlementStat(start, end)
	for _, v := range settlementData {
		v.Date = now.Format("2006-01-02")
		v.StartTime = start.Unix()
		v.EndTime = end.Unix()
		v.CreateAt = start.Unix()
	}
	zap.L().Debug("aggs now", zap.Any("start", start), zap.Any("end", end), zap.Any("settlementData", len(settlementData)))
	if len(settlementData) > 0 {
		DataAnalysisSave(settlementData)
	}
}
