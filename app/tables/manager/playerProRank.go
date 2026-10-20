package manager

import "github.com/shopspring/decimal"

//
type PlayerProRank struct {
	AgentId    int             `gorm:"column:agent_id;uniqueIndex:;"`
	UserId     int             `gorm:"column:user_id;uniqueIndex:;"`
	RecordDate int64           `gorm:"record_date;uniqueIndex:;"`
	RecordType int             `gorm:"record_type;uniqueIndex:;"`
	Level      int             `gorm:"level"`
	IsWin      int             `gorm:"is_win"`
	Eff        decimal.Decimal `gorm:"eff"`
	Pro        decimal.Decimal `gorm:"pro"`
}

func (t *PlayerProRank) TableName() string {
	return "gp_player_pro_rank"
}
