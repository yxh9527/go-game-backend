package gogame

// RoomConfig 大厅 op=38 房间配置。Payload 存客户端 roomInfoData 对象 JSON。
type RoomConfig struct {
	Id               int64  `gorm:"column:id;primaryKey;autoIncrement;" json:"id"`
	GameType         int32  `gorm:"column:game_type;index;default:0;" json:"game_type"`
	SubGameType      int32  `gorm:"column:sub_game_type;default:0;" json:"sub_game_type"`
	RoomType         int32  `gorm:"column:room_type;index;default:0;" json:"room_type"`
	GameRoomConfigId int64  `gorm:"column:game_room_config_id;index;default:0;" json:"game_room_config_id"`
	PlayMode         int32  `gorm:"column:play_mode;default:0;" json:"play_mode"`
	RoomIndex        int32  `gorm:"column:room_index;default:0;" json:"room_index"`
	SmallBlind       int64  `gorm:"column:small_blind;default:0;" json:"small_blind"`
	BigBlind         int64  `gorm:"column:big_blind;default:0;" json:"big_blind"`
	CarryMin         int64  `gorm:"column:carry_min;default:0;" json:"carry_min"`
	HundredMinBet    int64  `gorm:"column:hundred_min_bet;default:0;" json:"hundred_min_bet"`
	HundredMaxBet    int64  `gorm:"column:hundred_max_bet;default:0;" json:"hundred_max_bet"`
	ForceShowCard    int32  `gorm:"column:force_show_card;default:0;" json:"force_show_card"`
	Category         string `gorm:"column:category;size:32;index;" json:"category"`
	Name             string `gorm:"column:name;size:128;" json:"name"`
	Payload          string `gorm:"column:payload;type:text;" json:"payload"`
	State            int32  `gorm:"column:state;default:1;" json:"state"`
	SortIndex        int32  `gorm:"column:sort_index;default:0;" json:"sort_index"`
	CreateTime       int64  `gorm:"column:create_time;default:0;" json:"create_time"`
	UpdateTime       int64  `gorm:"column:update_time;default:0;" json:"update_time"`
}

func (t *RoomConfig) TableName() string {
	return "gg_room_config"
}
