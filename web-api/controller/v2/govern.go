package v2

import (
	"app/config"
	"app/entity"
	"app/entity/view"
	"app/esindex"
	"app/tables/manager"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"web-api/dao"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/olivere/elastic/v7"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// 用户列表
func GovernList(ctx *gin.Context) {
	gameIdStr := ctx.Query("gameId")
	agentId, err := strconv.Atoi(ctx.Query("agentId"))
	if err != nil {
		agentId = -1
	}
	gameIds := make([]int, 0)
	for _, v := range strings.Split(gameIdStr, ",") {
		id, _ := strconv.Atoi(v)
		gameIds = append(gameIds, id)
	}
	games := make([]*manager.Game, 0)
	dao.Mysql().Manager.Where("number in ?", gameIds).Find(&games)
	poolDatas := make([]*view.PoolData, 0)
	if agentId >= 0 {
		for _, game := range games {
			// /fg/agent/{agentId}/pool/{symbol}
			key := esindex.AgentKey(fmt.Sprintf("%d", agentId), "pool", game.ConfName)
			c := config.CfgIns.GetPoolCfg(int64(agentId), game.ConfName)
			pd := &view.PoolData{
				Name:  game.Name,
				Value: c,
				Key:   key,
			}
			poolDatas = append(poolDatas, pd)
		}
	} else {
		for _, game := range games {
			// /fg/config/pool/{symbol}
			key := esindex.ConfigKey("pool", game.ConfName)
			c := config.CfgIns.GetPoolDefaultCfg(game.ConfName)
			pd := &view.PoolData{
				Name:  game.Name,
				Value: c,
				Key:   key,
			}
			poolDatas = append(poolDatas, pd)
		}
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: poolDatas})
}

// 周期重置信息
func PoolResetInfo(ctx *gin.Context) {
	result := dao.RedisIns().Client.Get(context.Background(), "reset_pool_config").Val()
	if result == "" {
		resetConfig := &view.ResetPoolConfig{
			Now:       false,
			Interval:  7,
			ResetTime: time.Now().Unix() + 7*24*60*60,
		}
		result, _ = jsoniter.MarshalToString(resetConfig)
		dao.RedisIns().Client.Set(context.Background(), "reset_pool_config", result, -1)
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: result})
}

// 设置重置周期
func PoolResetTimeRange(ctx *gin.Context) {
	interval, _ := strconv.Atoi(ctx.Query("interval"))
	if interval != 7 && interval != 30 && interval != 1 && interval != 3 && interval != 10 && interval != 14 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "操作失败,周期数据异常", Data: nil})
	} else {
		resetConfig := &view.ResetPoolConfig{
			Now:       false,
			Interval:  interval,
			ResetTime: time.Now().Unix() + int64(interval)*24*60*60,
		}
		str, _ := jsoniter.MarshalToString(resetConfig)
		dao.RedisIns().Client.Set(context.Background(), "reset_pool_config", str, -1)
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: nil})
	}
}

// 设置重置周期
func PoolResetNow(ctx *gin.Context) {
	result := dao.RedisIns().Client.Get(context.Background(), "reset_pool_config").Val()
	if result == "" {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "系统异常", Data: nil})
		return
	}
	resetConfig := &view.ResetPoolConfig{}
	err := jsoniter.UnmarshalFromString(result, resetConfig)
	if err != nil {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "系统异常", Data: nil})
		return
	}
	resetConfig.Now = true
	result, _ = jsoniter.MarshalToString(resetConfig)
	dao.RedisIns().Client.Set(context.Background(), "reset_pool_config", result, -1)
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: nil})
}

// 修改水池配置
func PoolEdit(ctx *gin.Context) {
	rawKey := ctx.Query("key")
	key := esindex.NormalizeKey(rawKey)
	value := ctx.Query("value")
	if value == "" || key == "" {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "系统异常", Data: nil})
		return
	}
	section, rest := esindex.ParsePath(key)
	isDefaultPool := section == "config" && len(rest) >= 2 && rest[0] == "pool"
	isAgentPool := section == "agent" && len(rest) >= 3 && rest[1] == "pool"
	if !isDefaultPool && !isAgentPool {
		zap.L().Error("水池配置key异常", zap.String("key", rawKey), zap.String("normalized", key))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "配置key异常", Data: nil})
		return
	}
	pool := &config.Pool{}
	err := jsoniter.UnmarshalFromString(value, pool)
	if err != nil {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "系统异常", Data: nil})
		return
	}
	if err = dao.UpsertPoolConfig(key, value); err != nil {
		zap.L().Error("保存水池配置到数据库失败", zap.Any("err", err), zap.String("key", key))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "系统异常", Data: nil})
		return
	}
	err = dao.RedisIns().Set(key, value, -1)
	if err != nil {
		zap.L().Error("保存配置失败", zap.Any("err", err))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "系统异常", Data: nil})
		return
	}
	if rawKey != key {
		_ = dao.RedisIns().Del(rawKey)
	}
	dao.ParseConfig(key, value)
	dataStr, _ := jsoniter.MarshalToString(map[string]interface{}{
		"key":  key,
		"data": value,
	})
	str, _ := jsoniter.MarshalToString(map[string]interface{}{
		"event": "config",
		"data":  dataStr,
	})
	if pubErr := dao.RedisIns().Publish("message", str); pubErr != nil {
		zap.L().Error("发布水池配置失败", zap.Any("err", pubErr), zap.String("key", key))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "配置已保存，但同步到开奖服务失败", Data: nil})
		return
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: nil})
}

func GovernPlayer(ctx *gin.Context) {
	// agentId, _ := strconv.Atoi(ctx.Query("agentId"))
	// str := string(dao.GetEtcdValue(fmt.Sprintf("/config/user_ctl/%d", agentId)))
	// if str == "" {
	// 	str = string(dao.GetEtcdValue("/config/user_ctl/default"))
	// }
	// ctl := &view.UserCtl{}
	// if str != "" {
	// 	jsoniter.UnmarshalFromString(str, ctl)
	// }
	// ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: ctl})
}

func GovernEditPlayer(ctx *gin.Context) {
	// agentId, _ := strconv.Atoi(ctx.Query("agentId"))
	// value := ctx.Query("value")
	// dao.PutEtcdValue(fmt.Sprintf("/config/user_ctl/%d", agentId), value)
	// ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: value})
}

// 用户控制参数修改
func GetUserCtl(ctx *gin.Context) {
	str := dao.RedisIns().Client.Get(context.Background(), "player_statistics_interval").Val()
	if str != "" {
		arr := strings.Split(str, "|")
		if len(arr) >= 3 {
			inter, _ := strconv.Atoi(arr[2])
			endTime, _ := strconv.Atoi(arr[1])
			ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: &entity.GovernUserCtl{T: int64(inter), E: int64(endTime)}})
			return
		}
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: nil})
}

// 用户控制参数修改
func SetUserCtl(ctx *gin.Context) {
	day, _ := strconv.ParseInt(ctx.Query("t"), 10, 0)
	t := day * 24 * 60 * 60
	now := time.Now()
	dao.RedisIns().Client.Set(context.Background(), "player_statistics_interval", fmt.Sprintf("%d|%d|%d", now.Unix(), now.Unix()+t, day), -1)
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: nil})
}

