package view

import (
	"app/tables/manager"

	"github.com/shopspring/decimal"
)

type UserAndGameSummaryItem struct {
	User []*manager.PlayerDataSummary `json:"user"`
	Game []*manager.GameDataSummary   `json:"game"`
}

type MonthData struct {
	Cur  *UserAndGameSummaryItem `json:"cur"`
	Prev *UserAndGameSummaryItem `json:"prev"`
}

type WeekData struct {
	Cur  *UserAndGameSummaryItem `json:"cur"`
	Prev *UserAndGameSummaryItem `json:"prev"`
	Next *UserAndGameSummaryItem `json:"next"`
}

type UserAndGameSummaryResult struct {
	Month *MonthData `json:"month"`
	Week  *WeekData  `json:"week"`
}

type UserAndGameSummaryDetailResult struct {
	User []*manager.PlayerDataSummary `json:"user"`
	Game []*manager.GameDataSummary   `json:"game"`
}

type AutoSingleCtrl struct {
	TotalEffect       decimal.Decimal `json:"totalEffect"`       // 打吗
	TotalProfLoss     decimal.Decimal `json:"totalProfLoss"`     //yk
	TotalProfLossRate decimal.Decimal `json:"totalProfLossRate"` //ykb
	ControlRate       decimal.Decimal `json:"controlRate"`       //kz比例
	Score             decimal.Decimal `json:"score"`             //kz积分
}
