package v1

import (
	"app/entity"
	"app/tables/manager"
	"crypto/md5"
	"fmt"
	"net/http"
	"strconv"
	"web-api/common"
	"web-api/dao"

	"github.com/gin-gonic/gin"
)

// 管理账号列表
func AdminList(ctx *gin.Context) {
	var total int64 = 0
	uType, _ := strconv.Atoi(ctx.Query("uType"))
	page, _ := strconv.Atoi(ctx.Query("page"))
	pageSize, _ := strconv.Atoi(ctx.Query("pageSize"))
	if pageSize <= 0 {
		pageSize = 30
	}
	offset := (page - 1) * pageSize
	accounts := make([]*manager.SystemUser, 0)
	db := dao.Mysql().Manager.Model(manager.SystemUser{}).Debug()
	db.Count(&total)
	if uType == 0 {
		db.Offset(offset).Limit(pageSize).Find(&accounts)
	} else {
		db.Offset(offset).Limit(pageSize).Where("uType = ?", uType).Find(&accounts)
	}
	result := &entity.ManagerAccountList{Data: accounts, Total: total, LastPage: common.CountPage(total, int64(pageSize)), Page: int64(page), PerPage: int64(pageSize)}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: result, Msg: "成功"})
}

// 添加账号
func AdminAdd(ctx *gin.Context) {
	uType, _ := strconv.Atoi(ctx.PostForm("uType"))
	uName := ctx.PostForm("uName")
	account := ctx.PostForm("account")
	password := ctx.PostForm("password")
	ipLimit := ctx.PostForm("ipLimit")
	var count int64 = 0

	if uType <= 0 || uType > 3 || len(password) <= 0 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Data: nil, Msg: "账号类型异常!"})
		return
	}

	if ipLimit == "" {
		ipLimit = "*"
	}

	dao.Mysql().Manager.Where("account = ?", account).Count(&count)
	if count > 0 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusBadRequest, Data: nil, Msg: "用户名已存在!"})
		return
	}
	p := fmt.Sprintf("%x", md5.Sum([]byte(password)))
	admin := &manager.SystemUser{
		UType:    uType,
		UName:    uName,
		Account:  account,
		IpLimit:  ipLimit,
		Password: p,
	}

	if dao.Mysql().Manager.Save(admin).RowsAffected > 0 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "成功"})
	} else {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Data: nil, Msg: "系统错误!"})
	}
}

// 删除账号
func AdminDel(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Query("id"))
	admin := &manager.SystemUser{}
	dao.Mysql().Manager.Debug().Delete(admin, id)
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "成功"})
}

// 修改账号
func AdminEdit(ctx *gin.Context) {
	uType, _ := strconv.Atoi(ctx.PostForm("uType"))
	id, _ := strconv.Atoi(ctx.PostForm("id"))
	uName := ctx.PostForm("uName")
	account := ctx.PostForm("account")
	password := ctx.PostForm("password")
	ipLimit := ctx.PostForm("ipLimit")

	if ipLimit == "" {
		ipLimit = "*"
	}

	admin := &manager.SystemUser{}
	dao.Mysql().Manager.Model(manager.SystemUser{}).Debug().Where("id=?", id).Take(admin)
	if len(password) > 0 {
		p := fmt.Sprintf("%x", md5.Sum([]byte(password)))
		admin.Password = p
	}
	admin.UType = uType
	admin.UName = uName
	admin.Account = account
	admin.IpLimit = ipLimit
	if dao.Mysql().Manager.Model(manager.SystemUser{}).Debug().Where("id=?", id).Updates(admin).RowsAffected > 0 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "成功"})
	} else {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Data: nil, Msg: "系统错误!"})
	}
}

// 修改状态
func AdminChangeState(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.PostForm("id"))
	isForzen, _ := strconv.Atoi(ctx.PostForm("isForzen"))

	if dao.Mysql().Manager.Model(manager.SystemUser{}).Debug().Exec("update gp_system_user set isForzen=? where id=?", isForzen, id).RowsAffected > 0 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: nil, Msg: "成功"})
	} else {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Data: nil, Msg: "系统错误!"})
	}
}
