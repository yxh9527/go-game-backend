package view

import "github.com/shopspring/decimal"

type ResetPoolConfig struct {
	Now       bool  `json:"now"`
	Interval  int   `json:"interval"`
	ResetTime int64 `json:"resetTime"`
}

type CustomConfig struct {
	WaitTime          interface{} `json:"wait_time,omitempty"`
	BetLimits         interface{} `json:"bet_limits,omitempty"`
	OneGroupWeight    interface{} `json:"one_group_weight,omitempty"`
	TwoGroupWeight    interface{} `json:"two_group_weight,omitempty"`
	ThreeGroupWeight  interface{} `json:"three_group_weight,omitempty"`
	FourGroupWeight   interface{} `json:"four_group_weight,omitempty"`
	FiveGroupWeight   interface{} `json:"five_group_weight,omitempty"`
	SixGroupWeight    interface{} `json:"six_group_weight,omitempty"`
	LowerMultiple     interface{} `json:"lower_multiple,omitempty"`
	BetTable          interface{} `json:"bet_table,omitempty"`
	SpecialModeWeight interface{} `json:"special_mode_weight,omitempty"`
	AwardConfig       interface{} `json:"award_config"`
}

type GameListConfig struct {
	GameList []int `json:"gameList"`
	ShowType struct {
		Hot    []int `json:"hot"`
		New    []int `json:"new"`
		Expect []int `json:"expect"`
	} `json:"showType"`
	Maintain []int `json:"maintain"`
}

type UserSingleControl struct {
	Rate       decimal.Decimal `json:"rate"`
	UserId     int64           `json:"userId"`
	RateScore  decimal.Decimal `json:"rate_score"`
	LeftScore  decimal.Decimal `json:"left_score"`
	CreateTime int64           `json:"createTime,omitempty"`
	CtrlType   int             `json:"ctrl_type"` //0 自动控制 1 管理员控制
}
