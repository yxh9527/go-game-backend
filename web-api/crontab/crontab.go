package crontab

import (
	"app/entity/view"
	"app/esindex"
	"app/tables/manager"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"web-api/common"
	"web-api/dao"

	jsoniter "github.com/json-iterator/go"
	"github.com/olivere/elastic/v7"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

var c *cron.Cron = nil

// 清空水池
func clearPool() {
	rpc := &view.ResetPoolConfig{}
	str := dao.RedisIns().Client.Get(context.Background(), "reset_pool_config").Val()
	if str != "" {
		jsoniter.UnmarshalFromString(str, rpc)
	}
	if rpc.ResetTime == 0 || rpc.Interval == 0 {
		zap.L().Error("重置水池异常配置异常", zap.Any("data", rpc), zap.Any("str", str))
		return
	}
	if !rpc.Now && rpc.ResetTime > time.Now().Unix() {
		return
	}
	msg, _ := jsoniter.MarshalToString(map[string]interface{}{
		"event": "resetPool",
		"data":  "{}",
	})
	dao.RedisIns().Publish("message", msg)
	rpc.Now = false
	rpc.ResetTime = time.Now().Unix() + int64(rpc.Interval)*24*60*60
	s, _ := jsoniter.MarshalToString(rpc)
	dao.RedisIns().Client.Set(context.Background(), "reset_pool_config", s, -1)
}

// 清理数据
func clearSettlementData() {
	t := time.Now().Unix() - 3*30*24*60*60
	zap.L().Debug("定时清理数据开始")
	dao.Es().DeleteByQuery(esindex.Settlement()).Query(elastic.NewBoolQuery().Filter(elastic.NewRangeQuery("playedDate").Lte(t))).WaitForCompletion(false).Do(context.Background())
	dao.Es().DeleteByQuery(esindex.FlowingWaterLegacy()).Query(elastic.NewBoolQuery().Filter(elastic.NewRangeQuery("beginTime").Lte(t))).WaitForCompletion(false).Do(context.Background())
	dao.Es().DeleteByQuery(esindex.PoolRecordLog()).Query(elastic.NewBoolQuery().Filter(elastic.NewRangeQuery("beginTime").Lte(t))).WaitForCompletion(false).Do(context.Background())
	zap.L().Debug("定时清理数据结束")
}

// 清理玩家日志
func clearPlayerLog() {
	dao.Mysql().Manager.Model(manager.UserScoreLog{}).Debug().Where("createtime<=?", time.Now().Unix()-10*24*60*60).Delete(manager.UserScoreLog{})
	zap.L().Debug("定时清理玩家积分日志记录")
}

// 清理玩家控制信息
func userCtlTime() {
	psi := dao.RedisIns().Client.Get(context.Background(), "player_statistics_interval").Val()
	if psi == "" {
		return
	}
	arr := strings.Split(psi, "|")
	if len(arr) >= 3 {
		timeOut, _ := strconv.ParseInt(arr[1], 10, 64)
		now := time.Now().Unix()
		if now > timeOut {
			interval, _ := strconv.ParseInt(arr[2], 10, 64)
			newStr := fmt.Sprintf("%d|%d|%s", now, now+interval, arr[2])
			piple := dao.RedisIns().Client.Pipeline()
			piple.Set(context.Background(), "player_statistics_interval", newStr, -1)
			piple.Del(context.Background(), "userTotalProfLoss")
			piple.Del(context.Background(), "userTotalEffBet")
			piple.Del(context.Background(), "userBetCount")
			piple.Exec(context.Background())
		}
	}
	zap.L().Debug("清理玩家控制信息")
}

func getUserSettlementStatisData(start, end int64) []*view.UserGameDataStatisItem {
	//统计玩家数据
	aggs := elastic.NewTermsAggregation().Field("userId")
	aggs.SubAggregation("effectiveBetsTotal", elastic.NewSumAggregation().Field("effectiveBets"))
	aggs.SubAggregation("profitLossTotal", elastic.NewSumAggregation().Field("profitLoss"))
	aggs.SubAggregation("pumpTotal", elastic.NewSumAggregation().Field("pump"))
	aggs.SubAggregation("playTimesTotal", elastic.NewSumAggregation().Field("playTimes"))
	aggs.SubAggregation("chipsTotal", elastic.NewSumAggregation().Field("chips"))
	query := elastic.NewRangeQuery("createTime").Gte(start).Lt(end)
	boolQuery := elastic.NewBoolQuery().Must(query)
	resp, _ := dao.Es().Search().Index(esindex.Settlement()).Query(boolQuery).Aggregation("userId", aggs).Pretty(true).Size(0).Do(context.Background())
	result := make([]*view.UserGameDataStatisItem, 0, 64)
	items, ok := resp.Aggregations.Terms("userId")
	if ok {
		for _, item := range items.Buckets {
			userId := item.Key.(float64)
			count := item.DocCount
			effectivebets, _ := item.Sum("effectiveBetsTotal")
			profitLoss, _ := item.Sum("profitLossTotal")
			chips, _ := item.Sum("chipsTotal")
			pump, _ := item.Sum("pumpTotal")
			playTimes, _ := item.Sum("playTimesTotal")
			result = append(result, &view.UserGameDataStatisItem{
				UserId:        uint32(userId),
				EffectiveBets: *effectivebets.Value,
				ProfitLoss:    *profitLoss.Value,
				Pump:          *pump.Value,
				Chips:         *chips.Value,
				Count:         int(count),
				PlayTimes:     int(*playTimes.Value),
			})
		}
	}
	return result
}

// 格式化参数
func genParams(params interface{}) string {
	k := reflect.TypeOf(params).Kind()
	switch k {
	case reflect.String:
		return "%s"
	case reflect.Float32, reflect.Float64:
		return "%f"
	default:
		return "%d"
	}
}

// 格式化when条件
func genWhenFormatter(filed string, value interface{}, param interface{}) string {
	str := "WHEN %s =" + genParams(value) + " THEN " + genParams(param)
	return fmt.Sprintf(str, filed, value, param)
}

// 更新玩家统计数据,默认添加，其他情况特殊处理
// tableName:  表名
// filed: 条件筛选字段名
// columns: 需要更新的列名
// data: 数据
func genBatchUserDataSql(tableName, filed string, columns map[string]string, data []*view.UserGameDataStatisItem) string {
	params := make(map[string][]string, 0)
	for k, _ := range columns {
		params[k] = make([]string, 0, 16)
	}
	ids := make([]string, 0, 64)
	for _, item := range data {
		item.UpdateTime = time.Now().Unix()
		b, _ := json.Marshal(item)
		tmp := make(map[string]interface{})
		json.Unmarshal(b, &tmp)
		//格式化then条件
		for k, v := range tmp {
			if k == filed {
				ids = append(ids, fmt.Sprintf(genParams(v), v))
				continue
			}
			params[k] = append(params[k], genWhenFormatter(filed, item.UserId, v))
		}
	}
	//formatter when
	when := make(map[string]string)
	for k, v := range params {
		when[k] = strings.Join(v, " ")
	}
	//组装case
	c := make([]string, 0, 64)
	for k, v := range when {
		formatter := ""
		op := columns[k]
		switch op {
		case "+":
			formatter = "`%s` = `%s` + CASE %s END"
			c = append(c, fmt.Sprintf(formatter, k, k, v))
		case "-":
			formatter = "`%s` = `%s` - CASE %s END"
			c = append(c, fmt.Sprintf(formatter, k, k, v))
		case "=":
			formatter = "`%s` = CASE %s END"
			c = append(c, fmt.Sprintf(formatter, k, v))
		}
	}
	//formatter case
	cstr := strings.Join(c, ",")
	//组装where
	where := fmt.Sprintf("%s IN (%s)", filed, strings.Join(ids, ","))
	//组装完整sql
	return fmt.Sprintf("UPDATE %s SET %s WHERE %s;", tableName, cstr, where)
}

// 统计玩家游戏数据
func userGameDataStatis() {
	var startTime, endTime int64 = 0, 0
	//获取上次统计玩家游戏数据的时间
	str := dao.RedisIns().Client.Get(context.Background(), "statistics_user_data_time").Val()
	if str == "" {
		startTime = 0
	} else {
		startTime, _ = strconv.ParseInt(str, 10, 0)
		zap.L().Debug("上次统计用户时间", zap.Any("time", startTime))
	}
	endTime = time.Now().Unix() - 5*60
	userDatas := getUserSettlementStatisData(startTime, endTime)
	zap.L().Debug("玩家数据", zap.Any("data", userDatas))
	//生成批量更新sql
	tmp := make([]*view.UserGameDataStatisItem, 0, 32)
	for _, v := range userDatas {
		tmp = append(tmp, v)
		if len(tmp) >= 20 {
			sql := genBatchUserDataSql("gp_user", "id", map[string]string{
				"totalNumber":   "+",
				"totalEffBet":   "+",
				"totalProfLoss": "+",
				"totalPump":     "+",
				"times":         "+",
				"updateTime":    "=",
			}, tmp)
			//执行sql
			if err := dao.Mysql().Manager.Exec(sql).Error; err != nil {
				zap.L().Error("更新失败", zap.Any("err", err))
			}
			tmp = make([]*view.UserGameDataStatisItem, 0, 32)
		}
	}
	if len(tmp) > 0 {
		sql := genBatchUserDataSql("gp_user", "id", map[string]string{
			"totalNumber":   "+",
			"totalEffBet":   "+",
			"totalProfLoss": "+",
			"totalPump":     "+",
			"times":         "+",
			"updateTime":    "=",
		}, tmp)
		//执行sql
		if err := dao.Mysql().Manager.Exec(sql).Error; err != nil {
			zap.L().Error("更新失败", zap.Any("err", err))
		}
	}
	dao.RedisIns().Client.Set(context.Background(), "statistics_user_data_time", endTime, -1)
}

func NewCrontab() {
	if c == nil {
		c = cron.New(cron.WithParser(cron.NewParser(
			cron.SecondOptional|cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.Dow|cron.Descriptor,
		)), cron.WithChain(
			cron.Recover(cron.DefaultLogger), // or use cron.DefaultLogger
		))
	}
	//清空水池
	c.AddFunc("*/1 * * * *", clearPool)
	//定时清理数据
	c.AddFunc("*/10 * * * *", clearSettlementData)
	//清理玩家日志
	c.AddFunc("0 */1 * * *", clearPlayerLog)
	//每小时代理打点数据 每小时的第五分钟统计上一个小时的数据
	c.AddFunc("5 */1 * * *", common.AgentAggsByHour)
	//每天打点数据
	c.AddFunc("10 0 * * *", common.AgentAggsByDay)
	//清理玩家控制信息
	c.AddFunc("*/1 * * * *", userCtlTime)
	//定时聚合当前数据
	c.AddFunc("*/5 * * * *", common.AgentAggsNow)
	//统计玩家游戏数据
	c.AddFunc("*/10 * * * *", userGameDataStatis)
	//启动计划任务
	c.Start()
}
