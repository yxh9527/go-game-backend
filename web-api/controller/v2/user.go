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
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func getUserListFromMysql(ctx *gin.Context, total *int64, pageSize, offset int) []*manager.User {
	agentId, errAgentId := strconv.Atoi(ctx.Query("agentId"))
	inCtl, errInCtl := strconv.Atoi(ctx.Query("inCtl"))
	webId, errWebId := strconv.Atoi(ctx.Query("webId"))
	id, errId := strconv.Atoi(ctx.Query("id"))

	name := ctx.Query("name")
	userId := ctx.Query("userId")
	order := ctx.Query("order")
	users := make([]*manager.User, 0)
	dao.Mysql().Manager.Debug().Model(&manager.User{}).Count(total)
	db := dao.Mysql().Manager.Debug()
	switch order {
	case "totalProfLoss":
		db = db.Order("totalProfLoss asc")
	case "-totalProfLoss":
		db = db.Order("totalProfLoss desc")
	case "logTime":
		db = db.Order("logTime asc")
	case "-logTime":
		db = db.Order("logTime desc")
	}
	if errAgentId == nil {
		db = db.Where("agentId=?", agentId)
	}
	if errWebId == nil {
		db = db.Where("webId=?", webId)
	}
	if userId != "" {
		db = db.Where("userId like ?", "%"+userId+"%")
	}
	if name != "" {
		db = db.Where("nickName like ?", "%"+name+"%")
	}
	if errId == nil {
		db = db.Where("id=?", id)
	}
	if errInCtl == nil {
		//TODO::后续需要将用户控制的状态同步到数据库
		db = db.Where("isCtl=?", inCtl)
	}
	db.Offset(offset).Limit(pageSize).Find(&users)
	return users
}

// 获取月统计数据
func getUsersMonthAggs(startTime, endTime int64, ids []interface{}) []*view.UserInfoAggs {
	if startTime == 0 || endTime == 0 {
		now := time.Now()
		t := fmt.Sprintf("%d-%d-01 00:00:00", now.Year(), now.Month())
		pt, _ := time.ParseInLocation("2006-01-02 15:04:05", t, time.Local)
		startTime = pt.Unix()
		endTime = pt.AddDate(0, 1, 0).Unix()
	}
	source := elastic.NewFetchSourceContext(true).Exclude("init", "log")
	termsQuery := elastic.NewTermsQuery("userId", ids...)
	timeRange := elastic.NewRangeQuery("playedDate").Gte(startTime * 1000).Lt(endTime * 1000)
	query := elastic.NewRangeQuery("isTourist").Lte(0)
	boolQuery := elastic.NewBoolQuery().Must(termsQuery, timeRange, query)
	aggs := elastic.NewTermsAggregation().Field("userId").Size(len(ids))
	aggs = aggs.SubAggregation("effectiveBetsTotal", elastic.NewSumAggregation().Field("bet"))
	aggs = aggs.SubAggregation("profitLossTotal", elastic.NewSumAggregation().Field("win"))
	aggs = aggs.SubAggregation("docCount", elastic.NewSumAggregation().Field("doc_count"))
	resp, err := dao.Es().Search().Index(esindex.Settlement()).
		Size(0).
		FetchSourceContext(source).
		Query(boolQuery).
		Aggregation("userId", aggs).
		Pretty(true).
		Do(context.Background())
	if err != nil {
		zap.L().Error("获取注单信息失败", zap.Any("err", err))
		return nil
	}
	v, b := resp.Aggregations.Terms("userId")
	if !b {
		return nil
	}
	result := make([]*view.UserInfoAggs, 0)
	for _, bucket := range v.Buckets {
		b, _ := json.Marshal(bucket)
		item := &view.UserInfoAggs{}
		json.Unmarshal(b, item)
		result = append(result, item)
	}
	return result
}

// 获取周期统计数据
func getUsersCycleAggs(ids []int64) (map[int64]decimal.Decimal, map[int64]decimal.Decimal) {
	pipe := dao.RedisIns().Client.Pipeline()
	cycleProfitloss := make(map[int64]decimal.Decimal)
	for i := 0; i < len(ids); i++ {
		pipe.ZScore(context.Background(), "userTotalProfLoss", fmt.Sprintf("%d", ids[i]))
	}
	profloss, _ := pipe.Exec(context.Background())
	for i := 0; i < len(profloss); i++ {
		if profloss[i].Err() == nil {
			r, _ := profloss[i].(*redis.FloatCmd)
			cycleProfitloss[ids[i]] = decimal.NewFromFloat(r.Val())
		}
	}
	cycleEffectiveBets := make(map[int64]decimal.Decimal)
	pipe = dao.RedisIns().Client.Pipeline()
	for i := 0; i < len(ids); i++ {
		pipe.ZScore(context.Background(), "userTotalEffBet", fmt.Sprintf("%d", ids[i]))
	}
	effects, _ := pipe.Exec(context.Background())
	for i := 0; i < len(effects); i++ {
		if effects[i].Err() == nil {
			r, _ := effects[i].(*redis.FloatCmd)
			cycleEffectiveBets[ids[i]] = decimal.NewFromFloat(r.Val())
		}

	}
	return cycleProfitloss, cycleEffectiveBets
}

