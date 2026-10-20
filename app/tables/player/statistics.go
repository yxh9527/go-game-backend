package player

type Statistics struct {
	Uid       int64 `gorm:"column:uid;primaryKey;" json:"uid"`   //userId
	Gid       int64 `gorm:"column:gid;primaryKey;" json:"gid"`   //gameId
	AreaId    byte  `gorm:"column:aid;primaryKey;" json:"aid"`   //areaId
	Times     int32 `gorm:"column:times" json:"times"`           //本级别手数
	BetTimes  int32 `gorm:"column:bet_times" json:"bet_times"`   //下注入池手数
	IncrTimes int32 `gorm:"column:incr_times" json:"incr_times"` //加注入池手数
}

func (p *Statistics) TableName() string {
	return "statistics"
}