func hourDataCombineEff(result []*view.UserGameDataByHourItem) []*view.UserGameDataByHourEff {
	tmp := make(map[int64]float64)
	for _, t := range result {
		if v, ok := tmp[t.RecordTime]; ok {
			tmp[t.RecordTime] = v + t.Value
		} else {
			tmp[t.RecordTime] = t.Value
		}
	}
	ret := make([]*view.UserGameDataByHourEff, 0)
	for k, v := range tmp {
		ret = append(ret, &view.UserGameDataByHourEff{
			RecordTime: k,
			Value:      v,
		})
	}
	return ret
}

func hourDataCombinePro(result []*view.UserGameDataByHourItem) []*view.UserGameDataByHourPro {
	tmp := make(map[int64]float64)
	for _, t := range result {
		if v, ok := tmp[t.RecordTime]; ok {
			tmp[t.RecordTime] = v + t.Value
		} else {
			tmp[t.RecordTime] = t.Value
		}
	}
	ret := make([]*view.UserGameDataByHourPro, 0)
	for k, v := range tmp {
		ret = append(ret, &view.UserGameDataByHourPro{
			RecordTime: k,
			Value:      v,
		})
	}
	return ret
}

func hourDataCombineCnt(result []*view.UserGameDataByHourItem) []*view.UserGameDataByHourCnt {
	tmp := make(map[int64]float64)
	for _, t := range result {
		if v, ok := tmp[t.RecordTime]; ok {
			tmp[t.RecordTime] = v + t.Value
		} else {
			tmp[t.RecordTime] = t.Value
		}
	}
	ret := make([]*view.UserGameDataByHourCnt, 0)
	for k, v := range tmp {
		ret = append(ret, &view.UserGameDataByHourCnt{
			RecordTime: k,
			Value:      v,
		})
	}
	return ret
}

func hourDataCombineActive(result []*view.UserGameDataByHourItem) []*view.UserGameDataByHourActive {
	tmp := make(map[int64]float64)
	for _, t := range result {
		if v, ok := tmp[t.RecordTime]; ok {
			tmp[t.RecordTime] = v + t.Value
		} else {
			tmp[t.RecordTime] = t.Value
		}
	}
	ret := make([]*view.UserGameDataByHourActive, 0)
	for k, v := range tmp {
		ret = append(ret, &view.UserGameDataByHourActive{
			RecordTime: k,
			Value:      v,
		})
	}
	return ret
}

func UserAndGameDataByHour(ctx *gin.Context) {
	startTime, _ := strconv.ParseInt(ctx.Query("startTime"), 10, 0)
	endTime, _ := strconv.ParseInt(ctx.Query("endTime"), 10, 0)
	str := ctx.Query("gameId")
	ids := make([]int, 0, 8)
	if len(str) > 0 {
		for _, v := range strings.Split(str, ",") {
			id, err := strconv.Atoi(v)
			if err == nil {
				ids = append(ids, id)
			}
		}
	}
	effs := make([]*view.UserGameDataByHourItem, 0)
	pros := make([]*view.UserGameDataByHourItem, 0)
	cnts := make([]*view.UserGameDataByHourItem, 0)
	actives := make([]*view.UserGameDataByHourItem, 0)
	if len(ids) > 0 {
		dao.Mysql().Manager.Model(manager.GameDataHour{}).Debug().Select("`record_time`,`eff` as `value`").Where("game_id in ? and record_time BETWEEN ? and ?", ids, startTime, endTime).Find(&effs)
		dao.Mysql().Manager.Model(manager.GameDataHour{}).Debug().Select("`record_time`,`pro` as `value`").Where("game_id in ? and record_time BETWEEN ? and ?", ids, startTime, endTime).Find(&pros)
		dao.Mysql().Manager.Model(manager.GameDataHour{}).Debug().Select("`record_time`,`cnt` as `value`").Where("game_id in ? and record_time BETWEEN ? and ?", ids, startTime, endTime).Find(&cnts)
		dao.Mysql().Manager.Model(manager.GameDataHour{}).Debug().Select("`record_time`,`active` as `value`").Where("game_id in ? and record_time BETWEEN ? and ?", ids, startTime, endTime).Find(&actives)
	} else {
		dao.Mysql().Manager.Model(manager.GameDataHour{}).Debug().Select("`record_time`,`eff` as `value`").Where("record_time BETWEEN ? and ?", startTime, endTime).Find(&effs)
		dao.Mysql().Manager.Model(manager.GameDataHour{}).Debug().Select("`record_time`,`pro` as `value`").Where("record_time BETWEEN ? and ?", startTime, endTime).Find(&pros)
		dao.Mysql().Manager.Model(manager.GameDataHour{}).Debug().Select("`record_time`,`cnt` as `value`").Where("record_time BETWEEN ? and ?", startTime, endTime).Find(&cnts)
		dao.Mysql().Manager.Model(manager.GameDataHour{}).Debug().Select("`record_time`,`active` as `value`").Where("record_time BETWEEN ? and ?", startTime, endTime).Find(&actives)
		dao.Mysql().Manager.Model(manager.GameDataHour{}).Debug().Select("`record_time`,count(active) as `value`").Where("record_time BETWEEN ? and ?", startTime, endTime).Find(&actives)
	}
	result := entity.UserAgentGameDataByHourData{}
	result.Effects = hourDataCombineEff(effs)
	result.Pros = hourDataCombinePro(pros)
	result.Counts = hourDataCombineCnt(cnts)
	result.Active = hourDataCombineActive(actives)
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: result})
}