// 用户列表
func UserList(ctx *gin.Context) {
	var total int64 = 0
	page, _ := strconv.Atoi(ctx.Query("page"))
	startTime, _ := strconv.ParseInt(ctx.Query("startTime"), 10, 0)
	endTime, _ := strconv.ParseInt(ctx.Query("endTime"), 10, 0)
	//默认一页只有三十条数据
	pageSize := 30
	offset := (page - 1) * pageSize
	users := getUserListFromMysql(ctx, &total, pageSize, offset)
	idsInterface, ids := make([]interface{}, 0), make([]int64, 0)
	userItems := make([]*entity.UserListItem, 0)
	//获取玩家sc信息
	piple := dao.RedisIns().Client.Pipeline()
	for _, user := range users {
		userItems = append(userItems, &entity.UserListItem{User: user})
		idsInterface = append(idsInterface, user.Id)
		ids = append(ids, user.Id)
		piple.HGet(context.Background(), "user_ctl", fmt.Sprintf("%d", user.Id))
	}
	results, err := piple.Exec(context.Background())
	//获取月统计数据
	aggs := getUsersMonthAggs(startTime, endTime, idsInterface)
	for k, user := range userItems {
		for _, item := range aggs {
			if user.Id == item.Key {
				user.MonthEffectiveBets = item.Aggs.EffectiveBetsTotal.Value.InexactFloat64()
				user.MonthProfitLoss = item.Aggs.ProfitLossTotal.Value.InexactFloat64()
				user.MonthDocCount = item.DocCount
			}
		}
		if err == nil {
			if results[k].Err() == nil {
				s := (results[k].(*redis.StringCmd)).Val()
				c := &view.UserSingleControl{}
				jsoniter.UnmarshalFromString(s, c)
				user.InCtl = 1
				user.LeftScore = c.LeftScore.InexactFloat64()
				user.Rate = c.Rate.InexactFloat64()
				user.RateScore = c.RateScore.InexactFloat64()
			}
		}
	}
	//获取周期统计数据
	profs, effecs := getUsersCycleAggs(ids)
	for _, user := range userItems {
		for id, item := range profs {
			if user.Id == id {
				user.CycleProfitLoss = item.InexactFloat64()
			}
		}

		for id, item := range effecs {
			if user.Id == id {
				user.CycleEffectiveBets = item.InexactFloat64()
			}
		}
	}
	data := &entity.UserList{
		LastPage: common.CountPage(total, int64(pageSize)),
		Page:     int64(page),
		PerPage:  int64(pageSize),
		Total:    total,
		Data:     userItems,
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: data})
}

func getUserListWithAgentIdFromMysql(ctx *gin.Context, total *int64, pageSize, offset int) []*manager.User {
	inCtl, errInCtl := strconv.Atoi(ctx.Query("inCtl"))
	webId, errWebId := strconv.Atoi(ctx.Query("webId"))

	name := ctx.Query("name")
	userId := ctx.Query("userId")
	order := ctx.Query("order")
	users := make([]*manager.User, 0)
	dao.Mysql().Manager.Debug().Model(&manager.User{}).Count(total)
	db := dao.Mysql().Manager.Debug()
	switch order {
	case "totalProfLoss":
		db = db.Order("totalProfLoss asc")
	case "-totalProfLoss":
		db = db.Order("totalProfLoss desc")
	case "logTime":
		db = db.Order("logTime asc")
	case "-logTime":
		db = db.Order("logTime desc")
	}
	db = db.Where("agentId=?", ctx.GetInt64("agentId"))
	if errWebId == nil {
		db = db.Where("webId=?", webId)
	}
	if userId != "" {
		db = db.Where("userId like ?", "%"+userId+"%")
	}
	if name != "" {
		db = db.Where("nickName like ?", "%"+name+"%")
	}
	if errInCtl == nil {
		//TODO::后续需要将用户控制的状态同步到数据库
		db = db.Where("isCtl=?", inCtl)
	}
	db.Offset(offset).Limit(pageSize).Find(&users)
	return users
}

func UserListByAgentId(ctx *gin.Context) {
	var total int64 = 0
	page, _ := strconv.Atoi(ctx.Query("page"))
	startTime, _ := strconv.ParseInt(ctx.Query("startTime"), 10, 0)
	endTime, _ := strconv.ParseInt(ctx.Query("endTime"), 10, 0)
	//默认一页只有三十条数据
	pageSize := 30
	offset := (page - 1) * pageSize
	users := getUserListWithAgentIdFromMysql(ctx, &total, pageSize, offset)
	idsInterface, ids := make([]interface{}, 0), make([]int64, 0)
	userItems := make([]*entity.UserListItem, 0)
	for _, user := range users {
		userItems = append(userItems, &entity.UserListItem{User: user})
		idsInterface = append(idsInterface, user.Id)
		ids = append(ids, user.Id)
	}
	//获取月统计数据
	aggs := getUsersMonthAggs(startTime, endTime, idsInterface)
	for _, user := range userItems {
		for _, item := range aggs {
			if user.Id == item.Key {
				user.MonthEffectiveBets = item.Aggs.EffectiveBetsTotal.Value.InexactFloat64()
				user.MonthProfitLoss = item.Aggs.ProfitLossTotal.Value.InexactFloat64()
				user.MonthDocCount = item.DocCount
			}
		}
	}
	//获取周期统计数据
	profs, effecs := getUsersCycleAggs(ids)
	for _, user := range userItems {
		for id, item := range profs {
			if user.Id == id {
				user.CycleProfitLoss = item.InexactFloat64()
			}
		}

		for id, item := range effecs {
			if user.Id == id {
				user.CycleEffectiveBets = item.InexactFloat64()
			}
		}
	}
	data := &entity.UserList{
		LastPage: common.CountPage(total, int64(pageSize)),
		Page:     int64(page),
		PerPage:  int64(pageSize),
		Total:    total,
		Data:     userItems,
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: data})
}
