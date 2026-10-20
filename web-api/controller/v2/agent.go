package v2

import (
	"app/entity"
	"app/entity/view"
	"app/esindex"
	"app/tables/manager"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"web-api/common"
	"web-api/dao"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
)

func AgentAdd(ctx *gin.Context) {
	nickName := ctx.PostForm("nickName")
	email := ctx.PostForm("email")
	webId, _ := strconv.Atoi(ctx.PostForm("webId"))
	contacts := ctx.PostForm("contacts")
	phone := ctx.PostForm("phone")
	remarks := ctx.PostForm("remarks")
	if len(nickName) > 50 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Data: nil, Msg: "名称超出最大长度,只能50个字符"})
	}
	if len(contacts) > 50 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Data: nil, Msg: "负责人超出最大长度,只能50个字符"})
	}
	if len(phone) > 50 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Data: nil, Msg: "联系方式超出最大长度,只能50个字符"})
	}
	agent := &manager.Agent{
		AesKey:      common.GenRandString(32),
		Md5Key:      common.GenRandString(32),
		AgentKey:    common.GenRandString(32),
		NickName:    nickName,
		Email:       email,
		WebId:       int64(webId),
		Contacts:    contacts,
		Phone:       phone,
		Remarks:     remarks,
		IsPermanent: 1,
		State:       1,
		CreateTime:  int32(time.Now().Unix()),
		UpdateTime:  int32(time.Now().Unix()),
		Stock:       0,
		IsFrozen:    0,
		Level:       1,
		IsDel:       0,
	}

	var count int64 = 0
	dao.Mysql().Manager.Where("nickName = ?", nickName).Count(&count)
	if count > 0 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Data: nil, Msg: "已经有相同的代理"})
		ctx.Abort()
		return
	}
	dao.Mysql().Manager.Create(agent)
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "操作成功"})
}

func getAgentDataAggs(startTime, endTime, agentId, webId int64) []*view.AgentInfoAggs {
	result := make([]*view.AgentInfoAggs, 0)
	querys := make([]elastic.Query, 0)
	if startTime > 0 && endTime > 0 {
		querys = append(querys, elastic.NewRangeQuery("createAt").Gte(startTime).Lt(endTime))
	}
	if agentId >= 0 {
		querys = append(querys, elastic.NewTermQuery("agentId", agentId))
	}
	if webId > 0 {
		querys = append(querys, elastic.NewTermQuery("webId", webId))
	}
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	aggs := elastic.NewTermsAggregation().Field("agentId").Size(10000)
	aggs = aggs.SubAggregation("effectiveBetsTotal", elastic.NewSumAggregation().Field("effectiveBetsTotal"))
	aggs = aggs.SubAggregation("profitLossTotal", elastic.NewSumAggregation().Field("profitLossTotal"))
	aggs = aggs.SubAggregation("pumpTotal", elastic.NewSumAggregation().Field("pumpTotal"))
	aggs = aggs.SubAggregation("userTotal", elastic.NewSumAggregation().Field("userTotal"))
	aggs = aggs.SubAggregation("userBetsTotal", elastic.NewSumAggregation().Field("userBetsTotal"))
	aggs = aggs.SubAggregation("revenueTotal", elastic.NewSumAggregation().Field("revenueTotal"))
	aggs = aggs.SubAggregation("chipsTotal", elastic.NewSumAggregation().Field("chipsTotal"))
	aggs = aggs.SubAggregation("docCount", elastic.NewSumAggregation().Field("doc_count"))
	resp, err := dao.Es().Search().Index(esindex.DataAnalysis()).
		Size(0).
		Query(boolQuery).
		Aggregation("agentId", aggs).
		Pretty(true).
		Do(context.Background())
	if err != nil {
		return result
	}
	v, _ := resp.Aggregations.Terms("agentId")
	for _, bucket := range v.Buckets {
		b, _ := json.Marshal(bucket)
		item := &view.AgentInfoAggs{}
		json.Unmarshal(b, item)
		result = append(result, item)
	}
	return result
}

