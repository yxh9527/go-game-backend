package v1

import (
	"app/entity"
	"app/tables/manager"
	"crypto/md5"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"
	"web-api/common"
	"web-api/dao"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
)

func Login(ctx *gin.Context) {
	xRealIp := ctx.GetHeader("x-real-ip")
	name := ctx.PostForm("name")
	systemUser := &manager.SystemUser{}
	pass := ctx.PostForm("password")
	pass = fmt.Sprintf("%x", md5.Sum([]byte(pass)))
	result := dao.Mysql().Manager.Where("account=? and password=?", name, pass).Take(systemUser)
	if systemUser.IpLimit != "*" {
		bFind := false
		arr := strings.Split(systemUser.IpLimit, ",")
		for _, v := range arr {
			if v == xRealIp {
				bFind = true
			}
		}
		if !bFind {
			ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusForbidden, Msg: "登陆受限"})
			ctx.Abort()
			return
		}
	}
	if result.RowsAffected <= 0 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusUnauthorized, Msg: "账号或密码错误!"})
		ctx.Abort()
		return
	}
	if systemUser.IsForzen > 0 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusUnauthorized, Msg: "账号已被冻结"})
		ctx.Abort()
		return
	}
	li := &entity.LoginInfo{
		Id:        int32(systemUser.Id),
		Name:      systemUser.Account,
		LoginTime: time.Now().Unix(),
		UserType:  int32(systemUser.UType),
		AgentId:   systemUser.AgentId,
		Rand:      rand.Int63n(999999),
	}
	str, _ := jsoniter.MarshalToString(li)
	token, _ := common.AesEncrypt(entity.AesKey, entity.AesIV, []byte(str))
	systemUser.LoginTime = time.Now().Unix()
	systemUser.LoginNum += systemUser.LoginNum
	dao.Mysql().Manager.Save(systemUser)
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "登陆成功", Data: &entity.LoginResult{
		Token: token,
		User: &entity.LoginResultUser{
			Id:       int(systemUser.Id),
			Name:     systemUser.Account,
			UserType: systemUser.UType,
		},
	}})
}

func AgentLogin(ctx *gin.Context) {
	xRealIp := ctx.GetHeader("x-real-ip")
	name := ctx.PostForm("name")
	systemUser := &manager.SystemUser{}
	pass := ctx.PostForm("password")
	pass = fmt.Sprintf("%x", md5.Sum([]byte(pass)))
	result := dao.Mysql().Manager.Where("account=? and password=?", name, pass).Take(systemUser)
	if systemUser.IpLimit != "*" {
		bFind := false
		arr := strings.Split(systemUser.IpLimit, ",")
		for _, v := range arr {
			if v == xRealIp {
				bFind = true
			}
		}
		if !bFind {
			ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusForbidden, Msg: "登陆受限"})
			ctx.Abort()
			return
		}
	}
	if result.RowsAffected <= 0 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusUnauthorized, Msg: "账号或密码错误!"})
		ctx.Abort()
		return
	}
	if systemUser.UType != 3 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusUnauthorized, Msg: "账号或密码错误!"})
		ctx.Abort()
		return
	}
	if systemUser.IsForzen > 0 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusUnauthorized, Msg: "账号已被冻结"})
		ctx.Abort()
		return
	}
	li := &entity.LoginInfo{
		Id:        int32(systemUser.Id),
		Name:      systemUser.Account,
		LoginTime: time.Now().Unix(),
		UserType:  int32(systemUser.UType),
		AgentId:   systemUser.AgentId,
		Rand:      rand.Int63n(999999),
	}
	str, _ := jsoniter.MarshalToString(li)
	token, _ := common.AesEncrypt(entity.AesKey, entity.AesIV, []byte(str))
	systemUser.LoginTime = time.Now().Unix()
	systemUser.LoginNum += systemUser.LoginNum
	dao.Mysql().Manager.Save(systemUser)
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "登陆成功", Data: &entity.LoginResult{
		Token: token,
		User: &entity.LoginResultUser{
			Id:       int(systemUser.Id),
			Name:     systemUser.Account,
			UserType: systemUser.UType,
		},
	}})
}
