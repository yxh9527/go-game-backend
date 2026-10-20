package v2

import (
	"app/entity"
	"app/entity/view"
	"app/esindex"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"web-api/dao"

	"github.com/gin-gonic/gin"
	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
)

func SettlementList(ctx *gin.Context) {
	startTime, errStart := strconv.ParseInt(ctx.Query("startTime"), 10, 0)
	endTime, errEnd := strconv.ParseInt(ctx.Query("endTime"), 10, 0)
	gameId := ctx.Query("gameId")
	userId, errUserId := strconv.ParseInt(ctx.Query("userId"), 10, 0)
	agentId, errAgentId := strconv.ParseInt(ctx.Query("agentId"), 10, 0)
	officeNumber := ctx.Query("officeNumber")
	account := ctx.Query("account")
	nickName := ctx.Query("nickName")
	hash := ctx.Query("hash")
	strNce := ctx.Query("complete")
	page, _ := strconv.Atoi(ctx.Query("page"))
	pageSize, _ := strconv.Atoi(ctx.Query("pageSize"))
	if pageSize <= 0 {
		pageSize = 30
	}
	offset := (page - 1) * pageSize
	querys := make([]elastic.Query, 0)
	if gameId != "" {
		ids := strings.Split(gameId, ",")
		gameIds := make([]interface{}, 0)
		for _, v := range ids {
			id, _ := strconv.Atoi(v)
			gameIds = append(gameIds, id)
		}
		querys = append(querys, elastic.NewTermsQuery("gameId", gameIds...))
	}
	if errUserId == nil {
		querys = append(querys, elastic.NewTermQuery("userId", userId))
	}
	if errAgentId == nil && agentId > 0 {
		querys = append(querys, elastic.NewTermQuery("agentId", agentId))
	}
	if errStart == nil && errEnd == nil {
		querys = append(querys, elastic.NewRangeQuery("playedDate").Gte(startTime*1000).Lt(endTime*1000))
	}
	if officeNumber != "" {
		querys = append(querys, elastic.NewMatchPhraseQuery("roundID", officeNumber))
	}
	if hash != "" {
		querys = append(querys, elastic.NewMatchQuery("hash", hash))
	}
	if strNce != "" {
		complete, _ := strconv.ParseBool(strNce)
		querys = append(querys, elastic.NewMatchQuery("complete", complete))
	}
	if account != "" {
		querys = append(querys, elastic.NewMatchQuery("account", account))
	}
	if nickName != "" {
		querys = append(querys, elastic.NewMatchQuery("nickName", nickName))
	}
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	resp, err := dao.Es().Search().Index(esindex.Settlement()).
		FetchSourceContext(elastic.NewFetchSourceContext(true).Exclude("init")).
		Query(boolQuery).
		Pretty(true).
		Size(pageSize).
		Sort("playedDate", false).
		From(offset).
		Do(context.Background())
	if err != nil {
		zap.L().Error("获取注单信息失败", zap.Any("err", err))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "获取注单信息失败"})
		return
	}
	settlements := make([]*view.Settlement, 0)
	for _, v := range resp.Hits.Hits {
		b, _ := v.Source.MarshalJSON()
		settlement := &view.Settlement{}
		json.Unmarshal(b, settlement)
		settlements = append(settlements, settlement)
	}
	result := &entity.SettlementList{Data: settlements, Total: resp.Hits.TotalHits.Value, Page: int64(page), PageSize: int64(pageSize)}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: result, Msg: "成功"})
}