func AgentInfo(ctx *gin.Context) {
	webId, _ := strconv.ParseInt(ctx.Query("webId"), 10, 0)
	startTime, _ := strconv.ParseInt(ctx.Query("startTime"), 10, 0)
	endTime, _ := strconv.ParseInt(ctx.Query("endTime"), 10, 0)
	agentId, err := strconv.ParseInt(ctx.Query("agentId"), 10, 0)
	if err != nil {
		agentId = -1
	}
	data := getAgentDataAggs(startTime, endTime, agentId, webId)
	db := dao.Mysql().Manager.Model(manager.User{}).Debug()
	td := make([]*view.AgentTotalCount, 0)
	rd := make([]*view.AgentTotalCount, 0)
	//select count(*) as userNum,agentId from gp_user inner join gp_agent on gp_agent.id = gp_user.agentId where gp_user.webId =23 group by agentId
	db.Model(view.AgentTotalCount{}).Debug().Select("count(*) as userNum,agentId").Joins("inner join gp_agent on gp_agent.id = gp_user.agentId").Where("gp_user.webId=?", webId).Group("agentId").Find(&td)
	db.Model(view.AgentTotalCount{}).Debug().Select("count(*) as userNum,agentId").Joins("inner join gp_agent on gp_agent.id = gp_user.agentId").Where("gp_user.webId=? and gp_user.createTime BETWEEN ? and ?", webId, startTime, endTime).Group("agentId").Find(&rd)
	db = dao.Mysql().Manager.Model(manager.UserScoreLog{})
	tus := make([]*view.UserScoreSum, 0)
	if agentId > 0 {
		db.Model(view.UserScoreSum{}).Debug().Select("sum(score) as scoreSum,agentId,logType").Joins("inner join gp_agent on gp_agent.id = gp_user_score_log.agentId").Where("gp_agent.webId=? and gp_user_score_log.createTime BETWEEN ? and ? and agentId=?", webId, startTime, endTime, agentId).Group("`agentId`,`logType`").Find(&tus)
	} else {
		db.Model(view.UserScoreSum{}).Debug().Select("sum(score) as scoreSum,agentId,logType").Joins("inner join gp_agent on gp_agent.id = gp_user_score_log.agentId").Where("gp_agent.webId=? and gp_user_score_log.createTime BETWEEN ? and ?", webId, startTime, endTime).Group("`agentId`,`logType`").Find(&tus)
	}
	result := make([]map[string]interface{}, 0, 64)
	zap.L().Debug("====>", zap.Any("td", td), zap.Any("rd", rd))
	for _, v := range data {
		for _, t := range td {
			if t.AgentId == v.Key {
				v.Aggs.TotalRegUser = t.UserNum
			}
		}

		for _, t := range rd {
			if t.AgentId == v.Key {
				v.Aggs.RangeRegUser = t.UserNum
			}
		}

		for _, t := range tus {
			if t.AgentId == v.Key {
				if t.LogType == 1 {
					v.Aggs.ScoreUp = t.ScoreSum
				} else {
					v.Aggs.ScoreDown = t.ScoreSum
				}
			}
		}
		result = append(result, map[string]interface{}{
			"effectiveBetsTotal": v.Aggs.EffectiveBetsTotal.Value,
			"profitLossTotal":    v.Aggs.ProfitLossTotal.Value,
			"pumpTotal":          v.Aggs.PumpTotal.Value,
			"userTotal":          v.Aggs.UserTotal.Value,
			"userBetsTotal":      v.Aggs.UserBets.Value,
			"revenueTotal":       v.Aggs.RevenueTotal.Value,
			"totalRegUser":       v.Aggs.TotalRegUser,
			"rangeRegUser":       v.Aggs.RangeRegUser,
			"score_up":           v.Aggs.ScoreUp,
			"score_down":         v.Aggs.ScoreDown,
			"chipsTotal":         v.Aggs.ChipsTotal,
		})
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: result, Msg: "操作成功"})
}

