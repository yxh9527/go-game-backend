package manager

//
type SystemUserMsg struct {
	Id         int64  `gorm:"column:id;primaryKey;autoIncrement;not null;"`
	ReceiveId  int64  `gorm:"column:receiveId;index:system_user_msg;comment:接收人id;"`
	MsgId      int64  `gorm:"column:msgId;size:32;comment:消息id;"`
	Title      string `gorm:"column:title;comment:标题"`
	Info       string `gorm:"column:info;comment:内容"`
	IsRead     byte   `gorm:"column:isRead;index:system_user_msg;comment:是否阅读 1是 0否"`
	CreateTime int64  `gorm:"column:createTime;comment:创建时间;"`
	IsDel      byte   `gorm:"column:isDel;index:system_user_msg;comment:是否删除 1是 0否;"`
	SendId     int64  `gorm:"column:sendId;comment:发件人id;"`
}

func (t *SystemUserMsg) TableName() string {
	return "gp_system_user_msg"
}
