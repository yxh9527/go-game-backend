package v2

import (
	"app/entity"
	"app/entity/view"
	"app/esindex"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"web-api/common"
	"web-api/dao"

	"github.com/gin-gonic/gin"
	"github.com/olivere/elastic/v7"
)

// 流水列表
func BillList(ctx *gin.Context) {
	startTime, errStart := strconv.ParseInt(ctx.Query("startTime"), 10, 0)
	endTime, errEnd := strconv.ParseInt(ctx.Query("endTime"), 10, 0)
	symbol := ctx.Query("symbol")
	userId, errUserId := strconv.ParseInt(ctx.Query("userId"), 10, 0)
	officeNumber := ctx.Query("officeNumber")
	page, _ := strconv.Atoi(ctx.Query("page"))
	pageSize, _ := strconv.Atoi(ctx.Query("pageSize"))
	if pageSize <= 0 {
		pageSize = 30
	}
	offset := (page - 1) * pageSize
	querys := make([]elastic.Query, 0)
	if symbol != "" {
		querys = append(querys, elastic.NewTermQuery("symbol", symbol))
	}
	if errUserId == nil {
		querys = append(querys, elastic.NewTermQuery("userId", userId))
	}
	if errStart == nil && errEnd == nil {
		querys = append(querys, elastic.NewRangeQuery("createTime").Gte(startTime).Lt(endTime))
	}
	if officeNumber != "" {
		querys = append(querys, elastic.NewMatchPhraseQuery("roundId", officeNumber))
	}
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	resp, _ := dao.Es().Search().Index(esindex.FlowingWater()).
		Query(boolQuery).
		Pretty(true).
		Size(pageSize).
		From(offset).
		Sort("createTime", false).
		Do(context.Background())
	bills := make([]*view.Bill, 0)
	for _, v := range resp.Hits.Hits {
		b, _ := v.Source.MarshalJSON()
		bill := &view.Bill{}
		json.Unmarshal(b, bill)
		bill.Normalize()
		bills = append(bills, bill)
	}
	result := &entity.BillList{Data: bills, Total: resp.Hits.TotalHits.Value, LastPage: common.CountPage(resp.Hits.TotalHits.Value, int64(pageSize)), Page: int64(page), PerPage: int64(pageSize)}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: result, Msg: "成功"})
}