func agentGameDataAggs(startTime, endTime int64) []*elastic.AggregationBucketKeyItem {
	result := make([]*elastic.AggregationBucketKeyItem, 0)
	startRange := elastic.NewRangeQuery("startTime").Gte(startTime)
	endRange := elastic.NewRangeQuery("endTime").Lte(endTime)
	boolQuery := elastic.NewBoolQuery().Must(startRange, endRange)
	aggs := elastic.NewTermsAggregation().Field("agentId")
	gameAggs := elastic.NewTermsAggregation().Field("gameId").Size(1000)
	gameAggs = gameAggs.SubAggregation("effectiveBetsTotal", elastic.NewSumAggregation().Field("effectiveBetsTotal"))
	gameAggs = gameAggs.SubAggregation("profitLossTotal", elastic.NewSumAggregation().Field("profitLossTotal"))
	gameAggs = gameAggs.SubAggregation("pumpTotal", elastic.NewSumAggregation().Field("pumpTotal"))
	gameAggs = gameAggs.SubAggregation("userTotal", elastic.NewSumAggregation().Field("userTotal"))
	gameAggs = gameAggs.SubAggregation("revenueTotal", elastic.NewSumAggregation().Field("revenueTotal"))
	gameAggs = gameAggs.SubAggregation("docCount", elastic.NewSumAggregation().Field("doc_count"))
	aggs.SubAggregation("games", gameAggs)
	resp, err := dao.Es().Search().Index(esindex.DataAnalysis()).
		Size(0).
		Query(boolQuery).
		Aggregation("agents", aggs).
		Pretty(true).
		Do(context.Background())
	if err != nil {
		return result
	}
	r, _ := resp.Aggregations.Terms("agents")
	return r.Buckets
}

func AgGroup(ctx *gin.Context) {
	date := ctx.Query("date")
	rangeType := ctx.Query("range_type")
	var nowStart, nowEnd, lastStart, lastEnd int64 = 0, 0, 0, 0
	if date == "" {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Data: nil, Msg: "参数错误"})
		return
	}
	t := fmt.Sprintf("%s 00:00:00", date)
	pt, _ := time.ParseInLocation("2006-01-02 15:04:05", t, time.Local)
	if rangeType == "day" {
		nowStart = pt.Unix()
		nowEnd = nowStart + 24*60*60
		lastStart = pt.AddDate(0, 0, -1).Unix()
		lastEnd = nowStart
	} else {
		t = fmt.Sprintf("%04d-%02d-01 00:00:00", pt.Year(), pt.Month())
		pt, _ := time.ParseInLocation("2006-01-02 15:04:05", t, time.Local)
		nowStart = pt.Unix()
		nowEnd = pt.AddDate(0, 1, 0).Unix()
		lastStart = pt.AddDate(0, -1, 0).Unix()
		lastEnd = nowStart
	}
	agents := make([]*manager.Agent, 0)
	dao.Mysql().Manager.Model(manager.Agent{}).Debug().Find(&agents)
	nowRs := agentGameDataAggs(nowStart, nowEnd)
	lastRs := agentGameDataAggs(lastStart, lastEnd)
	results := make([]*entity.AgGroupData, 0)
	for _, agent := range agents {
		var profitLossTotal float64 = 0.0
		var userTotal float64 = 0
		var docCount float64 = 0
		var effectiveBetsTotal float64 = 0.0
		var pumpTotal float64 = 0.0
		ag := &entity.AgGroupData{
			AgentId:  int(agent.Id),
			NickName: agent.NickName,
			Now:      &entity.AgGroupItem{},
			Last:     &entity.AgGroupItem{},
		}
		var nowGames *elastic.AggregationBucketKeyItems = nil
		for _, v := range nowRs {
			id := v.Key.(float64)
			if id == float64(agent.Id) {
				nowGames, _ = v.Aggregations.Terms("games")
				for _, game := range nowGames.Buckets {
					data, _ := game.Sum("profitLossTotal")
					profitLossTotal += (*data.Value)
					data, _ = game.Sum("effectiveBetsTotal")
					effectiveBetsTotal += (*data.Value)
					data, _ = game.Sum("pumpTotal")
					pumpTotal += (*data.Value)
					data, _ = game.Sum("docCount")
					docCount += (*data.Value)
					data, _ = game.Sum("userTotal")
					userTotal += (*data.Value)
				}
			}
		}
		if nowGames == nil {
			ag.Now = &entity.AgGroupItem{
				DocCount:           int(docCount),
				DocCount2:          1,
				ProfitLossTotal:    profitLossTotal,
				EffectiveBetsTotal: effectiveBetsTotal,
				Key:                int64(agent.Id),
				PumpTotal:          pumpTotal,
				UserTotal:          userTotal,
			}
		} else {
			ag.Now = &entity.AgGroupItem{
				DocCount:           int(docCount),
				DocCount2:          1,
				ProfitLossTotal:    profitLossTotal,
				EffectiveBetsTotal: effectiveBetsTotal,
				Key:                int64(agent.Id),
				PumpTotal:          pumpTotal,
				UserTotal:          userTotal,
				Games:              nowGames.Aggregations,
			}
		}
		var lastGames *elastic.AggregationBucketKeyItems = nil
		for _, v := range lastRs {
			id := v.Key.(float64)
			if id == float64(agent.Id) {
				lastGames, _ = v.Aggregations.Terms("games")
				for _, game := range lastGames.Buckets {
					data, _ := game.Sum("profitLossTotal")
					profitLossTotal += (*data.Value)
					data, _ = game.Sum("effectiveBetsTotal")
					effectiveBetsTotal += (*data.Value)
					data, _ = game.Sum("pumpTotal")
					pumpTotal += (*data.Value)
					data, _ = game.Sum("docCount")
					docCount += (*data.Value)
					data, _ = game.Sum("userTotal")
					userTotal += (*data.Value)
				}
			}
		}
		if lastGames == nil {
			ag.Last = &entity.AgGroupItem{
				DocCount:           int(docCount),
				DocCount2:          1,
				ProfitLossTotal:    profitLossTotal,
				EffectiveBetsTotal: effectiveBetsTotal,
				Key:                int64(agent.Id),
				PumpTotal:          pumpTotal,
				UserTotal:          userTotal,
			}
		} else {
			ag.Last = &entity.AgGroupItem{
				DocCount:           int(docCount),
				DocCount2:          1,
				ProfitLossTotal:    profitLossTotal,
				EffectiveBetsTotal: effectiveBetsTotal,
				Key:                int64(agent.Id),
				PumpTotal:          pumpTotal,
				UserTotal:          userTotal,
				Games:              lastGames.Aggregations,
			}
		}
		results = append(results, ag)
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: results})
}

