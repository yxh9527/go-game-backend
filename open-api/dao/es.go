package dao

import (
	"app/config"
	"app/esindex"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
)

type EsClient struct {
	es *elastic.Client
}

var esClient *EsClient = nil

type tracelog struct{}

// 实现输出
func (tracelog) Printf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

func InitES(c *config.RunConfig) error {
	strs := []string{}
	for _, v := range c.Elastic.Host {
		strs = append(strs, v)
	}
	if c, err := elastic.NewClient(
		elastic.SetURL(strs...),
		elastic.SetBasicAuth(c.Elastic.UserName, c.Elastic.Password),
		elastic.SetSniff(false),
		elastic.SetErrorLog(log.New(os.Stderr, "ELASTIC ", log.LstdFlags)),
		elastic.SetInfoLog(log.New(os.Stdout, "", log.LstdFlags)),
		elastic.SetTraceLog(new(tracelog))); err == nil {
		esClient = &EsClient{es: c}
	} else {
		zap.L().Error("创建es客户端失败", zap.Error(err), zap.Any("strs", c), zap.Any("strs", strs))
		return errors.New("创建es链接失败")
	}
	return nil
}

func Es() *EsClient {
	return esClient
}

type SettlementRecord struct {
	Id             string  `json:"id"`
	Account        string  `json:"account"`
	UserId         int64   `json:"userId"`
	GameName       string  `json:"gameName"`
	GameId         int64   `json:"gameId"`
	AgentId        int64   `json:"agentId"`
	PlayedDate     int64   `json:"playedDate"`
	RoundId        string  `json:"roundID"`
	Bet            float64 `json:"bet"`
	Win            float64 `json:"win"`
	Balance        float64 `json:"balance"`
	BalanceCash    float64 `json:"balance_cash"`
	BalanceBonus   float64 `json:"balance_bonus"`
	Revenue        float64 `json:"revenue"`
	Pump           float64 `json:"pump"`
	NickName       string  `json:"nickName"`
	RowVersion     int64   `json:"rowVersion"`
	Symbol         string  `json:"symbol"`
	Currency       string  `json:"currency"`
	CurrencySymbol string  `json:"currencySymbol"`
	IsTourist      int32   `json:"isTourist"`
}

func (esc *EsClient) GetListWithRowVersion(agentId, rowVersion int64) []interface{} {
	querys := make([]elastic.Query, 0, 64)
	querys = append(querys, elastic.NewRangeQuery("rowVersion").Gt(rowVersion))
	querys = append(querys, elastic.NewTermQuery("agentId", agentId))
	querys = append(querys, elastic.NewTermQuery("complete", true))
	query := elastic.NewBoolQuery().Must(querys...)
	sourceQuery := elastic.NewFetchSourceContext(true).Exclude("init", "log")
	resp, err := esc.es.Search().Index(esindex.Settlement()).
		Size(1000).
		Query(query).
		Sort("playedDate", true).
		FetchSourceContext(sourceQuery).
		Do(context.Background())
	if err != nil {
		zap.L().Error("游戏结算列表查询失败", zap.Error(err))
		return nil
	} else {
		result := make([]interface{}, 0, 1000)
		for _, v := range resp.Hits.Hits {
			s := &SettlementRecord{}
			if e := json.Unmarshal(v.Source, s); e != nil {
				zap.L().Error("反序列化结果集失败", zap.Error(e))
				continue
			}
			s.Id = v.Id
			result = append(result, s)
		}
		return result
	}
}

func (esc *EsClient) GetListWithTimeRange(agentId, startTime, endTime int64) []interface{} {
	querys := make([]elastic.Query, 0, 64)
	querys = append(querys, elastic.NewRangeQuery("playedDate").Gte(startTime*1000).Lt(endTime*1000))
	querys = append(querys, elastic.NewTermQuery("agentId", agentId))
	querys = append(querys, elastic.NewTermQuery("complete", true))
	query := elastic.NewBoolQuery().Must(querys...)
	sourceQuery := elastic.NewFetchSourceContext(true).Exclude("init", "log")
	resp, err := esc.es.Search().Index(esindex.Settlement()).
		Size(10000).
		Query(query).
		Sort("playedDate", true).
		FetchSourceContext(sourceQuery).
		Do(context.Background())
	if err != nil {
		zap.L().Error("游戏结算列表查询失败", zap.Error(err))
		return nil
	} else {
		result := make([]interface{}, 0, 1000)
		for _, v := range resp.Hits.Hits {
			s := &SettlementRecord{}
			if e := json.Unmarshal(v.Source, s); e != nil {
				zap.L().Error("反序列化结果集失败", zap.Error(e))
				continue
			}
			s.Id = v.Id
			result = append(result, s)
		}
		return result
	}
}
