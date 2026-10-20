package rpc

import (
	"app/esindex"
	"data-center/dao"
	"fmt"
	"micro_service/services"
	"net"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"
)

type Server struct {
	address string // grpc服务自身的地址
	port    int    // grpc服务监听的端口
	handler *DataCenterService
}

func NewServer(address string, port int, handler *DataCenterService) *Server {
	return &Server{
		address: address,
		port:    port,
		handler: handler,
	}
}

func (s *Server) Serve(rds *dao.RedisDao, db *dao.DBDao) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		zap.L().Fatal("failed to listen", zap.Any("err", err))
	}
	// 创建一个grpc Server服务对象,Handler非必传
	ss := grpc.NewServer(grpc.KeepaliveParams(
		keepalive.ServerParameters{
			// MaxConnectionAge:      2 * time.Minute,
			// MaxConnectionAgeGrace: 30 * time.Second,
			Time:    60 * time.Second,
			Timeout: 30 * time.Second,
		},
	), grpc.MaxConcurrentStreams(1000), // 设置每个连接的最大并发流数
		grpc.MaxRecvMsgSize(1024*1024*100),
		grpc.MaxSendMsgSize(1024*1024*100),
		grpc.ReadBufferSize(1024*1024),
		grpc.WriteBufferSize(1024*1024))
	// 注册服务
	services.RegisterDataCenterServiceServer(ss, s.handler)
	// 注册ETCD
	service, err := NewDefNamingService(rds, esindex.ServiceName("datacenter"), s.address, int32(s.port))
	if err != nil {
		zap.L().Fatal("failed to create NamingService", zap.Any("err", err))
	}
	// 启动RPC并监听
	zap.L().Debug("server listening", zap.Any("addr", lis.Addr()))
	go func() {
		if err := ss.Serve(lis); err != nil {
			service.ClearRegistryInfo()
			zap.L().Fatal("failed to serve", zap.Any("err", err))
		}
	}()
}

func (s *Server) Stop() {
	zap.L().Info("服务器关闭完成")
}