func UserAndGameSummary(ctx *gin.Context) {
	agentId, err := strconv.Atoi(ctx.Query("agentId"))
	str := ctx.Query("gameId")
	ids := make([]int, 0)

	if err != nil {
		agentId = -1
	}

	if str != "" {
		for _, v := range strings.Split(str, ",") {
			id, _ := strconv.Atoi(v)
			ids = append(ids, id)
		}
	}
	now := time.Now()
	//上月开始时间
	preMonth := now.AddDate(0, -1, 0)
	t := fmt.Sprintf("%04d-%02d-01 00:00:00", preMonth.Year(), preMonth.Month())
	pt, _ := time.ParseInLocation("2006-01-02 15:04:05", t, time.Local)
	preMonthStartTimestamp := pt.Unix()
	//上月结束时间
	t = fmt.Sprintf("%04d-%02d-01 00:00:00", now.Year(), now.Month())
	pt, _ = time.ParseInLocation("2006-01-02 15:04:05", t, time.Local)
	preMonthEndTimestamp := pt.Unix()
	//上上月开始时间
	pre2Month := now.AddDate(0, -2, 0)
	t = fmt.Sprintf("%04d-%02d-01 00:00:00", pre2Month.Year(), pre2Month.Month())
	pt, _ = time.ParseInLocation("2006-01-02 15:04:05", t, time.Local)
	pre2MonthStartTimestamp := pt.Unix()
	//上上月结束时间
	pre2MonthEndTimestamp := preMonthStartTimestamp
	//上周的开始时间
	preWeekDay := now.AddDate(0, 0, -1*int(now.Weekday())-7)
	t = fmt.Sprintf("%04d-%02d-%02d 00:00:00", preWeekDay.Year(), preWeekDay.Month(), preWeekDay.Day())
	pt, _ = time.ParseInLocation("2006-01-02 15:04:05", t, time.Local)
	preWeekStartTimestamp := pt.Unix()
	//上周的结束时间
	preWeekEndTimestamp := preWeekStartTimestamp + 7*24*60*60
	//上上周的开始时间
	pre2WeekDay := now.AddDate(0, 0, -1*int(now.Weekday())-7*2)
	t = fmt.Sprintf("%04d-%02d-%02d 00:00:00", pre2WeekDay.Year(), pre2WeekDay.Month(), pre2WeekDay.Day())
	pt, _ = time.ParseInLocation("2006-01-02 15:04:05", t, time.Local)
	pre2WeekStartTimestamp := pt.Unix()
	//上上周的结束时间
	pre2WeekEndTimestamp := preWeekStartTimestamp

	zap.L().Debug("时间对照",
		zap.Any("preMonth", preMonth.String()),
		zap.Any("上月开始时间", preMonthStartTimestamp), zap.Any("上月结束时间", preMonthEndTimestamp),
		zap.Any("上上月开始时间", pre2MonthStartTimestamp), zap.Any("上上月结束时间", pre2MonthEndTimestamp),
		zap.Any("上周开始时间", preWeekStartTimestamp), zap.Any("上周结束时间", preWeekEndTimestamp),
		zap.Any("上上周开始时间", pre2WeekStartTimestamp), zap.Any("上上周结束时间", pre2WeekEndTimestamp),
	)

	preMonthGameData := make([]*manager.GameDataSummary, 0)
	pre2MonthGameData := make([]*manager.GameDataSummary, 0)
	preWeekGameData := make([]*manager.GameDataSummary, 0)
	pre2WeekGameData := make([]*manager.GameDataSummary, 0)
	curWeekGameData := make([]*manager.GameDataSummary, 0)

	preMonthUserData := make([]*manager.PlayerDataSummary, 0)
	pre2MonthUserData := make([]*manager.PlayerDataSummary, 0)
	preWeekUserData := make([]*manager.PlayerDataSummary, 0)
	pre2WeekUserData := make([]*manager.PlayerDataSummary, 0)

	if agentId != -1 {
		dao.Mysql().Manager.Model(manager.PlayerDataSummary{}).Debug().Select("regist,all_active,new_active").Where("agent_id=? and record_time >= ? and record_time<?", agentId, preMonthStartTimestamp, preMonthEndTimestamp).Find(&preMonthUserData)
		dao.Mysql().Manager.Model(manager.PlayerDataSummary{}).Debug().Select("regist,all_active,new_active").Where("agent_id=? and record_time >= ? and record_time<?", agentId, pre2MonthStartTimestamp, pre2MonthEndTimestamp).Find(&pre2MonthUserData)
		dao.Mysql().Manager.Model(manager.PlayerDataSummary{}).Debug().Select("regist,all_active,new_active").Where("agent_id=? and record_time >= ? and record_time<?", agentId, preWeekStartTimestamp, preWeekEndTimestamp).Find(&preWeekUserData)
		dao.Mysql().Manager.Model(manager.PlayerDataSummary{}).Debug().Select("regist,all_active,new_active").Where("agent_id=? and record_time >= ? and record_time<?", agentId, pre2WeekStartTimestamp, pre2WeekEndTimestamp).Find(&pre2WeekUserData)
		if len(ids) > 0 {
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("agent_id=? and game_id in ? and record_time >= ? and record_time<?", agentId, ids, preMonthStartTimestamp, preMonthEndTimestamp).Find(&preMonthGameData)
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("agent_id=? and game_id in ? and record_time >= ? and record_time<?", agentId, ids, pre2MonthStartTimestamp, pre2MonthEndTimestamp).Find(&pre2MonthGameData)
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("agent_id=? and game_id in ? and record_time >= ? and record_time<?", agentId, ids, preWeekStartTimestamp, preWeekEndTimestamp).Find(&preWeekGameData)
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("agent_id=? and game_id in ? and record_time >= ? and record_time<?", agentId, ids, pre2WeekStartTimestamp, pre2WeekEndTimestamp).Find(&pre2WeekGameData)
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("agent_id=? and game_id in ? and record_time >= ?", agentId, ids, preWeekEndTimestamp).Find(&curWeekGameData)
		} else {
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("agent_id=? and record_time >= ? and record_time<?", agentId, preMonthStartTimestamp, preMonthEndTimestamp).Find(&preMonthGameData)
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("agent_id=? and record_time >= ? and record_time<?", agentId, pre2MonthStartTimestamp, pre2MonthEndTimestamp).Find(&pre2MonthGameData)
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("agent_id=? and record_time >= ? and record_time<?", agentId, preWeekStartTimestamp, preWeekEndTimestamp).Find(&preWeekGameData)
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("agent_id=? and record_time >= ? and record_time<?", agentId, pre2WeekStartTimestamp, pre2WeekEndTimestamp).Find(&pre2WeekGameData)
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("agent_id=? and record_time >= ?", agentId, preWeekEndTimestamp).Find(&curWeekGameData)
		}
	} else {
		dao.Mysql().Manager.Model(manager.PlayerDataSummary{}).Debug().Select("sum(regist) as regist,sum(all_active) as all_active,sum(new_active) as new_active").Where("record_time >= ? and record_time<?", preMonthStartTimestamp, preMonthEndTimestamp).Group("record_time").Find(&preMonthUserData)
		dao.Mysql().Manager.Model(manager.PlayerDataSummary{}).Debug().Select("sum(regist) as regist,sum(all_active) as all_active,sum(new_active) as new_active").Where("record_time >= ? and record_time<?", pre2MonthStartTimestamp, pre2MonthEndTimestamp).Group("record_time").Find(&pre2MonthUserData)
		dao.Mysql().Manager.Model(manager.PlayerDataSummary{}).Debug().Select("sum(regist) as regist,sum(all_active) as all_active,sum(new_active) as new_active").Where("record_time >= ? and record_time<?", preWeekStartTimestamp, preWeekEndTimestamp).Group("record_time").Find(&preWeekUserData)
		dao.Mysql().Manager.Model(manager.PlayerDataSummary{}).Debug().Select("sum(regist) as regist,sum(all_active) as all_active,sum(new_active) as new_active").Where("record_time >= ? and record_time<?", pre2WeekStartTimestamp, pre2WeekEndTimestamp).Group("record_time").Find(&pre2WeekUserData)

		if len(ids) > 0 {
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("game_id in ? and record_time >= ? and record_time<?", ids, preMonthStartTimestamp, preMonthEndTimestamp).Find(&preMonthGameData)
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("game_id in ? and record_time >= ? and record_time<?", ids, pre2MonthStartTimestamp, pre2MonthEndTimestamp).Find(&pre2MonthGameData)
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("game_id in ? and record_time >= ? and record_time<?", ids, preWeekStartTimestamp, preWeekEndTimestamp).Find(&preWeekGameData)
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("game_id in ? and record_time >= ? and record_time<?", ids, pre2WeekStartTimestamp, pre2WeekEndTimestamp).Find(&pre2WeekGameData)
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("game_id in ? and record_time >= ?", ids, preWeekEndTimestamp).Find(&curWeekGameData)
		} else {
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("record_time >= ? and record_time<?", preMonthStartTimestamp, preMonthEndTimestamp).Find(&preMonthGameData)
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("record_time >= ? and record_time<?", pre2MonthStartTimestamp, pre2MonthEndTimestamp).Find(&pre2MonthGameData)
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("record_time >= ? and record_time<?", preWeekStartTimestamp, preWeekEndTimestamp).Find(&preWeekGameData)
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("record_time >= ? and record_time<?", pre2WeekStartTimestamp, pre2WeekEndTimestamp).Find(&pre2WeekGameData)
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("record_time >= ?", preWeekEndTimestamp).Find(&curWeekGameData)
		}
	}

	result := &view.UserAndGameSummaryResult{
		Month: &view.MonthData{},
		Week:  &view.WeekData{},
	}
	result.Month.Cur = &view.UserAndGameSummaryItem{
		User: preMonthUserData,
		Game: preMonthGameData,
	}
	result.Month.Prev = &view.UserAndGameSummaryItem{
		User: pre2MonthUserData,
		Game: pre2MonthGameData,
	}
	result.Week.Cur = &view.UserAndGameSummaryItem{
		User: preWeekUserData,
		Game: preWeekGameData,
	}
	result.Week.Prev = &view.UserAndGameSummaryItem{
		User: pre2WeekUserData,
		Game: pre2WeekGameData,
	}
	result.Week.Next = &view.UserAndGameSummaryItem{
		Game: curWeekGameData,
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: result})
}

