package unified

import (
	"app/config"
	"bufio"
	"context"
	datadao "data-center/dao"
	"encoding/json"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"lottery/dao"
	"micro_service/services"
	"net"
	"os"
	"sync"
	"testing"
)

type scratchFixtureLottery struct{ *LotteryService }

func (s scratchFixtureLottery) GetBalance(_ context.Context, req *services.GetBalanceReq) (*services.GetBalanceResp, error) {
	balance, code := s.getPlayerCurrency(req.UserId)
	return &services.GetBalanceResp{Code: code, Currency: decimalFromCent(balance).StringFixed(2), CurrencyCent: balance}, nil
}

type scratchFixtureStorage struct {
	services.UnimplementedGogameStorageServiceServer
	client *redis.Client
}

func (s scratchFixtureStorage) Call(ctx context.Context, req *services.GogameStorageRequest) (*services.GogameStorageResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if values := md.Get("x-gogame-storage-token"); len(values) != 1 || values[0] != "scratch-fixture-token" {
		return nil, status.Error(codes.PermissionDenied, "invalid fixture token")
	}
	if req.Operation != "cashRead" {
		return nil, status.Error(codes.InvalidArgument, "fixture only provides cash reads")
	}
	var account datadao.CashAccount
	if json.Unmarshal([]byte(req.Payload), &account) != nil {
		return nil, status.Error(codes.InvalidArgument, "bad account")
	}
	raw, err := datadao.CashRead(ctx, s.client, account)
	if err != nil {
		return nil, err
	}
	return &services.GogameStorageResponse{Phase: "result", Payload: string(raw)}, nil
}
func TestScratchRPCFixture(t *testing.T) {
	if os.Getenv("GOGAME_SCRATCH_RPC_FIXTURE") != "1" {
		t.Skip("standalone Node/Go integration fixture")
	}
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()
	config.CfgIns = &config.Configs{Lock: &sync.RWMutex{}, Currency: &config.CurrencyMgr{Data: map[string]decimal.Decimal{"CNY": decimal.NewFromInt(1)}}, Pool: &config.PoolMgr{Default: map[string]*config.Pool{}, Agent: map[string]*config.Pool{}}}
	for _, kind := range []string{"mj", "queen"} {
		config.CfgIns.Pool.Default["fixture-"+kind] = &config.Pool{Symbol: "fixture-" + kind, Pool: map[int32]*config.PoolItem{1: {Base: decimal.NewFromInt(100000000)}}}
	}
	sold := map[string]bool{}
	svc := &LotteryService{rds: dao.NewFinanceRedis(client), roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{}, testSkipPoolSideEffects: true, testScratchMirror: func(r *financeRound) error {
		sold[fmt.Sprintf("%d:%d:%s", r.Agent, r.GameID, r.Scratch.Serial)] = true
		return nil
	}}
	svc.testScratchSold = func(_ context.Context, scope dao.ScratchScope, serials []string) (map[string]bool, error) {
		result := map[string]bool{}
		for _, serial := range serials {
			result[serial] = sold[scope.Market()+":"+serial]
		}
		return result, nil
	}
	svc.testResolveRuntime = func(agent, game, level uint32) (*roundRuntime, services.ErrorCode) {
		if agent != 7 || level != 1 || (game != 91061 && game != 91063) {
			return nil, services.ErrorCode_PARAMS_INVALID
		}
		room := uint32(61)
		symbol := "fixture-mj"
		if game == 91063 {
			room = 63
			symbol = "fixture-queen"
		}
		return &roundRuntime{AgentID: agent, GameID: game, Level: level, RoomType: room, Symbol: symbol, PoolSymbol: symbol + "_1"}, services.ErrorCode_OK
	}
	for _, uid := range []uint32{42, 43} {
		client.HSet(context.Background(), fmt.Sprintf("player_%d", uid), "id", uid, "agent_id", 7, "currency_type", "CNY", "currency", 1000000, "cash_revision", 0)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	services.RegisterLotteryServiceServer(server, scratchFixtureLottery{svc})
	services.RegisterGogameStorageServiceServer(server, scratchFixtureStorage{client: client})
	go server.Serve(listener)
	defer server.Stop()
	fmt.Println("SCRATCH_TEST_RPC " + listener.Addr().String())
	bufio.NewReader(os.Stdin).ReadString('\n')
}
