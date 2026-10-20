package dao

import (
	"app/config"
	"app/esindex"
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	SYSTEM_CTRL    = 1 //系统控制
	AUTO_CTRL      = 2 //自动控制
	DefaultRevenue = 0.03
)

var gameCache *GameCacheMgr = nil
var singleCtrl *SingleCtrlMgr = nil

// 玩家数据对象
type User struct {
	TotalEffectBet decimal.Decimal //总有效投注
	TotalProfLoss  decimal.Decimal //玩家总返奖累计（完局直接加 award，award 不会为负）
	Count          decimal.Decimal //下注局数
	UpdateTime     int64           //更新时间
	UserId         uint32          //玩家id
	AgentId        int64           //代理id
	Account        string          //平台账号
	IsTourist      int32           //是否是游客账号 0 否  1 是
}

// 游戏数据对象
type Game struct {
	TotalEffectBet decimal.Decimal //总有效投注
	TotalChips     decimal.Decimal //总打码    （下注金额||返奖金额）绝对值，谁大取谁 的总和
	TotalProfLoss  decimal.Decimal //代理赔付累计（仅 award>0 时加 award，含预占）
	TotalRevenue   decimal.Decimal //总税收
	UpdateTime     int64           //更新时间
	AgentId        int64           //代理id
	Symbol         string          //游戏标识
}

// addGamePayout FG：仅 award>0 计入代理游戏总赔付。
func addGamePayout(game *Game, award decimal.Decimal) {
	if game == nil {
		return
	}
	if award.GreaterThan(decimal.Zero) {
		game.TotalProfLoss = game.TotalProfLoss.Add(award.Truncate(4))
	}
}

// addUserAward 玩家总返奖：award 不会为负，完局直接累加。
func addUserAward(user *User, award decimal.Decimal) {
	if user == nil {
		return
	}
	user.TotalProfLoss = user.TotalProfLoss.Add(award.Truncate(4))
}

// positiveAward 水池预判只用正赔付，负 award（玩家亏损）不占赔付额度。
func positiveAward(award decimal.Decimal) decimal.Decimal {
	if award.GreaterThan(decimal.Zero) {
		return award
	}
	return decimal.Zero
}

type Ctrl struct {
	Rate       decimal.Decimal `json:"rate"`
	UserId     uint32          `json:"userId"`
	RateScore  decimal.Decimal `json:"rate_score"`
	LetScore   decimal.Decimal `json:"left_score"`
	TYPE       int             `json:"type"`
	IsDel      bool            `json:"-"` //是否删除
	UpdateTime int64           `json:"-"`
}

type RoundItem struct {
	MaxPay  decimal.Decimal `json:"maxPay"`  //最大赔付
	Over    bool            `json:"over"`    //是否完成
	AgentId int64           `json:"agentId"` //所属代理
	RoundId string          `json:"roundId"` //注单id
	Saved   bool            `json:"saved"`   //是否保存
}

type AgentData struct {
	lock       *sync.RWMutex
	Id         int64
	WebId      int64
	gameCache  map[string]*Game      //key:{symbol}
	userCache  map[uint32]*User      //
	RoundCache map[string]*RoundItem //key:注单号 value:最大赔付值
}

func (ad *AgentData) GetUser(userId uint32) *User {
	u := ad.userCache[userId]
	if u == nil {
		e, p, c, acc, isTourist := RedisIns().GetUserStatData(userId)
		u = &User{
			TotalEffectBet: e,
			TotalProfLoss:  p,
			UserId:         userId,
			AgentId:        ad.Id,
			Count:          c,
			Account:        acc,
			IsTourist:      isTourist,
		}
		ad.SetUser(userId, u)
	}
	return u
}

func (ad *AgentData) SetUser(id uint32, u *User) {
	u.UserId = id
	u.AgentId = ad.Id
	ad.userCache[id] = u
}

func (ad *AgentData) GetGame(symbol string) *Game {
	g := ad.gameCache[symbol]
	if g == nil {
		e, c, p, r, _ := RedisIns().GetGameStatData(ad.Id, symbol)
		g = &Game{
			TotalEffectBet: e,
			TotalChips:     c,
			TotalProfLoss:  p,
			TotalRevenue:   r,
			AgentId:        ad.Id,
		}
		ad.SetGame(symbol, g)
	}
	return g
}

func (ad *AgentData) SetGame(symbol string, g *Game) {
	g.AgentId = ad.Id
	g.Symbol = symbol
	ad.gameCache[symbol] = g
}

type SingleCtrlMgr struct {
	lock *sync.RWMutex
	r    *rand.Rand
	sc   map[uint32]*Ctrl //单控列表
}

func (s *SingleCtrlMgr) getSingleCtrl(uid uint32) *Ctrl {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.sc[uid]
}

