package v2

import (
	"app/config"
	"app/entity"
	"app/entity/view"
	"app/esindex"
	"app/tables/manager"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	. "web-api/common"
	"web-api/dao"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/olivere/elastic/v7"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 获取游戏列表
func GameList(ctx *gin.Context) {
	games := make([]*manager.Game, 0)
	dao.Mysql().Manager.Find(&games)
	gis := make([]*entity.GameListItem, 0)
	for _, game := range games {
		gi := &entity.GameListItem{
			Id:       int(game.Id),
			Number:   game.Number,
			GameType: game.GameType,
			Name:     game.Name,
			NameZH:   game.NameZH,
			Symbol:   game.ConfName,
		}
		gis = append(gis, gi)
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: gis})
}

func ApiConfigList(ctx *gin.Context) {
	var total int64 = 0
	page, _ := strconv.Atoi(ctx.Query("page"))
	pageSize, _ := strconv.Atoi(ctx.Query("pageSize"))
	if pageSize <= 0 {
		pageSize = 30
	}
	offset := (page - 1) * pageSize

	apiConfig := make([]*manager.ApiConfig, 0)
	dao.Mysql().Manager.Model(manager.ApiConfig{}).Count(&total)
	dao.Mysql().Manager.Order("id desc").Offset(offset).Limit(pageSize).Find(&apiConfig)

	result := &entity.ApiConfigList{Data: apiConfig, Total: total, LastPage: CountPage(total, int64(pageSize)), Page: int64(page), PerPage: int64(pageSize)}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: result, Msg: "成功"})
}

func GetGameUrl(ctx *gin.Context) {
	data, err := dao.RedisIns().Get(esindex.ConfigKey("system"))
	if err != nil {
		zap.L().Error("获取客户端地址配置失败", zap.Any("err", err))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "失败"})
		return
	}
	sys := &config.SystemConfig{}
	err = jsoniter.UnmarshalFromString(data, sys)
	if err != nil {
		zap.L().Error("解析系统配置失败", zap.Any("err", err))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "失败"})
		return
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: sys, Msg: "成功"})
}

func UpdateGameUrl(ctx *gin.Context) {
	gameUrl := ctx.Query("gameUrl")
	replay := ctx.Query("replay")
	arr := strings.Split(gameUrl, ",")
	rs := strings.Split(replay, ",")
	data, err := dao.RedisIns().Get(esindex.ConfigKey("system"))
	if err != nil {
		zap.L().Error("获取客户端地址配置失败", zap.Any("err", err))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "失败"})
		return
	}
	sys := &config.SystemConfig{}
	err = jsoniter.UnmarshalFromString(data, sys)
	if err != nil {
		zap.L().Error("解析系统配置失败", zap.Any("err", err))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "失败"})
		return
	}
	sys.GameUrls = arr
	sys.Replays = rs
	zap.L().Debug("获取系统配置", zap.Any("sys", sys))
	r, _ := jsoniter.MarshalToString(sys)
	dao.RedisIns().Set(esindex.ConfigKey("system"), r, -1)
	dataStr, _ := jsoniter.MarshalToString(map[string]interface{}{
		"key":  esindex.ConfigKey("system"),
		"data": r,
	})
	str, _ := jsoniter.MarshalToString(map[string]interface{}{
		"event": "config",
		"data":  dataStr,
	})
	dao.RedisIns().Publish("message", str)
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "成功"})
}

func ApiConfigAdd(ctx *gin.Context) {
	name := ctx.Query("name")
	client_api_urls := ctx.Query("client_api_urls")
	hall_urls := ctx.Query("hall_urls")
	max_score, _ := strconv.Atoi(ctx.Query("max_score"))
	min_score, _ := strconv.Atoi(ctx.Query("min_score"))
	apiConfig := &manager.ApiConfig{
		Name:          name,
		Min:           min_score,
		Max:           max_score,
		ClientApiUrls: client_api_urls,
		HallUrls:      hall_urls,
	}
	dao.Mysql().Manager.Create(apiConfig)
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "成功"})
}

func ApiConfigDel(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Query("id"))
	dao.Mysql().Manager.Model(manager.ApiConfig{}).Delete("id=?", id)
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "成功"})
}

