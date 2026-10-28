package gogame

// Match 比赛配置与运行状态。进入进行中后开赛时间、报名费、奖励结构锁定。
type Match struct {
	Id                 int64  `gorm:"column:id;primaryKey;autoIncrement;" json:"id"`
	AgentId            int64  `gorm:"column:agent_id;index;default:0;" json:"agent_id"`
	ChampionshipName   string `gorm:"column:championship_name;type:text;" json:"championship_name"`
	Consults           string `gorm:"column:consults;type:text;" json:"consults"`
	GameType           int32  `gorm:"column:game_type;index;default:19;" json:"game_type"`
	RoomType           int32  `gorm:"column:room_type;default:101;" json:"room_type"`
	ChampionshipType   int32  `gorm:"column:championship_type;default:1;" json:"championship_type"`
	RegisterType       int32  `gorm:"column:register_type;default:4;" json:"register_type"`
	RegisterGold       int64  `gorm:"column:register_gold;default:0;" json:"register_gold"`
	ServiceGold        int64  `gorm:"column:service_gold;default:0;" json:"service_gold"`
	Currency           string `gorm:"column:currency;size:16;default:CNY;" json:"currency"`
	RegisterTicket     int32  `gorm:"column:register_ticket;default:0;" json:"register_ticket"`
	TicketId           int64  `gorm:"column:ticket_id;default:0;" json:"ticket_id"`
	AllowTicketPay     int32  `gorm:"column:allow_ticket_pay;default:1;" json:"allow_ticket_pay"`
	TicketValue        int64  `gorm:"column:ticket_value;default:0;" json:"ticket_value"`
	TicketName         string `gorm:"column:ticket_name;size:128;" json:"ticket_name"`
	InitPoint          int64  `gorm:"column:init_point;default:0;" json:"init_point"`
	MinPlayer          int32  `gorm:"column:min_player;default:2;" json:"min_player"`
	MaxPlayer          int32  `gorm:"column:max_player;default:0;" json:"max_player"`
	DelayRegisterLevel int32  `gorm:"column:delay_register_level;default:0;" json:"delay_register_level"`
	DelayRegisterTime  int32  `gorm:"column:delay_register_time;default:0;" json:"delay_register_time"`
	RegisterTimeLimit  int32  `gorm:"column:register_time_limit;default:0;" json:"register_time_limit"`
	RestTime           int32  `gorm:"column:rest_time;default:0;" json:"rest_time"`
	RestTimeInterval   int32  `gorm:"column:rest_time_interval;default:0;" json:"rest_time_interval"`
	IncreaseBindTime   int32  `gorm:"column:increase_bind_time;default:0;" json:"increase_bind_time"`
	RebuyTime          int32  `gorm:"column:rebuy_time;default:0;" json:"rebuy_time"`
	MaxBuy             int32  `gorm:"column:max_buy;default:0;" json:"max_buy"`
	ForceShowCard      int32  `gorm:"column:force_show_card;default:1;" json:"force_show_card"`
	FixPot             int32  `gorm:"column:fix_pot;default:0;" json:"fix_pot"`
	TotalPot           int64  `gorm:"column:total_pot;default:0;" json:"total_pot"`
	FreeTypePot        int64  `gorm:"column:free_type_pot;default:0;" json:"free_type_pot"`
	BonusPot           int64  `gorm:"column:bonus_pot;default:0;" json:"bonus_pot"`
	BountyPot          int64  `gorm:"column:bounty_pot;default:0;" json:"bounty_pot"`
	BlindsId           int64  `gorm:"column:blinds_id;default:0;" json:"blinds_id"`
	BlindType          int32  `gorm:"column:blind_type;default:2;" json:"blind_type"`
	AwardId            int64  `gorm:"column:award_id;default:0;" json:"award_id"`
	AwardJSON          string `gorm:"column:award_json;type:text;" json:"award_json"`
	State              int32  `gorm:"column:state;index;default:0;" json:"state"`
	StartTime          int64  `gorm:"column:start_time;index;default:0;" json:"start_time"`
	EndTime            int64  `gorm:"column:end_time;default:0;" json:"end_time"`
	SortIndex          int32  `gorm:"column:sort_index;default:0;" json:"sort_index"`
	CurrentPlayer      int32  `gorm:"column:current_player;default:0;" json:"current_player"`
	CurrentLivePlayer  int32  `gorm:"column:current_live_player;default:0;" json:"current_live_player"`
	BlindLevel         int32  `gorm:"column:blind_level;default:0;" json:"blind_level"`
	BigBlind           int64  `gorm:"column:big_blind;default:0;" json:"big_blind"`
	SmallBlind         int64  `gorm:"column:small_blind;default:0;" json:"small_blind"`
	SettlementType     int32  `gorm:"column:settlement_type;default:0;" json:"settlement_type"`
	Domain             string `gorm:"column:domain;size:512;" json:"domain"`
	Shield             string `gorm:"column:shield;size:128;" json:"shield"`
	CreateTime         int64  `gorm:"column:create_time;default:0;" json:"create_time"`
	UpdateTime         int64  `gorm:"column:update_time;default:0;" json:"update_time"`
}

func (t *Match) TableName() string {
	return "gg_match"
}

func (t *Match) RunningLocked() bool {
	return t.State == MatchStateRunning || t.State == MatchStateFinished
}
