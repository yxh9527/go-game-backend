package cfgparse

import (
	"app/config"
	"app/esindex"

	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

// Parse 解析 Redis 配置 key（格式：/{Prefix}/config/... 或 /{Prefix}/agent/...）。
// 分割后第一段必须等于 esindex.Prefix（代码常量，如 fg），否则视为其他服务配置，直接跳过。
func Parse(key string, value string) {
	if config.CfgIns == nil {
		zap.L().Error("配置实例未初始化", zap.Any("key", key))
		return
	}
	if value == "" {
		zap.L().Error("配置数据异常", zap.Any("key", key), zap.Any("data", value))
		return
	}
	if !esindex.MatchKeyPrefix(key) {
		zap.L().Debug("skip config: prefix mismatch",
			zap.String("key", key),
			zap.String("expectPrefix", esindex.Prefix))
		return
	}
	section, rest := esindex.ParsePath(key)
	if section == "" {
		zap.L().Error("配置数据异常", zap.Any("key", key), zap.Any("data", value))
		return
	}
	switch section {
	case "config":
		if len(rest) < 1 {
			zap.L().Error("配置数据异常", zap.Any("key", key), zap.Any("data", value))
			return
		}
		switch rest[0] {
		case "system":
			tmp := &config.SystemConfig{}
			if err := jsoniter.UnmarshalFromString(value, tmp); err == nil {
				config.CfgIns.SetSystemConfig(tmp)
			} else {
				zap.L().Error("加载系统配置失败", zap.Any("err", err), zap.Any("value", value))
			}
		case "currency":
			tmp := config.CfgIns.Currency
			if err := jsoniter.UnmarshalFromString(value, tmp); err == nil {
				config.CfgIns.SetCurrency(tmp)
				zap.L().Debug("加载currency配置文件成功", zap.Any("data", tmp))
			} else {
				zap.L().Error("加载系统配置失败", zap.Any("err", err), zap.Any("value", value))
			}
		case "pool":
			if len(rest) < 2 {
				zap.L().Error("pool配置key异常", zap.Any("key", key))
				return
			}
			tmp := &config.Pool{}
			if err := jsoniter.UnmarshalFromString(value, tmp); err == nil {
				tmp.Symbol = rest[1]
				zap.L().Debug("加载pool配置文件成功", zap.Any("data", tmp), zap.String("symbol", rest[1]))
				config.CfgIns.SetDefaultPool(rest[1], tmp)
			} else {
				zap.L().Error("加载pool配置失败", zap.Any("err", err))
			}
		case "ctrl":
			if len(rest) < 2 {
				zap.L().Error("ctrl配置key异常", zap.Any("key", key))
				return
			}
			tmp := &config.AwardConfig{}
			if err := jsoniter.UnmarshalFromString(value, tmp); err == nil {
				if rest[1] != "default" && tmp.GameId == 0 {
					zap.L().Debug("ctrl 配置异常", zap.Any("data", tmp))
					return
				}
				config.CfgIns.SetCtrl(rest[1], tmp)
			} else {
				zap.L().Error("加载ctrl配置失败", zap.Any("err", err))
			}
		case "autoCtrl":
			tmp := &config.AutoCtrlMgr{
				Ctrls: make([]*config.AutoCtrlItem, 0, 32),
			}
			if err := jsoniter.UnmarshalFromString(value, &tmp.Ctrls); err == nil {
				config.CfgIns.SetAutoCtrl(tmp)
			} else {
				tmp2 := &config.AutoCtrlMgr{}
				if err2 := jsoniter.UnmarshalFromString(value, tmp2); err2 == nil {
					config.CfgIns.SetAutoCtrl(tmp2)
				} else {
					zap.L().Error("加载autoCtrl配置失败", zap.Any("err", err))
				}
			}
		case "controlCoefficient":
			tmp := config.DefaultControlCoefficientConfig()
			if err := jsoniter.UnmarshalFromString(value, tmp); err == nil {
				config.CfgIns.SetControlCoefficient(tmp)
				zap.L().Debug("加载controlCoefficient配置成功", zap.Any("data", tmp))
			} else {
				zap.L().Error("加载控制系数配置失败", zap.Any("err", err), zap.Any("value", value))
			}
		}
	case "agent":
		if len(rest) >= 3 && rest[1] == "pool" {
			tmp := &config.Pool{}
			if err := jsoniter.UnmarshalFromString(value, tmp); err == nil {
				tmp.Symbol = rest[2]
				config.CfgIns.SetAgentPool(rest[0], tmp)
			} else {
				zap.L().Error("加载代理pool配置失败", zap.Any("err", err))
			}
		}
	}
}
