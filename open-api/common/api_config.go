package common

import (
	"app/tables/manager"
	"open-api/dao"
	"sync"
	"time"

	"go.uber.org/zap"
)

type ApiConfigMgr struct {
	l sync.RWMutex
	d map[int64]*manager.ApiConfig
}

var apiConfigs *ApiConfigMgr = nil

func InitApiConfigMgr() {
	if apiConfigs == nil {
		apiConfigs = &ApiConfigMgr{
			l: sync.RWMutex{},
			d: make(map[int64]*manager.ApiConfig),
		}
		apiConfigs.load()
		go func() {
			if err := recover(); err != nil {
				zap.L().Error("panic", zap.Any("动态域名定时更新异常", err))
			}
			for {
				time.Sleep(time.Second * 300)
				apiConfigs.load()
			}
		}()
	}
}

func ApiConfigs() *ApiConfigMgr {
	return apiConfigs
}

func (a *ApiConfigMgr) load() {
	apiConfigs := dao.Mysql().LoadApiConfigs()
	if apiConfigs == nil {
		zap.L().Error("加载动态域名配置失败")
		return
	}

	a.l.Lock()
	defer a.l.Unlock()

	for _, v := range apiConfigs {
		a.d[v.Id] = v
	}
}

func (a *ApiConfigMgr) Get(effectBet int) *manager.ApiConfig {
	a.l.RLock()
	defer a.l.RUnlock()
	maxId := 0
	for k, v := range a.d {
		maxId = int(k)
		if v.Min <= effectBet && v.Max > effectBet {
			return v
		}
	}
	return a.d[int64(maxId)]
}
