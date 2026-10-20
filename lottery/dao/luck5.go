package dao

import (
	"app/esindex"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"micro_service/services"
)

// 发牌时固定资金身份、原池与汇率，结算不再按当前配置选择水池。
type Luck5Round struct {
	UserId       uint32          `json:"userId"`
	AgentId      uint32          `json:"agentId"`
	GameId       uint32          `json:"gameId"`
	CurrencyType string          `json:"currencyType"`
	RoundId      string          `json:"roundId"`
	BetCent      int64           `json:"betCent"`
	ReserveCent  int64           `json:"reserveCent"`
	Symbol       string          `json:"symbol"`
	Exchange     decimal.Decimal `json:"exchange"`
	Account      string          `json:"account"`
	WebId        uint32          `json:"webId"`
	GameName     string          `json:"gameName"`
	IsTourist    int32           `json:"isTourist"`
	DealBalance  int64           `json:"dealBalance"`
	Signature    string          `json:"signature"`
	WinCent      int64           `json:"winCent"`
	RecordJson   string          `json:"recordJson"`
	Balance      int64           `json:"balance"`
	FundsDone    bool            `json:"fundsDone"`
	RecordDone   bool            `json:"recordDone"`
}

// 局号全局唯一，不能让相同局号在另一个用户、代理或币种下再次扣注。
func luck5Key(roundId string) string {
	digest := sha256.Sum256([]byte(roundId))
	return "luck5:round:" + hex.EncodeToString(digest[:])
}

// 钱包使用整数分；既有水池和统计按配置汇率换算到池单位。
func luck5Amount(cent int64) decimal.Decimal {
	return decimal.NewFromInt(cent).Shift(-2)
}

