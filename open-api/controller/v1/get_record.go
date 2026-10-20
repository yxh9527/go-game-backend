package v1

import (
	"app/tables/manager"
	"net/http"
	"net/url"
	. "open-api/common"
	. "open-api/dao"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func GetList(ctx *gin.Context, params url.Values, agent *manager.Agent) {
	rowVersion := ctx.Query("rowVersion")
	if rowVersion != "" {
		rv, err := strconv.ParseInt(rowVersion, 10, 64)
		if err != nil {
			ctx.JSON(http.StatusOK, GetJsonObj(API_GAME_DATA.String(), &SimpleResp{Code: int(CODE_CHECK_FIELD_LOSS)}))
			return
		}
		result := Es().GetListWithRowVersion(agent.Id, rv)
		if result != nil {
			ctx.JSON(http.StatusOK, GetJsonObj(API_GAME_DATA.String(), &DataResp{Code: int(CODE_OK), List: result}))
		} else {
			ctx.JSON(http.StatusOK, GetJsonObj(API_GAME_DATA.String(), &SimpleResp{Code: int(CODE_DATA_NOT)}))
		}
	} else {
		startTime, _ := strconv.ParseInt(params.Get("startTime"), 10, 64)
		endTime, _ := strconv.ParseInt(params.Get("endTime"), 10, 64)
		result := Es().GetListWithTimeRange(agent.Id, startTime, endTime)
		if result != nil {
			ctx.JSON(http.StatusOK, GetJsonObj(API_GAME_DATA.String(), &DataResp{Code: int(CODE_OK), List: result}))
		} else {
			ctx.JSON(http.StatusOK, GetJsonObj(API_GAME_DATA.String(), &SimpleResp{Code: int(CODE_DATA_NOT)}))
		}
	}
}

func GetScore(ctx *gin.Context, params url.Values, agent *manager.Agent) {
	acc := params.Get("account")
	if acc == "" {
		zap.L().Error("参数异常,获取玩家积分失败")
		ctx.JSON(http.StatusOK, GetJsonObj(API_BALANCE.String(), &SimpleResp{Code: int(CODE_CHECK_FIELD_LOSS)}))
		return
	}
	agentPlayer := Mysql().GetAgentPlayerInfoByAgentIdAndAcc(agent.Id, acc)
	if agentPlayer == nil {
		zap.L().Error("参数异常,玩家不存在", zap.Any("acc", acc))
		ctx.JSON(http.StatusOK, GetJsonObj(API_BALANCE.String(), &SimpleResp{Code: int(CODE_ACCOUNT_NOT)}))
		return
	}

	if Redis().InGame(agentPlayer.Id) {
		zap.L().Error("玩家正在游戏中", zap.Any("acc", acc), zap.Any("id", agentPlayer.Id))
		ctx.JSON(http.StatusOK, GetJsonObj(API_BALANCE.String(), &SimpleResp{Code: int(CODE_ACCOUNT_IN_GAME)}))
		return
	}
	//注意单位 redis里面存的是分为单位，mysql里面存的是实际的钱
	currency, ok := Redis().GetGameCurrency(agentPlayer.Id)
	if ok {
		m, _ := currency.Float64()
		//redis 是以分为单位
		ctx.JSON(http.StatusOK, GetJsonObj(API_BALANCE.String(), &ScoreResp{Code: int(CODE_OK), Money: m / 100}))
	} else {
		player := Mysql().GetGamePlayer(agent.Id, agentPlayer.Id)
		if player != nil {
			m, _ := player.Score.Float64()
			ctx.JSON(http.StatusOK, GetJsonObj(API_BALANCE.String(), &ScoreResp{Code: int(CODE_OK), Money: m}))
		}
	}
}
