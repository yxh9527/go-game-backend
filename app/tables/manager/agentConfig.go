package manager

import (
	"github.com/shopspring/decimal"
)

// 代理配置
type AgentConfig struct {
	Id               int64           `gorm:"column:id;primaryKey;autoIncrement;"`
	AgentId          int64           `gorm:"column:agentId;index:index_agentId"`
	WarningPoint     decimal.Decimal `gorm:"column:warningPoint"`
	StopPoint        decimal.Decimal `gorm:"column:stopPoint"`
	LowerLevelNumber int             `gorm:"column:lowerLevelNumber;"`
	RealmName        string          `gorm:"column:realmName"`
	CreateTime       int             `gorm:"column:createTime"`
	UpdateTime       int64           `gorm:"column:updateTime"`
	IsDel            byte            `gorm:"column:isDel"`
	ChackBackUrl     string          `gorm:"column:chackBackUrl"`
	Pomp             decimal.Decimal `gorm:"column:pomp"`
}

func (t *AgentConfig) TableName() string {
	return "gp_agent_conf"
}