func UserAndGameSummaryDetail(ctx *gin.Context) {
	agentId, agentIdError := strconv.Atoi(ctx.Query("agentId"))
	str := ctx.Query("gameId")
	startTime, _ := strconv.ParseInt(ctx.Query("startTime"), 10, 0)
	endTime, _ := strconv.ParseInt(ctx.Query("endTime"), 10, 0)
	ids := make([]int, 0)
	if str != "" {
		for _, v := range strings.Split(str, ",") {
			id, _ := strconv.Atoi(v)
			ids = append(ids, id)
		}
	}
	curMonthGameData := make([]*manager.GameDataSummary, 0)
	curMonthUserData := make([]*manager.PlayerDataSummary, 0)
	if agentIdError == nil {
		dao.Mysql().Manager.Model(manager.PlayerDataSummary{}).Debug().Select("sum(regist) as regist,sum(all_active) as all_active,sum(new_active) as new_active,record_time").Where("agent_id=? and record_time >= ? and record_time<?", agentId, startTime, endTime).Find(&curMonthUserData)
		if len(ids) > 0 {
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("agent_id=? and record_time >= ? and record_time<? and game_id in?", agentId, startTime, endTime, ids).Find(&curMonthGameData)
		} else {
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("agent_id=? and record_time >= ? and record_time<?", agentId, startTime, endTime).Find(&curMonthGameData)
		}
	} else {
		dao.Mysql().Manager.Model(manager.PlayerDataSummary{}).Debug().Select("sum(regist) as regist,sum(all_active) as all_active,sum(new_active) as new_active,record_time").Where("record_time >= ? and record_time<?", startTime, endTime).Group("record_time").Find(&curMonthUserData)
		if len(ids) > 0 {
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("record_time >= ? and record_time<? and game_id in?", startTime, endTime, ids).Find(&curMonthGameData)
		} else {
			dao.Mysql().Manager.Model(manager.GameDataSummary{}).Debug().Where("record_time >= ? and record_time<?", startTime, endTime).Find(&curMonthGameData)
		}
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: view.UserAndGameSummaryDetailResult{
		User: curMonthUserData,
		Game: curMonthGameData,
	}})
}

func ExportAgentData(ctx *gin.Context) {
	startTime, _ := strconv.ParseInt(ctx.Query("startTime"), 10, 0)
	endTime, _ := strconv.ParseInt(ctx.Query("endTime"), 10, 0)
	//统计
	webIdAggs := elastic.NewTermsAggregation().Field("webId")
	aggs := elastic.NewTermsAggregation().Field("agentId").Size(10000)
	gameAggs := elastic.NewTermsAggregation().Field("gameId").Size(3000)
	gameAggs.SubAggregation("effectiveBetsTotal", elastic.NewSumAggregation().Field("exBet"))
	gameAggs.SubAggregation("profitLossTotal", elastic.NewSumAggregation().Field("exWin"))
	gameAggs.SubAggregation("userTotal", elastic.NewCardinalityAggregation().Field("userId"))
	gameAggs.SubAggregation("revenueTotal", elastic.NewSumAggregation().Field("exRevenue"))
	gameAggs.SubAggregation("pumpTotal", elastic.NewSumAggregation().Field("pump"))
	gameAggs.SubAggregation("chipsTotal", elastic.NewSumAggregation().Field("chips"))
	aggs.SubAggregation("gameId", gameAggs)
	webIdAggs.SubAggregation("agentId", aggs)
	zap.L().Debug("====>", zap.Any("startTime", startTime), zap.Any("endTime", endTime))
	query1 := elastic.NewRangeQuery("playedDate").Gte(startTime * 1000).Lt(endTime * 1000)
	query2 := elastic.NewRangeQuery("isTourist").Lte(0)
	boolQuery := elastic.NewBoolQuery().Must(query1, query2)
	resp, err := dao.Es().Search().Index(esindex.Settlement()).Query(boolQuery).Aggregation("webId", webIdAggs).Size(0).Do(context.Background())
	if err != nil {
		zap.L().Error("获取注单打点数据异常", zap.Any("err", err))
		return
	}
	zap.L().Debug("resp", zap.Any("resp", resp))
	result := make(map[string]*view.DataAnalysisItem)
	webIdItems, ok := resp.Aggregations.Terms("webId")
	if ok {
		for _, w := range webIdItems.Buckets {
			webId := w.Key.(float64)
			items, ok := w.Aggregations.Terms("agentId")
			if ok {
				for _, item := range items.Buckets {
					agentId := item.Key.(float64)
					itemGames, ook := item.Aggregations.Terms("gameId")
					if !ook {
						continue
					}
					for _, id := range itemGames.Buckets {
						gameId := id.Key.(float64)
						effectivebets, _ := id.Sum("effectiveBetsTotal")
						profitLoss, _ := id.Sum("profitLossTotal")
						chips, _ := id.Sum("chipsTotal")
						pump, _ := id.Sum("pumpTotal")
						userCount, _ := id.Cardinality("userTotal")
						revenue, _ := id.Sum("revenueTotal")
						cfg := config.CfgIns.GetPoolCfgByGameId(int64(agentId), int64(gameId))
						tmp := &view.DataAnalysisItem{
							WebId:              int(webId),
							AgentId:            int(agentId),
							GameId:             int64(gameId),
							DocCount:           int(id.DocCount),
							ChipsTotal:         *chips.Value,
							ProfitLossTotal:    *profitLoss.Value,
							EffectiveBetsTotal: *effectivebets.Value,
							PumpTotal:          *pump.Value,
							UserTotal:          int(*userCount.Value),
							RevenueTotal:       *revenue.Value,
							GameName:           "",
						}
						if cfg != nil {
							if len(cfg.NameZH) > 0 {
								tmp.GameName = fmt.Sprintf("%s [%s]", cfg.Name, cfg.NameZH)
							} else {
								tmp.GameName = cfg.Name
							}
							tmp.Symbol = cfg.Symbol
						}
						result[fmt.Sprintf("%d-%s", int(agentId), tmp.Symbol)] = tmp
					}
				}
			}
		}
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: result, Msg: "成功"})
}