func (s *SingleCtrlMgr) setSingleCtrlScore(uid uint32, delta decimal.Decimal) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if ctl := s.sc[uid]; ctl != nil {
		ctl.LetScore = ctl.LetScore.Sub(delta)
		if ctl.LetScore.LessThanOrEqual(decimal.Zero) {
			ctl.IsDel = true
		}
		ctl.UpdateTime = time.Now().Unix()
	}
}

func (s *SingleCtrlMgr) randInt(n int) int {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.r == nil {
		s.r = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return s.r.Intn(n)
}

func (s *SingleCtrlMgr) addSingleCtrl(uid uint32, rate, ctrl decimal.Decimal, t int) {
	s.lock.Lock()
	defer s.lock.Unlock()

	tmp := &Ctrl{
		LetScore:   ctrl,
		UserId:     uid,
		RateScore:  ctrl,
		Rate:       rate,
		IsDel:      false,
		UpdateTime: time.Now().Unix(),
		TYPE:       t,
	}

	if ctrl := s.sc[uid]; ctrl != nil {
		if t == SYSTEM_CTRL {
			s.sc[uid] = tmp
		} else {
			if ctrl.TYPE == AUTO_CTRL {
				if ctrl.IsDel {
					s.sc[uid] = tmp
				}
			}
		}
	} else {
		s.sc[uid] = tmp
	}
}

func (s *SingleCtrlMgr) delSingleCtrl(uid uint32) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.sc, uid)
}

// 同步入库
func (s *SingleCtrlMgr) flushDB() {
	s.lock.Lock()
	defer s.lock.Unlock()

	piple, n, delIds := RedisIns().cli.Pipeline(), 0, make([]uint32, 0, 64)
	for k, v := range s.sc {
		if v.UpdateTime <= 0 {
			continue
		}
		n++
		itemKey := fmt.Sprintf("%d", k)
		str, _ := jsoniter.MarshalToString(v)
		if v.IsDel {
			piple.HDel(context.Background(), "user_ctl", itemKey)
			delIds = append(delIds, k)
		} else {
			piple.HSet(context.Background(), "user_ctl", itemKey, str)
		}
		if n >= 50 {
			_, err := piple.Exec(context.Background())
			if err != nil {
				zap.L().Error("批量更新失败", zap.Any("err", err))
			}
			n, piple = 0, RedisIns().cli.Pipeline()
		}
		v.UpdateTime = 0
	}
	if n > 0 {
		_, err := piple.Exec(context.Background())
		if err != nil {
			zap.L().Error("批量更新失败", zap.Any("err", err))
		}
	}
	//清理缓存
	for _, v := range delIds {
		delete(s.sc, v)
	}
}

type GameCacheMgr struct {
	lock          *sync.RWMutex
	agents        map[int64]*AgentData //key agetId
	ctrlThreshold decimal.Decimal      //单控阈值
	gcmRand       *rand.Rand
}

func (gcm *GameCacheMgr) GetAgent(agentId int64) *AgentData {
	gcm.lock.Lock()
	defer gcm.lock.Unlock()

	agent := gcm.agents[agentId]
	if agent == nil {
		agent = &AgentData{
			gameCache:  make(map[string]*Game),
			userCache:  make(map[uint32]*User),
			lock:       &sync.RWMutex{},
			Id:         agentId,
			RoundCache: make(map[string]*RoundItem),
		}
		gcm.agents[agentId] = agent
	}
	return agent
}

func (gcm *GameCacheMgr) GetUser(agentId, userId int64) *User {
	agent := gcm.GetAgent(agentId)
	if agent == nil {
		return nil
	}
	//细分代理锁
	agent.lock.Lock()
	defer agent.lock.Unlock()

	return agent.GetUser(uint32(userId))
}

func (gcm *GameCacheMgr) poolType(pool decimal.Decimal, pv *config.Pool) (int, decimal.Decimal) {
	if pool.LessThanOrEqual(decimal.Zero) {
		return config.POOL_BREAK_DOWN, decimal.Zero
	}
	if pool.LessThanOrEqual(pv.Pool[1].Normal) {
		return config.POOL_LOW, pv.Pool[1].MinRate
	} else if pool.GreaterThan(pv.Pool[1].Normal) && pool.LessThanOrEqual(pv.Pool[1].Max) {
		return config.POOL_NORMAL, pv.Pool[1].NormalRate
	} else {
		return config.POOL_HIGH, pv.Pool[1].MaxRate
	}
}