func SettlementListWithAgentId(ctx *gin.Context) {
	startTime, errStart := strconv.ParseInt(ctx.Query("startTime"), 10, 0)
	endTime, errEnd := strconv.ParseInt(ctx.Query("endTime"), 10, 0)
	gameId := ctx.Query("gameId")
	userId, errUserId := strconv.ParseInt(ctx.Query("userId"), 10, 0)
	agentId := ctx.GetInt64("agentId")
	difficulty, errDifficulty := strconv.ParseInt(ctx.Query("difficulty"), 10, 0)
	officeNumber := ctx.Query("officeNumber")
	page, _ := strconv.Atoi(ctx.Query("page"))
	pageSize, _ := strconv.Atoi(ctx.Query("pageSize"))
	if pageSize <= 0 {
		pageSize = 30
	}
	offset := (page - 1) * pageSize
	querys := make([]elastic.Query, 0)
	if gameId != "" {
		ids := strings.Split(gameId, ",")
		gameIds := make([]interface{}, 0)
		for _, v := range ids {
			id, _ := strconv.Atoi(v)
			gameIds = append(gameIds, id)
		}
		querys = append(querys, elastic.NewTermsQuery("gameId", gameIds...))
	}
	if errDifficulty == nil {
		querys = append(querys, elastic.NewTermQuery("difficulty", difficulty))
	}
	if errUserId == nil {
		querys = append(querys, elastic.NewTermQuery("userId", userId))
	}
	if errStart == nil && errEnd == nil {
		querys = append(querys, elastic.NewRangeQuery("playedDate").Gte(startTime).Lt(endTime))
	}
	if officeNumber != "" {
		querys = append(querys, elastic.NewMatchPhraseQuery("roundID", officeNumber))
	}
	querys = append(querys, elastic.NewTermQuery("agentId", agentId))
	querys = append(querys, elastic.NewRangeQuery("isTourist").Lte(0))
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	resp, _ := dao.Es().Search().Index(esindex.Settlement()).
		Query(boolQuery).
		Pretty(true).
		Size(pageSize).
		Sort("playedDate", false).
		From(offset).
		Do(context.Background())
	settlements := make([]*view.Settlement, 0)
	for _, v := range resp.Hits.Hits {
		b, _ := v.Source.MarshalJSON()
		settlement := &view.Settlement{}
		json.Unmarshal(b, settlement)
		settlements = append(settlements, settlement)
	}
	result := &entity.SettlementList{Data: settlements, Total: resp.Hits.TotalHits.Value, Page: int64(page), PageSize: int64(pageSize)}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: result, Msg: "成功"})
}

func ExportSettlmentCountWithAgentId(ctx *gin.Context) {
	startTime, errStart := strconv.ParseInt(ctx.Query("startTime"), 10, 0)
	endTime, errEnd := strconv.ParseInt(ctx.Query("endTime"), 10, 0)
	gameId := ctx.Query("gameId")
	userId, errUserId := strconv.ParseInt(ctx.Query("userId"), 10, 0)
	agentId, errAgentId := strconv.ParseInt(ctx.Query("agentId"), 10, 0)
	officeNumber := ctx.Query("officeNumber")
	account := ctx.Query("account")
	nickName := ctx.Query("nickName")
	hash := ctx.Query("hash")
	strNce := ctx.Query("complete")
	querys := make([]elastic.Query, 0)
	if gameId != "" {
		ids := strings.Split(gameId, ",")
		gameIds := make([]interface{}, 0)
		for _, v := range ids {
			id, _ := strconv.Atoi(v)
			gameIds = append(gameIds, id)
		}
		querys = append(querys, elastic.NewTermsQuery("gameId", gameIds...))
	}
	if errUserId == nil {
		querys = append(querys, elastic.NewTermQuery("userId", userId))
	}
	if errAgentId == nil && agentId > 0 {
		querys = append(querys, elastic.NewTermQuery("agentId", agentId))
	}
	if errStart == nil && errEnd == nil {
		querys = append(querys, elastic.NewRangeQuery("playedDate").Gte(startTime*1000).Lt(endTime*1000))
	}
	if officeNumber != "" {
		querys = append(querys, elastic.NewMatchPhraseQuery("roundID", officeNumber))
	}
	if hash != "" {
		querys = append(querys, elastic.NewMatchQuery("hash", hash))
	}
	if strNce != "" {
		complete, _ := strconv.ParseBool(strNce)
		querys = append(querys, elastic.NewMatchQuery("complete", complete))
	}
	if account != "" {
		querys = append(querys, elastic.NewMatchQuery("account", account))
	}
	if nickName != "" {
		querys = append(querys, elastic.NewMatchQuery("nickName", nickName))
	}
	querys = append(querys, elastic.NewRangeQuery("isTourist").Lte(0))
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	aggs := elastic.NewValueCountAggregation().Field("roundID")
	resp, err := dao.Es().Search().Index(esindex.Settlement()).
		Query(boolQuery).
		Aggregation("count", aggs).
		Pretty(true).
		Do(context.Background())
	if err != nil {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Data: nil, Msg: "失败"})
		zap.L().Error("查询出错", zap.Any("err", err))
		return
	}
	v, b := resp.Aggregations.ValueCount("count")
	if b {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: v.Value, Msg: "成功"})
	} else {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Data: nil, Msg: "失败"})
		zap.L().Error("获取数量失败", zap.Any("b", b))
	}
}

