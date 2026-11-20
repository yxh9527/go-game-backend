package gogame

// Reuse the backend's existing durable game-storage table. No new financial
// journal or startup-replay table is introduced for scratch tickets.
type ScratchTicketStorage struct {
	ID             uint64 `gorm:"column:id;primaryKey;autoIncrement;"`
	UserId         uint32 `gorm:"column:user_id;not null;uniqueIndex:uk_game_storage;"`
	GameId         uint32 `gorm:"column:game_id;not null;uniqueIndex:uk_game_storage;"`
	CurrencySymbol string `gorm:"column:currency_symbol;size:32;not null;uniqueIndex:uk_game_storage;"`
	StorageKey     string `gorm:"column:storage_key;size:128;not null;uniqueIndex:uk_game_storage;"`
	Payload        string `gorm:"column:payload;type:longtext;"`
	ExpireAt       int64  `gorm:"column:expire_at;index;"`
	UpdatedAt      int64  `gorm:"column:updated_at;"`
}

func (ScratchTicketStorage) TableName() string { return "fg_game_storage" }
