package manager

//
type MsgType struct {
	Id    int64  `gorm:"column:id;primaryKey;autoIncrement;not null;"`
	Title string `gorm:"column:title;size:50;comment:消息标题"`
	Class int    `gorm:"column:class;comment:1游戏消息类型 2站内消息类型;"`
}

func (t *MsgType) TableName() string {
	return "gp_msg_type"
}
