package player

type GameType struct {
	Id         int64  `gorm:"column:id;primaryKey;autoIncrement;" json:"id"` //id
	Name       string `gorm:"column:name;size:128;" json:"name"`             //name
	CreateTime int64  `gorm:"column:create_time" json:"create_time"`
}

func (p *GameType) TableName() string {
	return "game_type"
}
