package v1

import (
	"app/entity"
	"app/tables/manager"
	"net/http"
	"strconv"
	"time"
	"web-api/common"
	"web-api/dao"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Logout(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "成功"})
}

// 三级联动数据
func LinkAgent(ctx *gin.Context) {
	webs := make([]*manager.Web, 0)
	if dao.Mysql().Manager.Find(&webs).RowsAffected <= 0 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "暂无平台数据"})
		ctx.Abort()
		return
	}
	result := make([]*entity.LinkAgentResult, 0)
	for _, v := range webs {
		agents := make([]*manager.Agent, 0)
		dao.Mysql().Manager.Where("webId = ?", v.Id).Find(&agents)
		agentsData := make([]*entity.LinkAgentData, 0)
		for _, agent := range agents {
			item := &entity.LinkAgentData{
				Id:   int(agent.Id),
				Name: agent.NickName,
				// GameIds: agent.GameIds,
			}
			agentsData = append(agentsData, item)
		}
		result = append(result, &entity.LinkAgentResult{
			Id:     int(v.Id),
			Name:   v.NickName,
			Agents: agentsData,
		})
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: result, Msg: "成功"})
}

// 管理账号列表
func WebList(ctx *gin.Context) {
	var total int64 = 0
	name := ctx.Query("name")
	page, _ := strconv.Atoi(ctx.Query("page"))
	pageSize, _ := strconv.Atoi(ctx.Query("pageSize"))
	if pageSize <= 0 {
		pageSize = 30
	}
	zap.L().Debug("webList 请求参数", zap.Any("page", page), zap.Any("pageSize", pageSize))
	dao.Mysql().Manager.Model(manager.Web{}).Count(&total)
	offset := (page - 1) * pageSize
	webs := make([]*manager.Web, 0)
	if len(name) <= 0 {
		dao.Mysql().Manager.Debug().Order("id desc").Offset(int(offset)).Limit(int(pageSize)).Find(&webs)
	} else {
		dao.Mysql().Manager.Debug().Order("id desc").Offset(int(offset)).Limit(int(pageSize)).Where("nickName like ?", "%"+name+"%").Find(&webs)
	}
	result := &entity.WebListResult{
		Data:     webs,
		LastPage: common.CountPage(total, int64(pageSize)),
		Total:    total,
		Page:     int64(page),
		PerPage:  int64(pageSize),
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: result, Msg: "成功"})
}

// 添加平台账号
func WebAdd(ctx *gin.Context) {
	nickName := ctx.PostForm("nickName")
	contacts := ctx.PostForm("contacts")
	phone := ctx.PostForm("phone")
	email := ctx.PostForm("email")
	if len(nickName) <= 0 || len(nickName) > 50 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Msg: "昵称不能超过50个字符"})
		ctx.Abort()
		return
	}
	if len(contacts) <= 0 || len(contacts) > 50 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Msg: "负责人名称不能超过50个字符"})
		ctx.Abort()
		return
	}
	if len(phone) <= 0 || len(phone) > 50 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Msg: "电话号码不能超过50个字符"})
		ctx.Abort()
		return
	}
	if len(email) <= 0 || len(email) > 50 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Msg: "邮箱不能超过50个字符"})
		ctx.Abort()
		return
	}
	web := &manager.Web{
		NickName:  nickName,
		Contacts:  contacts,
		Phone:     phone,
		Email:     email,
		StartTime: time.Now().Unix(),
		EndTime:   time.Now().AddDate(999, 0, 0).Unix(),
	}
	dao.Mysql().Manager.Create(web)
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "成功"})
}

// 修改平台账号
func WebEdit(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.PostForm("id"))
	nickName := ctx.PostForm("nickName")
	contacts := ctx.PostForm("contacts")
	phone := ctx.PostForm("phone")
	email := ctx.PostForm("email")
	remarks := ctx.PostForm("remarks")
	if len(nickName) <= 0 || len(nickName) > 50 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Msg: "昵称不能超过50个字符"})
		ctx.Abort()
		return
	}
	if len(contacts) <= 0 || len(contacts) > 50 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Msg: "负责人名称不能超过50个字符"})
		ctx.Abort()
		return
	}
	if len(phone) <= 0 || len(phone) > 50 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Msg: "电话号码不能超过50个字符"})
		ctx.Abort()
		return
	}
	if len(email) <= 0 || len(email) > 50 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Msg: "邮箱不能超过50个字符"})
		ctx.Abort()
		return
	}

	if len(remarks) <= 0 || len(remarks) > 50 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Msg: "备注不能超过50个字符"})
		ctx.Abort()
		return
	}
	web := &manager.Web{
		Id:        int64(id),
		NickName:  nickName,
		Contacts:  contacts,
		Phone:     phone,
		Email:     email,
		StartTime: time.Now().Unix(),
		EndTime:   time.Now().AddDate(999, 0, 0).Unix(),
		Remarks:   remarks,
	}
	dao.Mysql().Manager.Save(web)
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "成功"})
}