func AgentStatInfo(ctx *gin.Context) {
	webId, _ := strconv.ParseInt(ctx.Query("webId"), 10, 0)
	startTime, _ := strconv.ParseInt(ctx.Query("startTime"), 10, 0)
	endTime, _ := strconv.ParseInt(ctx.Query("endTime"), 10, 0)
	agentId := ctx.GetInt64("agentId")
	data := getAgentDataAggs(startTime, endTime, agentId, webId)
	db := dao.Mysql().Manager.Model(manager.User{}).Debug()
	td := make([]*view.AgentTotalCount, 0)
	rd := make([]*view.AgentTotalCount, 0)
	//select count(*) as userNum,agentId from gp_user inner join gp_agent on gp_agent.id = gp_user.agentId where gp_user.webId =23 group by agentId
	db.Model(view.AgentTotalCount{}).Debug().Select("count(*) as userNum,agentId").Joins("inner join gp_agent on gp_agent.id = gp_user.agentId").Where("gp_user.webId=?", webId).Group("agentId").Find(&td)
	db.Model(view.AgentTotalCount{}).Debug().Select("count(*) as userNum,agentId").Joins("inner join gp_agent on gp_agent.id = gp_user.agentId").Where("gp_user.webId=? and gp_user.createTime BETWEEN ? and ?", webId, startTime, endTime).Group("agentId").Find(&rd)
	db = dao.Mysql().Manager.Model(manager.UserScoreLog{})
	tus := make([]*view.UserScoreSum, 0)
	if agentId > 0 {
		db.Model(view.UserScoreSum{}).Debug().Select("sum(score) as scoreSum,agentId,logType").Joins("inner join gp_agent on gp_agent.id = gp_user_score_log.agentId").Where("gp_agent.webId=? and gp_user_score_log.createTime BETWEEN ? and ? and agentId=?", webId, startTime, endTime, agentId).Group("`agentId`,`logType`").Find(&tus)
	} else {
		db.Model(view.UserScoreSum{}).Debug().Select("sum(score) as scoreSum,agentId,logType").Joins("inner join gp_agent on gp_agent.id = gp_user_score_log.agentId").Where("gp_agent.webId=? and gp_user_score_log.createTime BETWEEN ? and ?", webId, startTime, endTime).Group("`agentId`,`logType`").Find(&tus)
	}
	result := make([]map[string]interface{}, 0, 64)
	for _, v := range data {
		for _, t := range td {
			if t.AgentId == v.Key {
				v.Aggs.TotalRegUser = t.UserNum
			}
		}
		for _, t := range rd {
			if t.AgentId == v.Key {
				v.Aggs.RangeRegUser = t.UserNum
			}
		}
		for _, t := range tus {
			if t.AgentId == v.Key {
				if t.LogType == 1 {
					v.Aggs.ScoreUp = t.ScoreSum
				} else {
					v.Aggs.ScoreDown = t.ScoreSum
				}
			}
		}
		result = append(result, map[string]interface{}{
			"effectiveBetsTotal": v.Aggs.EffectiveBetsTotal.Value,
			"profitLossTotal":    v.Aggs.ProfitLossTotal.Value,
			"pumpTotal":          v.Aggs.PumpTotal.Value,
			"userTotal":          v.Aggs.UserTotal.Value,
			"userBetsTotal":      v.Aggs.UserBets.Value,
			"revenueTotal":       v.Aggs.RevenueTotal.Value,
			"totalRegUser":       v.Aggs.TotalRegUser,
			"rangeRegUser":       v.Aggs.RangeRegUser,
			"score_up":           v.Aggs.ScoreUp,
			"score_down":         v.Aggs.ScoreDown,
			"chipsTotal":         v.Aggs.ChipsTotal,
		})
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: result, Msg: "操作成功"})
}

