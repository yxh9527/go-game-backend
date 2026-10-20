package manager

//
type Feedback struct {
	Id         int64  `gorm:"column:id;primaryKey;autoIncrement;"`
	AgentId    int64  `gorm:"column:agentId;"`
	UserId     string `gorm:"column:userId;size:50;"`
	GameId     int64  `gorm:"column:gameId"`
	Difficulty byte   `gorm:"column:difficulty"`
	Msg        string `gorm:"column:msg"`
	CreateTime int64  `gorm:"column:createTime"`
	State      int64  `gorm:"column:state;comment:状态 0未处理 1已处理"`
}

func (t *Feedback) TableName() string {
	return "gp_feedback"
}