func ReportFormList(ctx *gin.Context) {
	webId, _ := strconv.Atoi(ctx.Query("webId"))
	agentId, agentIdError := strconv.Atoi(ctx.Query("agentId"))
	str := ctx.Query("gameId")
	startTime, _ := strconv.ParseInt(ctx.Query("startTime"), 10, 0)
	endTime, _ := strconv.ParseInt(ctx.Query("endTime"), 10, 0)
	ids := make([]interface{}, 0)
	if str != "" {
		for _, v := range strings.Split(str, ",") {
			id, _ := strconv.Atoi(v)
			ids = append(ids, id)
		}
	}
	querys := make([]elastic.Query, 0)
	if startTime > 0 && endTime > 0 {
		querys = append(querys, elastic.NewRangeQuery("createAt").Gte(startTime).Lt(endTime))
	}
	if agentIdError == nil {
		querys = append(querys, elastic.NewTermQuery("agentId", agentId))
	}
	if webId > 0 {
		querys = append(querys, elastic.NewTermQuery("webId", webId))
	}
	if len(ids) > 0 {
		querys = append(querys, elastic.NewTermsQuery("gameId", ids...))
	}
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	aggs := elastic.NewTermsAggregation().Field("agentId").Size(10000)
	gameAggs := elastic.NewTermsAggregation().Field("gameId").Size(1000)
	gameAggs.SubAggregation("effectiveBetsTotal", elastic.NewSumAggregation().Field("effectiveBetsTotal"))
	gameAggs.SubAggregation("profitLossTotal", elastic.NewSumAggregation().Field("profitLossTotal"))
	gameAggs.SubAggregation("pumpTotal", elastic.NewSumAggregation().Field("pumpTotal"))
	gameAggs.SubAggregation("userTotal", elastic.NewSumAggregation().Field("userTotal"))
	gameAggs.SubAggregation("revenueTotal", elastic.NewSumAggregation().Field("revenueTotal"))
	gameAggs.SubAggregation("chipsTotal", elastic.NewSumAggregation().Field("chipsTotal"))
	gameAggs.SubAggregation("docCount", elastic.NewSumAggregation().Field("doc_count"))
	aggs.SubAggregation("gameId", gameAggs)
	resp, err := dao.Es().Search().Index(esindex.DataAnalysis()).
		Size(0).
		Query(boolQuery).
		Aggregation("agentId", aggs).
		Pretty(true).
		Do(context.Background())
	if err != nil {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "获取代理统计列表失败"})
		return
	}
	agents := make([]*manager.Agent, 0)
	dao.Mysql().Manager.Model(manager.Agent{}).Debug().Find(&agents)
	result := entity.ReportForm{
		Data: make([]*entity.ReportFormItem, 0, 64),
	}
	aggAgents, ok := resp.Aggregations.Terms("agentId")
	if !ok {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "获取代理统计列表失败"})
		return
	}
	for _, ag := range aggAgents.Buckets {
		agentId, _ := ag.Key.(float64)
		ga, _ := ag.Terms("gameId")
		for _, v := range ga.Buckets {
			gameId, _ := v.Key.(float64)
			item := &entity.ReportFormItem{}
			for _, agent := range agents {
				if agent.Id == int64(agentId) {
					item.AgentId = agent.Id
					item.AgentName = agent.NickName
					item.GameId = int(gameId)
					cfg := config.CfgIns.GetPoolCfgByGameId(agent.Id, int64(gameId))
					if cfg != nil {
						if len(cfg.NameZH) > 0 {
							item.GameName = fmt.Sprintf("%s [%s]", cfg.Name, cfg.NameZH)
						} else {
							item.GameName = cfg.Name
						}
					}
					break
				}
			}
			d, b := v.Sum("effectiveBetsTotal")
			if b {
				result.EffectiveBetsTotal += *d.Value
				item.EffectiveBetsTotal = *d.Value
			}
			d, b = v.Sum("profitLossTotal")
			if b {
				result.ProfitLossTotal += *d.Value
				item.ProfitLossTotal = *d.Value
			}
			d, b = v.Sum("pumpTotal")
			if b {
				result.PumpTotal += *d.Value
				item.PumpTotal = *d.Value
			}
			d, b = v.Sum("userTotal")
			if b {
				result.UserBetsTotal += *d.Value
				item.UserTotal = int(*d.Value)
			}
			d, b = v.Sum("revenueTotal")
			if b {
				result.RevenueTotal += *d.Value
				item.RevenueTotal = *d.Value
			}
			d, b = v.Sum("docCount")
			if b {
				result.DocCount += int64(*d.Value)
				item.Doc2Count = int(*d.Value)
			}
			d, b = v.Sum("chipsTotal")
			if b {
				result.ChipsTotal += *d.Value
				item.ChipsTotal = *d.Value
			}
			zap.L().Debug("原始盈亏数据", zap.Any("gameId", item.GameId), zap.Any("ProfitLossTotal", item.ProfitLossTotal))
			result.Data = append(result.Data, item)
		}
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: result})
}

// 获取时间段范围内活跃的代理
func GetTimeRangeActiveAgent(startTime, endTime, webId int64) []int64 {
	querys := make([]elastic.Query, 0)
	if startTime > 0 && endTime > 0 {
		querys = append(querys, elastic.NewRangeQuery("createAt").Gte(startTime).Lte(endTime))
	}
	if webId > 0 {
		querys = append(querys, elastic.NewTermQuery("webId", webId))
	}
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	resp, err := dao.Es().Search().Index(esindex.DataAnalysis()).
		Size(0).
		Query(boolQuery).
		Pretty(true).
		Aggregation("agentIds", elastic.NewTermsAggregation().Field("agentId").Size(10000)).
		Do(context.Background())
	if err != nil {
		zap.L().Error("查询失败", zap.Any("err", err))
		return nil
	}
	agentIds := make([]int64, 0, 32)
	r, b := resp.Aggregations.Terms("agentIds")
	if b {
		for _, v := range r.Buckets {
			agentIds = append(agentIds, int64(v.Key.(float64)))
		}
	}
	zap.L().Debug("result", zap.Any("data", r))
	return agentIds
}

// 根据筛选条件加载代理数据
func LoadAgents(webId, startTime, endTime, page, pageSize, agentId int64, result *entity.ReportForm) {
	agents := make([]*manager.Agent, 0, 32)
	ids := make([]interface{}, 0, 32)
	var total int64 = 0
	if agentId >= 0 {
		dao.Mysql().Manager.Model(&manager.Agent{}).Where("id = ? and webId=?", agentId, webId).Count(&total)
		dao.Mysql().Manager.Model(&manager.Agent{}).Where("id = ? and webId=?", agentId, webId).Find(&agents)
	} else {
		agentIds := GetTimeRangeActiveAgent(startTime, endTime, webId)
		total = int64(len(agentIds))
		start := (page - 1) * pageSize
		end := start + pageSize
		if end > total && start < total {
			agentIds = agentIds[start:]
		} else if end <= total {
			agentIds = agentIds[start:end]
		} else {
			agentIds = make([]int64, 0)
		}
		dao.Mysql().Manager.Model(&manager.Agent{}).Where("id in ?", agentIds).Find(&agents)
	}
	for _, agent := range agents {
		ids = append(ids, agent.Id)
	}
	querys := make([]elastic.Query, 0)
	if startTime > 0 && endTime > 0 {
		querys = append(querys, elastic.NewRangeQuery("createAt").Gte(startTime).Lt(endTime))
	}
	if len(ids) > 0 {
		querys = append(querys, elastic.NewTermsQuery("agentId", ids...))
	}
	if webId > 0 {
		querys = append(querys, elastic.NewTermQuery("webId", webId))
	}
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	aggs := elastic.NewTermsAggregation().Field("agentId").Size(10000)
	aggs.SubAggregation("effectiveBetsTotal", elastic.NewSumAggregation().Field("effectiveBetsTotal"))
	aggs.SubAggregation("profitLossTotal", elastic.NewSumAggregation().Field("profitLossTotal"))
	aggs.SubAggregation("pumpTotal", elastic.NewSumAggregation().Field("pumpTotal"))
	aggs.SubAggregation("userTotal", elastic.NewSumAggregation().Field("userTotal"))
	aggs.SubAggregation("revenueTotal", elastic.NewSumAggregation().Field("revenueTotal"))
	aggs.SubAggregation("chipsTotal", elastic.NewSumAggregation().Field("chipsTotal"))
	aggs.SubAggregation("docCount", elastic.NewSumAggregation().Field("doc_count"))
	resp, err := dao.Es().Search().Index(esindex.DataAnalysis()).
		Size(0).
		Query(boolQuery).
		Aggregation("agentId", aggs).
		Pretty(true).
		Do(context.Background())
	if err != nil {
		return
	}
	aggAgents, ok := resp.Aggregations.Terms("agentId")
	if !ok {
		return
	}
	for _, v := range aggAgents.Buckets {
		agentId, _ := v.Key.(float64)
		for _, agent := range agents {
			if int64(agentId) == agent.Id {
				item := &entity.ReportFormItem{
					DocCount:  0,
					UserTotal: 0,
				}
				item.AgentId = int64(agentId)
				item.AgentName = agent.NickName
				d, b := v.Sum("effectiveBetsTotal")
				if b {
					item.EffectiveBetsTotal = *d.Value
				}
				d, b = v.Sum("profitLossTotal")
				if b {
					item.ProfitLossTotal = *d.Value
				}
				d, b = v.Sum("pumpTotal")
				if b {
					item.PumpTotal = *d.Value
				}
				d, b = v.Sum("userTotal")
				if b {
					item.UserTotal = int(*d.Value)
				}
				d, b = v.Sum("revenueTotal")
				if b {
					item.RevenueTotal = *d.Value
				}
				d, b = v.Sum("docCount")
				if b {
					item.Doc2Count = int(*d.Value)
				}
				d, b = v.Sum("chipsTotal")
				if b {
					item.ChipsTotal = *d.Value
				}
				result.Data = append(result.Data, item)
				break
			}
		}
	}
	result.Total = total
	result.Page = page
	result.PerPage = pageSize
}

