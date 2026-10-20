package controller

import (
	"app/tables/manager"
	"crypto/md5"
	"fmt"
	"net/http"
	"net/url"
	. "open-api/common"
	v1 "open-api/controller/v1"
	"open-api/dao"
	"strconv"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Dispatch(c *gin.Context, agent *manager.Agent, plainTextBytes string) {
	if p, err := url.ParseQuery(plainTextBytes); err != nil {
		zap.L().Error("解密后的明文格式不对", zap.String("str", plainTextBytes))
		c.JSON(http.StatusOK, GetJsonObj("", &SimpleResp{Code: int(CODE_URL_DECRYPT_ERR)}))
	} else {
		s := p.Get("s")
		switch s {
		case API_LOGIN.String():
			v1.Login(c, p, agent)
		case API_ADD_SCORE.String():
			v1.AddScore(c, p, agent)
		case API_SUB_SCORE.String():
			v1.SubScore(c, p, agent)
		case API_KICK_OUT.String():
			v1.Kick(c, p, agent)
		case API_BALANCE.String():
			v1.GetScore(c, p, agent)
		case API_GAME_DATA.String():
			v1.GetList(c, p, agent)
		case API_WEB_AUTH.String():
			// v1.WebAuth(c, p, agent)
		case API_UPDATE_CURRENCYTYPE.String():
			v1.UpdateCurrencyType(c, p, agent)
		}
	}
}

// 初始化gin
func NewRouter() *gin.Engine {
	engine := gin.New()
	engine.Use(ginzap.RecoveryWithZap(zap.L(), true))
	g := engine
	limitC := make(chan int, 120)
	//设置跨域
	g.Use(Cors())
	g.GET("/api/gen", func(c *gin.Context) {
		aesKey := "1356CCA67971C527"
		aesIv := "BD36A1A5D6F936EC"
		agentMd5key := "2904D1C1DEFEA6F8F3110CB3ABCE2360"
		params, md5Key := "", ""
		timeStamp := time.Now().Unix()
		md5Key = fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%d%d%s", 0, timeStamp, agentMd5key))))
		at := c.Query("s")
		switch at {
		case API_LOGIN.String():
			currencyType := c.Query("currencyType")
			lang := c.Query("lang")
			account := c.Query("account")
			nickName := c.Query("nickName")
			ip := c.Query("ip")
			money := c.Query("money")
			symbol := c.Query("symbol")
			isTourist := c.Query("isTourist")
			src := fmt.Sprintf("s=%s&account=%s&nickName=%s&ip=%s&money=%s&symbol=%s&currencyType=%s&lang=%s&isTourist=%s", at, account, nickName, ip, money, symbol, currencyType, lang, isTourist)
			dst, err := AesEncrypt(aesKey, aesIv, []byte(src))
			if err != nil {
				zap.L().Error("加密失败", zap.Any("src", src), zap.Any("err", err))
			}
			params = dst
		case API_ADD_SCORE.String():
			account := c.Query("account")
			money := c.Query("money")
			src := fmt.Sprintf("s=%s&account=%s&money=%s", at, account, money)
			dst, err := AesEncrypt(aesKey, aesIv, []byte(src))
			if err != nil {
				zap.L().Error("加密失败", zap.Any("src", src), zap.Any("err", err))
			}
			params = dst
		case API_SUB_SCORE.String():
			account := c.Query("account")
			money := c.Query("money")
			src := fmt.Sprintf("s=%s&account=%s&money=%s", at, account, money)
			dst, err := AesEncrypt(aesKey, aesIv, []byte(src))
			if err != nil {
				zap.L().Error("加密失败", zap.Any("src", src), zap.Any("err", err))
			}
			params = dst
		case API_KICK_OUT.String():
			account := c.Query("accounts")
			src := fmt.Sprintf("s=%s&accounts=%s", at, account)
			dst, err := AesEncrypt(aesKey, aesIv, []byte(src))
			if err != nil {
				zap.L().Error("加密失败", zap.Any("src", src), zap.Any("err", err))
			}
			params = dst
		case API_GAME_DATA.String():
			startTime := c.Query("startTime")
			endTime := c.Query("endTime")
			src := fmt.Sprintf("s=%s&startTime=%s&endTime=%s", at, startTime, endTime)
			dst, err := AesEncrypt(aesKey, aesIv, []byte(src))
			if err != nil {
				zap.L().Error("加密失败", zap.Any("src", src), zap.Any("err", err))
			}
			params = dst
		case API_WEB_AUTH.String():
			account := c.Query("account")
			nickName := c.Query("nickName")
			ip := c.Query("ip")
			src := fmt.Sprintf("s=%s&account=%s&nickName=%s&ip=%s", at, account, nickName, ip)
			dst, err := AesEncrypt(aesKey, aesIv, []byte(src))
			if err != nil {
				zap.L().Error("加密失败", zap.Any("src", src), zap.Any("err", err))
			}
			params = dst
		case API_BALANCE.String():
			account := c.Query("account")
			src := fmt.Sprintf("s=%s&account=%s", at, account)
			dst, err := AesEncrypt(aesKey, aesIv, []byte(src))
			if err != nil {
				zap.L().Error("加密失败", zap.Any("src", src), zap.Any("err", err))
			}
			params = dst
		}
		c.Redirect(http.StatusPermanentRedirect, fmt.Sprintf("/channelHandle?agent=0&timestamp=%d&param=%s&key=%s", timeStamp, params, md5Key))
	})
	g.GET("/channelHandle", func(c *gin.Context) {
		limitC <- 1
		defer func() {
			<-limitC
		}()
		agentId, _ := strconv.ParseInt(c.Query("agent"), 10, 32)
		timestamp, _ := strconv.ParseInt(c.Query("timestamp"), 10, 64)
		param := c.Query("param")
		key := c.Query("key")
		zap.L().Debug("请求", zap.Any("agentId", agentId), zap.Any("param", param), zap.Any("key", key), zap.Any("timestamp", timestamp))
		//代理不存在
		agent := dao.GetAgentMgr().Get(agentId)
		if agent == nil {
			zap.L().Debug("代理不存在", zap.Any("agentId", agentId))
			c.JSON(http.StatusOK, GetJsonObj("", &SimpleResp{Code: int(CODE_AGENT_NOT)}))
			c.Abort()
			return
		}
		if len(agent.AesKey) < 32 {
			zap.L().Debug("代理数据不正确", zap.Any("AesKey", agent.AesKey))
			c.JSON(http.StatusOK, GetJsonObj("", &SimpleResp{Code: int(CODE_AGENT_NOT)}))
			c.Abort()
			return
		}
		//代理被禁用
		if agent.IsFrozen == 1 {
			zap.L().Debug("代理已被禁用", zap.Any("agentId", agentId))
			c.JSON(http.StatusOK, GetJsonObj("", &SimpleResp{Code: int(CODE_AGENT_PROHIBIT)}))
			c.Abort()
			return
		}
		//校验md5key
		localMd5 := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%d%d%s", agentId, timestamp, agent.Md5Key))))
		if localMd5 != key {
			zap.L().Debug("md5校验失败", zap.String("localMd5", localMd5), zap.String("key", key))
			c.JSON(http.StatusOK, GetJsonObj("", &SimpleResp{Code: int(CODE_AGENT_CHECK_ERR)}))
			c.Abort()
			return
		}
		//校验aeskey
		aesIv := agent.AesKey[0:16]
		aesKey := agent.AesKey[16:]
		plainTextBytes := AesDecrypt(aesKey, aesIv, param)
		zap.L().Debug("请求", zap.String("str", plainTextBytes))
		Dispatch(c, agent, plainTextBytes)
		zap.L().Debug("完成", zap.String("str", plainTextBytes))
	})

	return engine
}
