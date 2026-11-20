package gogame

type SmartTask struct {
	ID        string  `gorm:"column:id;size:180;primaryKey"`
	AgentID   int64   `gorm:"column:agent_id;index:idx_smart_owner,priority:1"`
	UserID    int64   `gorm:"column:user_id;index:idx_smart_owner,priority:2"`
	RoomType  int64   `gorm:"column:room_type;index:idx_smart_owner,priority:3"`
	RoomID    int64   `gorm:"column:room_id"`
	ActiveKey *string `gorm:"column:active_key;size:128;uniqueIndex"`
	Version   int64   `gorm:"column:version"`
	Payload   string  `gorm:"column:payload;type:mediumtext"`
}

func (*SmartTask) TableName() string { return "gg_smart_task" }
