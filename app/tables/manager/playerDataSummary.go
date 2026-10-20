package manager

// 游戏
type PlayerDataSummary struct {
	RecordTime int64 `gorm:"column:record_time;uniqueIndex:recordTime_agentId;" json:"record_time"`
	Regist     int   `gorm:"column:regist;" json:"regist"`
	AllActive  int   `gorm:"column:all_active;" json:"all_active"`
	NewActive  int   `gorm:"column:new_active;" json:"new_active"`
	AgentId    int   `gorm:"column:agent_id;uniqueIndex:recordTime_agentId;" json:"agent_id"`
	WebId      int   `gorm:"column:web_id" json:"web_id"`
}

func (t *PlayerDataSummary) TableName() string {
	return "gp_player_summary"
}
