package v2

import (
	"app/config"
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
)

// 库存预警
func StockList(ctx *gin.Context) {
	gameId := ctx.Query("gameId")
	pageSize, _ := strconv.Atoi(ctx.Query("pageSize"))
	page, _ := strconv.Atoi(ctx.Query("page"))
	agentId, _ := strconv.Atoi(ctx.Query("agentId"))
	querys := make([]elastic.Query, 0)
	ids := make([]interface{}, 0)
	for _, v := range strings.Split(gameId, ",") {
		id, _ := strconv.Atoi(v)
		cfg := config.CfgIns.GetPoolCfgByGameId(int64(agentId), int64(id))
		if cfg != nil {
			ids = append(ids, cfg.Symbol)
		}
	}
	querys = append(querys, elastic.NewTermsQuery("symbol", ids...))
	querys = append(querys, elastic.NewTermQuery("agentId", agentId))
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	resp, err := dao.Es().Search().Index(esindex.PoolRecordLog()).
		Query(boolQuery).
		Pretty(true).
		Size(pageSize).
		From((page-1)*pageSize).
		Sort("createTime", false).
		Do(context.Background())
	if err != nil {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "查询失败"})
		return
	}
	logs := make([]*view.PoolLogItem, 0)
	for _, v := range resp.Hits.Hits {
		b, _ := v.Source.MarshalJSON()
		log := &view.PoolLogItem{}
		json.Unmarshal(b, log)
		logs = append(logs, log)
	}
	result := &view.PoolLog{Data: logs, Page: page, PageSize: pageSize}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: result, Msg: "成功"})
}
