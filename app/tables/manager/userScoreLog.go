package manager

type UserScoreLog struct {
	Id         int64   `gorm:"column:id;primaryKey;autoIncrement;not null;" json:"id"`
	LogOn      string  `gorm:"column:logOn;size:50;index:LogOnIndex;" json:"logOn"`
	AgentId    int64   `gorm:"column:agentId;" json:"agentId"`
	UserId     int64   `gorm:"column:userId;" json:"userId"`
	OpenId     string  `gorm:"column:openId;comment:第三方平台id;" json:"openId"`
	Score      float64 `gorm:"column:score;comment:分数;" json:"score"`
	LogType    int     `gorm:"column:logType;comment:1上分 2下分;" json:"logType"`
	UserScore  int     `gorm:"column:userScore;comment:用户上分后金额;" json:"userScore"`
	CreateTime int32   `gorm:"column:createtime;comment:注册时间" json:"createtime"`
}

func (t *UserScoreLog) TableName() string {
	return "gp_user_score_log"
}
