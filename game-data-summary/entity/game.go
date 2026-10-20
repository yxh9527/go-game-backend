package entity

// 游戏
type Game struct {
	Id         int64  `gorm:"column:id;primaryKey;autoIncrement;" json:"id"`
	Number     int    `gorm:"column:number" json:"number"`
	Name       string `gorm:"column:name" json:"name"`
	NameZH     string `gorm:"column:nameZH" json:"nameZH"`
	ConfName   string `gorm:"column:confName" json:"-"`
	GameClass  int    `gorm:"column:gameClass" json:"-"`
	GameType   int    `gorm:"column:gameType" json:"-"`
	LimitTime  string `gorm:"column:limitTime" json:"-"`
	IsFrozen   int16  `gorm:"column:isFrozen" json:"-"`
	State      int16  `gorm:"column:state" json:"-"`
	CreateTime int    `gorm:"column:createTime" json:"-"`
	UpdateTime int    `gorm:"column:updateTime" json:"-"`
	Weight     int    `gorm:"column:weight" json:"-"`
	ShowType   int    `gorm:"column:showType" json:"-"`
	// Gamer      int    `gorm:"column:gamer"`
	IsShow int `gorm:"column:isShow" json:"-"`
}

func (t *Game) TableName() string {
	return "gp_game"
}
