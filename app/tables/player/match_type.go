package player

type MatchType struct {
	Id         int64  `gorm:"column:id;primaryKey;autoIncrement;" json:"id"` //id
	Name       string `gorm:"column:name;size:128;" json:"name"`             //name
	Icon       string `gorm:"column:icon;size:256" json:"icon"`
	CreateTime int64  `gorm:"column:create_time" json:"create_time"`
}

func (p *MatchType) TableName() string {
	return "match_type"
}
