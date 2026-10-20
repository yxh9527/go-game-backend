package view

import "github.com/shopspring/decimal"

type PoolItem struct {
	Min          decimal.Decimal `json:"min"`
	Normal       decimal.Decimal `json:"normal"`
	Max          decimal.Decimal `json:"max"`
	NormalRate   decimal.Decimal `json:"normalRate"`
	MinRate      decimal.Decimal `json:"minRate"`
	MaxRate      decimal.Decimal `json:"maxRate"`
	Control      decimal.Decimal `json:"control"`
	Revenue      decimal.Decimal `json:"revenue"`
	HighPayTimes decimal.Decimal `json:"highPayTimes"`
	HighPayGrown decimal.Decimal `json:"highPayGrown"`
	Avg          decimal.Decimal `json:"avg"`
	Offset       decimal.Decimal `json:"offset"`
	Base         decimal.Decimal `json:"base"`
}

type Pool struct {
	GameId int               `json:"gameId"`
	Data   map[int]*PoolItem `json:"pool"`
}

type PoolData struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
	Name  string      `json:"name"`
}

type PoolLogItem struct {
	AgentId    int             `json:"agentId"`
	Symbol     string          `json:"symbol"`
	PoolValue  float64         `json:"poolValue"`
	Normal     int             `json:"normal"`
	NormalRate decimal.Decimal `json:"normalRate"`
	Min        int             `json:"min"`
	MinRate    decimal.Decimal `json:"minRate"`
	Max        int             `json:"max"`
	MaxRate    decimal.Decimal `json:"maxRate"`
	Ctl        int             `json:"ctl"`
	Revenue    decimal.Decimal `json:"revenue"`
	CreateTime int64           `json:"createTime"`
}

type PoolLog struct {
	Data     []*PoolLogItem `json:"data"`
	Total    int            `json:"total"`
	PageSize int            `json:"pageSize"`
	Page     int            `json:"page"`
}
