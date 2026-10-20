package v1

import (
	"app/config"
	"app/tables/manager"
	"app/tables/player"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	. "open-api/common"
	"open-api/dao"
	. "open-api/dao"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

func Update3rdParams(userId int64, currencyType string) {
	key := fmt.Sprintf("player_%d", userId)
	txManager := Mysql().Manager.Begin()
	txPlayer := Mysql().Player.Begin()
	txManager.Model(&manager.User{}).Debug().Where("id=?", userId).Updates(map[string]interface{}{"currencyType": currencyType})
	txPlayer.Model(&player.Player{}).Debug().Where("user_id = ?", userId).Updates(map[string]interface{}{"currency_type": currencyType})
	if Redis().Exists(key) {
		if Redis().TTL(key) <= 5 {
			Redis().Del(key)
		} else {
			if !Redis().HSet(key, "currency_type", currencyType) {
				txManager.Rollback()
				txPlayer.Rollback()
			}
		}
	}
	txManager.Commit()
	txPlayer.Commit()
}

func GatewayList(totalEffectBet float64) []string {
	cm, urls := config.CfgIns.GetGatewayCfg(), make([]string, 0, 64)
	if cm.UpdateTime < time.Now().Unix() {
		tmp := make([]*manager.ApiConfig, 0, 32)
		dao.Mysql().Manager.Model(&manager.ApiConfig{}).Find(&tmp)
		cm.Urls = tmp
		cm.UpdateTime = time.Now().Unix() + 10
		zap.L().Debug("获取最新的gateway配置", zap.Any("cm", cm))
		config.CfgIns.SetGatewayCfg(&cm)
	}
	for _, v := range cm.Urls {
		if float64(v.Min) <= totalEffectBet && float64(v.Max) > totalEffectBet {
			urls = append(urls, strings.Split(v.HallUrls, ",")...)
		}
	}
	return urls
}

/** 把 open-api lang 映射成 clientApi /game 使用的 language。 */
func mapClientApiLanguage(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "", "zh", "zh-cn", "zh_cn", "cn":
		return "zh-cn"
	case "zh-tw", "zh_tw", "tw":
		return "zh-tw"
	case "en", "en-us", "en_us":
		return "en"
	default:
		return strings.TrimSpace(lang)
	}
}

/**
 * 生成指向 fgServer clientApi 的进游 URL。
 * 形如: {base}/game?type=h5&gamecode=api&language=zh-cn&userId=1234&token=SESSION@...
 */
func buildClientApiGameURL(baseURL, gamecode, lang string, userId int64, token string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	code := strings.TrimSpace(gamecode)
	values := url.Values{}
	values.Set("type", "h5")
	values.Set("gamecode", code)
	values.Set("language", mapClientApiLanguage(lang))
	values.Set("userId", strconv.FormatInt(userId, 10))
	values.Set("token", strings.TrimSpace(token))
	return fmt.Sprintf("%s/game?%s", base, values.Encode())
}

