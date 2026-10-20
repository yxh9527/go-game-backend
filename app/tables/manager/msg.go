package manager

//
type Msg struct {
	Id         int64  `gorm:"column:id;primaryKey;autoIncrement;not null;"`
	Number     string `gorm:"column:number;size:20;comment:消息序号"`
	Title      string `gorm:"column:title;size:50;comment:消息标题"`
	MsgType    int64  `gorm:"column:msgType;comment:消息类型"`
	StartTime  int    `gorm:"column:startTime;comment:发布时间"`
	EndTime    int    `gorm:"column:endTime;comment:停止时间"`
	ReceiveIds string `gorm:"column:receiveIds;size:1000;comment:收件ID集合"`
	WebId      int64  `gorm:"column:webId;comment:站点ID"`
	GameId     int64  `gorm:"column:gameId;comment:游戏ID"`
	AgentId    int64  `gorm:"column:agentId;comment:代理ID"`
	Info       string `gorm:"column:info;comment:消息内容"`
	Remarks    string `gorm:"column:remarks;comment:备注"`
	CreateTime int64  `gorm:"column:createTime;comment:创建时间"`
	SysUserId  int64  `gorm:"column:sysUserId;comment:创建人ID"`
	UpdateTime int64  `gorm:"column:updateTime;comment:更新时间"`
	MsgClass   int    `gorm:"column:msgClass;comment:1游戏消息 2站内消息;"`
}

func (t *Msg) TableName() string {
	return "gp_msg"
}
