package manager

import "github.com/shopspring/decimal"

//
type ProfitLoos struct {
	Id         int64           `gorm:"column:id;primaryKey;autoIncrement;not null;"`
	AgentId    int64           `gorm:"column:agentId;comment:代理id;index:profit_loss_agentId;"`
	GameId     int64           `gorm:"column:gameId;comment:游戏id;index:profit_loss_gameId;"`
	Difficulty int             `gorm:"column:difficulty;comment:难度类型"`
	ProfitLoss decimal.Decimal `gorm:"column:profitLoss;comment:盈亏"`
	PlTime     int             `gorm:"column:plTime;comment:时间"`
	CreateTime int             `gorm:"column:createTime;"`
}

func (t *ProfitLoos) TableName() string {
	return "gp_profit_loss"
}
