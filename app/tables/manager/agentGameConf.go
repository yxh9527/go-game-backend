package manager

// 代理配置
type AgentGameConf struct {
	Id         int64 `gorm:"column:id;primaryKey;autoIncrement;"`
	AgentId    int64 `gorm:"column:agentId;index:index_agentId"`
	GameId     int   `gorm:"column:gameId"`
	Difficulty int   `gorm:"column:difficulty"`
	CreateTime int   `gorm:"column:creatTime"`
	IsControl  int   `gorm:"column:isControl"`
}

func (t *AgentGameConf) TableName() string {
	return "gp_agent_game_conf"
}
