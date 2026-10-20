package dao

import (
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

const (
	nameServicePrefix = "grpc/registry"
	defLeaseSecond    = 15
)

type Endpoint struct {
	Name string `json:"name"`
	Addr string `json:"addr"`
	Port int    `json:"port"`
	Id   int64  `json:"id"`
}

type NamingService struct {
	Name   string
	Client *RedisDao
	Id     int64
}

func NewDefNamingService(client *RedisDao, serviceName, ip string, port int32) (*NamingService, error) {
	id := time.Now().Unix()
	key := fmt.Sprintf("/%s/%s/%s-%d", nameServicePrefix, serviceName, serviceName, id)
	ns := &NamingService{
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
	if err := client.Set(key, str, defLeaseSecond*3); err != nil {
		zap.L().Error("注册服务失败", zap.Error(err), zap.String("key", key))
		return nil, err
	}
	zap.L().Info("client-api 已注册", zap.String("key", key), zap.String("addr", ip), zap.Int32("port", port))
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				zap.L().Error("panic", zap.Any("err", rec))
			}
		}()
		ticker := time.NewTicker(time.Duration(defLeaseSecond) * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			ok, err := client.SetKeyTimeOut(key, defLeaseSecond*6)
			if err != nil {
				zap.L().Error("client-api 续约失败", zap.Error(err))
				continue
			}
			if !ok {
				zap.L().Error("client-api 续约失败", zap.Bool("ok", ok))
			}
		}
	}()
	return ns, nil
}

func (ns *NamingService) ClearRegistryInfo() {
	if ns == nil || ns.Client == nil {
		return
	}
	key := fmt.Sprintf("/%s/%s/%s-%d", nameServicePrefix, ns.Name, ns.Name, ns.Id)
	if err := ns.Client.Del(key); err != nil {
		zap.L().Error("清除服务注册失败", zap.Error(err), zap.String("key", key))
		return
	}
	zap.L().Info("client-api 已注销", zap.String("key", key))
}