func (gcm *GameCacheMgr) Complete(agentId int64, userId uint32, symbol string, bet, award, rate decimal.Decimal) {
	agent := gcm.GetAgent(agentId)
	//细分代理锁
	agent.lock.Lock()
	defer agent.lock.Unlock()

	game := agent.GetGame(symbol)
	user := agent.GetUser(uint32(userId))
	if user.IsTourist == 0 {
		chips := award
		//不会有返奖的情况，直接用bet计算税收
		if award.LessThan(bet) {
			//bet作为有效打码
			chips = bet
		}
		if chips.LessThan(decimal.Zero) {
			chips = chips.Abs()
		}
		game.TotalChips = game.TotalChips.Add(chips.Truncate(4))
		//以有效下注计算水池后   可以直接放在下注的时候计算税收
		// game.TotalRevenue = game.TotalRevenue.Add(bet.Mul(rate).Truncate(4))
		game.UpdateTime = time.Now().Unix()
		user.Count = user.Count.Add(decimal.NewFromInt(1))
		addUserAward(user, award)
		user.UpdateTime = time.Now().Unix()
	}
}

// 返还水池（仅退代理游戏预占/多扣赔付，不改玩家总返奖累计）
func (gcm *GameCacheMgr) ReturnPool(agentId int64, userId uint32, symbol string, delta decimal.Decimal) {
	agent := gcm.GetAgent(agentId)
	//细分代理锁
	agent.lock.Lock()
	defer agent.lock.Unlock()

	game := agent.GetGame(symbol)
	_ = agent.GetUser(uint32(userId))
	if delta.GreaterThan(decimal.Zero) {
		game.TotalProfLoss = game.TotalProfLoss.Sub(delta)
		game.UpdateTime = time.Now().Unix()
	}
}

// 下注
func (gcm *GameCacheMgr) ChangePool(agentId int64, userId int32, symbol, currencyType, recordId string, bet, award, rate decimal.Decimal) bool {
	agent := gcm.GetAgent(agentId)
	//细分代理锁
	agent.lock.Lock()
	defer agent.lock.Unlock()

	user := agent.GetUser(uint32(userId))
	if user.IsTourist == 0 {
		game := agent.GetGame(symbol)
		before := poolWithBase(game, agentId, symbol)
		// FG：仅 award>0 记入代理总赔付（含预占）
		addGamePayout(game, award)
		//增加水池
		game.TotalEffectBet = game.TotalEffectBet.Add(bet)
		//累计税收
		game.TotalRevenue = game.TotalRevenue.Add(bet.Mul(rate).Truncate(4))
		//触发更新
		game.UpdateTime = time.Now().Unix()
		//记录玩家有效投注
		user.TotalEffectBet = user.TotalEffectBet.Add(bet)
		//记录玩家局数
		// user.Count = user.Count.Add(decimal.NewFromInt(1))
		user.UpdateTime = time.Now().Unix()
		after := poolWithBase(game, agentId, symbol)
		zap.L().Debug("Pool", zap.Any("agentId", agentId), zap.Any("symbol", symbol), zap.Any("recordId", recordId), zap.Any("userId", userId), zap.Any("currencyType", currencyType), zap.Any("bet", bet), zap.Any("award", award), zap.Any("before", before), zap.Any("after", after))
	}
	return true
}

// RollbackPoolChange 撤销一次已经写入水池的下注变更，供余额更新失败时恢复现场。
func (gcm *GameCacheMgr) RollbackPoolChange(agentId int64, userId int32, symbol string, bet, award, rate decimal.Decimal) bool {
	agent := gcm.GetAgent(agentId)
	agent.lock.Lock()
	defer agent.lock.Unlock()

	user := agent.GetUser(uint32(userId))
	if user == nil || user.IsTourist != 0 {
		return true
	}

	game := agent.GetGame(symbol)
	game.TotalEffectBet = game.TotalEffectBet.Sub(bet)
	game.TotalProfLoss = game.TotalProfLoss.Sub(positiveAward(award).Truncate(4))
	game.TotalRevenue = game.TotalRevenue.Sub(bet.Mul(rate).Truncate(4))
	game.UpdateTime = time.Now().Unix()
	user.TotalEffectBet = user.TotalEffectBet.Sub(bet)
	user.UpdateTime = time.Now().Unix()
	return true
}

// 下注
func (gcm *GameCacheMgr) ChangePoolWithNoLock(agent *AgentData, user *User, game *Game, currencyType, recordId string, bet, award, revence decimal.Decimal) bool {
	if user.IsTourist == 0 {
		before := poolWithBase(game, agent.Id, game.Symbol)
		// FG：仅 award>0 记入代理总赔付（含预占）
		addGamePayout(game, award)
		//增加水池
		game.TotalEffectBet = game.TotalEffectBet.Add(bet)
		//
		game.TotalRevenue = game.TotalRevenue.Add(revence.Truncate(4))
		//触发更新
		game.UpdateTime = time.Now().Unix()
		//记录玩家有效投注
		user.TotalEffectBet = user.TotalEffectBet.Add(bet)
		//记录玩家局数
		// user.Count = user.Count.Add(decimal.NewFromInt(1))
		user.UpdateTime = time.Now().Unix()
		after := poolWithBase(game, agent.Id, game.Symbol)
		zap.L().Debug("Pool", zap.Any("agentId", agent.Id), zap.Any("symbol", game.Symbol), zap.Any("recordId", recordId), zap.Any("userId", user.UserId), zap.Any("currencyType", currencyType), zap.Any("bet", bet), zap.Any("award", award), zap.Any("before", before), zap.Any("after", after))
	}
	return true
}

