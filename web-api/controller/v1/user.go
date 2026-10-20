package v1

import (
	"app/entity"
	"app/entity/view"
	"app/tables/manager"
	"net/http"
	"strconv"
	"web-api/dao"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

func UserInfo(ctx *gin.Context) {
	userId, _ := strconv.ParseInt(ctx.Query("id"), 10, 0)
	userInfo := &view.UserInfo{}
	dao.Mysql().Manager.Model(manager.User{}).Debug().Select("`gp_user`.*,`gp_agent`.nickName as agentName,`gp_web`.nickName as webName").
		Joins("inner join `gp_agent` on `gp_user`.agentId=`gp_agent`.id inner join `gp_web` on `gp_user`.webId=`gp_web`.id").Where("`gp_user`.id=?", userId).Take(userInfo)
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: userInfo})
}

func UpdateUserState(ctx *gin.Context) {
	state, errState := strconv.Atoi(ctx.Query("state"))
	str := ctx.Query("id")
	if str == "" || errState != nil {
		zap.L().Debug("修改用户状态失败,参数错误", zap.Any("state", state), zap.Any("id", str))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Msg: "参数错误", Data: nil})
		return
	}
	ids := make([]int64, 0)
	err := jsoniter.UnmarshalFromString(str, &ids)
	if err != nil {
		zap.L().Debug("修改用户状态失败,参数错误", zap.Any("id", str), zap.Any("err", err))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Msg: "参数错误", Data: nil})
		return
	}
	dao.Mysql().Manager.Model(manager.User{}).Debug().Exec("update gp_user set state=? where id in ?", state, ids)
	// if state == 2 {
	// 	for _, v := range ids {
	// 		common.KickByPlayer(v)
	// 	}
	// }
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: nil})
}

func UserScoreLogs(ctx *gin.Context) {
	id, IDerr := strconv.Atoi(ctx.Query("id"))
	if IDerr != nil {
		zap.L().Debug("获取用户上下分数据失败", zap.Any("err", IDerr))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Msg: "参数错误", Data: nil})
		return
	}
	page, pageErr := strconv.Atoi(ctx.Query("page"))
	if pageErr != nil {
		page = 1
	}
	logs := make([]*manager.UserScoreLog, 0, 128)
	pageSize := 30
	var count int64 = 0
	dao.Mysql().Manager.Model(manager.UserScoreLog{}).Debug().Where("userId=?", id).Count(&count)
	dao.Mysql().Manager.Model(manager.UserScoreLog{}).Debug().Offset((page-1)*pageSize).Where("userId=?", id).Find(&logs)
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: map[string]interface{}{
		"total":     count,
		"data":      logs,
		"page_size": pageSize,
	}})
}
