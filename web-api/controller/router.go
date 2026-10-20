package controller

import (
	. "app/entity"
	"fmt"
	"net/http"
	. "web-api/common"
	v1 "web-api/controller/v1"
	v2 "web-api/controller/v2"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

var signKey = "6553795CC3FA62E6E62875C48DB35463"

// 路由注册，及绑定访问权限
func Register(g *gin.Engine, method, url string, cb gin.HandlerFunc, accTypes []ACCOUNT_TYPE) {
	UrlPloys().SetPloy(url, accTypes)
	if accTypes != nil {
		switch method {
		case "GET", "get":
			g.GET(url, CheckPermissions(), cb)
		case "POST", "post":
			g.POST(url, CheckPermissions(), cb)
		case "PUT", "put":
			g.PUT(url, CheckPermissions(), cb)
		}
	} else {
		switch method {
		case "GET", "get":
			g.GET(url, cb)
		case "POST", "post":
			g.POST(url, cb)
		case "PUT", "put":
			g.PUT(url, cb)
		}
	}
}

// 身份验证
func CheckPermissions() gin.HandlerFunc {
	return func(c *gin.Context) {
		// zap.L().Debug("======>request", zap.Any("uri", c.Request.URL.Path))
		token := c.Query("token")
		if token == "" {
			token = c.PostForm("token")
		}
		if token == "" {
			c.String(http.StatusForbidden, "没有权限！")
			c.Abort()
			return
		}
		//
		if c.Request.URL.Path == "/api/auth/v2/game/gameUrl" {
			return
		}
		if c.Request.URL.Path == "/api/auth/v2/queryOrder" {
			roundId := c.Query("roundId")
			account := c.Query("account")
			if token != HmacSha256ToHex(signKey, []byte(fmt.Sprintf("%s%s", roundId, account))) {
				c.String(http.StatusForbidden, "没有权限！")
				c.Abort()
				return
			}
		} else {
			str := AesDecrypt(AesKey, AesIV, token)
			loginInfo := &LoginInfo{}
			jsoniter.UnmarshalFromString(str, loginInfo)
			if loginInfo.Id <= 0 || !UrlPloys().CheckPloy(c.Request.URL.Path, int(loginInfo.UserType)) {
				c.String(http.StatusForbidden, "没有权限！")
				c.Abort()
				return
			}
			c.Set("agentId", int64(loginInfo.AgentId))
			c.Set("id", int64(loginInfo.Id))
		}
	}
}

// 初始化gin
func NewRouter() *gin.Engine {
	engine := gin.New()
	engine.Use(ginzap.RecoveryWithZap(zap.L(), true))
	g := engine
	//设置跨域
	g.Use(Cors())
	//登陆相关 无需权限验证
	Register(g, "post", "/api/auth/v1/login", v1.Login, nil)
	Register(g, "post", "/api/agent/v1/login", v1.AgentLogin, nil)
	//代理相关
	Register(g, "get", "/api/agent/v1/msg/my/list", v1.MsgList, []ACCOUNT_TYPE{S})
	Register(g, "get", "/api/agent/v2/report-form/list", v2.AgentReportFormList, []ACCOUNT_TYPE{S})
	Register(g, "get", "/api/agent/v2/report-form/lineChart", v2.AgentLineChart, []ACCOUNT_TYPE{S})
	Register(g, "get", "/api/agent/v1/user/list", v2.UserListByAgentId, []ACCOUNT_TYPE{S})
	Register(g, "get", "/api/agent/v1/user/info", v1.UserInfo, []ACCOUNT_TYPE{S})
	Register(g, "get", "/api/agent/v2/fw/list", v2.BillList, []ACCOUNT_TYPE{S})
	Register(g, "get", "/api/agent/v2/settlement/list", v2.SettlementListWithAgentId, []ACCOUNT_TYPE{S})
	Register(g, "get", "/api/agent/v2/govern/list4Agent", v1.Logout, []ACCOUNT_TYPE{S})
	Register(g, "get", "/api/agent/v1/msg/list", v1.Logout, []ACCOUNT_TYPE{S})
	Register(g, "get", "/api/agent/v2/games", v1.Logout, []ACCOUNT_TYPE{S})
	Register(g, "get", "/api/agent/v2/stat/agent/info", v2.AgentStatInfo, []ACCOUNT_TYPE{S})
	Register(g, "get", "/api/agent/v2/report-form/userChart", v2.UserChart, []ACCOUNT_TYPE{S})
	Register(g, "get", "/api/agent/v2/export/settlements/count", v2.ExportSettlmentCountWithAgentId, []ACCOUNT_TYPE{S})
	Register(g, "get", "/api/agent/v2/export/settlements", v2.ExportSettlementsWithAgentId, []ACCOUNT_TYPE{S})

	//需要权限验证
	Register(g, "get", "/api/auth/v1/logout", v1.Logout, []ACCOUNT_TYPE{M, A})
	Register(g, "get", "/api/auth/v1/web/list", v1.WebList, []ACCOUNT_TYPE{M, A})
	Register(g, "get", "/api/auth/v1/game/gamer-game-list", v1.GamerGameList, []ACCOUNT_TYPE{M, A})
	Register(g, "get", "/api/auth/v1/agent/list", v1.GetAgentList, []ACCOUNT_TYPE{M, A})
	Register(g, "get", "/api/auth/v1/agent/select-list", v1.GetAgentSelectList, []ACCOUNT_TYPE{M, A})
	Register(g, "get", "/api/auth/v1/agent/getInfo", v1.GetAgentInfo, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v1/agent/stop", v1.ServerStop, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v1/agent/start", v1.ServerStart, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v1/web/linkage", v1.LinkAgent, []ACCOUNT_TYPE{M, A})
	Register(g, "get", "/api/auth/v1/user/info", v1.UserInfo, []ACCOUNT_TYPE{M, A})
	Register(g, "get", "/api/auth/v1/user/upState", v1.UpdateUserState, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v1/log/list", nil, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v1/account/list", v1.AdminList, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v1/account/del", v1.AdminDel, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v1/agent/user/score/log", v1.UserScoreLogs, []ACCOUNT_TYPE{M, A})
	Register(g, "get", "/api/auth/v2/stat/agent/info", v2.AgentInfo, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v2/stat/agent/ag-group", v2.AgGroup, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v2/site/serverTime", v2.ServerTime, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v2/govern/userAndGameSummary", v2.UserAndGameSummary, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v2/govern/userAndGameSummaryDetail", v2.UserAndGameSummaryDetail, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v2/govern/userAndGameDataByHour", v2.UserAndGameDataByHour, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v2/report-form/list", v2.ReportFormList, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v2/export/agent/data", v2.ExportAgentData, []ACCOUNT_TYPE{M, A})
	Register(g, "get", "/api/auth/v2/report-form/listWithAgent", v2.ReportFormListWithAgent, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v2/report-form/listAgentGameAggs", v2.GetAgentGameDataAggs, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v2/report-form/history", v2.ReportFormHistory, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v2/user/list", v2.UserList, []ACCOUNT_TYPE{M, A})
	Register(g, "get", "/api/auth/v2/govern/user-record", v2.GovernUserRecord, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v2/fw/list", v2.BillList, []ACCOUNT_TYPE{M, A})
	Register(g, "get", "/api/auth/v2/game/list", v2.GameList, []ACCOUNT_TYPE{M, A})
	Register(g, "get", "/api/auth/v2/settlement/list", v2.SettlementList, []ACCOUNT_TYPE{M, A})
	Register(g, "get", "/api/auth/v2/export/settlements/count", v2.ExportSettlmentCountWithAgentId, []ACCOUNT_TYPE{M, A})
	Register(g, "get", "/api/auth/v2/export/settlements", v2.ExportSettlementsWithAgentId, []ACCOUNT_TYPE{M, A})
	Register(g, "get", "/api/auth/v2/govern/list", v2.GovernList, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v2/govern/poolResetTimeRange", v2.PoolResetTimeRange, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v2/govern/poolResetNow", v2.PoolResetNow, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v2/govern/poolResetInfo", v2.PoolResetInfo, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v2/govern/player", v2.GovernPlayer, []ACCOUNT_TYPE{M})
	Register(g, "put", "/api/auth/v2/govern/player", v2.GovernEditPlayer, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v2/govern/setUserCtl", v2.SetUserCtl, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v2/govern/getUserCtl", v2.GetUserCtl, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v2/govern/clearLock", v2.ClearPlayerLock, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v2/stock/list", v2.StockList, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v2/game/agent", v2.GameAgentList, []ACCOUNT_TYPE{M})
	Register(g, "get", "/api/auth/v2/game/gameUrl", v2.GameUrls, []ACCOUNT_TYPE{M, A})
	Register(g, "get", "/api/auth/v2/clear/gameState", v2.ClearGameState, []ACCOUNT_TYPE{M, A})
	Register(g, "get", "/api/auth/v2/queryOrder", v2.QueryOrder, []ACCOUNT_TYPE{M, A, S})
	Register(g, "get", "/api/auth/v2/exchange", v2.GetExchange, []ACCOUNT_TYPE{M, A, S})

	Register(g, "post", "/api/auth/v2/editExchange", v2.EditExchange, []ACCOUNT_TYPE{M, A, S})
	Register(g, "post", "/api/auth/v1/web/add", v1.WebAdd, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v1/web/edit", v1.WebEdit, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v2/agent/add", v2.AgentAdd, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v1/agent/edit", v1.AgentEdit, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v1/game/upState", v1.UpdateGameState, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v1/game/stopAll", v1.StopAllGame, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v1/game/startAll", v1.StartAllGame, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v1/account/add", v1.AdminAdd, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v1/account/edit", v1.AdminEdit, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v1/account/upState", v1.AdminChangeState, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v2/game/apiConfigList", v2.ApiConfigList, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v2/game/getGameUrl", v2.GetGameUrl, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v2/game/updateGameUrl", v2.UpdateGameUrl, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v2/game/apiConfigAdd", v2.ApiConfigAdd, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v2/game/apiConfigDel", v2.ApiConfigDel, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v2/game/apiConfigUpdate", v2.ApiConfigUpdate, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v2/govern/user", v2.GovernUser, []ACCOUNT_TYPE{M})
	Register(g, "put", "/api/auth/v2/govern/user", v2.GovernUser, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v2/govern/edit", v2.PoolEdit, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v2/game/getGameCurrency", v2.GameCurrencys, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v2/game/changeStatus", v2.ChangeStatus, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v2/user/saveAutoSingleControl", v2.SaveSingleCtrlConfig, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v2/user/getAutoSingleControl", v2.GetSingleCtrlConfig, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v2/game/getGameSettingData", v2.GetGameSettingData, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v2/game/saveGameSettingData", v2.SaveGameSettingData, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v2/game/getControlCoefficientData", v2.GetControlCoefficientData, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v2/game/saveControlCoefficientData", v2.SaveControlCoefficientData, []ACCOUNT_TYPE{M})
	Register(g, "post", "/api/auth/v2/game/syncAllPool", v2.SyncAllPool, []ACCOUNT_TYPE{M})
	return engine
}