// for test
func Login(ctx *gin.Context, params url.Values, agent *manager.Agent) {
	account := params.Get("account")
	nickName := params.Get("nickName")
	ip := params.Get("ip")
	money := params.Get("money")
	symbol := params.Get("symbol") //这里把gameId等价于symbol 减少修改量
	currencyType := params.Get("currencyType")
	lang := params.Get("lang")
	isTourist, _ := strconv.Atoi(params.Get("isTourist"))
	if lang == "" {
		lang = "zh"
	}
	game := GameCacheIns.GetGame(symbol)
	if game == nil {
		ctx.JSON(http.StatusOK, GetJsonObj(API_LOGIN.String(), &SimpleResp{Code: int(CODE_GAME_NOT)}))
		zap.L().Error("游戏不存在", zap.Any("gameId", symbol))
		return
	}
	if game.State != 1 {
		ctx.JSON(http.StatusOK, GetJsonObj(API_LOGIN.String(), &SimpleResp{Code: int(CODE_GAME_PROHIBIT)}))
		zap.L().Error("游戏维护中", zap.Any("gameId", symbol), zap.Any("state", game.State))
		return
	}
	var userId int64 = 0
	var pp *player.Player = nil
	//查询玩家是否已经存在
	player := Mysql().GetAgentPlayerInfoByAgentIdAndAcc(agent.Id, account)
	if player == nil {
		score, _ := strconv.ParseFloat(money, 64)
		//新建玩家信息
		player, pp = Mysql().AddNewPlayer(agent.Id, agent.WebId, score, account, nickName, ip, currencyType, int32(isTourist))
		if player.Id <= 0 {
			ctx.JSON(http.StatusOK, GetJsonObj(API_LOGIN.String(), &SimpleResp{Code: int(CODE_REQUEST_ERR)}))
			return
		}
		userId = int64(player.Id)
		p := ConvertUserEntityToHumanPlayer(pp)
		if pe := Redis().SetPlayer(p); pe != nil {
			zap.L().Warn("向redis写入玩家信息缓存失败", zap.Any("req", params), zap.Error(pe))
			ctx.JSON(http.StatusOK, GetJsonObj(API_ADD_SCORE.String(), &SimpleResp{Code: int(CODE_UP_ACCOUNT_SCORE_ERR)}))
			return
		}
		_, err := Redis().UpdatePlayerCurrency(uint32(pp.UserId), int64(100000*100))
		if err != nil {
			zap.L().Error("更新玩家游戏币和经验失败", zap.Any("req", params), zap.Error(err))
			ctx.JSON(http.StatusOK, GetJsonObj(API_ADD_SCORE.String(), &SimpleResp{Code: int(CODE_UP_ACCOUNT_SCORE_ERR)}))
			return
		}
	} else {
		userId = int64(player.Id)
		Update3rdParams(player.Id, currencyType)
	}
	aesKey, aesIv := agent.AesKey[16:], agent.AesKey[0:16]
	source := fmt.Sprintf("agentId=%d&userId=%d&times=%d", agent.Id, userId, time.Now().UnixMilli())
	res, _ := AesEncrypt(aesKey, aesIv, []byte(source))
	sessionKey := fmt.Sprintf("SESSION@%s", res)
	//创建用户session
	session := &Session{}
	sessionStr, _ := Redis().Get(sessionKey)
	if sessionStr != "" {
		if jsoniter.UnmarshalFromString(sessionStr, session) == nil {
			session.GameId = int64(game.Number)
			session.CurrencyType = currencyType
			session.Lang = lang
		}
	} else {
		session = &Session{
			AgentId:      agent.Id,
			UserId:       player.Id,
			GameId:       int64(game.Number),
			NickName:     player.NickName,
			AuthToken:    res,
			Mgckey:       sessionKey,
			Lang:         lang,
			Account:      player.UserId,
			LastAuthTime: time.Now().Unix(),
			AuthCount:    0,
			CurrencyType: currencyType,
			Symbol:       game.ConfName,
			IsTourist:    int32(isTourist),
		}
	}
	session.LastAuthTime = time.Now().Unix()
	str, err := jsoniter.MarshalToString(session)
	if err != nil {
		ctx.JSON(http.StatusOK, GetJsonObj(API_LOGIN.String(), &SimpleResp{Code: int(CODE_REQUEST_ERR)}))
		return
	}
	if err := Redis().Set(sessionKey, str, 20*60); err != nil {
		zap.L().Error("创建用户session失败", zap.Any("err", err))
		ctx.JSON(http.StatusOK, GetJsonObj(API_LOGIN.String(), &SimpleResp{Code: int(CODE_REQUEST_ERR)}))
		return
	}
	arr := config.CfgIns.System.GameUrls
	if len(arr) == 0 {
		ctx.JSON(http.StatusOK, GetJsonObj(API_LOGIN.String(), &SimpleResp{Code: int(CODE_REQUEST_ERR)}))
		return
	}
	// GameUrls 配置为 clientApi 根地址，例如 http://172.21.211.216:9700
	requestUrl := buildClientApiGameURL(
		arr[rand.Intn(len(arr))],
		game.ConfName,
		lang,
		player.Id,
		session.Mgckey,
	)
	ctx.Redirect(http.StatusFound, requestUrl)
}