type UserChartItem struct {
	GameId          int64   `json:"gameId"`
	ExProfitLoss    float64 `json:"exProfitLoss"`
	EffectiveBets   float64 `json:"effectiveBets"`
	CreateTime      int64   `json:"createTime"`
	ExEffectiveBets float64 `json:"exEffectiveBets"`
	ProfitLoss      float64 `json:"profitLoss"`
	DateStr         string  `json:"date"`
}

func UserChart(ctx *gin.Context) {
	userId, _ := strconv.ParseInt(ctx.Query("userId"), 10, 64)
	startTime, _ := strconv.ParseInt(ctx.Query("startTime"), 10, 64)
	endTime, _ := strconv.ParseInt(ctx.Query("endTime"), 10, 64)
	page, _ := strconv.ParseInt(ctx.Query("page"), 10, 64)
	if page > 0 {
		page -= page
	} else {
		page = 0
	}
	querys := make([]elastic.Query, 0, 16)
	querys = append(querys, elastic.NewRangeQuery("playedDate").Gte(startTime*1000).Lt(endTime*1000))
	querys = append(querys, elastic.NewRangeQuery("isTourist").Lte(0))
	querys = append(querys, elastic.NewTermQuery("userId", userId))
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	resp, err := dao.Es().Search().Index(esindex.Settlement()).
		FetchSourceContext(elastic.NewFetchSourceContext(true).Include("bet", "exBet", "exWin", "win", "playedDate", "symbol")).
		Size(10000).
		Query(boolQuery).
		From(int(page * 10000)).
		Pretty(true).
		Do(context.Background())
	if err != nil {
		zap.L().Error("UserChart", zap.Any("err", err))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "失败", Data: nil})
		return
	}
	data := make([]interface{}, 0, 128)
	for _, v := range resp.Hits.Hits {
		item := &UserChartItem{}
		str, _ := jsoniter.MarshalToString(v.Source)
		jsoniter.UnmarshalFromString(str, item)
		item.DateStr = time.Unix(item.CreateTime, 0).Format("2006-01-02 15:00:00")
		zap.L().Debug("item.str", zap.Any("str", item.DateStr))
		data = append(data, item)
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: map[string]interface{}{
		"total": resp.Hits.TotalHits.Value,
		"data":  data,
	}, Msg: "操作成功"})
}
