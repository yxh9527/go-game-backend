package unified

import (
	"app/config"
	"context"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"
	"lottery/dao"
	"micro_service/services"
	"sync"
	"testing"
)

func TestSeatNetServiceUsesAbsoluteProfitForStatisticsWithoutDebitingItAgain(t *testing.T) {
	mr := miniredis.RunT(t)
	dao.NewRedisDao([]string{mr.Addr()}, "", "")
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()
	savedCfg := config.CfgIns
	defer func() { config.CfgIns = savedCfg }()
	config.CfgIns = &config.Configs{Lock: &sync.RWMutex{}, Currency: &config.CurrencyMgr{Data: map[string]decimal.Decimal{"CNY": decimal.NewFromInt(1)}},
		Pool: &config.PoolMgr{Default: map[string]*config.Pool{"test-net": {Symbol: "test-net", GameId: 92024, Pool: map[int32]*config.PoolItem{1: {}}}}, Agent: map[string]*config.Pool{}}}
	runtime := &roundRuntime{AgentID: 7, GameID: 92024, Level: 1, Symbol: "test-net", PoolSymbol: "test-net_1"}
	for _, testCase := range []struct {
		name string
		nets []int64
		bad  bool
	}{
		{"simple", []int64{-123, 123}, false}, {"winner profit exceeds own opening carry", []int64{20000, -10000, -10000}, false},
		{"last loser insufficient", []int64{15000, -15000}, true}, {"zero draw", []int64{0, 0}, false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			roundID := "net-" + testCase.name
			round := &financeRound{RoundID: roundID, Agent: 7, GameID: 92024, Level: 1, Symbol: "test-net", PoolSymbol: "test-net_1", State: "BETTING", Bets: map[string]*financeBetSnapshot{}}
			req := &services.SettlementRequest{RoundId: roundID, SettlementId: roundID + "-settled", GameId: 92024, Agent: 7, Level: 1, Mode: financeModeSeatNet, OutcomeHash: "actual-complete-result"}
			for i, net := range testCase.nets {
				uid := uint32(42 + i)
				key := fmt.Sprintf("player_%d", uid)
				if err := client.HSet(context.Background(), key, "id", uid, "agent_id", 7, "currency_type", "CNY", "currency", 100000, "cash_revision", 3).Err(); err != nil {
					t.Fatal(err)
				}
				token := fmt.Sprintf("seat-%d", uid)
				carryKey := fmt.Sprintf("gogame:cash-carry:7:%d:CNY", uid)
				client.Set(context.Background(), carryKey, fmt.Sprintf(`{"token":"%s","tableKey":"7:niuniu:1","amount":10000,"revision":5}`, token), 0)
				carry := &services.FinanceCarryBinding{Token: token, TableKey: "7:niuniu:1", Revision: 5}
				betID := fmt.Sprintf("%s:rev:0:%d", roundID, uid)
				round.Bets[betID] = &financeBetSnapshot{BetID: betID, UserID: uid, CurrencyType: "CNY", Amount: "0", Accepted: true, Status: "ACTIVE", Carry: carry}
				value := decimalFromCent(net)
				effective := value.Abs()
				req.Items = append(req.Items, &services.SettlementItem{UserId: uid, CurrencyType: "CNY", BetAmount: effective.StringFixed(2), Profit: value.StringFixed(2), Payout: effective.Add(value).StringFixed(2), NetScore: value.StringFixed(2), Pump: "0.00", Record: `{"actual":true}`, Carry: carry})
			}
			svc := &LotteryService{rds: dao.RedisIns(), roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{roundID: round}, testSkipPoolSideEffects: true}
			if err := svc.saveRoundLocked(round); err != nil {
				t.Fatal(err)
			}
			req.BetDigest = svc.roundBetDigest(round)
			resp := svc.settleSeatNetLocked(round, runtime, req)
			if testCase.bad {
				if resp.Code == services.ErrorCode_OK {
					t.Fatal("insufficient last payer accepted")
				}
				for i := range testCase.nets {
					balance, _ := client.HGet(context.Background(), fmt.Sprintf("player_%d", 42+i), "currency").Int64()
					if balance != 100000 {
						t.Fatal("a preceding player was mutated")
					}
				}
				return
			}
			if resp.Code != services.ErrorCode_OK || resp.State != "SETTLED" {
				t.Fatalf("settlement=%+v", resp)
			}
			repeat := svc.settleSeatNetLocked(svc.rounds[roundID], runtime, req)
			if repeat.Code != services.ErrorCode_OK {
				t.Fatal("exact retry failed")
			}
			for i, net := range testCase.nets {
				balance, _ := client.HGet(context.Background(), fmt.Sprintf("player_%d", 42+i), "currency").Int64()
				if balance != 100000+net {
					t.Fatalf("user %d balance=%d expected=%d", 42+i, balance, 100000+net)
				}
				revision, _ := client.HGet(context.Background(), fmt.Sprintf("player_%d", 42+i), "cash_revision").Int64()
				if revision != 4 {
					t.Fatal("net transfer was repeated")
				}
			}
			changed := proto.Clone(req).(*services.SettlementRequest)
			changed.Items[0].Record = `{"changed":true}`
			if svc.settleSeatNetLocked(svc.rounds[roundID], runtime, changed).Code == services.ErrorCode_OK {
				t.Fatal("same settlement ID accepted changed content")
			}
		})
	}
}
