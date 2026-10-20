package consumer

import (
	"app/config"
	"encoding/json"
	"game-data-summary/dao"
	"game-data-summary/entity"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

var (
	BeginTime   uint32
	EndTime     uint32
	Games       []*entity.Game
	Day         int
	lastWeekDay int
	Hour        = -1
)

// 用户信息
type UserAnalyseData struct {
	UserId    int             `json:"userId"`    // 用户id
	Eff       decimal.Decimal `json:"eff"`       // 有效下注
	Pro       decimal.Decimal `json:"pro"`       // 盈亏
	BetsCount uint32          `json:"betsCount"` // 注单数
}

// 天，小时
var cacheUserAnalyseData = make([]map[int]map[int]map[int]*UserAnalyseData, 0, 0)

func AnalyseSettlementData(data *[]*config.DataItem) map[int]map[int]map[int]*UserAnalyseData {
	// 需要汇总数据: 活跃用户数，有效下注总数，盈亏总数，次留, 未次留, 注单数总数，盈亏排行
	// 代理归类
	// 游戏归类
	// 代理且游戏归类
	// 用户排行
	startTm := time.Now().UnixNano()

	userRank := make(map[int]map[int]map[int]*UserAnalyseData)

	// 预处理所有代理所有游戏
	userRank[-1] = make(map[int]map[int]*UserAnalyseData)
	userRank[-1][-1] = make(map[int]*UserAnalyseData)
	userRank[-1][-1][-1] = &UserAnalyseData{Eff: decimal.Zero, Pro: decimal.Zero}

	for _, item := range *data {
		if _, ok := userRank[item.AgentId]; !ok {
			userRank[item.AgentId] = make(map[int]map[int]*UserAnalyseData)
			if _, ok := userRank[item.AgentId][-1]; !ok {
				userRank[item.AgentId][-1] = make(map[int]*UserAnalyseData)
			}
		}
		if _, ok := userRank[item.AgentId][item.GameId]; !ok {
			userRank[item.AgentId][item.GameId] = make(map[int]*UserAnalyseData)
			if _, ok := userRank[-1][item.GameId]; !ok {
				userRank[-1][item.GameId] = make(map[int]*UserAnalyseData)
			}
		}
		if _, ok := userRank[item.AgentId][item.GameId][item.UserId]; !ok {
			userRank[item.AgentId][item.GameId][item.UserId] = &UserAnalyseData{Eff: decimal.Zero, Pro: decimal.Zero, UserId: item.UserId}
			if _, ok := userRank[item.AgentId][-1][item.UserId]; !ok {
				userRank[item.AgentId][-1][item.UserId] = &UserAnalyseData{Eff: decimal.Zero, Pro: decimal.Zero, UserId: item.UserId}
			}
			if _, ok := userRank[-1][item.GameId][item.UserId]; !ok {
				userRank[-1][item.GameId][item.UserId] = &UserAnalyseData{Eff: decimal.Zero, Pro: decimal.Zero, UserId: item.UserId}
			}
			if _, ok := userRank[-1][-1][item.UserId]; !ok {
				userRank[-1][-1][item.UserId] = &UserAnalyseData{Eff: decimal.Zero, Pro: decimal.Zero, UserId: item.UserId}
			}
		}
		userRank[-1][-1][item.UserId].Eff = userRank[-1][-1][item.UserId].Eff.Add(item.Eff)
		userRank[-1][-1][item.UserId].Pro = userRank[-1][-1][item.UserId].Pro.Add(item.Pro)
		userRank[-1][-1][item.UserId].BetsCount += 1

		userRank[-1][item.GameId][item.UserId].Eff = userRank[-1][item.GameId][item.UserId].Eff.Add(item.Eff)
		userRank[-1][item.GameId][item.UserId].Pro = userRank[-1][item.GameId][item.UserId].Pro.Add(item.Pro)
		userRank[-1][item.GameId][item.UserId].BetsCount += 1

		userRank[item.AgentId][-1][item.UserId].Eff = userRank[item.AgentId][-1][item.UserId].Eff.Add(item.Eff)
		userRank[item.AgentId][-1][item.UserId].Pro = userRank[item.AgentId][-1][item.UserId].Pro.Add(item.Pro)
		userRank[item.AgentId][-1][item.UserId].BetsCount += 1

		userRank[item.AgentId][item.GameId][item.UserId].Eff = userRank[item.AgentId][item.GameId][item.UserId].Eff.Add(item.Eff)
		userRank[item.AgentId][item.GameId][item.UserId].Pro = userRank[item.AgentId][item.GameId][item.UserId].Pro.Add(item.Pro)
		userRank[item.AgentId][item.GameId][item.UserId].BetsCount += 1
	}

	cacheDays := len(cacheUserAnalyseData)
	if cacheDays == 0 {
		cacheUserAnalyseData = append(cacheUserAnalyseData, userRank)
	} else {
		cacheDays -= 1
		for agentId, agentData := range userRank {
			if _, ok := cacheUserAnalyseData[cacheDays][agentId]; !ok {
				cacheUserAnalyseData[cacheDays][agentId] = agentData
			} else {
				for gameId, gameData := range agentData {
					if _, ok := cacheUserAnalyseData[cacheDays][agentId][gameId]; !ok {
						cacheUserAnalyseData[cacheDays][agentId][gameId] = gameData
					} else {
						for userId, userData := range gameData {
							if _, ok := cacheUserAnalyseData[cacheDays][agentId][gameId][userId]; !ok {
								cacheUserAnalyseData[cacheDays][agentId][gameId][userId] = userData
							} else {
								cacheUserAnalyseData[cacheDays][agentId][gameId][userId].Eff =
									cacheUserAnalyseData[cacheDays][agentId][gameId][userId].Eff.Add(userData.Eff)
								cacheUserAnalyseData[cacheDays][agentId][gameId][userId].Pro =
									cacheUserAnalyseData[cacheDays][agentId][gameId][userId].Pro.Add(userData.Pro)
							}
						}
					}
				}
			}
		}
	}

	// 缓存统计信息
	MarshalFile(cacheUserAnalyseData)

	endTm := time.Now().UnixNano()
	zap.L().Debug("AnalyseSettlementData", zap.Any("耗时", endTm-startTm))

	return userRank
}

func SaveAnalyseData1(db *dao.DbInfo, agentId, gameId, webId, trueGameId int) {
	startTm := time.Now().UnixNano()

	cacheDays := len(cacheUserAnalyseData)
	allEff := decimal.Zero
	allPro := decimal.Zero
	ss := decimal.Zero
	var betsCount uint32 = 0
	if agentData, ok := cacheUserAnalyseData[cacheDays-1][agentId]; ok {
		if gameData, ok := agentData[gameId]; ok {
			for _, data := range gameData {
				allEff = data.Eff
				allPro = data.Pro
				betsCount = data.BetsCount
			}
		}
	}
	if !allEff.Equal(decimal.Zero) {
		//总杀数
		ss = allPro.Div(allEff)
	}
	var keepAlive, notKeepAlive, activeUsers uint32 = 0, 0, 0
	if cacheDays >= 2 {
		if _, ok := cacheUserAnalyseData[cacheDays-2][agentId]; ok {
			if _, ok := cacheUserAnalyseData[cacheDays-2][agentId][gameId]; ok {
				//次留点数
				for userId, _ := range cacheUserAnalyseData[cacheDays-2][agentId][gameId] {
					if _, ok := cacheUserAnalyseData[cacheDays-1][agentId][gameId][userId]; ok {
						keepAlive += 1
					} else {
						notKeepAlive += 1
					}
				}
				//活跃用户数
				activeUsers = uint32(len(cacheUserAnalyseData[cacheDays-1][agentId][gameId]))
			}
		}
	}
	db.AddGameSummaryData(agentId, BeginTime, uint32(webId), trueGameId, allEff, allPro, ss, keepAlive, notKeepAlive, activeUsers, betsCount)
	zap.L().Debug("游戏总有效投注统计", zap.Any("代理id", agentId),
		zap.Any("游戏id", gameId),
		zap.Any("入库游戏id", trueGameId),
		zap.Any("WebId", webId),
		zap.Any("当日该游戏总有效投注", allEff),
		zap.Any("当日该游戏总盈亏", allPro),
		zap.Any("当日该游戏总杀数", ss),
		zap.Any("当日该游戏次留点数", keepAlive),
		zap.Any("当日该游戏未次留点数", notKeepAlive),
		zap.Any("当日该游戏活跃用户数", activeUsers),
		zap.Any("当日该游戏注单数", betsCount))

	endTm := time.Now().UnixNano()
	zap.L().Debug("SaveAnalyseData1", zap.Any("耗时", endTm-startTm))
}

func SaveAnalyseData2(db *dao.DbInfo, agentId int, recordType uint32) {
	startTm := time.Now().UnixNano()

	cacheDays := len(cacheUserAnalyseData)
	allUsers := make([]*UserAnalyseData, 0, 0)
	if recordType == 1 { // 日
		if agentData, ok := cacheUserAnalyseData[cacheDays-1][agentId]; ok {
			for _, v := range agentData[-1] {
				allUsers = append(allUsers, v)
			}
		}
	} else { // 周
		sumUserData := make(map[int]UserAnalyseData)
		for i := 0; i < 7 && i < cacheDays; i += 1 {
			if agentData, ok := cacheUserAnalyseData[i][agentId]; ok {
				for userId, v := range agentData[-1] {
					if v2, ok := sumUserData[userId]; !ok {
						sumUserData[userId] = *v
					} else {
						v2.Eff = v2.Eff.Add(v.Eff)
						v2.Pro = v2.Pro.Add(v.Pro)
						v2.BetsCount += v.BetsCount
					}
				}
			}
		}

		for _, v := range sumUserData {
			allUsers = append(allUsers, &v)
		}
	}

	sort.Slice(allUsers, func(i, j int) bool {
		return allUsers[i].Pro.GreaterThan(allUsers[j].Pro)
	})

	userCount := len(allUsers)
	for i := 0; i < 20 && i < userCount; i += 1 {
		if allUsers[i].Pro.GreaterThan(decimal.Zero) {
			db.AddPlayerProRank(-1, uint32(allUsers[i].UserId), BeginTime, recordType, uint32(i+1), 1, allUsers[i].Eff, allUsers[i].Pro)
		}
		if allUsers[userCount-i-1].Pro.LessThan(decimal.Zero) {
			db.AddPlayerProRank(-1, uint32(allUsers[userCount-i-1].UserId), BeginTime, recordType, uint32(i+1), 0, allUsers[userCount-i-1].Eff, allUsers[userCount-i-1].Pro)
		}
	}

	endTm := time.Now().UnixNano()
	zap.L().Debug("SaveAnalyseData2", zap.Any("耗时", endTm-startTm))
}

func TaskByHour(db *dao.DbInfo, es *dao.EsInfo) {
	nt := time.Now()
	startTm := nt.UnixNano()
	//还未到新的一天
	if Hour == nt.Hour() {
		return
	}
	Hour = nt.Hour()
	nt = nt.Add(-1 * time.Second * 60 * 60)
	BeginTime = uint32(nt.Unix() - int64(nt.Minute()*60+nt.Second()))
	EndTime = BeginTime + 60*60
	zap.L().Debug("开始小时计划任务", zap.Any("beginTime", BeginTime), zap.Any("endTime", EndTime))
	lastHourSettlementData := es.GetAllSettlementData(int64(BeginTime), int64(EndTime))
	lastHourRankData := AnalyseSettlementData(&lastHourSettlementData)
	db.AddPlayerActiveCountByHour(BeginTime, uint32(len(lastHourRankData[-1][-1])))
	//一个小时内 所有活跃的玩家
	for _, game := range Games {
		var allPlayers, cnt uint32 = 0, 0
		var eff, pro = decimal.Zero, decimal.Zero
		if _, ok := lastHourRankData[-1][int(game.Number)]; ok {
			allPlayers = uint32(len(lastHourRankData[-1][int(game.Number)]))
			for _, data := range lastHourRankData[-1][int(game.Number)] {
				eff = eff.Add(data.Eff)
				pro = pro.Add(data.Pro)
				cnt += data.BetsCount
			}
		}
		db.AddGameDataByHour(BeginTime, cnt, uint32(game.Number), allPlayers, eff, pro)
		zap.L().Debug("小时数据统计", zap.Any("gameId", game.Number),
			zap.Any("active", allPlayers),
			zap.Any("eff", eff),
			zap.Any("pro", pro),
			zap.Any("cnt", cnt))
	}
	zap.L().Debug("结束小时计划任务，统计游戏相关数据完成", zap.Any("耗时", time.Now().UnixNano()-startTm))
}

// 处理计划任务
func TaskByDay(db *dao.DbInfo, es *dao.EsInfo) {
	nt := time.Now()
	startTm := nt.UnixNano()
	//还未到新的一天
	if Day == nt.Day() || nt.Hour() != 0 {
		return
	}
	cacheDays := len(cacheUserAnalyseData) - 1
	Day = nt.Day()
	nt = nt.AddDate(0, 0, -1)
	BeginTime = uint32(nt.Unix() - int64(nt.Hour()*60*60+nt.Minute()*60+nt.Second()))
	EndTime = BeginTime + 24*60*60
	zap.L().Debug("开始计划任务，统计游戏相关数据....")
	agents := db.GetAgentList()
	for _, agent := range agents {
		//新注册且活跃
		if _, ok := cacheUserAnalyseData[cacheDays][int(agent.AgentId)]; ok {
			newActivePlayers := cacheUserAnalyseData[cacheDays][int(agent.AgentId)][-1]
			//所有活跃用户
			activePlayers := len(newActivePlayers)
			//新注册玩家列表
			newPlayerMap := db.GetNewPlayersList(agent.AgentId, agent.WebId, BeginTime, EndTime)
			if newPlayerMap == nil {
				return
			}
			registers := len(newPlayerMap)
			c := 0
			for k, _ := range newPlayerMap {
				if _, ok := newActivePlayers[k]; ok {
					c++
				}
			}
			zap.L().Debug("统计结果", zap.Any("新注册", registers), zap.Any("当日总活跃", activePlayers), zap.Any("新注册活跃", c))
			db.AddPlayerSummaryData(int(agent.AgentId), BeginTime, agent.WebId, uint32(registers), uint32(activePlayers), uint32(c))
			for _, game := range Games {
				SaveAnalyseData1(db, int(agent.AgentId), int(game.Number), int(agent.WebId), int(game.Number))
				SaveAnalyseData1(db, -1, int(game.Number), int(agent.WebId), int(game.Number))
			}
			SaveAnalyseData2(db, int(agent.AgentId), 1)
		} else {
			zap.L().Debug("该日代理无数据", zap.Any("代理ID", agent.AgentId))
		}
	}
	SaveAnalyseData1(db, -1, -1, 0, -1)
	SaveAnalyseData2(db, -1, 1)
	if cacheDays == 7 {
		cacheUserAnalyseData = cacheUserAnalyseData[1:]
	}
	cacheUserAnalyseData = append(cacheUserAnalyseData, make(map[int]map[int]map[int]*UserAnalyseData))
	zap.L().Debug("结束日计划任务，统计游戏相关数据完成", zap.Any("耗时", time.Now().UnixNano()-startTm))
}

func TaskByWeek(db *dao.DbInfo, es *dao.EsInfo) {
	nt := time.Now()
	startTm := nt.UnixNano()
	//还没到新的一周 (每周从周一开始算)
	if lastWeekDay == nt.Day() || nt.Weekday() != 1 {
		return
	}
	lastWeekDay = nt.Day()
	subDay := int(nt.Weekday())
	if subDay == 0 { // 星期天转换下在处理
		subDay = 7
	}
	subDay = subDay - 1
	nt = nt.AddDate(0, 0, -1*subDay)
	EndTime = uint32(nt.Unix() - int64(nt.Hour()*60*60+nt.Minute()*60+nt.Second()))
	BeginTime = EndTime - 24*60*60*7
	zap.L().Debug("开始周计划任务")
	agents := db.GetAgentList()
	for _, agent := range agents {
		for _, game := range Games {
			var agentWeekActive, allWeekActive uint32 = 0, 0
			for i := 0; i < len(cacheUserAnalyseData) && i < 7; i += 1 {
				if _, ok := cacheUserAnalyseData[i][int(agent.AgentId)]; ok {
					if _, ok := cacheUserAnalyseData[i][int(agent.AgentId)][int(game.Number)]; ok {
						agentWeekActive = uint32(len(cacheUserAnalyseData[i][int(agent.AgentId)][int(game.Number)]))
						allWeekActive = uint32(len(cacheUserAnalyseData[i][-1][int(game.Number)]))
					}
				}
			}
			db.AddGameWeekActive(int(agent.AgentId), uint32(game.Number), agentWeekActive, BeginTime, EndTime)
			db.AddGameWeekActive(-1, uint32(game.Number), allWeekActive, BeginTime, EndTime)
			zap.L().Debug("游戏周活跃信息统计", zap.Any("代理id", agent.AgentId),
				zap.Any("时间", BeginTime),
				zap.Any("游戏id", game.Number),
				zap.Any("活跃人数", agentWeekActive),
				zap.Any("所有代理活跃人数", allWeekActive))
		}
		// 本周盈亏排行
		SaveAnalyseData2(db, int(agent.AgentId), 2)
		SaveAnalyseData2(db, -1, 2)
	}
	zap.L().Debug("结束周计划任务，统计游戏相关数据完成", zap.Any("耗时", time.Now().UnixNano()-startTm))
}

func MarshalFile(data interface{}) {
	path, err := os.Executable()
	if err != nil {
		zap.L().Error("加载文件失败", zap.Any("err", err))
		return
	}
	dir := filepath.Dir(path)
	byteData, err := json.Marshal(data)
	if err != nil {
		zap.L().Debug("marshalFile序列化失败", zap.Any("error", err))
		return
	}

	f, err := os.OpenFile(dir+"/summary_cache.dat", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		zap.L().Debug("marshalFile OpenFile failed", zap.Any("error", err))
		return
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			zap.L().Debug("marshalFile failed", zap.Any("error", err))
		}
	}(f)
	f.Seek(0, io.SeekStart)
	_, err = f.WriteAt(byteData, 0)
	if err != nil {
		zap.L().Debug("marshalFile WriteFile failed", zap.Any("error", err))
		return
	} else {
		zap.L().Debug("marshalFile WriteFile 成功")
	}
}

func UnmarshalFile() {
	path, err := os.Executable()
	if err != nil {
		zap.L().Error("加载文件失败", zap.Any("err", err))
		return
	}
	dir := filepath.Dir(path)
	f, err := os.Open(dir + "/summary_cache.dat")
	if err != nil {
		zap.L().Debug("unmarshalFile 打开缓存文件失败", zap.Any("error", err))
		return
	}

	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			zap.L().Debug("unmarshalFile failed", zap.Any("error", err))
		}
	}(f)

	data, err := io.ReadAll(f)
	if err != nil {
		zap.L().Debug("unmarshalFile read fd failed", zap.Any("error", err))
		return
	}
	err = json.Unmarshal(data, &cacheUserAnalyseData)
	if err != nil {
		zap.L().Debug("unmarshalFile 失败", zap.Any("error", err))
	} else {
		zap.L().Debug("unmarshalFile 成功")
	}
}
