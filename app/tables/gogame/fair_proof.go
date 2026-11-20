package gogame

type FairProof struct {
	ID         string `gorm:"column:id;size:64;primaryKey"`
	AgentID    int64  `gorm:"column:agent_id;index:idx_fair_room,priority:1"`
	RoomID     int64  `gorm:"column:room_id;index:idx_fair_room,priority:2"`
	Inning     int64  `gorm:"column:inning;index:idx_fair_room,priority:3"`
	RoomType   int64  `gorm:"column:room_type"`
	Version    int64  `gorm:"column:version"`
	CreatedAt  int64  `gorm:"column:created_at;index"`
	Commitment string `gorm:"column:commitment;size:64"`
	Payload    string `gorm:"column:payload;type:mediumtext"`
}

func (*FairProof) TableName() string { return "gg_fair_proof" }
