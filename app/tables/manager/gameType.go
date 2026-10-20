package manager

// GameType 游戏分类
type GameType struct {
	Id   int64  `gorm:"column:id;primaryKey;autoIncrement;" json:"id"`
	Name string `gorm:"column:name;size:64;not null;" json:"name"`
}

func (t *GameType) TableName() string {
	return "gp_game_type"
}
