package v1

import (
	"app/tables/manager"
	"net/http"
	"net/url"
	. "open-api/common"
	. "open-api/dao"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func AddScore(ctx *gin.Context, params url.Values, agent *manager.Agent) {
	acc := params.Get("account")
	moneyStr := params.Get("money")
	money, err := strconv.ParseFloat(moneyStr, 64)
	if err != nil {
		zap.L().Error("参数错误", zap.Any("money", money))
		ctx.JSON(http.StatusOK, GetJsonObj(API_ADD_SCORE.String(), &SimpleResp{Code: int(CODE_CHECK_FIELD_LOSS)}))
		return
	}

	if money < 0 {
		zap.L().Error("金额不能小于0", zap.Any("money", money))
		ctx.JSON(http.StatusOK, GetJsonObj(API_ADD_SCORE.String(), &SimpleResp{Code: int(CODE_UP_ACCOUNT_SCORE_ERR)}))
		return
	}

	agentPlayer := Mysql().GetAgentPlayerInfoByAgentIdAndAcc(agent.Id, acc)
	if agentPlayer == nil {
		zap.L().Error("玩家不存在", zap.Int64("agentId", agent.Id), zap.Any("acc", acc))
		ctx.JSON(http.StatusOK, GetJsonObj(API_ADD_SCORE.String(), &SimpleResp{Code: int(CODE_ACCOUNT_NOT)}))
		return
	}
	str := Redis().UserLock(agentPlayer.Id, 30)
	if str != "" {
		defer func() {
			Redis().UserUnlock(agentPlayer.Id, str)
		}()
	} else {
		zap.L().Error("玩家正在游戏中", zap.Any("agentId", agent.Id), zap.Any("acc", acc))
		ctx.JSON(http.StatusOK, GetJsonObj(API_ADD_SCORE.String(), &SimpleResp{Code: int(CODE_ACCOUNT_IN_GAME)}))
		return
	}
	player := Mysql().GetGamePlayer(agent.Id, agentPlayer.Id)
	if player == nil {
		zap.L().Error("玩家不存在", zap.Any("agentId", agent.Id), zap.Any("acc", acc))
		ctx.JSON(http.StatusOK, GetJsonObj(API_ADD_SCORE.String(), &SimpleResp{Code: int(CODE_ACCOUNT_NOT)}))
		return
	}
	currencyScore, err := Redis().UpdatePlayerCurrency(uint32(player.UserId), int64(money*100))
	if err == redis.Nil {
		p := ConvertUserEntityToHumanPlayer(player)
		if pe := Redis().SetPlayer(p); pe != nil {
			zap.L().Warn("向redis写入玩家信息缓存失败", zap.Any("req", params), zap.Error(pe))
			ctx.JSON(http.StatusOK, GetJsonObj(API_ADD_SCORE.String(), &SimpleResp{Code: int(CODE_UP_ACCOUNT_SCORE_ERR)}))
			return
		}
		currencyScore, err = Redis().UpdatePlayerCurrency(uint32(player.UserId), int64(money*100))
	}
	if err != nil {
		zap.L().Error("更新玩家游戏币和经验失败", zap.Any("req", params), zap.Error(err))
		ctx.JSON(http.StatusOK, GetJsonObj(API_ADD_SCORE.String(), &SimpleResp{Code: int(CODE_UP_ACCOUNT_SCORE_ERR)}))
		return
	}
	score := decimal.NewFromInt(currencyScore)
	if !Mysql().AddScoreLog(int(player.UserId), agent.Id, acc, money, 1, score.Div(decimal.NewFromInt(100))) {
		zap.L().Error("记录积分变更日志失败", zap.Any("agentId", agent.Id), zap.Any("acc", acc), zap.Any("playerId", player.UserId), zap.Any("score", player.Score))
	}
	zap.L().Debug("增加积分成功")
	ctx.JSON(http.StatusOK, GetJsonObj(API_ADD_SCORE.String(), &SimpleResp{Code: int(CODE_OK)}))
}
