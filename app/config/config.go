package config

import (
	"app/tables/manager"
	"fmt"
	"sync"

	"github.com/shopspring/decimal"
)

var CfgIns *Configs = nil

type SystemConfig struct {
	GameUrls []string `json:"game_url"`
	Replays  []string `json:"replays"`
}

// 启动配置
type RunConfig struct {
	Redis struct {
		Host []string `yaml:"host"`
		User string   `yaml:"user"`
		Pwd  string   `yaml:"pwd"`
	} `yaml:"redis"`
	Mysql map[string]struct {
		Host     string `yaml:"host"`
		Port     int32  `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
	} `yaml:"mysql"`
	Elastic struct {
		Host     []string `yaml:"hosts"`
		UserName string   `yaml:"username"`
		Password string   `yaml:"password"`
	} `yaml:"elastic"`
	ServerIp   string `yaml:"server_ip"`
	ServerPort int    `yaml:"server_port"`
	Log        string `yaml:"log"`
}

type PoolItem struct {
	Min        decimal.Decimal `json:"min"`
	Normal     decimal.Decimal `json:"normal"`
	Max        decimal.Decimal `json:"max"`
	NormalRate decimal.Decimal `json:"normalRate"`
	MinRate    decimal.Decimal `json:"minRate"`
	MaxRate    decimal.Decimal `json:"maxRate"`
	Control    decimal.Decimal `json:"control"`
	Revenue    decimal.Decimal `json:"revenue"`
	Base       decimal.Decimal `json:"base"`
}

type Pool struct {
	Symbol string              `json:"symbol"`
	Name   string              `json:"name"`
	NameZH string              `json:"nameZH"`
	GameId int64               `json:"gameId"`
	Pool   map[int32]*PoolItem `json:"pool"`
}

const (
	POOL_BREAK_DOWN = -1
	POOL_LOW        = 0
	POOL_NORMAL     = 1
	POOL_HIGH       = 2
)

type OddsItem struct {
	Odds decimal.Decimal `json:"odds"`
	M    decimal.Decimal `json:"multiple"`
	Rate decimal.Decimal `json:"rate"`
}

// 当Min==Max == 0时表示默认配置
// 配置算法 “其他情况时单控不介入” 流程
type AwardOddsConfigItem struct {
	Id       int64           `json:"id"`        //id
	Name     string          `json:"name"`      //名字
	Min      decimal.Decimal `json:"min"`       //盈亏比最小值 百分比
	Max      decimal.Decimal `json:"max"`       //盈亏比最大值 百分比
	PoolOdds []*OddsItem     `json:"pool_odds"` //中奖概率 低、中、高 水位开奖概率 百分比
}

type AwardConfig struct {
	OddsConfig []*AwardOddsConfigItem `json:"award_config"` //key==> AwardOddsConfigItem.min-AwardOddsConfigItem.max 组成的字符串
	GameId     int64                  `json:"gameId"`
	Symbol     string                 `json:"symbol"`
}

type AutoCtrlItem struct {
	TotalEffect       decimal.Decimal `json:"totalEffect"`       // 打吗
	TotalProfLoss     decimal.Decimal `json:"totalProfLoss"`     //yk
	TotalProfLossRate decimal.Decimal `json:"totalProfLossRate"` //ykb
	ControlRate       decimal.Decimal `json:"controlRate"`       //kz比例
	Score             decimal.Decimal `json:"score"`             //kz积分
}

type AutoCtrlMgr struct {
	Ctrls []*AutoCtrlItem
}

// ControlCoefficientRange describes a configurable payout coefficient rule.
// Bet ranges use (min, max] semantics; a max of zero means no upper bound.
// Single-control ranges use min as the threshold and ignore max.
type ControlCoefficientRange struct {
	Min         decimal.Decimal `json:"min"`
	Max         decimal.Decimal `json:"max"`
	Coefficient decimal.Decimal `json:"coefficient"`
}

type ControlCoefficientConfig struct {
	Bet          []*ControlCoefficientRange `json:"bet"`
	Single       []*ControlCoefficientRange `json:"single"`
	SingleRounds decimal.Decimal            `json:"single_rounds"`
}

func DefaultControlCoefficientConfig() *ControlCoefficientConfig {
	return &ControlCoefficientConfig{
		Bet: []*ControlCoefficientRange{
			{Min: decimal.Zero, Max: decimal.RequireFromString("0.5"), Coefficient: decimal.RequireFromString("0.25")},
			{Min: decimal.RequireFromString("0.5"), Max: decimal.NewFromInt(2), Coefficient: decimal.RequireFromString("0.5")},
			{Min: decimal.NewFromInt(2), Max: decimal.NewFromInt(4), Coefficient: decimal.RequireFromString("0.6")},
			{Min: decimal.NewFromInt(4), Max: decimal.NewFromInt(8), Coefficient: decimal.RequireFromString("0.9")},
			{Min: decimal.NewFromInt(8), Max: decimal.Zero, Coefficient: decimal.NewFromInt(1)},
		},
		Single: []*ControlCoefficientRange{
			{Min: decimal.NewFromInt(1), Coefficient: decimal.RequireFromString("0.9")},
			{Min: decimal.NewFromInt(2), Coefficient: decimal.RequireFromString("0.8")},
			{Min: decimal.NewFromInt(3), Coefficient: decimal.RequireFromString("0.7")},
			{Min: decimal.NewFromInt(4), Coefficient: decimal.RequireFromString("0.6")},
			{Min: decimal.NewFromInt(5), Coefficient: decimal.RequireFromString("0.5")},
			{Min: decimal.NewFromInt(6), Coefficient: decimal.RequireFromString("0.4")},
			{Min: decimal.NewFromInt(7), Coefficient: decimal.RequireFromString("0.35")},
			{Min: decimal.NewFromInt(8), Coefficient: decimal.RequireFromString("0.3")},
			{Min: decimal.NewFromInt(9), Coefficient: decimal.RequireFromString("0.25")},
		},
		SingleRounds: decimal.NewFromInt(500),
	}
}

// pool配置管理
type PoolMgr struct {
	//key:symbol
	Default map[string]*Pool
	//key:agentId-symbol
	Agent map[string]*Pool
}

// pool配置管理
type AwardMgr struct {
	//key symbol
	Data map[string]*AwardConfig
}

type CurrencyMgr struct {
	Data map[string]decimal.Decimal `json:"currency"`
}

type GatewaysMgr struct {
	Urls       []*manager.ApiConfig
	UpdateTime int64
}

type Configs struct {
	Lock        *sync.RWMutex
	Pool        *PoolMgr
	Award       *AwardMgr
	Coefficient *ControlCoefficientConfig
	System      *SystemConfig
	Currency    *CurrencyMgr
	AC          *AutoCtrlMgr
	Gateway     *GatewaysMgr
}

func (cfg *Configs) SetGatewayCfg(c *GatewaysMgr) {
	cfg.Lock.Lock()
	defer cfg.Lock.Unlock()
	cfg.Gateway = c
}

func (cfg *Configs) GetGatewayCfg() GatewaysMgr {
	cfg.Lock.Lock()
	defer cfg.Lock.Unlock()
	return *cfg.Gateway
}

func (cfg *Configs) SetSystemConfig(c *SystemConfig) {
	cfg.Lock.Lock()
	defer cfg.Lock.Unlock()
	cfg.System = c
}

func (cfg *Configs) GetSystemConfig() SystemConfig {
	cfg.Lock.Lock()
	defer cfg.Lock.Unlock()
	return *cfg.System
}

func (cfg *Configs) SetDefaultPool(symbol string, cm *Pool) {
	cfg.Lock.Lock()
	defer cfg.Lock.Unlock()
	cfg.Pool.Default[symbol] = cm
}

func (cfg *Configs) SetAgentPool(agentId string, cm *Pool) {
	cfg.Lock.Lock()
	defer cfg.Lock.Unlock()

	cfg.Pool.Agent[fmt.Sprintf("%s-%s", agentId, cm.Symbol)] = cm
}

func (cfg *Configs) SetCurrency(cm *CurrencyMgr) {
	cfg.Lock.Lock()
	defer cfg.Lock.Unlock()
	cfg.Currency = cm
}

func (cfg *Configs) SetAutoCtrl(ac *AutoCtrlMgr) {
	cfg.Lock.Lock()
	defer cfg.Lock.Unlock()
	cfg.AC = ac
}

func (cfg *Configs) SetControlCoefficient(c *ControlCoefficientConfig) {
	cfg.Lock.Lock()
	defer cfg.Lock.Unlock()
	cfg.Coefficient = c
}

func (cfg *Configs) IsSingleControlEnabled(roundCount decimal.Decimal) bool {
	cfg.Lock.RLock()
	c := cfg.Coefficient
	cfg.Lock.RUnlock()
	if c == nil {
		c = DefaultControlCoefficientConfig()
	}
	return roundCount.GreaterThan(c.SingleRounds)
}

func (cfg *Configs) SetCtrl(symbol string, ac *AwardConfig) {
	cfg.Lock.Lock()
	defer cfg.Lock.Unlock()

	cfg.Award.Data[symbol] = ac
}

func (cfg *Configs) GetAutoCtrl(effectBet, profitLoss decimal.Decimal) *AutoCtrlItem {
	cfg.Lock.RLock()
	defer cfg.Lock.RUnlock()
	if cfg.AC == nil || effectBet.Equal(decimal.Zero) {
		return nil
	}
	var validAsc *AutoCtrlItem = nil
	rate := profitLoss.Div(effectBet).Mul(decimal.NewFromInt(100))
	for _, v := range cfg.AC.Ctrls {
		if v == nil {
			continue
		}
		if v.TotalEffect.LessThanOrEqual(effectBet) &&
			v.TotalProfLoss.LessThanOrEqual(profitLoss) &&
			v.TotalProfLossRate.LessThanOrEqual(rate) {
			if validAsc == nil {
				validAsc = v
			} else if validAsc.ControlRate.Abs().LessThan(v.ControlRate.Abs()) {
				validAsc = v
			}
		}
	}
	return validAsc
}

func (cfg *Configs) GetControlCoefficient(averageBet, effectBet, profitLoss, roundCount decimal.Decimal, singleControl bool) decimal.Decimal {
	cfg.Lock.RLock()
	c := cfg.Coefficient
	cfg.Lock.RUnlock()
	if c == nil {
		c = DefaultControlCoefficientConfig()
	}
	coefficient := decimal.NewFromInt(1)
	for _, item := range c.Bet {
		if item == nil || averageBet.LessThanOrEqual(item.Min) || (item.Max.GreaterThan(decimal.Zero) && averageBet.GreaterThan(item.Max)) {
			continue
		}
		coefficient = item.Coefficient
		break
	}
	if singleControl && roundCount.GreaterThan(c.SingleRounds) && !effectBet.Equal(decimal.Zero) {
		ratio := profitLoss.Div(effectBet)
		for _, item := range c.Single {
			if item != nil && ratio.GreaterThan(item.Min) && item.Coefficient.LessThan(coefficient) {
				coefficient = item.Coefficient
			}
		}
	}
	if coefficient.LessThan(decimal.Zero) {
		return decimal.Zero
	}
	if coefficient.GreaterThan(decimal.NewFromInt(1)) {
		return decimal.NewFromInt(1)
	}
	return coefficient
}

func (cfg *Configs) GetPoolCfg(agentId int64, symbol string) *Pool {
	cfg.Lock.RLock()
	defer cfg.Lock.RUnlock()
	if cfg.Pool == nil {
		return nil
	}
	if p := cfg.Pool.Agent[fmt.Sprintf("%d-%s", agentId, symbol)]; p != nil {
		return p
	}
	if p := cfg.Pool.Agent[fmt.Sprintf("%d_%s", agentId, symbol)]; p != nil {
		return p
	}
	return cfg.Pool.Default[symbol]
}

func (cfg *Configs) GetPoolDefaultCfg(symbol string) *Pool {
	cfg.Lock.RLock()
	defer cfg.Lock.RUnlock()
	if cfg.Pool == nil {
		return nil
	}
	return cfg.Pool.Default[symbol]
}

func (cfg *Configs) GetPoolCfgByGameId(agentId, gameId int64) *Pool {
	cfg.Lock.RLock()
	defer cfg.Lock.RUnlock()
	if cfg.Pool == nil {
		return nil
	}
	for _, c := range cfg.Pool.Agent {
		if c != nil && c.GameId == gameId {
			return c
		}
	}
	for _, c := range cfg.Pool.Default {
		if c != nil && c.GameId == gameId {
			return c
		}
	}
	return nil
}

func (cfg *Configs) GetPoolDefaultCfgByGameId(gameId int64) *Pool {
	cfg.Lock.RLock()
	defer cfg.Lock.RUnlock()
	if cfg.Pool == nil {
		return nil
	}
	for _, c := range cfg.Pool.Default {
		if c != nil && c.GameId == gameId {
			return c
		}
	}
	return nil
}

func (cfg *Configs) GetExchange(currencyType string) (decimal.Decimal, bool) {
	cfg.Lock.RLock()
	defer cfg.Lock.RUnlock()
	exchange, ok := cfg.Currency.Data[currencyType]
	return exchange, ok
}

func (cfg *Configs) GetAwardOddsSingleCtrlConfig(symbol string) *AwardOddsConfigItem {
	cfg.Lock.RLock()
	defer cfg.Lock.RUnlock()
	var res *AwardOddsConfigItem = nil
	if items, ok := cfg.Award.Data["default"]; ok {
		for _, v := range items.OddsConfig {
			if v.Name == "single" {
				res = v
			}
		}
	}
	return res
}

func (cfg *Configs) GetAwardOddsDefaultCtrlConfig(symbol string) *AwardOddsConfigItem {
	cfg.Lock.RLock()
	defer cfg.Lock.RUnlock()
	var res *AwardOddsConfigItem = nil
	if items, ok := cfg.Award.Data["default"]; ok {
		for _, v := range items.OddsConfig {
			if v.Name == "default" {
				res = v
			}
		}
	}
	return res
}

// 根据盈亏比获取奖励配置
func (cfg *Configs) GetAwardOddsConfigWithProfitOdds(symbol string, odds decimal.Decimal) *AwardOddsConfigItem {
	cfg.Lock.RLock()
	defer cfg.Lock.RUnlock()
	var res *AwardOddsConfigItem = nil
	if items, ok := cfg.Award.Data["default"]; ok {
		for _, v := range items.OddsConfig {
			if v.Name == "single" {
				continue
			}
			if v.Min.LessThanOrEqual(odds) && v.Max.GreaterThan(odds) {
				res = v
			}
		}
	}
	return res
}

type DataItem struct {
	UserId  int             `json:"userId"`
	GameId  int             `json:"gameId"`
	AgentId int             `json:"agentId"`
	Eff     decimal.Decimal `json:"bet"`
	Pro     decimal.Decimal `json:"win"`
}

type RankResult struct {
	AgentId uint32
	UserId  uint32
	Eff     decimal.Decimal
	Pro     decimal.Decimal
}