// 根据筛选条件查询所有统计数据
func AggsAllWithAgent(startTime, endTime, webId, agentId int64) (map[string]float64, error) {
	result := map[string]float64{
		"effectiveBetsTotal": 0.0,
		"profitLossTotal":    0.0,
		"pumpTotal":          0.0,
		"userTotal":          0.0,
		"revenueTotal":       0.0,
		"chipsTotal":         0.0,
		"docCount":           0.0,
	}
	zap.L().Debug("params", zap.Any("startTime", startTime), zap.Any("endTime", endTime), zap.Any("agentId", agentId))
	querys := make([]elastic.Query, 0)
	if startTime > 0 && endTime > 0 {
		querys = append(querys, elastic.NewRangeQuery("createAt").Gte(startTime).Lte(endTime))
	}
	if agentId >= 0 {
		querys = append(querys, elastic.NewTermQuery("agentId", agentId))
	}
	if webId > 0 {
		querys = append(querys, elastic.NewTermQuery("webId", webId))
	}
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	resp, err := dao.Es().Search().Index(esindex.DataAnalysis()).
		Size(0).
		Query(boolQuery).
		Aggregation("effectiveBetsTotal", elastic.NewSumAggregation().Field("effectiveBetsTotal")).
		Aggregation("profitLossTotal", elastic.NewSumAggregation().Field("profitLossTotal")).
		Aggregation("pumpTotal", elastic.NewSumAggregation().Field("pumpTotal")).
		Aggregation("userTotal", elastic.NewSumAggregation().Field("userTotal")).
		Aggregation("revenueTotal", elastic.NewSumAggregation().Field("revenueTotal")).
		Aggregation("chipsTotal", elastic.NewSumAggregation().Field("chipsTotal")).
		Aggregation("docCount", elastic.NewSumAggregation().Field("doc_count")).
		Pretty(true).
		Do(context.Background())
	if err != nil {
		zap.L().Error("查询失败", zap.Any("err", err))
		return nil, err
	}
	agents := make([]*manager.Agent, 0)
	dao.Mysql().Manager.Model(manager.Agent{}).Debug().Find(&agents)
	eff, ok := resp.Aggregations.Sum("effectiveBetsTotal")
	if ok {
		result["effectiveBetsTotal"] = *eff.Value
	}
	pro, ok := resp.Aggregations.Sum("profitLossTotal")
	if ok {
		result["profitLossTotal"] = *pro.Value
	}
	pum, ok := resp.Aggregations.Sum("pumpTotal")
	if ok {
		result["pumpTotal"] = *pum.Value
	}
	ut, ok := resp.Aggregations.Sum("userTotal")
	if ok {
		result["userTotal"] = *ut.Value
	}
	rt, ok := resp.Aggregations.Sum("revenueTotal")
	if ok {
		result["revenueTotal"] = *rt.Value
	}
	chips, ok := resp.Aggregations.Sum("chipsTotal")
	if ok {
		result["chipsTotal"] = *chips.Value
	}
	dc, ok := resp.Aggregations.Sum("docCount")
	if ok {
		result["docCount"] = *dc.Value
	}
	return result, nil
}

func ReportFormListWithAgent(ctx *gin.Context) {
	page, _ := strconv.ParseInt(ctx.Query("page"), 10, 64)
	pageSize, _ := strconv.ParseInt(ctx.Query("pageSize"), 10, 64)
	webId, _ := strconv.ParseInt(ctx.Query("webId"), 10, 64)
	agentId, ae := strconv.ParseInt(ctx.Query("agentId"), 10, 64)
	if ae != nil {
		agentId = -1
	}
	startTime, _ := strconv.ParseInt(ctx.Query("startTime"), 10, 0)
	endTime, _ := strconv.ParseInt(ctx.Query("endTime"), 10, 0)
	res, err := AggsAllWithAgent(startTime, endTime, webId, agentId)
	if err != nil {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "失败"})
		return
	}
	result := &entity.ReportForm{
		Data:               make([]*entity.ReportFormItem, 0, 32),
		EffectiveBetsTotal: res["effectiveBetsTotal"],
		ProfitLossTotal:    res["profitLossTotal"],
		PumpTotal:          res["pumpTotal"],
		UserBetsTotal:      res["userTotal"],
		RevenueTotal:       res["revenueTotal"],
		ChipsTotal:         res["chipsTotal"],
		DocCount:           int64(res["docCount"]),
	}
	LoadAgents(webId, startTime, endTime, page, pageSize, agentId, result)
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: result})
}