func ApiConfigUpdate(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Query("id"))
	name := ctx.Query("name")
	client_api_urls := ctx.Query("client_api_urls")
	hall_urls := ctx.Query("hall_urls")
	max_score, _ := strconv.Atoi(ctx.Query("max_score"))
	min_score, _ := strconv.Atoi(ctx.Query("min_score"))

	apiConfig := &manager.ApiConfig{
		Id:            int64(id),
		Name:          name,
		Min:           min_score,
		Max:           max_score,
		ClientApiUrls: client_api_urls,
		HallUrls:      hall_urls,
	}

	dao.Mysql().Manager.Save(apiConfig)
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "成功"})
}

func GameAgentList(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.Query("page"))
	pageSize, _ := strconv.Atoi(ctx.Query("pageSize"))
	name := ctx.Query("name")
	list := make([]*manager.Game, 0)
	offset := (page - 1) * pageSize
	var db *gorm.DB = dao.Mysql().Manager.Model(manager.Game{}).Debug()
	if len(name) > 0 {
		tag := "%" + name + "%"
		db = db.Where("name like ? or nameZH like ?", tag, tag)
	}
	var count int64 = 0
	db.Count(&count)
	if page > 0 && pageSize > 0 {
		db = db.Offset(offset).Limit(pageSize)
	}
	db.Find(&list)
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: map[string]interface{}{
		"total": count,
		"list":  list,
	}, Msg: "成功"})
}

func GameUrls(ctx *gin.Context) {
	// agentId, _ := strconv.Atoi(ctx.Query("page"))
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: map[string]interface{}{
		"data": config.CfgIns.GetSystemConfig(),
	}, Msg: "成功"})
}

func ClearGameState(ctx *gin.Context) {
	playerId, _ := strconv.Atoi(ctx.Query("playerId"))
	gameId, _ := strconv.Atoi(ctx.Query("gameId"))
	currency := ctx.Query("currency")
	game := &manager.Game{}
	dao.Mysql().Manager.Where("number=?", gameId).Take(game)
	dao.Es().DeleteByQuery(esindex.GameStates()).Query(elastic.NewBoolQuery().Filter(elastic.NewTermQuery("_id", fmt.Sprintf("%d-%s-%s", playerId, game.ConfName, currency)))).WaitForCompletion(false).Do(context.Background())
	zap.L().Debug("清理玩家指定游戏状态成功", zap.Any("playerId", playerId), zap.Any("name", game.NameZH), zap.Any("symbol", game.ConfName), zap.Any("currency", currency))
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: map[string]interface{}{
		"data": config.CfgIns.GetSystemConfig(),
	}, Msg: "成功"})
}

func QueryOrder(ctx *gin.Context) {
	account := ctx.Query("account")
	roundId := ctx.Query("roundId")
	querys := make([]elastic.Query, 0)
	if account != "" {
		//此平台account也是 nickName
		querys = append(querys, elastic.NewMatchQuery("nickName", account))
	}
	if roundId != "" {
		querys = append(querys, elastic.NewMatchPhraseQuery("roundID", roundId))
	}
	querys = append(querys, elastic.NewRangeQuery("isTourist").Lte(0))
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	resp, err := dao.Es().Search().Index(esindex.Settlement()).
		FetchSourceContext(elastic.NewFetchSourceContext(true).Exclude("init", "log")).
		Query(boolQuery).
		Pretty(true).
		Sort("playedDate", false).
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
	result := &entity.SettlementList{Data: settlements, Total: resp.Hits.TotalHits.Value}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: result, Msg: "成功"})
}

// asc[]: {"totalProfLoss":0,"totalProfLossRate":0,"totalEffect":0,"controlRate":0,"score":0}
func ChangeStatus(ctx *gin.Context) {
	asc := ctx.Query("asc")
	items := make([]*view.AutoSingleCtrl, 0, 32)
	if asc == "" {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "参数异常", Data: nil})
		return
	}
	if err := jsoniter.UnmarshalFromString(asc, items); err != nil {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "参数异常", Data: nil})
		return
	}
	err := dao.RedisIns().Set(esindex.ConfigKey("autoCtrl"), asc, -1)
	if err != nil {
		zap.L().Error("保存自动单控配置失败", zap.Any("err", err))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "保存自动单控配置失败", Data: nil})
		return
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: nil})
}

