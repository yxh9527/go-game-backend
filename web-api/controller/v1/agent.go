package v1

import (
	"app/entity"
	"app/entity/view"
	"app/tables/manager"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"web-api/common"
	"web-api/dao"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/redis/go-redis/v9"
)

func GetAgentList(ctx *gin.Context) {
	var total int64 = 0
	nickName := ctx.Query("name")
	page, _ := strconv.Atoi(ctx.Query("page"))
	pageSize, _ := strconv.Atoi(ctx.Query("pageSize"))
	if pageSize <= 0 {
		pageSize = 30
	}
	offset := (page - 1) * pageSize
	agents := make([]*view.AgentListItem, 0)
	db := dao.Mysql().Manager.Debug().Model(manager.Agent{}).Select("`gp_agent`.*,`gp_web`.nickName as webName").Order("id desc").Joins("inner join `gp_web` on `gp_agent`.webId=`gp_web`.id")
	db.Count(&total)
	db = db.Offset(offset).Limit(pageSize)
	if len(nickName) > 0 {
		db.Where("gp_agent.nickName like ?", "%"+nickName+"%").Find(&agents)
	} else {
		db.Find(&agents)
	}
	result := &entity.AgentList{Data: agents, Total: total, LastPage: common.CountPage(total, int64(pageSize)), Page: int64(page), PerPage: int64(pageSize)}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: result, Msg: "成功"})
}

func GetAgentSelectList(ctx *gin.Context) {
	nickName := ctx.Query("name")
	agents := make([]*manager.Agent, 0)
	if len(nickName) > 0 {
		dao.Mysql().Manager.Debug().Select("id,nickName,level").Where("nickName like ?", "%"+nickName+"%").Find(&agents)
	} else {
		dao.Mysql().Manager.Debug().Select("id,nickName,level").Find(&agents)
	}
	result := make([]*entity.AgentSelectItem, 0)
	for _, agent := range agents {
		sAgents := make([]*manager.Agent, 0)
		dao.Mysql().Manager.Debug().Select("id,nickName,level").Where("upperLevel=?", agent.Id).Find(&sAgents)
		result = append(result, &entity.AgentSelectItem{
			Id:      int(agent.Id),
			Name:    agent.NickName,
			Level:   int(agent.Level),
			SubList: sAgents,
		})
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: result, Msg: "成功"})
}

func GetAgentInfo(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Query("id"))
	agent := &manager.Agent{}
	agentGames := make([]*manager.AgentGame, 0)
	dao.Mysql().Manager.Where("id=?", id).Take(agent)
	dao.Mysql().Manager.Where("agentId=?", id).Find(&agentGames)
	aagi := make([]*entity.AddAgentGameIdsItem, 0)
	tmp := &entity.AddAgentGameIdsItem{}
	tmp.Id = 1
	for _, ag := range agentGames {
		tmp.Val = append(tmp.Val, int(ag.GameId))
	}
	aagi = append(aagi, tmp)
	nameList := make([]string, 0)
	games := make([]*manager.Game, 0)
	dao.Mysql().Manager.Debug().Where("id in ?", tmp.Val).Find(&games)
	for _, v := range games {
		nameList = append(nameList, v.Name)
	}
	agentInfo := &entity.AgentInfo{
		Agent: agent,
		GameList: []*entity.AgentInfoGameList{
			{
				Id:       aagi[0].Id,
				Val:      aagi[0].Val,
				NameList: nameList,
			},
		},
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: agentInfo, Msg: "成功"})
}

func AgentEdit(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.PostForm("id"))
	remarks := ctx.PostForm("remarks")
	isFrozen, _ := strconv.Atoi(ctx.PostForm("isFrozen"))
	dao.Mysql().Manager.Exec("update gp_agent set isFrozen=? where id=?", isFrozen, id)
	if len(remarks) > 0 {
		dao.Mysql().Manager.Exec("update gp_agent set remarks=? where id=?", remarks, id)
	}
	res := dao.RedisIns().Client.ZAdd(context.Background(), "agent_stop", redis.Z{Member: fmt.Sprintf("%d", id), Score: 1})
	if res.Err() == nil {
		dataStr, _ := jsoniter.MarshalToString(map[string]interface{}{
			"id":     id,
			"status": isFrozen,
		})
		str, _ := jsoniter.MarshalToString(map[string]interface{}{
			"event": "agentStatuChange",
			"data":  dataStr,
		})
		dao.RedisIns().Publish("message", str)
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "成功"})
	}
}

func ServerStop(ctx *gin.Context) {
	// _, err := dao.Etcd().Put(ctx, "/config/system_stop", "{\"stop\":1}")
	// if err != nil {
	// 	zap.L().Error("关闭系统失败", zap.Any("err", err))
	// 	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Data: nil, Msg: "失败"})
	// 	return
	// }
	// // common.KickByServer()
	// ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "成功"})
}

func ServerStart(ctx *gin.Context) {
	// _, err := dao.Etcd().Delete(ctx, "/config/system_stop")
	// if err != nil {
	// 	zap.L().Error("开启系统失败", zap.Any("err", err))
	// 	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Data: nil, Msg: "失败"})
	// 	return
	// }
	// ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "成功"})
}

func MsgList(ctx *gin.Context) {
	msgType := ctx.Query("msgType")
	msgs := make([]*manager.SystemUserMsg, 0)
	sql := dao.Mysql().Manager.Model(manager.SystemUserMsg{}).Debug().
		Select("a.id,a.title,a.isRead,a.createTime,b.account as sendName").
		Joins("a inner join gp_system_user b on a.sendId=b.id").
		Where("a.receiveId=? and a.isDel=0", ctx.GetInt64("id"))
	if len(msgType) > 0 {
		if msgType == "1" {
			sql = sql.Where("a.isRead=?", 0)
		} else {
			sql = sql.Where("a.isRead=?", 1)
		}
	}
	sql.Order("a.createTime desc").Find(&msgs)
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"code": 200,
		"data": msgs,
		"msg":  "成功",
	})
}