func poolCfgBase(agentId int64, symbol string) decimal.Decimal {
	pc := config.CfgIns.GetPoolCfg(agentId, symbol)
	if pc == nil || pc.Pool == nil {
		return decimal.Zero
	}
	item := pc.Pool[1]
	if item == nil {
		return decimal.Zero
	}
	return item.Base
}

func poolRaw(game *Game) decimal.Decimal {
	return (game.TotalEffectBet.Sub(game.TotalProfLoss)).Sub(game.TotalRevenue)
}

func poolWithBase(game *Game, agentId int64, symbol string) decimal.Decimal {
	return poolRaw(game).Add(poolCfgBase(agentId, symbol))
}

func (gcm *GameCacheMgr) GetPool(agentId int64, symbol string) decimal.Decimal {
	agent := gcm.GetAgent(agentId)
	//细分代理锁
	agent.lock.Lock()
	defer agent.lock.Unlock()

	game := agent.GetGame(symbol)
	// pool = 有效投注 - 总赔付 - 税收 + Base（与 Lottery 口径一致）
	return poolWithBase(game, agentId, symbol)
}

func (gcm *GameCacheMgr) CheckPoolWithChange(agentId int64, symbol, recordId, currencyType string, award, bet, revenue decimal.Decimal, userId uint32) bool {
	agent := gcm.GetAgent(agentId)
	//细分代理锁
	agent.lock.Lock()
	defer agent.lock.Unlock()

	game := agent.GetGame(symbol)
	//水池计算方式  pool = 有效投注 - 总赔付 - 税收 + Base；预判只扣正赔付
	payout := positiveAward(award)
	pool := (game.TotalEffectBet.Add(bet).Sub(game.TotalProfLoss.Add(payout))).Sub(game.TotalRevenue.Add(revenue)).Add(poolCfgBase(agentId, symbol))
	if pool.LessThanOrEqual(decimal.Zero) {
		return false
	}
	user := agent.GetUser(uint32(userId))
	if user.IsTourist == 0 {
		before := poolWithBase(game, agentId, symbol)
		addGamePayout(game, award)
		//增加水池
		game.TotalEffectBet = game.TotalEffectBet.Add(bet)
		//
		game.TotalRevenue = game.TotalRevenue.Add(revenue.Truncate(4))
		//触发更新
		game.UpdateTime = time.Now().Unix()
		//记录玩家有效投注
		user.TotalEffectBet = user.TotalEffectBet.Add(bet)
		//记录玩家局数
		// user.Count = user.Count.Add(decimal.NewFromInt(1))
		user.UpdateTime = time.Now().Unix()
		after := poolWithBase(game, agentId, symbol)
		zap.L().Debug("Pool", zap.Any("agentId", agentId), zap.Any("symbol", symbol), zap.Any("recordId", recordId), zap.Any("userId", userId), zap.Any("currencyType", currencyType), zap.Any("bet", bet), zap.Any("award", award), zap.Any("before", before), zap.Any("after", after))
	}

	return true
}

func (gcm *GameCacheMgr) CheckPoolWithOutBet(agentId int64, symbol, recordId, currencyType string, award, bet, revenue decimal.Decimal, userId uint32) bool {
	agent := gcm.GetAgent(agentId)
	//细分代理锁
	agent.lock.Lock()
	defer agent.lock.Unlock()

	game := agent.GetGame(symbol)
	//水池计算方式  pool = 有效投注 - 总赔付 - 税收 + Base
	pool := poolWithBase(game, agentId, symbol)
	payout := positiveAward(award)
	if payout.GreaterThan(decimal.Zero) && pool.LessThan(payout) {
		return false
	}
	user := agent.GetUser(uint32(userId))
	if user.IsTourist == 0 {
		before := poolWithBase(game, agentId, symbol)
		addGamePayout(game, award)
		//增加水池
		game.TotalEffectBet = game.TotalEffectBet.Add(bet)
		//
		game.TotalRevenue = game.TotalRevenue.Add(revenue.Truncate(4))
		//触发更新
		game.UpdateTime = time.Now().Unix()
		//记录玩家有效投注
		user.TotalEffectBet = user.TotalEffectBet.Add(bet)
		//记录玩家局数
		// user.Count = user.Count.Add(decimal.NewFromInt(1))
		user.UpdateTime = time.Now().Unix()
		after := poolWithBase(game, agentId, symbol)
		zap.L().Debug("Pool", zap.Any("agentId", agentId), zap.Any("symbol", symbol), zap.Any("recordId", recordId), zap.Any("userId", userId), zap.Any("currencyType", currencyType), zap.Any("bet", bet), zap.Any("award", award), zap.Any("before", before), zap.Any("after", after))
	}

	return true
}