// production
// func Login(ctx *gin.Context, params url.Values, agent *manager.Agent) {
// 	account := params.Get("account")
// 	// nickName := params.Get("nickName")
// 	// ip := params.Get("ip")
// 	// money := params.Get("money")
// 	symbol := params.Get("symbol") //这里把gameId等价于symbol 减少修改量
// 	currencyType := params.Get("currencyType")
// 	lang := params.Get("lang")
// 	isTourist, _ := strconv.Atoi(params.Get("isTourist"))
// 	if lang == "" {
// 		lang = "zh"
// 	}
// 	game := GameCacheIns.GetGame(symbol)
// 	if game == nil {
// 		ctx.JSON(http.StatusOK, GetJsonObj(API_LOGIN.String(), &SimpleResp{Code: int(CODE_GAME_NOT)}))
// 		zap.L().Error("游戏不存在", zap.Any("gameId", symbol))
// 		return
// 	}
// 	if game.State != 1 {
// 		ctx.JSON(http.StatusOK, GetJsonObj(API_LOGIN.String(), &SimpleResp{Code: int(CODE_GAME_PROHIBIT)}))
// 		zap.L().Error("游戏维护中", zap.Any("gameId", symbol), zap.Any("state", game.State))
// 		return
// 	}
// 	var userId int64 = 0
// 	//查询玩家是否已经存在
// 	player := Mysql().GetAgentPlayerInfoByAgentIdAndAcc(agent.Id, account)
// 	if player == nil {
// 		ctx.JSON(http.StatusOK, GetJsonObj(API_LOGIN.String(), &SimpleResp{Code: int(CODE_ACCOUNT_NOT)}))
// 		zap.L().Error("参数错误", zap.Any("gameId", symbol), zap.Any("state", game.State))
// 		return
// 	} else {
// 		userId = int64(player.Id)
// 		Update3rdParams(player.Id, currencyType)
// 	}
// 	aesKey, aesIv := agent.AesKey[16:], agent.AesKey[0:16]
// 	source := fmt.Sprintf("agentId=%d&userId=%d&times=%d", agent.Id, userId, time.Now().UnixMilli())
// 	res, _ := AesEncrypt(aesKey, aesIv, []byte(source))
// 	sessionKey := fmt.Sprintf("SESSION@%s", res)
// 	//创建用户session
// 	session := &Session{}
// 	sessionStr, _ := Redis().Get(sessionKey)
// 	if sessionStr != "" {
// 		if jsoniter.UnmarshalFromString(sessionStr, session) == nil {
// 			session.GameId = int64(game.Number)
// 			session.CurrencyType = currencyType
// 			session.Lang = lang
// 		}
// 	} else {
// 		session = &Session{
// 			AgentId:      agent.Id,
// 			UserId:       player.Id,
// 			GameId:       int64(game.Number),
// 			NickName:     player.NickName,
// 			AuthToken:    res,
// 			Mgckey:       sessionKey,
// 			Lang:         lang,
// 			Account:      player.UserId,
// 			LastAuthTime: time.Now().Unix(),
// 			AuthCount:    0,
// 			CurrencyType: currencyType,
// 			Symbol:       game.ConfName,
// 			IsTourist:    int32(isTourist),
// 		}
// 	}
// 	session.LastAuthTime = time.Now().Unix()
// 	str, err := jsoniter.MarshalToString(session)
// 	if err != nil {
// 		ctx.JSON(http.StatusOK, GetJsonObj(API_LOGIN.String(), &SimpleResp{Code: int(CODE_REQUEST_ERR)}))
// 		return
// 	}
// 	if err := Redis().Set(sessionKey, str, 20*60); err != nil {
// 		zap.L().Error("创建用户session失败", zap.Any("err", err))
// 		ctx.JSON(http.StatusOK, GetJsonObj(API_LOGIN.String(), &SimpleResp{Code: int(CODE_REQUEST_ERR)}))
// 		return
// 	}
// 	arr := config.CfgIns.System.GameUrls
// 	if len(arr) == 0 {
// 		ctx.JSON(http.StatusOK, GetJsonObj(API_LOGIN.String(), &SimpleResp{Code: int(CODE_REQUEST_ERR)}))
// 		return
// 	}
// 	// GameUrls 配置为 clientApi 根地址，例如 http://172.21.211.216:9700
// 	requestUrl := buildClientApiGameURL(
// 		arr[rand.Intn(len(arr))],
// 		game.ConfName,
// 		lang,
// 		player.Id,
// 		session.Mgckey,
// 	)
// 	ctx.PureJSON(http.StatusOK, GetJsonObj(API_LOGIN.String(), &LoginResp{
// 		Code: int(CODE_OK),
// 		Url:  requestUrl,
// 	}))
// }

func UpdateCurrencyType(ctx *gin.Context, params url.Values, agent *manager.Agent) {
	account := params.Get("account")
	currencyType := params.Get("currencyType")
	var userId int64 = 0
	//查询玩家是否已经存在
	player := Mysql().GetAgentPlayerInfoByAgentIdAndAcc(agent.Id, account)
	if player == nil {
		ctx.JSON(http.StatusOK, GetJsonObj(API_LOGIN.String(), &SimpleResp{Code: int(CODE_REQUEST_ERR)}))
		return
	} else {
		userId = int64(player.Id)
	}
	Update3rdParams(userId, currencyType)
	ctx.PureJSON(http.StatusOK, GetJsonObj(API_LOGIN.String(), map[string]interface{}{
		"code": CODE_OK,
	}))
}