func ExportSettlementsWithAgentId(ctx *gin.Context) {
	startTime, errStart := strconv.ParseInt(ctx.Query("startTime"), 10, 0)
	endTime, errEnd := strconv.ParseInt(ctx.Query("endTime"), 10, 0)
	gameId := ctx.Query("gameId")
	userId, errUserId := strconv.ParseInt(ctx.Query("userId"), 10, 0)
	agentId, errAgentId := strconv.ParseInt(ctx.Query("agentId"), 10, 0)
	officeNumber := ctx.Query("officeNumber")
	account := ctx.Query("account")
	nickName := ctx.Query("nickName")
	hash := ctx.Query("hash")
	strNce := ctx.Query("complete")
	querys := make([]elastic.Query, 0)
	if gameId != "" {
		ids := strings.Split(gameId, ",")
		gameIds := make([]interface{}, 0)
		for _, v := range ids {
			id, _ := strconv.Atoi(v)
			gameIds = append(gameIds, id)
		}
		querys = append(querys, elastic.NewTermsQuery("gameId", gameIds...))
	}
	if errUserId == nil {
		querys = append(querys, elastic.NewTermQuery("userId", userId))
	}
	if errAgentId == nil && agentId > 0 {
		querys = append(querys, elastic.NewTermQuery("agentId", agentId))
	}
	if errStart == nil && errEnd == nil {
		querys = append(querys, elastic.NewRangeQuery("playedDate").Gte(startTime*1000).Lt(endTime*1000))
	}
	if officeNumber != "" {
		querys = append(querys, elastic.NewMatchPhraseQuery("roundID", officeNumber))
	}
	if hash != "" {
		querys = append(querys, elastic.NewMatchQuery("hash", hash))
	}
	if strNce != "" {
		complete, _ := strconv.ParseBool(strNce)
		querys = append(querys, elastic.NewMatchQuery("complete", complete))
	}
	if account != "" {
		querys = append(querys, elastic.NewMatchQuery("account", account))
	}
	if nickName != "" {
		querys = append(querys, elastic.NewMatchQuery("nickName", nickName))
	}
	querys = append(querys, elastic.NewRangeQuery("isTourist").Lte(0))
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	resp, err := dao.Es().Search().Index(esindex.Settlement()).
		FetchSourceContext(elastic.NewFetchSourceContext(true).Exclude("init", "log")).
		Size(10000).
		Query(boolQuery).
		Pretty(true).
		Sort("playedDate", false).
		Do(context.Background())
	if err != nil {
		zap.L().Error("获取列表失败", zap.Any("err", err))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Data: nil, Msg: "失败"})
		return
	}
	settlements := make([]*view.Settlement, 0)
	for _, v := range resp.Hits.Hits {
		b, _ := v.Source.MarshalJSON()
		settlement := &view.Settlement{}
		json.Unmarshal(b, settlement)
		settlements = append(settlements, settlement)
	}
	result := &entity.SettlementList{Data: settlements, Total: resp.Hits.TotalHits.Value}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: result, Msg: "成功"})
}
