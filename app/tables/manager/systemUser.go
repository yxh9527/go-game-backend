package manager

//
type SystemUser struct {
	Id         int64  `gorm:"column:id;primaryKey;autoIncrement;not null;" json:"id"`
	Account    string `gorm:"column:account;size:20;comment:账号;" json:"account"`
	Password   string `gorm:"column:password;size:32;comment:游戏id;" json:"-"`
	UType      int    `gorm:"column:uType;comment:1总控账号 2信息账号 3代理账号" json:"uType"`
	AgentId    int64  `gorm:"column:agentId;comment:代理id" json:"agentId"`
	UName      string `gorm:"column:uName;size:30;comment:账号名" json:"uName"`
	IpLimit    string `gorm:"column:ipLimit;default:*;comment:ip限制" json:"ipLimit"`
	CreateTime int64  `gorm:"column:createTime;comment:创建时间;" json:"createTime"`
	LoginTime  int64  `gorm:"column:loginTime;comment:登陆时间;" json:"loginTime"`
	LoginNum   int64  `gorm:"column:loginNum;comment:登陆次数;" json:"loginNum"`
	IsForzen   int    `gorm:"column:isForzen;comment:是否冻结1是 0否;" json:"isForzen"`
}

func (t *SystemUser) TableName() string {
	return "gp_system_user"
}