func SaveSingleCtrlConfig(ctx *gin.Context) {
	asc := ctx.QueryArray("asc[]")
	items := make([]*view.AutoSingleCtrl, 0, 32)
	if len(asc) <= 0 {
		zap.L().Error("参数为空")
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "参数异常", Data: nil})
		return
	}
	hundred := decimal.NewFromInt(100)
	for _, v := range asc {
		a := &view.AutoSingleCtrl{}
		if err := jsoniter.UnmarshalFromString(v, a); err != nil {
			zap.L().Error("解析参数失败", zap.Any("err", err))
			ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "参数异常", Data: nil})
			return
		}
		if a.ControlRate.Abs().GreaterThan(hundred) {
			ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "控制概率必须在0~100范围内", Data: nil})
			return
		}
		if a.Score.LessThan(decimal.Zero) {
			ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "控制分数不能为负", Data: nil})
			return
		}
		items = append(items, a)
	}
	str, _ := jsoniter.MarshalToString(items)
	err := dao.RedisIns().Set(esindex.ConfigKey("autoCtrl"), str, -1)
	if err != nil {
		zap.L().Error("保存自动单控配置失败", zap.Any("err", err))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "保存自动单控配置失败", Data: nil})
		return
	}

	publishConfigMessage(esindex.ConfigKey("autoCtrl"), str)
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: nil})
}

func GetSingleCtrlConfig(ctx *gin.Context) {
	str, err := dao.RedisIns().Get(esindex.ConfigKey("autoCtrl"))
	if err != nil {
		zap.L().Error("获取自动单控配置失败", zap.Any("err", err))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "保存自动单控配置失败", Data: nil})
		return
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: str})
}

func SaveGameSettingData(ctx *gin.Context) {
	str := ctx.Query("config")
	ac := &config.AwardConfig{
		OddsConfig: make([]*config.AwardOddsConfigItem, 0, 16),
	}
	err := jsoniter.UnmarshalFromString(str, &ac.OddsConfig)
	if err != nil {
		zap.L().Error("配置异常", zap.Any("err", err))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "失败", Data: str})
		return
	}
	key := esindex.ConfigKey("ctrl", "default")
	str, _ = jsoniter.MarshalToString(ac)
	err = dao.RedisIns().Set(key, str, -1)
	if err != nil {
		zap.L().Error("保存配置失败", zap.Any("err", err))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "失败", Data: str})
		return
	}
	publishConfigMessage(key, str)
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: str})
}

func GetGameSettingData(ctx *gin.Context) {
	key := esindex.ConfigKey("ctrl", "default")
	str, err := dao.RedisIns().Get(key)
	if err != nil {
		zap.L().Error("获取配置失败", zap.Any("err", err))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "失败", Data: str})
	} else {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: str})
	}
}

func SyncAllPool(ctx *gin.Context) {
	str := ctx.Query("config")
	pool := &config.Pool{}
	if err := jsoniter.UnmarshalFromString(str, pool); err != nil {
		zap.L().Error("解析配置失败", zap.Any("err", err))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "失败", Data: nil})
		return
	}
	srcGame := &manager.Game{}
	dao.Mysql().Manager.Model(manager.Game{}).Where("number=?", pool.GameId).Take(srcGame)
	games := make([]*manager.Game, 0, 64)
	dao.Mysql().Manager.Model(manager.Game{}).Where("gameType=?", srcGame.GameType).Find(&games)
	n := 0
	piple := dao.RedisIns().Client.Pipeline()
	cs := make(map[string]string)
	for _, g := range games {
		key := esindex.ConfigKey("pool", g.ConfName)
		pool.Name = g.Name
		pool.NameZH = g.NameZH
		pool.GameId = int64(g.Number)
		pool.Symbol = g.ConfName
		ps, err := jsoniter.MarshalToString(pool)
		if err != nil {
			continue
		}
		cs[key] = ps
		piple.Set(context.Background(), key, ps, -1)
		n++
		if n >= 100 {
			n = 0
			_, err = piple.Exec(context.Background())
			if err != nil {
				zap.L().Error("保存配置失败", zap.Any("err", err))
				ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "失败", Data: nil})
				return
			}
			piple = dao.RedisIns().Client.Pipeline()
		}
	}
	if n > 0 {
		_, err := piple.Exec(context.Background())
		if err != nil {
			zap.L().Error("保存配置失败", zap.Any("err", err))
			ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "失败", Data: nil})
			return
		}
	}
	//广播更新配置
	for key, c := range cs {
		dataStr, _ := jsoniter.MarshalToString(map[string]interface{}{
			"key":  key,
			"data": c,
		})
		msg, _ := jsoniter.MarshalToString(map[string]interface{}{
			"event": "config",
			"data":  dataStr,
		})
		dao.RedisIns().Publish("message", msg)
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: nil})
}

