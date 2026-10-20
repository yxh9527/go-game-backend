package v1

import (
	"app/entity"
	"app/tables/manager"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"web-api/dao"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
)

func GamerGameList(ctx *gin.Context) {
	games := make([]*manager.Game, 0)
	dao.Mysql().Manager.Find(&games)
	result := []*entity.GamerListResult{
		{Id: 1, Name: "", Data: games},
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: result, Msg: "成功"})
}

func UpdateGameState(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.PostForm("id"))
	isFrozen, _ := strconv.Atoi(ctx.PostForm("isFrozen"))
	g := &manager.Game{}
	dao.Mysql().Manager.Model(manager.Game{}).Debug().Where("number=?", id).Take(g)
	dao.Mysql().Manager.Model(manager.Game{}).Debug().Exec("update gp_game set state=? where number=?", isFrozen, id)
	dataStr, _ := jsoniter.MarshalToString(map[string]interface{}{
		"symbol": g.ConfName,
		"status": isFrozen,
	})
	str, _ := jsoniter.MarshalToString(map[string]interface{}{
		"event": "gameStatuChange",
		"data":  dataStr,
	})
	dao.RedisIns().Publish("message", str)

	action := "stop"
	if isFrozen == 1 {
		action = "start"
	}

	dao.RedisIns().Publish("server-message", fmt.Sprintf("{\"event\":\"%s\",\"msg\":\"%s\",\"ids\":[%d]}", action, "gameStatusChange", id))
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "成功"})
}

func StopAllGame(ctx *gin.Context) {
	games := make([]*manager.Game, 0, 64)
	dao.Mysql().Manager.Model(manager.Game{}).Debug().Where("state=1").Find(&games)
	dao.Mysql().Manager.Model(manager.Game{}).Debug().Exec("update gp_game set state=2 where state !=0 ")
	ids := make([]string, 0, 32)
	for _, game := range games {
		dataStr, _ := jsoniter.MarshalToString(map[string]interface{}{
			"symbol": game.ConfName,
			"status": 2,
		})
		str, _ := jsoniter.MarshalToString(map[string]interface{}{
			"event": "gameStatuChange",
			"data":  dataStr,
		})
		dao.RedisIns().Publish("message", str)
		ids = append(ids, fmt.Sprintf("%d", game.Number))
	}
	dao.RedisIns().Publish("server-message", fmt.Sprintf("{\"event\":\"stop\",\"msg\":\"%s\",\"ids\":[%s]}", "gameStatusChange", strings.Join(ids, ",")))
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "成功"})
}

func StartAllGame(ctx *gin.Context) {
	games := make([]*manager.Game, 0, 64)
	dao.Mysql().Manager.Model(manager.Game{}).Debug().Where("state=2").Find(&games)
	dao.Mysql().Manager.Model(manager.Game{}).Debug().Exec("update gp_game set state=1  where state !=0 ")
	ids := make([]string, 0, 32)
	for _, game := range games {
		dataStr, _ := jsoniter.MarshalToString(map[string]interface{}{
			"symbol": game.ConfName,
			"status": 1,
		})
		str, _ := jsoniter.MarshalToString(map[string]interface{}{
			"event": "gameStatuChange",
			"data":  dataStr,
		})
		dao.RedisIns().Publish("message", str)
		ids = append(ids, fmt.Sprintf("%d", game.Number))
	}
	dao.RedisIns().Publish("server-message", fmt.Sprintf("{\"event\":\"start\",\"msg\":\"%s\",\"ids\":[%s]}", "gameStatusChange", strings.Join(ids, ",")))
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "成功"})
}
