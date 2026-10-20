package player

type MatchAward struct {
	Id         int64  `gorm:"column:id;primaryKey;autoIncrement;" json:"id"` //id
	Name       string `gorm:"column:name;size:128;" json:"name"`             //name
	CreateTime int64  `gorm:"column:create_time" json:"create_time"`
	Detail     string `gorm:"column:detail" json:"detail"` //用逗号分隔 奖励分配比例
}

func (p *MatchAward) TableName() string {
	return "match_award"
}