// 在代理锁内读取持久化阶段，网络结果未知时也能按原局号确认。
func (rd *RedisDao) loadLuck5Round(ctx context.Context, roundId string) (*Luck5Round, error) {
	raw, err := rd.cli.Get(ctx, luck5Key(roundId)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	round := &Luck5Round{}
	if err := jsoniter.UnmarshalFromString(raw, round); err != nil {
		return nil, err
	}
	return round, nil
}

// 身份核对不能只看用户与局号，否则原池预扣可能被另一代理或币种使用。
func sameLuck5Identity(round, request *Luck5Round) bool {
	return round.UserId == request.UserId && round.AgentId == request.AgentId &&
		round.GameId == request.GameId && round.CurrencyType == request.CurrencyType && round.RoundId == request.RoundId
}

// 重复发牌先返回原扣注结果，配置或玩家缓存变化不影响已成功的局。
func (rd *RedisDao) Luck5DealResult(ctx context.Context, request *Luck5Round) (*services.Luck5Resp, error) {
	round, err := rd.loadLuck5Round(ctx, request.RoundId)
	if err != nil || round == nil {
		return nil, err
	}
	if !sameLuck5Identity(round, request) || round.BetCent != request.BetCent || round.ReserveCent != request.ReserveCent {
		return &services.Luck5Resp{Code: services.ErrorCode_PARAMS_INVALID, Error: "dealConflict"}, nil
	}
	return &services.Luck5Resp{Code: services.ErrorCode_OK, Ret: true, CurrencyCent: round.DealBalance}, nil
}

// 资金事务同时提交钱包、原池统计和阶段结果；只在成功确认后推进内存缓存。
func (rd *RedisDao) commitLuck5Funds(ctx context.Context, agent *AgentData, round *Luck5Round, settle bool) (services.ErrorCode, error) {
	playerKey := "player_" + strconv.FormatUint(uint64(round.UserId), 10)
	roundKey := luck5Key(round.RoundId)
	game := agent.GetGame(round.Symbol)
	user := agent.GetUser(round.UserId)
	nextGame := *game
	nextUser := *user
	bet := luck5Amount(round.BetCent).Mul(round.Exchange)
	win := luck5Amount(round.WinCent).Mul(round.Exchange)
	reserve := luck5Amount(round.ReserveCent).Mul(round.Exchange)
	if round.IsTourist == 0 {
		if settle {
			nextGame.TotalProfLoss = nextGame.TotalProfLoss.Sub(reserve.Sub(win))
			chips := decimal.Max(bet, win)
			nextGame.TotalChips = nextGame.TotalChips.Add(chips)
			nextUser.TotalProfLoss = nextUser.TotalProfLoss.Add(win)
			nextUser.Count = nextUser.Count.Add(decimal.NewFromInt(1))
		} else {
			nextGame.TotalEffectBet = nextGame.TotalEffectBet.Add(bet)
			nextGame.TotalProfLoss = nextGame.TotalProfLoss.Add(reserve)
			nextUser.TotalEffectBet = nextUser.TotalEffectBet.Add(bet)
		}
	}
	code := services.ErrorCode_SYSTEM_ERROR
	storedRaw, readErr := rd.cli.Get(ctx, roundKey).Result()
	if readErr != nil && !errors.Is(readErr, redis.Nil) {
		return code, readErr
	}
	if settle {
		stored := &Luck5Round{}
		if readErr != nil || jsoniter.UnmarshalFromString(storedRaw, stored) != nil || stored.FundsDone || !sameLuck5Identity(stored, round) {
			return code, errors.New("round stage changed")
		}
	} else if readErr == nil {
		return code, errors.New("round already exists")
	}
	current, err := rd.cli.HGet(ctx, playerKey, "currency").Int64()
	if err != nil {
		return code, err
	}
	if current < 0 {
		return code, errors.New("invalid player balance")
	}
	if settle {
		if round.WinCent > math.MaxInt64-current {
			code = services.ErrorCode_PARAMS_INVALID
			return code, errors.New("balance overflow")
		}
		round.Balance = current + round.WinCent
		round.FundsDone = true
	} else {
		if current < round.BetCent {
			code = services.ErrorCode_NO_ENOUGH_MONEY
			return code, errors.New("balanceInsufficient")
		}
		round.DealBalance = current - round.BetCent
		round.Balance = round.DealBalance
	}
	raw, err := jsoniter.MarshalToString(round)
	if err != nil {
		return code, err
	}
	_, err = rd.cli.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.HSet(ctx, playerKey, "currency", round.Balance)
		pipe.Expire(ctx, playerKey, 20*time.Minute)
		pipe.SAdd(ctx, "dirty_list_imp", round.UserId)
		pipe.Set(ctx, roundKey, raw, 0)
		if round.IsTourist == 0 {
			member := fmt.Sprintf("%d_%s", agent.Id, round.Symbol)
			userMember := strconv.FormatUint(uint64(round.UserId), 10)
			if settle {
				pipe.ZIncrBy(ctx, esindex.AgentProfitLossData(), win.Sub(reserve).InexactFloat64(), member)
				pipe.ZIncrBy(ctx, esindex.AgentChipsData(), decimal.Max(bet, win).InexactFloat64(), member)
				pipe.ZIncrBy(ctx, "userTotalProfLoss", win.InexactFloat64(), userMember)
				pipe.ZIncrBy(ctx, "userBetCount", 1, userMember)
			} else {
				pipe.ZIncrBy(ctx, esindex.AgentEffectData(), bet.InexactFloat64(), member)
				pipe.ZIncrBy(ctx, esindex.AgentProfitLossData(), reserve.InexactFloat64(), member)
				pipe.ZIncrBy(ctx, "userTotalEffBet", bet.InexactFloat64(), userMember)
			}
		}
		return nil
	})
	if err == nil {
		code = services.ErrorCode_OK
	}
	if err != nil {
		// EXEC 回包丢失时确认持久化阶段；不盲目再扣钱包或恢复旧池快照。
		stored, readErr := rd.loadLuck5Round(context.Background(), round.RoundId)
		if readErr != nil || stored == nil || !sameLuck5Identity(stored, round) ||
			stored.BetCent != round.BetCent || stored.ReserveCent != round.ReserveCent ||
			(settle && (!stored.FundsDone || stored.Signature != round.Signature)) || (!settle && stored.FundsDone) {
			delete(agent.gameCache, round.Symbol)
			delete(agent.userCache, round.UserId)
			return code, err
		}
		*round = *stored
	}
	nextGame.UpdateTime = time.Now().Unix()
	nextUser.UpdateTime = nextGame.UpdateTime
	*game = nextGame
	*user = nextUser
	return services.ErrorCode_OK, nil
}

