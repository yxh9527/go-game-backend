package gogame

// PlayerPanelHand is immutable settled-hand reporting data, never a wallet ledger.
type PlayerPanelHand struct {
	RecordKey     string `gorm:"column:record_key;size:64;primaryKey"`
	AgentId       uint32 `gorm:"column:agent_id;index:idx_panel_account,priority:1"`
	PlayerId      uint32 `gorm:"column:player_id;index:idx_panel_account,priority:2"`
	Currency      string `gorm:"column:currency;size:12;index:idx_panel_account,priority:3"`
	RoomId        int64  `gorm:"column:room_id;index"`
	EndedAt       int64  `gorm:"column:ended_at;index:idx_panel_account,priority:4"`
	Net           int64  `gorm:"column:net"`
	Raised        bool   `gorm:"column:raised"`
	Voluntary     bool   `gorm:"column:voluntary"`
	CountersKnown bool   `gorm:"column:counters_known"`
	Payload       string `gorm:"column:payload;type:mediumtext"`
}

func (*PlayerPanelHand) TableName() string { return "gg_player_panel_hand" }

type PlayerNote struct {
	AgentId       uint32 `gorm:"column:agent_id;primaryKey"`
	PlayerId      uint32 `gorm:"column:player_id;primaryKey"`
	TargetId      uint32 `gorm:"column:target_id;primaryKey"`
	RemarkName    string `gorm:"column:remark_name;size:128"`
	RemarkMessage string `gorm:"column:remark_message;size:1024"`
	Mark          int    `gorm:"column:mark"`
	UpdatedAt     int64  `gorm:"column:updated_at"`
}

func (*PlayerNote) TableName() string { return "gg_player_note" }

type PlayerEmojiDaily struct {
	AgentId  uint32 `gorm:"column:agent_id;primaryKey"`
	PlayerId uint32 `gorm:"column:player_id;primaryKey"`
	Day      string `gorm:"column:day;size:10;primaryKey"`
	Used     int    `gorm:"column:used"`
}

func (*PlayerEmojiDaily) TableName() string { return "gg_player_emoji_daily" }
