package manager

//
type Web struct {
	Id          int64  `gorm:"column:id;primaryKey;autoIncrement;not null;" json:"id"`
	Lable       string `gorm:"column:lable;size:30;comment:站点标识;" json:"lable"`
	NickName    string `gorm:"column:nickName;size:50;index:web_name;comment:站点名称;" json:"nickName"`
	Email       string `gorm:"column:email;size:50;comment:邮箱" json:"email"`
	Contacts    string `gorm:"column:contacts;size:50;comment:负责人" json:"contacts"`
	Phone       string `gorm:"column:phone;size:50;comment:负责人联系方式" json:"phone"`
	RealmName   string `gorm:"column:realmName;size:500;comment:域名" json:"realmName"`
	Remarks     string `gorm:"column:remarks;size:500;comment:备注;" json:"remarks"`
	StartTime   int64  `gorm:"column:startTime;index:web_time;comment:生效开始时间;" json:"startTime"`
	EndTime     int64  `gorm:"column:endTime;index:web_time;comment:生效结束时间;" json:"endTime"`
	IsPermanent byte   `gorm:"column:isPermanent;comment:是否永久 1是 0否;" json:"isPermanent"`
	IsFrozen    byte   `gorm:"column:isFrozen;comment:是否冻结 1是 0否" json:"isFrozen"`
	State       byte   `gorm:"column:state;comment:状态 1正常 2维护" json:"state"`
	CreateTime  int64  `gorm:"column:createTime;comment:创建时间" json:"createTime"`
	UpdataTime  int64  `gorm:"column:updataTime;comment:修改时间" json:"updataTime"`
	IsDel       byte   `gorm:"column:isDel;comment:是否删除 1是 0否" json:"isDel"`
}

func (t *Web) TableName() string {
	return "gp_web"
}