// 获取指定代理游戏统计详情
func GetAgentGameDataAggs(ctx *gin.Context) {
	agentId, ae := strconv.ParseInt(ctx.Query("agentId"), 10, 64)
	startTime, se := strconv.ParseInt(ctx.Query("startTime"), 10, 0)
	endTime, ee := strconv.ParseInt(ctx.Query("endTime"), 10, 0)
	querys := make([]elastic.Query, 0)
	if se == nil && ee == nil {
		querys = append(querys, elastic.NewRangeQuery("createAt").Gte(startTime).Lt(endTime))
	}
	if ae == nil {
		querys = append(querys, elastic.NewTermQuery("agentId", agentId))
	}
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	aggs := elastic.NewTermsAggregation().Field("gameId").Size(10000)
	aggs.SubAggregation("effectiveBetsTotal", elastic.NewSumAggregation().Field("effectiveBetsTotal"))
	aggs.SubAggregation("profitLossTotal", elastic.NewSumAggregation().Field("profitLossTotal"))
	aggs.SubAggregation("pumpTotal", elastic.NewSumAggregation().Field("pumpTotal"))
	aggs.SubAggregation("userTotal", elastic.NewSumAggregation().Field("userTotal"))
	aggs.SubAggregation("revenueTotal", elastic.NewSumAggregation().Field("revenueTotal"))
	aggs.SubAggregation("chipsTotal", elastic.NewSumAggregation().Field("chipsTotal"))
	aggs.SubAggregation("docCount", elastic.NewSumAggregation().Field("doc_count"))
	resp, err := dao.Es().Search().Index(esindex.DataAnalysis()).
		Size(0).
		Query(boolQuery).
		Aggregation("gameId", aggs).
		Pretty(true).
		Do(context.Background())
	if err != nil {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "获取代理统计列表失败"})
		return
	}
	agents := make([]*manager.Agent, 0)
	dao.Mysql().Manager.Model(manager.Agent{}).Debug().Where("id=?", agentId).Find(&agents)
	if len(agents) <= 0 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "获取代理信息失败"})
		return
	}
	result := entity.AgentGameAggs{
		Data: make([]*entity.ReportFormItem, 0, 64),
	}
	gameAggs, ok := resp.Aggregations.Terms("gameId")
	if !ok {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "获取统计结果失败"})
		return
	}
	for _, v := range gameAggs.Buckets {
		gameId, _ := v.Key.(float64)
		item := &entity.ReportFormItem{}
		item.AgentId = agentId
		item.AgentName = agents[0].NickName
		item.GameId = int(gameId)
		cfg := config.CfgIns.GetPoolCfgByGameId(agentId, int64(gameId))
		if cfg != nil {
			if len(cfg.NameZH) > 0 {
				item.GameName = fmt.Sprintf("%s [%s]", cfg.Name, cfg.NameZH)
			} else {
				item.GameName = cfg.Name
			}
			item.Symbol = cfg.Symbol
		}
		d, b := v.Sum("effectiveBetsTotal")
		if b {
			item.EffectiveBetsTotal = *d.Value
		}
		d, b = v.Sum("profitLossTotal")
		if b {
			item.ProfitLossTotal = *d.Value
		}
		d, b = v.Sum("pumpTotal")
		if b {
			item.PumpTotal = *d.Value
		}
		d, b = v.Sum("userTotal")
		if b {
			item.UserTotal = int(*d.Value)
		}
		d, b = v.Sum("revenueTotal")
		if b {
			item.RevenueTotal = *d.Value
		}
		d, b = v.Sum("docCount")
		if b {
			item.Doc2Count = int(*d.Value)
		}
		d, b = v.Sum("chipsTotal")
		if b {
			item.ChipsTotal = *d.Value
		}
		result.Data = append(result.Data, item)
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: result})
}

func AgentReportFormList(ctx *gin.Context) {
	agentId := ctx.GetInt64("agentId")
	str := ctx.Query("gameId")
	startTime, _ := strconv.ParseInt(ctx.Query("startTime"), 10, 0)
	endTime, _ := strconv.ParseInt(ctx.Query("endTime"), 10, 0)
	ids := make([]interface{}, 0)
	if str != "" {
		for _, v := range strings.Split(str, ",") {
			id, _ := strconv.Atoi(v)
			ids = append(ids, id)
		}
	}
	querys := make([]elastic.Query, 0)
	if startTime > 0 && endTime > 0 {
		querys = append(querys, elastic.NewRangeQuery("createAt").Gte(startTime).Lte(endTime))
	}
	querys = append(querys, elastic.NewTermQuery("agentId", agentId))
	if len(ids) > 0 {
		querys = append(querys, elastic.NewTermsQuery("gameId", ids...))
	}
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	aggs := elastic.NewTermsAggregation().Field("agentId")
	gameAggs := elastic.NewTermsAggregation().Field("gameId").Size(1000)
	gameAggs.SubAggregation("effectiveBetsTotal", elastic.NewSumAggregation().Field("effectiveBetsTotal"))
	gameAggs.SubAggregation("profitLossTotal", elastic.NewSumAggregation().Field("profitLossTotal"))
	gameAggs.SubAggregation("pumpTotal", elastic.NewSumAggregation().Field("pumpTotal"))
	gameAggs.SubAggregation("userTotal", elastic.NewSumAggregation().Field("userTotal"))
	gameAggs.SubAggregation("revenueTotal", elastic.NewSumAggregation().Field("revenueTotal"))
	gameAggs.SubAggregation("chipsTotal", elastic.NewSumAggregation().Field("chipsTotal"))
	gameAggs.SubAggregation("docCount", elastic.NewSumAggregation().Field("doc_count"))
	aggs.SubAggregation("gameId", gameAggs)
	resp, err := dao.Es().Search().Index(esindex.DataAnalysis()).
		Size(0).
		Query(boolQuery).
		Aggregation("agentId", aggs).
		Pretty(true).
		Do(context.Background())
	if err != nil {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "失败"})
		return
	}
	agents := make([]*manager.Agent, 0)
	dao.Mysql().Manager.Model(manager.Agent{}).Debug().Find(&agents)
	result := entity.ReportForm{
		Data: make([]*entity.ReportFormItem, 0, 32),
	}
	aggAgents, _ := resp.Aggregations.Terms("agentId")
	for _, ag := range aggAgents.Buckets {
		ga, _ := ag.Terms("gameId")
		for _, v := range ga.Buckets {
			gameId, _ := v.Key.(float64)
			item := &entity.ReportFormItem{}
			for _, agent := range agents {
				id, _ := v.Key.(int64)
				if agent.Id == id {
					item.AgentId = agent.Id
					item.AgentName = agent.NickName
					item.GameId = int(gameId)
					cfg := config.CfgIns.GetPoolCfgByGameId(agent.Id, int64(gameId))
					if cfg != nil {
						if len(cfg.NameZH) > 0 {
							item.GameName = fmt.Sprintf("%s [%s]", cfg.Name, cfg.NameZH)
						} else {
							item.GameName = cfg.Name
						}
					}
					break
				}
			}
			d, b := v.Sum("effectiveBetsTotal")
			if b {
				result.EffectiveBetsTotal += *d.Value
				item.EffectiveBetsTotal = *d.Value
			}
			d, b = v.Sum("profitLossTotal")
			if b {
				result.ProfitLossTotal += *d.Value
				item.ProfitLossTotal = *d.Value
			}
			d, b = v.Sum("pumpTotal")
			if b {
				result.PumpTotal += *d.Value
				item.PumpTotal = *d.Value
			}
			d, b = v.Sum("userTotal")
			if b {
				result.UserBetsTotal += *d.Value
				item.UserTotal = int(*d.Value)
			}
			d, b = v.Sum("revenueTotal")
			if b {
				result.RevenueTotal += *d.Value
				item.RevenueTotal = *d.Value
			}
			d, b = v.Sum("docCount")
			if b {
				result.DocCount += int64(*d.Value)
				item.Doc2Count = int(*d.Value)
			}
			d, b = v.Sum("chipsTotal")
			if b {
				result.ChipsTotal += *d.Value
				item.ChipsTotal = *d.Value
			}
			item.ProfitLossTotal = item.EffectiveBetsTotal - item.ProfitLossTotal
			result.Data = append(result.Data, item)
		}
	}
	result.ProfitLossTotal = result.EffectiveBetsTotal - result.ProfitLossTotal
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: result})
}