// 发牌的余额检查、池检查和资金更新都在既有代理锁内完成。
func (gcm *GameCacheMgr) Luck5Deal(ctx context.Context, request *Luck5Round) (*services.Luck5Resp, error) {
	agent := gcm.GetAgent(int64(request.AgentId))
	agent.lock.Lock()
	defer agent.lock.Unlock()
	if response, err := RedisIns().Luck5DealResult(ctx, request); err != nil || response != nil {
		return response, err
	}
	balance, err := RedisIns().GetPlayerCurrency(request.UserId)
	if err != nil {
		return nil, err
	}
	if balance < request.BetCent {
		return &services.Luck5Resp{Code: services.ErrorCode_NO_ENOUGH_MONEY, CurrencyCent: balance, Error: "balanceInsufficient"}, nil
	}
	game := agent.GetGame(request.Symbol)
	user := agent.GetUser(request.UserId)
	request.Account = user.Account
	request.IsTourist = user.IsTourist
	bet := luck5Amount(request.BetCent).Mul(request.Exchange)
	reserve := luck5Amount(request.ReserveCent).Mul(request.Exchange)
	if user.IsTourist == 0 && poolWithBase(game, agent.Id, request.Symbol).Add(bet).LessThan(reserve) {
		return &services.Luck5Resp{Code: services.ErrorCode_NO_ENOUGH_POOL_MONEY, CurrencyCent: balance, Error: "poolInsufficient"}, nil
	}
	if code, err := RedisIns().commitLuck5Funds(ctx, agent, request, false); err != nil {
		return &services.Luck5Resp{Code: code, CurrencyCent: balance, Error: "dealFundsFailed"}, err
	}
	return &services.Luck5Resp{Code: services.ErrorCode_OK, Ret: true, CurrencyCent: request.DealBalance}, nil
}

// 结算只动已预扣资金；资金阶段已提交时重试只保存原记录。
func (gcm *GameCacheMgr) Luck5Settle(ctx context.Context, request *Luck5Round, saveRecord func(*Luck5Round) error) (*services.Luck5Resp, error) {
	agent := gcm.GetAgent(int64(request.AgentId))
	agent.lock.Lock()
	defer agent.lock.Unlock()
	round, err := RedisIns().loadLuck5Round(ctx, request.RoundId)
	if err != nil {
		return nil, err
	}
	if round == nil || !sameLuck5Identity(round, request) {
		return &services.Luck5Resp{Code: services.ErrorCode_PARAMS_INVALID, Error: "roundIdentityMismatch"}, nil
	}
	if request.WinCent < 0 || request.WinCent > round.ReserveCent {
		return &services.Luck5Resp{Code: services.ErrorCode_PARAMS_INVALID, Error: "winExceedsReserve"}, nil
	}
	if round.FundsDone && round.Signature != request.Signature {
		return &services.Luck5Resp{Code: services.ErrorCode_PARAMS_INVALID, Error: "settleConflict"}, nil
	}
	if round.RecordDone {
		return &services.Luck5Resp{Code: services.ErrorCode_OK, Ret: true, CurrencyCent: round.Balance}, nil
	}
	if !round.FundsDone {
		round.WinCent = request.WinCent
		round.RecordJson = request.RecordJson
		round.Signature = request.Signature
		if code, err := RedisIns().commitLuck5Funds(ctx, agent, round, true); err != nil {
			return &services.Luck5Resp{Code: code, Error: "settleFundsFailed"}, err
		}
	}
	if err := saveRecord(round); err != nil {
		return &services.Luck5Resp{Code: services.ErrorCode_SYSTEM_ERROR, CurrencyCent: round.Balance, Error: "recordSaveFailed"}, err
	}
	round.RecordDone = true
	raw, err := jsoniter.MarshalToString(round)
	if err == nil {
		err = RedisIns().cli.Set(ctx, luck5Key(round.RoundId), raw, 0).Err()
	}
	if err != nil {
		return &services.Luck5Resp{Code: services.ErrorCode_SYSTEM_ERROR, CurrencyCent: round.Balance, Error: "completionSaveFailed"}, err
	}
	return &services.Luck5Resp{Code: services.ErrorCode_OK, Ret: true, CurrencyCent: round.Balance}, nil
}
