package manager

import "github.com/shopspring/decimal"

//
type Statistics struct {
	Id               int64           `gorm:"column:id;primaryKey;autoIncrement;not null;"`
	AgentId          int64           `gorm:"column:agentId;comment:代理id;index:statistics_agentId;"`
	GameId           int64           `gorm:"column:gameId;comment:游戏id;index:statistics_gameId;"`
	Difficulty       int             `gorm:"column:difficulty;comment:难度类型"`
	NewUserNumber    int64           `gorm:"column:newUserNumber;comment:新增"`
	ActiveUserNumber int64           `gorm:"column:activeUserNumber;comment:活跃"`
	GameNumber       int64           `gorm:"column:gameNumber;comment:场次"`
	EffectiveBet     decimal.Decimal `gorm:"column:effectiveBet;comment:effect bet;"`
	Bet              decimal.Decimal `gorm:"column:bet;comment:bet;"`
	ProfitLoss       decimal.Decimal `gorm:"column:profitLoss;comment:profitLoss;"`
	Pump             decimal.Decimal `gorm:"column:pump;comment:cs;"`
	CreateTime       int64           `gorm:"column:createTime;comment:统计时间;index:statistics_createTime;"`
}

func (t *Statistics) TableName() string {
	return "gp_statistics"
}