// 保存注单信息
func (gcm *GameCacheMgr) SaveRoundData(agentId int64, roundId string, maxPay decimal.Decimal, playerId uint32) {
	agent := gcm.GetAgent(agentId)
	//细分代理锁
	agent.lock.Lock()
	defer agent.lock.Unlock()

	user := agent.GetUser(playerId)
	if user != nil && user.IsTourist == 0 {
		if ri := agent.RoundCache[roundId]; ri != nil {
			zap.L().Error("重复缓存对局数据", zap.Any("data", ri))
		} else {
			agent.RoundCache[roundId] = &RoundItem{
				MaxPay:  maxPay,
				Over:    false,
				AgentId: agentId,
				RoundId: roundId,
				Saved:   false,
			}
		}
	}
}

func (gcm *GameCacheMgr) GetPlayerAccount(agentId, userId int64) string {
	agent := gcm.GetAgent(agentId)
	//细分代理锁
	agent.lock.Lock()
	defer agent.lock.Unlock()

	user := agent.GetUser(uint32(userId))
	if user != nil {
		return user.Account
	}
	return ""
}

// 获取注单信息
func (gcm *GameCacheMgr) GetRoundData(agentId int64, roundId string) *RoundItem {
	agent := gcm.GetAgent(agentId)
	//细分代理锁
	agent.lock.RLock()
	defer agent.lock.RUnlock()

	ri := agent.RoundCache[roundId]
	if ri == nil {
		zap.L().Debug("无预扣信息", zap.Any("agentId", agentId), zap.Any("roundId", roundId))
	}
	return ri
}

func (gcm *GameCacheMgr) FinishRoundData(agentId int64, roundId string) *RoundItem {
	agent := gcm.GetAgent(agentId)
	agent.lock.Lock()
	defer agent.lock.Unlock()

	ri := agent.RoundCache[roundId]
	if ri == nil {
		return nil
	}
	ri.Over = true
	copy := *ri
	return &copy
}

