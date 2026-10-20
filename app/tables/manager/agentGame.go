package manager

// 代理配置
type AgentGame struct {
	Id            int64 `gorm:"column:id;primaryKey;autoIncrement;" json:"id"`
	AgentId       int64 `gorm:"column:agentId;index:index_agentId" json:"agentId"`
	GameId        int   `gorm:"column:gameId" json:"gameId"`
	IsFrozen      int64 `gorm:"column:isFrozen" json:"isFrozen"`
	State         int   `gorm:"column:state" json:"state"`
	MainStartTime int   `gorm:"column:mainStartTime" json:"mainStartTime"`
	MainEndTime   int   `gorm:"column:mainEndTime" json:"mainEndTime"`
	CreateTime    int   `gorm:"column:creatTime" json:"creatTime"`
	UpdateTime    int   `gorm:"column:updateTime" json:"updateTime"`
}

func (t *AgentGame) TableName() string {
	return "gp_agent_game"
}
