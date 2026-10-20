package manager

//
type SystemConf struct {
	Id          int64 `gorm:"column:id;primaryKey;autoIncrement;not null;"`
	SystemState int   `gorm:"column:systemState"`
}

func (t *SystemConf) TableName() string {
	return "gp_system_conf"
}