/*
平均打码量A：请求 averageBet>0 且 betAmount>0 时用 averageBet，否则 betAmount>0 时用 betAmount；
前端在每局开奖前传入开奖金额，若满足开奖则不干预；不满足则开奖失败
按游戏分水池房间：低 、中 、 高水位；

低水位：水池余额*百分比、A*100(可配置倍数)；取两者最小值
正常水位：水池余额*百分比、A*500(可配置倍数)；取两者取最小值
高水位：水池余额*百分比、A*1000(可配置倍数)；取两者取最小值

单控时：水池余额*百分比、A*20(可配置倍数)；取两者最小值

单控：
自动单控
手动单控
单控数值：盈利金额大于设定值时对应的单控制数值；
单控数值使用；
触发进入单控后：总是按以下条件进行中奖条件判定：

单控时：水池余额*百分比、A*20(可配置倍数)；取两者最小值。
*/
func (gcm *GameCacheMgr) Lottery(agentId int64, userId int32, pc *config.Pool, symbol, currencyType string, bet, award decimal.Decimal, roundId string, averageBet decimal.Decimal) (decimal.Decimal, bool) {
	agent := gcm.GetAgent(agentId)
	//细分代理锁
	agent.lock.Lock()
	defer agent.lock.Unlock()

	user := agent.GetUser(uint32(userId))
	game := agent.GetGame(symbol)

	revence := bet.Mul(pc.Pool[1].Revenue).Truncate(4)
	pool := (bet.Add(game.TotalEffectBet).Sub(game.TotalProfLoss)).Sub(game.TotalRevenue.Add(revence)).Add(pc.Pool[1].Base)
	//判断pool等级
	t, r := gcm.poolType(pool, pc)
	//计算可赔付值
	if t == config.POOL_BREAK_DOWN {
		return pool, false
	}
	//判断是否自动ctrl
	if ac := config.CfgIns.GetAutoCtrl(user.TotalEffectBet, user.TotalProfLoss); ac != nil {
		zap.L().Debug("Lottery:自动控制", zap.Any("agentId", agentId), zap.Any("symbol", symbol), zap.Any("roundId", roundId), zap.Any("playerId", userId), zap.Any("ac", ac))
		SCIns().addSingleCtrl(uint32(userId), ac.ControlRate, ac.Score, AUTO_CTRL)
	}
	zap.L().Debug("Lottery:当前水池值", zap.Any("agentId", agentId), zap.Any("symbol", symbol), zap.Any("roundId", roundId), zap.Any("playerId", userId), zap.Any("pool", pool))
	rate := decimal.Zero
	if !user.TotalEffectBet.Equal(decimal.Zero) {
		rate = (user.TotalProfLoss).Div(user.TotalEffectBet)
	}
	cItems := config.CfgIns.GetAwardOddsConfigWithProfitOdds(symbol, rate)
	if cItems == nil {
		//如果都不满足则获取默认配置
		cItems = config.CfgIns.GetAwardOddsDefaultCtrlConfig(symbol)
	}
	//判断是否进入单控 1获取玩家单控信息 若有单控根据随机值判断是否进入单控 否则直接进入正常流程
	ctrl := singleCtrl.getSingleCtrl(uint32(userId))
	if ctrl != nil && !ctrl.IsDel {
		randNum := SCIns().randInt(100)
		if randNum <= int(ctrl.Rate.Abs().IntPart()) {
			//进入单控
			cItems = config.CfgIns.GetAwardOddsSingleCtrlConfig(symbol)
			if cItems == nil {
				zap.L().Debug("Lottery:获取单控配置信息失败", zap.Any("agentId", agentId), zap.Any("symbol", symbol), zap.Any("roundId", roundId), zap.Any("playerId", userId), zap.Any("pool", pool))
				return pool, false
			}
			singleCtrl.setSingleCtrlScore(uint32(userId), award)
		}
	}
	item := cItems.PoolOdds[t]
	m := item.M
	//水池余额*百分比
	p1 := pool.Mul(r)
	avg := bet
	if averageBet.GreaterThan(decimal.Zero) {
		avg = averageBet
	}
	p2 := avg.Mul(m)
	//p1、p2取最小值
	p := p1
	if p1.GreaterThan(p2) {
		p = p2
		zap.L().Debug("Lottery:使用平均投注倍数上限", zap.Any("agentId", agentId), zap.Any("symbol", symbol), zap.Any("roundId", roundId), zap.Any("playerId", userId), zap.Any("可赔付", p), zap.Any("平均投注", avg), zap.Any("总投注", user.TotalEffectBet), zap.Any("投注数量", user.Count), zap.Any("配置详情", item))
	} else {
		zap.L().Debug("Lottery:使用水池值", zap.Any("agentId", agentId), zap.Any("symbol", symbol), zap.Any("roundId", roundId), zap.Any("playerId", userId), zap.Any("可赔付", p), zap.Any("平均投注", avg), zap.Any("总投注", user.TotalEffectBet), zap.Any("总盈亏", game.TotalProfLoss), zap.Any("总税收", game.TotalRevenue), zap.Any("基数", pc.Pool[1].Base), zap.Any("投注数量", user.Count), zap.Any("倍数", m))
	}
	coefficient := config.CfgIns.GetControlCoefficient(avg, user.TotalEffectBet, user.TotalProfLoss, user.Count, config.CfgIns.IsSingleControlEnabled(user.Count))
	controlledLimit := p.Mul(coefficient).Truncate(4)
	zap.L().Debug("Lottery:pool配置", zap.Any("agentId", agentId), zap.Any("symbol", symbol), zap.Any("roundId", roundId), zap.Any("playerId", userId), zap.Any("rate", rate), zap.Any("水池状态", t), zap.Any("item", cItems), zap.Any("倍数", m), zap.Any("控制系数", coefficient), zap.Any("原始可赔付", p), zap.Any("控制后上限", controlledLimit), zap.Any("最终赔付", award))
	if controlledLimit.LessThan(award) {
		zap.L().Debug("Lottery:赔付失败", zap.Any("agentId", agentId), zap.Any("symbol", symbol), zap.Any("roundId", roundId), zap.Any("playerId", userId), zap.Any("可赔付", controlledLimit), zap.Any("返奖值", award))
		return pool, false
	}
	zap.L().Debug("Lottery:赔付成功", zap.Any("agentId", agentId), zap.Any("symbol", symbol), zap.Any("roundId", roundId), zap.Any("playerId", userId), zap.Any("可赔付", controlledLimit), zap.Any("返奖值", award))

	CacheIns().ChangePoolWithNoLock(agent, user, game, currencyType, roundId, bet, award, revence)

	return pool, true
}

func GetAgentChanges() ([]*User, []*Game) {
	gameCache.lock.RLock()
	defer gameCache.lock.RUnlock()

	users := make([]*User, 0, 64)
	games := make([]*Game, 0, 64)
	for _, agent := range gameCache.agents {
		agent.lock.RLock()
		for _, item := range agent.gameCache {
			if item.UpdateTime > 0 {
				games = append(games, item)
			}
		}
		for _, item := range agent.userCache {
			if item.UpdateTime > 0 {
				users = append(users, item)
			}
		}
		agent.lock.RUnlock()
	}
	return users, games
}

func ResetPool(data interface{}) {
	gameCache.lock.Lock()
	defer gameCache.lock.Unlock()

	for _, agent := range gameCache.agents {
		for _, item := range agent.gameCache {
			item.TotalChips = decimal.Zero
			item.TotalEffectBet = decimal.Zero
			item.TotalProfLoss = decimal.Zero
			item.TotalRevenue = decimal.Zero
		}
	}

	piple := RedisIns().cli.Pipeline()
	zap.L().Debug("清理水池开始")
	piple.Del(context.Background(), esindex.AgentEffectData())
	piple.Del(context.Background(), esindex.AgentChipsData())
	piple.Del(context.Background(), esindex.AgentRevenueData())
	piple.Del(context.Background(), esindex.AgentProfitLossData())
	_, err := piple.Exec(context.Background())
	if err != nil {
		zap.L().Error("Redis重置pool失败!", zap.Any("err", err))
	}
	zap.L().Debug("清理水池结束")
}

