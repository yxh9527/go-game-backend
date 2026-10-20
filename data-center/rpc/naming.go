package rpc

import (
	"data-center/dao"
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

const (
	nameServicePrefix = "grpc/registry"
	// 默认的租赁时间
	defLeaseSecond = 15
)

// 服务元数据信息
type Endpoint struct {
	Name string `json:"name"`
	Addr string `json:"addr"`
	Port int    `json:"port"`
	Id   int64  `json:"id"`
}

// 命名服务结构体
type NamingService struct {
	Name   string
	Client *dao.RedisDao
	Id     int64
}

func NewDefNamingService(client *dao.RedisDao, serviceName, ip string, port int32) (*NamingService, error) {
	id := time.Now().Unix()
	key := fmt.Sprintf("/%s/%s/%s-%d", nameServicePrefix, serviceName, serviceName, id)
	ns := NamingService{
		Name:   serviceName,
		Client: client,
		Id:     id,
	}
	endPoint := &Endpoint{
		Name: serviceName,
		Addr: ip,
		Port: int(port),
		Id:   id,
	}
	str, _ := jsoniter.MarshalToString(endPoint)
	err := client.Set(key, str, defLeaseSecond*3)
	if err != nil {
		zap.L().Fatal("注册服务失败", zap.Any("err", err))
	}
	go func() {
		defer func() {
			if err := recover(); err != nil {
				zap.L().Error("panic", zap.Any("err", err))
			}
		}()
		ticker := time.NewTicker((defLeaseSecond) * time.Second)
		for range ticker.C {
			if b, err := client.SetKeyTimeOut(key, defLeaseSecond*6); err != nil {
				zap.L().Error("data-center 续约失败", zap.Any("err", err))
			} else {
				if !b {
					zap.L().Error("data-center 续约失败", zap.Any("b", b))
				}
			}
		}
	}()
	return &ns, nil
}

func (ns *NamingService) ClearRegistryInfo() {
	key := fmt.Sprintf("/%s/%s/%s-%d", nameServicePrefix, ns.Name, ns.Name, ns.Id)
	ns.Client.Del(key)
}
