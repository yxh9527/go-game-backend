package gogame

// MatchPlayer 参赛玩家。扣费成功后才写入。
type MatchPlayer struct {
	Id          int64  `gorm:"column:id;primaryKey;autoIncrement;" json:"id"`
	MatchId     int64  `gorm:"column:match_id;uniqueIndex:uk_match_player;default:0;" json:"match_id"`
	PlayerId    int64  `gorm:"column:player_id;uniqueIndex:uk_match_player;index;default:0;" json:"player_id"`
	Name        string `gorm:"column:name;size:128;" json:"name"`
	Avatar      string `gorm:"column:avatar;size:256;" json:"avatar"`
	Points      int64  `gorm:"column:points;default:0;" json:"points"`
	Rank        int32  `gorm:"column:rank;default:0;" json:"rank"`
	Status      int32  `gorm:"column:status;default:1;" json:"status"`
	RoomId      int64  `gorm:"column:room_id;default:0;" json:"room_id"`
	PaidGold    int64  `gorm:"column:paid_gold;default:0;" json:"paid_gold"`
	PaidService int64  `gorm:"column:paid_service;default:0;" json:"paid_service"`
	Currency    string `gorm:"column:currency;size:16;" json:"currency"`
	TicketUsed  int32  `gorm:"column:ticket_used;default:0;" json:"ticket_used"`
	DisuseTime  int64  `gorm:"column:disuse_time;default:0;" json:"disuse_time"`
	RewardJSON  string `gorm:"column:reward_json;type:text;" json:"reward_json"`
	CreateTime  int64  `gorm:"column:create_time;default:0;" json:"create_time"`
	UpdateTime  int64  `gorm:"column:update_time;default:0;" json:"update_time"`
}

func (t *MatchPlayer) TableName() string {
	return "gg_match_player"
}

// MatchRoom 比赛分桌，对应客户端 MTTRoomInfos。房间号会换桌。
type MatchRoom struct {
	Id         int64 `gorm:"column:id;primaryKey;autoIncrement;" json:"id"`
	MatchId    int64 `gorm:"column:match_id;index;default:0;" json:"match_id"`
	RoomId     int64 `gorm:"column:room_id;index;default:0;" json:"room_id"`
	RoomIndex  int32 `gorm:"column:room_index;default:0;" json:"room_index"`
	PlayerNums int32 `gorm:"column:player_nums;default:0;" json:"player_nums"`
	Status     int32 `gorm:"column:status;default:1;" json:"status"`
	UpdateTime int64 `gorm:"column:update_time;default:0;" json:"update_time"`
}

func (t *MatchRoom) TableName() string {
	return "gg_match_room"
}
