package view

import (
	"strings"

	"github.com/shopspring/decimal"
)

type Bill struct {
	AgentId        int64           `json:"agentId"`
	UserId         int64           `json:"userId"`
	GameId         int             `json:"gameId"`
	Symbol         string          `json:"symbol"`
	OfficeNumber   string          `json:"roundId"`
	Bets           decimal.Decimal `json:"bet"`          // 下注（负）；接口返回时 desc=返奖 置 0
	Award          decimal.Decimal `json:"award"`        // 返奖/到账（>=0）
	UserScore      decimal.Decimal `json:"currentScore"` // 账变后余额
	FlowingWaterOn string          `json:"flowingWaterOn"`
	CreatTime      int64           `json:"createTime"`
	CurrencyType   string          `json:"currency"`
	GameName       string          `json:"gameName"`
	Msg            string          `json:"desc"`
}

// Normalize 对齐流水读约定：award 为返奖；desc=返奖 时下注置 0。
func (b *Bill) Normalize() {
	if b == nil {
		return
	}
	if b.Award.LessThan(decimal.Zero) {
		b.Award = decimal.Zero
	}
	if strings.TrimSpace(b.Msg) != "返奖" {
		return
	}
	b.Bets = decimal.Zero
}
