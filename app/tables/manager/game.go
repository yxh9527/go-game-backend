package manager

// 游戏
type Game struct {
	Id             int64   `gorm:"column:id;primaryKey;autoIncrement;" json:"id"`
	Number         int     `gorm:"column:number;uniqueIndex:uq_gp_game_number" json:"number"`
	Name           string  `gorm:"column:name" json:"name"`
	NameZH         string  `gorm:"column:nameZH" json:"nameZH"`
	ConfName       string  `gorm:"column:confName" json:"-"`
	GameClass      int     `gorm:"column:gameClass" json:"-"`
	GameType       int     `gorm:"column:gameType" json:"-"`
	ClientGameType *uint32 `gorm:"column:clientGameType;type:int unsigned;uniqueIndex:uq_gp_game_client_room,priority:1" json:"clientGameType,omitempty"`
	RoomType       *uint32 `gorm:"column:roomType;type:int unsigned;uniqueIndex:uq_gp_game_client_room,priority:2" json:"roomType,omitempty"`
	LimitTime      string  `gorm:"column:limitTime" json:"-"`
	IsFrozen       int16   `gorm:"column:isFrozen" json:"isFrozen"`
	State          int16   `gorm:"column:state" json:"state"`
	CreateTime     int     `gorm:"column:createTime" json:"-"`
	UpdateTime     int     `gorm:"column:updateTime" json:"-"`
	Weight         int     `gorm:"column:weight" json:"-"`
	ShowType       int     `gorm:"column:showType" json:"-"`
	IsShow         int     `gorm:"column:isShow" json:"-"`
}

func (t *Game) TableName() string {
	return "gp_game"
}
