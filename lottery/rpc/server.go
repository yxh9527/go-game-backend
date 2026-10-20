package rpc

import (
	"app/esindex"
	"fmt"
	"lottery/dao"
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
	handler *LotteryService
}

func NewServer(address string, port int, handler *LotteryService) *Server {
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

	/*
		//If a client is idle for ? seconds, send a GOAWAY
		'grpc.max_connection_idle_ms': 15_000,
		//If any connection is alive for more than ? seconds, send a GOAWAY【Default：~(1 << 31)】
		//when the connection is sent by goaway, The client will receive a status with code `CANCELLED` and the server handler's call object will emit either a 'cancelled' event or an 'end' event.
		'grpc.max_connection_age_ms': 5_000,
		// Allow ? seconds for pending RPCs to complete before forcibly closing connections【Default：~(1 << 31).通常和`max_connection_age_ms`成对出现，此参数取决于该服务每次链接的处理时间】
		'grpc.max_connection_age_grace_ms': 3_000,
		// Ping the client every ? seconds to ensure the connection is still active【Recommendation ：1 minute (60000 ms)】
		'grpc.keepalive_time_ms': 60_000,
		// Wait ? second for the ping ack before assuming the connection is dead【Default：20 seconds (20000 ms)】
		'grpc.keepalive_timeout_ms': 20_000
	*/

	// 创建一个grpc Server服务对象,Handler非必传
	ss := grpc.NewServer(grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionAge:      2 * time.Minute,
		MaxConnectionAgeGrace: 30 * time.Second,
		Time:                  30 * time.Second,
		Timeout:               15 * time.Second,
	}))
	// 注册服务
	services.RegisterLotteryServiceServer(ss, s.handler)
	// 注册ETCD
	service, err := NewDefNamingService(rds, esindex.ServiceName("lottery"), s.address, int32(s.port))
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