func ReportFormHistory(ctx *gin.Context) {
	var total int64 = 0
	page, _ := strconv.Atoi(ctx.Query("page"))
	pageSize, _ := strconv.Atoi(ctx.Query("pageSize"))
	if pageSize <= 0 {
		pageSize = 30
	}
	offset := (page - 1) * pageSize
	agentId, agentIdError := strconv.Atoi(ctx.Query("agentId"))
	startTime, _ := strconv.ParseInt(ctx.Query("startTime"), 10, 0)
	endTime, _ := strconv.ParseInt(ctx.Query("endTime"), 10, 0)
	str := ctx.Query("gameId")
	ids := make([]interface{}, 0)
	if str != "" {
		for _, v := range strings.Split(str, ",") {
			id, _ := strconv.Atoi(v)
			ids = append(ids, id)
		}
	}
	querys := make([]elastic.Query, 0)
	if startTime > 0 {
		querys = append(querys, elastic.NewRangeQuery("startTime").Gte(startTime))
	}
	if endTime > 0 {
		querys = append(querys, elastic.NewRangeQuery("endTime").Lte(endTime))
	}
	if agentIdError == nil {
		querys = append(querys, elastic.NewTermQuery("agentId", agentId))
	}
	if len(ids) > 0 {
		querys = append(querys, elastic.NewTermsQuery("gameId", ids...))
	}
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	resp, err := dao.Es().Search().Index(esindex.DataAnalysisRange()).
		Sort("endTime", false).
		Size(pageSize).
		From(offset).
		Query(boolQuery).
		Pretty(true).
		Do(context.Background())
	if err != nil {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "失败"})
		return
	}
	total = resp.Hits.TotalHits.Value
	res := &entity.ReportFormHistory{
		Total:   total,
		Page:    int64(page),
		PerPage: int64(pageSize),
	}
	for _, v := range resp.Hits.Hits {
		res.Data = append(res.Data, v.Source)
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: res})
}

func ClearPlayerLock(ctx *gin.Context) {
	ids := ctx.Query("ids")
	userIds := make([]string, 0)
	userIds = append(userIds, strings.Split(ids, ",")...)
	users := make([]*manager.User, 0)
	dao.Mysql().Manager.Where("userId in ?", userIds).Find(&users)
	piple := dao.RedisIns().Client.Pipeline()
	for _, user := range users {
		piple.Del(context.Background(), fmt.Sprintf("user_lock_%d", user.Id))
		piple.Del(context.Background(), fmt.Sprintf("score_lock_%d", user.Id))
	}
	piple.Exec(context.Background())
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: nil})
}

func GovernUserRecord(ctx *gin.Context) {
	userId, _ := strconv.ParseInt(ctx.Query("userId"), 10, 64)
	page, _ := strconv.Atoi(ctx.Query("page"))
	pageSize := 30
	offset := (page - 1) * pageSize
	querys := make([]elastic.Query, 0)
	if userId > 0 {
		querys = append(querys, elastic.NewTermQuery("userId", userId))
	}
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	resp, err := dao.Es().Search().Index(esindex.UserController()).
		Query(boolQuery).
		Sort("createTime", false).
		Pretty(true).
		Size(pageSize).
		From(offset).
		Do(context.Background())
	res := &entity.GovernUserRecord{
		Total:    0,
		Page:     int64(page),
		PageSize: int64(pageSize),
		Data:     make([]interface{}, 0, 32),
	}
	if err != nil {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: res})
		return
	}
	res.Total = resp.Hits.TotalHits.Value
	for _, v := range resp.Hits.Hits {
		res.Data = append(res.Data, v.Source)
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: res})
}

func GovernUser(ctx *gin.Context) {
	strUserId := ctx.Query("userId")
	userId, _ := strconv.ParseInt(strUserId, 10, 64)
	rate, _ := decimal.NewFromString(ctx.Query("rate"))
	rate_score, _ := decimal.NewFromString(ctx.Query("rate_score"))
	if userId <= 0 {
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "参数异常", Data: nil})
		return
	}

	if rate.Equal(decimal.Zero) && rate_score.Equal(decimal.Zero) {
		dao.RedisIns().Client.HDel(context.Background(), "user_ctl", strUserId)
		dao.Mysql().Manager.Model(manager.User{}).Debug().Exec("update gp_user set isCtl=0 where id=?", userId)
		dataStr, _ := jsoniter.MarshalToString(&entity.DelPlayerSingleCtrl{Uid: uint32(userId)})
		msg, _ := jsoniter.MarshalToString(map[string]interface{}{
			"event": "delSingleCtrl",
			"data":  dataStr,
		})
		dao.RedisIns().Publish("message", msg)
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: nil})
		return
	}

	str := dao.RedisIns().Client.HGet(context.Background(), "user_ctl", strUserId).Val()
	uc := &view.UserSingleControl{}
	jsoniter.UnmarshalFromString(str, uc)

	uc.Rate = rate
	uc.LeftScore = rate_score
	uc.RateScore = rate_score
	uc.UserId = userId
	uc.CtrlType = 1

	tmpStr, _ := jsoniter.MarshalToString(uc)

	dao.RedisIns().Client.HSet(context.Background(), "user_ctl", strUserId, tmpStr)
	dao.Mysql().Manager.Model(manager.User{}).Debug().Exec("update gp_user set isCtl=1,controlNumber=controlNumber+1 where id=?", userId)
	uc.CreateTime = time.Now().Unix()
	dao.Es().Index().Index(esindex.UserController()).BodyJson(uc).Do(context.Background())

	dataStr, _ := jsoniter.MarshalToString(&entity.AddPlayerSingleCtrl{
		Uid:         uint32(userId),
		RemainScore: rate_score,
		CtrlScore:   rate_score,
		UpdateTime:  uc.CreateTime,
		Rate:        rate,
	})
	msg, _ := jsoniter.MarshalToString(map[string]interface{}{
		"event": "addSingleCtrl",
		"data":  dataStr,
	})
	dao.RedisIns().Publish("message", msg)
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: nil})
}

func AgentLineChart(ctx *gin.Context) {
	agentId := ctx.GetInt64("agentId")
	startTime, _ := strconv.ParseInt(ctx.Query("startTime"), 10, 64)
	endTime, _ := strconv.ParseInt(ctx.Query("endTime"), 10, 64)
	gameId, _ := strconv.ParseInt(ctx.Query("gameId"), 10, 64)
	page, _ := strconv.ParseInt(ctx.Query("page"), 10, 64)
	if page > 0 {
		page -= page
	} else {
		page = 0
	}
	querys := make([]elastic.Query, 0, 16)
	querys = append(querys, elastic.NewRangeQuery("startTime").Gte(startTime))
	querys = append(querys, elastic.NewRangeQuery("endTime").Lte(endTime))
	querys = append(querys, elastic.NewTermQuery("agentId", agentId))
	if gameId > 0 {
		querys = append(querys, elastic.NewTermQuery("gameId", gameId))
	}
	boolQuery := elastic.NewBoolQuery().Must(querys...)
	resp, err := dao.Es().Search().Index(esindex.DataAnalysisRange()).
		Size(1000).
		Query(boolQuery).
		From(int(page * 1000)).
		Pretty(true).
		Do(context.Background())
	if err != nil {
		zap.L().Error("AgentLineChart", zap.Any("err", err))
		ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusInternalServerError, Msg: "失败", Data: nil})
		return
	}
	result := make([]interface{}, 0, 128)
	for _, v := range resp.Hits.Hits {
		result = append(result, v.Source)
	}
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Msg: "成功", Data: map[string]interface{}{"total": resp.Hits.TotalHits.Value, "data": result}})
}
