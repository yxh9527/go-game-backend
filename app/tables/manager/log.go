package manager

//
type Log struct {
	Id         int64  `gorm:"column:id;primaryKey;autoIncrement;not null;"`
	LogType    int    `gorm:"column:logType"`
	Source     string `gorm:"column:source;size:50"`
	SysUserID  int64  `gorm:"column:sysUserID"`
	Url        string `gorm:"column:url"`
	Text       string `gorm:"column:text"`
	Ip         string `gorm:"column:ip;size:30;"`
	CreateTime int    `gorm:"column:createTime"`
}

func (t *Log) TableName() string {
	return "gp_log"
}
