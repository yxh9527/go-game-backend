package manager

import "github.com/shopspring/decimal"

// 游戏
type GameDataSummary struct {
	RecordTime   int64           `gorm:"column:record_time;uniqueIndex:recordTime_agentId_gameId;" json:"record_time"`
	AgentId      int64           `gorm:"column:agent_id;uniqueIndex:recordTime_agentId_gameId;" json:"agent_id"`
	GameId       int             `gorm:"column:game_id;uniqueIndex:recordTime_agentId_gameId;" json:"game_id"`
	WebId        int             `gorm:"column:web_id;" json:"web_id"`
	Eff          decimal.Decimal `gorm:"column:eff" json:"eff"`
	Pro          decimal.Decimal `gorm:"column:pro" json:"pro"`
	Ss           decimal.Decimal `gorm:"column:ss" json:"ss"`
	KeepAlive    int             `gorm:"column:keep_alive" json:"keep_alive"`
	NotKeepAlive int             `gorm:"column:not_keep_alive" json:"not_keep_alive"`
	ActiveCont   int             `gorm:"column:active_cont" json:"active_cout"`
	BetsCount    int             `gorm:"column:bets_count" json:"bets_count"`
}

func (t *GameDataSummary) TableName() string {
	return "gp_game_data_summary"
}
