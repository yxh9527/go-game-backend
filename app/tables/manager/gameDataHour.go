package manager

import "github.com/shopspring/decimal"

// 游戏
type GameDataHour struct {
	RecordTime int64           `gorm:"column:record_time;uniqueIndex:record_time_game_id;"`
	GameId     int             `gorm:"column:game_id;uniqueIndex:record_time_game_id;"`
	Eff        decimal.Decimal `gorm:"column:eff"`
	Pro        decimal.Decimal `gorm:"column:pro"`
	Cnt        int             `gorm:"column:cnt"`
	Active     int             `gorm:"column:active"`
}

func (t *GameDataHour) TableName() string {
	return "gp_game_data_hour"
}
