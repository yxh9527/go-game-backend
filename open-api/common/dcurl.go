package common

import (
	"math/rand"
	"sync"
	"time"

	"go.uber.org/zap"
)


type DataCenterRpcUrlMgr struct {
	l    *sync.RWMutex
	urls []string
}

var dcrum *DataCenterRpcUrlMgr

func GetDataCenterRpcUrl() string {
	dcrum.l.RLock()
	defer dcrum.l.RUnlock()
	n := len(dcrum.urls)
	if n <= 0 {
		zap.L().Error("data-center 启动异常")
		return ""
	}
	return dcrum.urls[rand.Intn(n)]
}

func load() {
	// resp, err := dao.Redis().Get("/grpc/registry/pp-datacenter", clientv3.WithPrefix())
	// if err != nil {
	// 	zap.L().Error("刷新 data-center url 失败", zap.Error(err))
	// 	return
	// }
	// var str []string
	// for _, v := range resp.Kvs {
	// 	cfg := &MicroRegistry{}
	// 	if err := json.Unmarshal(v.Value, cfg); err != nil {
	// 		zap.L().Error("反序列化proxy配置失败", zap.Error(err))
	// 	} else {
	// 		str = append(str, "http://"+cfg.Nodes[0].Metadata.HttpAddr+"/update_game_currency")
	// 	}
	// }
	// if len(str) <= 0 {
	// 	zap.L().Error("刷新 data-center url 失败")
	// 	return
	// }
	// dcrum.l.Lock()
	// dcrum.urls = str
	// dcrum.l.Unlock()
}

func DataCenterUrlInit() {
	if dcrum == nil {
		dcrum = &DataCenterRpcUrlMgr{
			l:    &sync.RWMutex{},
			urls: make([]string, 0, 20),
		}
	}
	gw := &sync.WaitGroup{}
	gw.Add(1)
	go func() {
		gw.Done()
		defer func() {
			if err := recover(); err != nil {
				zap.L().Error("panic", zap.Any("err", err))
			}
		}()
		load()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			load()
		}
	}()
	gw.Wait()
}
