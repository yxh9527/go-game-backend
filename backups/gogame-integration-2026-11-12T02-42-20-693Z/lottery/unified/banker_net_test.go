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
	"time"
)

func TestBankerSettlementAtomicallyCreditsBettorsTransfersBankerNetAndReleasesOnlyItsReservation(t *testing.T) {
	for _, test := range []struct {
		name string
		net  int64
		bad  string
	}{
		{"banker loses", -200, ""}, {"banker wins", 500, ""}, {"zero draw", 0, ""}, {"bad last carry", -200, "revision"}, {"cap exceeded", -11000, "cap"}, {"missing bettor", -200, "missing"}, {"wrong game", -200, "game"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			mr := miniredis.RunT(t)
			client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
			defer client.Close()
			rd := dao.NewFinanceRedis(client)
			old := config.CfgIns
			defer func() { config.CfgIns = old }()
			config.CfgIns = &config.Configs{Lock: &sync.RWMutex{}, Currency: &config.CurrencyMgr{Data: map[string]decimal.Decimal{"CNY": decimal.NewFromInt(1)}}}
			binding := &services.FinanceCarryBinding{Token: "banker-43", TableKey: "7:baccarat:123", Revision: 5}
			round := &financeRound{RoundID: "banker-case", Agent: 7, GameID: 92058, Level: 1, Symbol: "test-bank", PoolSymbol: "test-bank_1", State: "RESERVED", Bets: map[string]*financeBetSnapshot{
				"bettor": {BetID: "bettor", UserID: 42, CurrencyType: "CNY", Amount: "10.00", Accepted: true, Status: "ACTIVE", AreaID: "1"},
				"banker": {BetID: "banker", UserID: 43, CurrencyType: "CNY", Amount: "0.00", Accepted: true, Status: "ACTIVE", AreaID: "banker:10000", Carry: binding},
			}}
			payout := int64(1200)
			if test.net > 0 {
				payout = 500
			}
			reserved := decimalFromCent(payout)
			if test.net > 0 {
				reserved = reserved.Add(decimalFromCent(test.net))
			}
			round.Reservation = &financeReservation{ReservationID: "this-reservation", TotalPayoutCNY: reserved.String(), OutcomeHash: "sealed-result", Status: "RESERVED", ExpiresAt: time.Now().Add(time.Minute).Unix()}
			runtime := &roundRuntime{AgentID: 7, GameID: 92058, Level: 1, Symbol: "test-bank", PoolSymbol: "test-bank_1", RoomType: 58}
			if test.bad == "game" {
				runtime.RoomType = 50
			}
			svc := &LotteryService{rds: rd, roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{round.RoundID: round}, testSkipPoolSideEffects: true,
				testResolveRuntime: func(uint32, uint32, uint32) (*roundRuntime, services.ErrorCode) { return runtime, 0 }}
			for _, uid := range []uint32{42, 43} {
				balance := int64(100000)
				if uid == 42 {
					balance = 99000
				}
				client.HSet(ctx, fmt.Sprintf("player_%d", uid), "id", uid, "agent_id", 7, "currency_type", "CNY", "currency", balance, "cash_revision", 3)
			}
			client.Set(ctx, "gogame:cash-carry:7:43:CNY", `{"token":"banker-43","tableKey":"7:baccarat:123","amount":20000,"revision":5}`, time.Minute)
			if test.bad == "revision" {
				client.Set(ctx, "gogame:cash-carry:7:43:CNY", `{"token":"banker-43","tableKey":"7:baccarat:123","amount":20000,"revision":6}`, time.Minute)
			}
			rd.SeedReservationLedgerActive(7, runtime.PoolSymbol, reserved, "this-reservation", round.RoundID)
			client.ZAdd(ctx, "agent_reserved_data", redis.Z{Member: "7_test-bank_1", Score: reserved.Add(decimal.NewFromInt(25)).InexactFloat64()})
			digest := svc.roundBetDigest(round)
			round.Reservation.BetDigest = digest
			if err := svc.saveRoundLocked(round); err != nil {
				t.Fatal(err)
			}
			net := decimalFromCent(test.net)
			req := &services.SettlementRequest{SettlementId: "stable-bank-settlement", RoundId: round.RoundID, GameId: 92058, Agent: 7, Level: 1, Mode: financeModeBankerNet, BetDigest: digest, ReservationId: "this-reservation", OutcomeHash: "sealed-result", Items: []*services.SettlementItem{
				{UserId: 42, CurrencyType: "CNY", BetAmount: "10.00", Payout: decimalFromCent(payout).StringFixed(2), Profit: decimalFromCent(payout - 1000).StringFixed(2), Pump: "0.00", Record: `{"bettor":true}`},
				{UserId: 43, CurrencyType: "CNY", BetAmount: net.Abs().StringFixed(2), Payout: net.Abs().Add(net).StringFixed(2), Profit: net.StringFixed(2), NetScore: net.StringFixed(2), Pump: "0.00", Carry: binding, Record: `{"banker":true}`},
			}}
			if test.bad == "missing" {
				req.Items = req.Items[1:]
			}
			beforeQ := client.ZScore(ctx, "agent_reserved_data", "7_test-bank_1").Val()
			if test.bad == "" {
				wrongMode := proto.Clone(req).(*services.SettlementRequest)
				wrongMode.Mode = financeModeNormal
				if result, _ := svc.Settlement(ctx, wrongMode); result.Code == 0 {
					t.Fatal("ordinary settlement bypassed registered banker liability")
				}
			}
			response, err := svc.Settlement(ctx, req)
			if test.bad != "" {
				if err != nil {
					t.Fatal(err)
				}
				if response.Code == 0 {
					t.Fatal("invalid batch accepted")
				}
				if client.HGet(ctx, "player_42", "currency").Val() != "99000" || client.HGet(ctx, "player_43", "currency").Val() != "100000" || client.ZScore(ctx, "agent_reserved_data", "7_test-bank_1").Val() != beforeQ {
					t.Fatal("rejected batch partially mutated funds or reservation")
				}
				return
			}
			if err != nil || response.Code != 0 || response.State != "SETTLED" {
				t.Fatal(response, err)
			}
			if client.HGet(ctx, "player_42", "currency").Val() != fmt.Sprint(99000+payout) || client.HGet(ctx, "player_43", "currency").Val() != fmt.Sprint(100000+test.net) {
				t.Fatal("incorrect atomic payout")
			}
			if client.ZScore(ctx, "agent_reserved_data", "7_test-bank_1").Val() != 25 {
				t.Fatal("settlement consumed another reservation")
			}
			ledger, _ := rd.GetReservationLedgerRaw("this-reservation")
			if dao.ParseReservationLedgerStatus(ledger) != "RELEASED" {
				t.Fatal("reservation not terminal")
			}
			for i := 0; i < 2; i++ {
				response, err = svc.Settlement(ctx, req)
				if err != nil || response.Code != 0 {
					t.Fatal("original retry failed", response, err)
				}
			}
			// Simulate a lost acknowledgment after Redis committed but before the
			// old in-memory round was replaced. The wallet marker must be read
			// before requiring an ACTIVE ledger that has already been released.
			svc.rounds[round.RoundID] = round
			response, err = svc.Settlement(ctx, req)
			if err != nil || response.Code != 0 {
				t.Fatal("lost-ack original request could not confirm committed money", response, err)
			}
			if client.HGet(ctx, "player_43", "cash_revision").Val() != "4" || client.HGet(ctx, "player_42", "cash_revision").Val() != "4" {
				t.Fatal("repeat settlement changed money twice")
			}
			changed := proto.Clone(req).(*services.SettlementRequest)
			changed.Items[0].Record = "changed"
			response, _ = svc.Settlement(ctx, changed)
			if response.Code == 0 {
				t.Fatal("changed original settlement accepted")
			}
		})
	}
}