// 获取单控阈值
func getSingleCtrlThreshold() {
	value, err := RedisIns().Get("single_ctrl_threshold")
	if err != nil {
		return
	}
	t, _ := decimal.NewFromString(value)
	gameCache.ctrlThreshold = t
	zap.L().Debug("获取单控阈值", zap.Any("value", t))
}

func SaveAgentData(games []*Game) {
	piple, n := RedisIns().cli.Pipeline(), 0
	for _, item := range games {
		zap.L().Debug("定时缓存代理数据", zap.Any("item", item))
		if item.UpdateTime > 0 {
			key := fmt.Sprintf("%d_%s", item.AgentId, item.Symbol)
			n++
			piple.ZAdd(context.Background(), esindex.AgentEffectData(), redis.Z{Score: item.TotalEffectBet.InexactFloat64(), Member: key})
			piple.ZAdd(context.Background(), esindex.AgentChipsData(), redis.Z{Score: item.TotalChips.InexactFloat64(), Member: key})
			piple.ZAdd(context.Background(), esindex.AgentProfitLossData(), redis.Z{Score: item.TotalProfLoss.InexactFloat64(), Member: key})
			piple.ZAdd(context.Background(), esindex.AgentRevenueData(), redis.Z{Score: item.TotalRevenue.InexactFloat64(), Member: key})
			item.UpdateTime = 0
		}
		if n >= 100 {
			_, err := piple.Exec(context.Background())
			if err != nil {
				zap.L().Error("定时缓存代理游戏统计数据失败", zap.Any("error", err))
			}
			n = 0
			piple = redisDao.cli.Pipeline()
		}
	}
	if n > 0 {
		_, err := piple.Exec(context.Background())
		if err != nil {
			zap.L().Error("定时缓存代理游戏统计数据失败", zap.Any("error", err))
		}
	}
}

func SaveUserData(users []*User) {
	piple, n := RedisIns().cli.Pipeline(), 0
	for _, item := range users {
		zap.L().Debug("定时缓存玩家数据", zap.Any("item", item))
		if item.UpdateTime > 0 {
			n++
			k := fmt.Sprintf("%d", item.UserId)
			piple.ZAdd(context.Background(), "userTotalEffBet", redis.Z{Score: item.TotalEffectBet.InexactFloat64(), Member: k})
			piple.ZAdd(context.Background(), "userTotalProfLoss", redis.Z{Score: item.TotalProfLoss.InexactFloat64(), Member: k})
			piple.ZAdd(context.Background(), "userBetCount", redis.Z{Score: item.Count.InexactFloat64(), Member: k})
			item.UpdateTime = 0
		}
		if n >= 100 {
			_, err := piple.Exec(context.Background())
			if err != nil {
				zap.L().Error("定时缓存玩家游戏统计数据失败", zap.Any("error", err))
			}
			n = 0
			piple = redisDao.cli.Pipeline()
		}
	}
	if n > 0 {
		_, err := piple.Exec(context.Background())
		if err != nil {
			zap.L().Error("定时缓存玩家游戏统计数据失败", zap.Any("error", err))
		}
	}
}

// 加载对局数据
func LoadAllRoundData() {
	values := make([]string, 0, 128)
	iter := RedisIns().cli.HScan(context.Background(), "rund_records", 0, "*", 100).Iterator()
	for iter.Next(context.Background()) {
		data := iter.Val()
		values = append(values, data)
	}
	if err := iter.Err(); err != nil {
		zap.L().Fatal("加载对局数据失败", zap.Any("err", err))
	}
	for i := 1; i < len(values); i += 2 {
		zap.L().Debug("玩家注单缓存信息", zap.Any("data", values[i]))
		ri := &RoundItem{}
		err := jsoniter.UnmarshalFromString(values[i], ri)
		if err != nil {
			zap.L().Fatal("玩家注单缓存信息", zap.Any("data", values[i]), zap.Any("err", err))
		} else {
			agent := gameCache.GetAgent(ri.AgentId)
			ri.Saved = true
			agent.RoundCache[ri.RoundId] = ri
		}
	}
}

