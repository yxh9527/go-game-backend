package manager

//
type PlayerDataHour struct {
	RecordTime  int64 `gorm:"column:record_time;uniqueIndex:record_time_game_id;"`
	ActiveCount int   `gorm:"column:active_count"`
}

func (t *PlayerDataHour) TableName() string {
	return "gp_player_data_hour"
}
