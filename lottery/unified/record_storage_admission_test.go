package unified

import (
	"context"
	"micro_service/services"
	"sync"
	"testing"
)

func TestNewUnifiedWagerStopsBeforeCreatingRoundWhenESIsUnconfigured(t *testing.T) {
	service := &LotteryService{roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{},
		testResolveRuntime: func(uint32, uint32, uint32) (*roundRuntime, services.ErrorCode) {
			return &roundRuntime{AgentID: 7, GameID: 92058, Level: 1, Symbol: "actual", PoolSymbol: "actual_1", RoomType: 58}, services.ErrorCode_OK
		}}
	response, err := service.Bet(context.Background(), &services.BetRequest{RequestId: "new-request", RoundId: "new-round", Agent: 7, GameId: 92058, Level: 1, Items: []*services.BetItem{{BetId: "new-bet", UserId: 42, CurrencyType: "CNY", Amount: "1.00"}}})
	if err != nil || response.Code != services.ErrorCode_SYSTEM_ERROR || len(service.rounds) != 0 {
		t.Fatal(response, err, service.rounds)
	}
}