// 同步注单数据信息
func SyncRoundData() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				zap.L().Error("panic", zap.Any("error", err))
			}
		}()
		ticker := time.NewTicker(time.Second * 60 * 1)
		for range ticker.C {
			zap.L().Debug("开始注单数据同步")
			rc := make([]*RoundItem, 0, 128)
			ro := make([]*RoundItem, 0, 128)
			gameCache.lock.Lock()
			for _, agent := range gameCache.agents {
				for _, item := range agent.RoundCache {
					if !item.Saved {
						rc = append(rc, item)
					}
					if item.Over {
						ro = append(ro, item)
					}
				}
			}
			gameCache.lock.Unlock()
			n, ok := 0, true
			piple := RedisIns().cli.Pipeline()
			for _, v := range rc {
				n++
				str, _ := jsoniter.MarshalToString(v)
				piple.HSet(context.Background(), "rund_records", v.RoundId, str)
				if n >= 50 {
					_, err := piple.Exec(context.Background())
					if err != nil {
						if err != redis.Nil {
							zap.L().Fatal("获取rund_records失败", zap.Any("err", err))
						} else {
							return
						}
						ok = false
						zap.L().Error("批量保存注单信息失败，稍后重试", zap.Any("err", err))
						break
					} else {
						piple = redisDao.cli.Pipeline()
						n = 0
					}
				}
			}
			if n > 0 {
				_, err := piple.Exec(context.Background())
				if err != nil {
					ok = false
					zap.L().Error("批量保存注单信息失败，稍后重试", zap.Any("err", err))
				}
			}
			if ok {
				for _, v := range rc {
					v.Saved = true
				}
			}
			// 删除已经完成预扣及返还的注单信息
			n, ok = 0, true
			piple = RedisIns().cli.Pipeline()
			for _, v := range ro {
				n++
				piple.HDel(context.Background(), "rund_records", v.RoundId)
				if n >= 50 {
					_, err := piple.Exec(context.Background())
					if err != nil {
						if err != redis.Nil {
							zap.L().Fatal("获取配置失败", zap.Any("err", err))
						}
						ok = false
						zap.L().Error("批量保存注单信息失败，稍后重试", zap.Any("err", err))
						break
					} else {
						piple = redisDao.cli.Pipeline()
						n = 0
					}
				}
			}
			if n > 0 {
				_, err := piple.Exec(context.Background())
				if err != nil {
					ok = false
					zap.L().Error("批量保存注单信息失败，稍后重试", zap.Any("err", err))
				}
			}
			if ok {
				for _, v := range ro {
					agent := gameCache.GetAgent(v.AgentId)
					if agent != nil {
						agent.lock.Lock()
						delete(agent.RoundCache, v.RoundId)
						agent.lock.Unlock()
					}
				}
			}
			zap.L().Debug("完成注单数据同步")
		}
	}()
}

// 初始化缓存
func CahceInit() {
	//游戏数据加载
	if gameCache == nil {
		gameCache = &GameCacheMgr{
			lock:          &sync.RWMutex{},
			agents:        make(map[int64]*AgentData),
			ctrlThreshold: decimal.NewFromInt(100000),
			gcmRand:       rand.New(rand.NewSource(time.Now().Unix())),
		}
		getSingleCtrlThreshold() //更新单控阈值
		go func() {
			defer func() {
				if err := recover(); err != nil {
					zap.L().Error("panic", zap.Any("err", err))
				}
			}()
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				users, games := GetAgentChanges()
				SaveAgentData(games)     //定时保存代理数据
				SaveUserData(users)      //定时保存玩家数据
				getSingleCtrlThreshold() //更新单控阈值
			}
		}()
	}
	//单控数据加载
	if singleCtrl == nil {
		singleCtrl = &SingleCtrlMgr{
			lock: &sync.RWMutex{},
			r:    rand.New(rand.NewSource(time.Now().UnixNano())),
			sc:   make(map[uint32]*Ctrl),
		}
		//加载所有的控制信息
		values := make([]string, 0, 128)
		iter := RedisIns().cli.HScan(context.Background(), "user_ctl", 0, "*", 100).Iterator()
		for iter.Next(context.Background()) {
			data := iter.Val()
			values = append(values, data)
		}
		if err := iter.Err(); err != nil {
			zap.L().Fatal("加载控制信息失败", zap.Any("err", err))
		}
		for i := 1; i < len(values); i += 2 {
			zap.L().Debug("玩家控制信息数据", zap.Any("data", values[i]))
			ctl := &Ctrl{}
			err := jsoniter.UnmarshalFromString(values[i], ctl)
			if err != nil {
				zap.L().Fatal("解析控制信息失败", zap.Any("data", values[i]), zap.Any("err", err))
			} else {
				singleCtrl.sc[ctl.UserId] = ctl
			}
		}
		go func() {
			defer func() {
				if err := recover(); err != nil {
					zap.L().Error("panic", zap.Any("err", err))
				}
			}()
			ticker := time.NewTicker(5 * 60 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				//定时缓存玩家单控信息
				singleCtrl.flushDB()
			}
		}()
	}
}

func CacheIns() *GameCacheMgr {
	return gameCache
}

func SCIns() *SingleCtrlMgr {
	return singleCtrl
}