func GetExchange(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: map[string]interface{}{
		"currency": config.CfgIns.Currency.Data,
	}, Msg: "成功"})
}

// 添加汇率配置
func EditExchange(ctx *gin.Context) {
	c := ctx.Query("config")
	tmp := &config.CurrencyMgr{}
	if err := jsoniter.UnmarshalFromString(c, tmp); err != nil {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Data: nil, Msg: "失败"})
		zap.L().Error("解析汇率配置失败", zap.Any("err", err), zap.Any("config", c))
		return
	}
	//更新配置
	config.CfgIns.SetCurrency(tmp)

	dataStr, _ := jsoniter.MarshalToString(config.CfgIns.Currency)

	dao.RedisIns().Set(esindex.ConfigKey("currency"), dataStr, -1)

	data := &entity.ConfigMsg{}
	data.Key = esindex.ConfigKey("currency")
	data.Data = dataStr

	str, _ := jsoniter.MarshalToString(data)
	msg, _ := jsoniter.MarshalToString(map[string]interface{}{
		"event": "config",
		"data":  str,
	})

	dao.RedisIns().Publish("message", msg)

	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: map[string]interface{}{
		"currency": config.CfgIns.Currency.Data,
	}, Msg: "成功"})
}

func GameCurrencys(ctx *gin.Context) {
	str, err := dao.RedisIns().Get(esindex.ConfigKey("currency"))
	if err != nil {
		zap.L().Error("保存自动单控配置失败", zap.Any("err", err))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "保存自动单控配置失败", Data: nil})
		return
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: str})
}

func GetControlCoefficientData(ctx *gin.Context) {
	key := esindex.ConfigKey("controlCoefficient")
	str, err := dao.RedisIns().Get(key)
	if err != nil || str == "" {
		str, _ = jsoniter.MarshalToString(config.DefaultControlCoefficientConfig())
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: str})
}

func SaveControlCoefficientData(ctx *gin.Context) {
	value := config.DefaultControlCoefficientConfig()
	str := ctx.Query("config")
	if err := jsoniter.UnmarshalFromString(str, value); err != nil {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Msg: "控制系数配置格式错误", Data: nil})
		return
	}
	if value.Bet == nil || value.Single == nil || value.SingleRounds.LessThan(decimal.Zero) {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Msg: "控制系数配置不能为空", Data: nil})
		return
	}
	for _, item := range append(value.Bet, value.Single...) {
		if item == nil || item.Coefficient.LessThan(decimal.Zero) || item.Coefficient.GreaterThan(decimal.NewFromInt(1)) {
			ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Msg: "控制系数必须在 0 到 1 之间", Data: nil})
			return
		}
	}
	str, _ = jsoniter.MarshalToString(value)
	key := esindex.ConfigKey("controlCoefficient")
	if err := dao.RedisIns().Set(key, str, -1); err != nil {
		zap.L().Error("保存控制系数配置失败", zap.Error(err))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "保存失败", Data: nil})
		return
	}
	config.CfgIns.SetControlCoefficient(value)
	publishConfigMessage(key, str)
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: str})
}

func publishConfigMessage(key, value string) {
	dataStr, _ := jsoniter.MarshalToString(map[string]interface{}{
		"key":  key,
		"data": value,
	})
	msg, _ := jsoniter.MarshalToString(map[string]interface{}{
		"event": "config",
		"data":  dataStr,
	})
	dao.RedisIns().Publish("message", msg)
}
