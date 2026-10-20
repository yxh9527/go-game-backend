package dao

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GameStorageRecord 长期游戏恢复状态，权威落库，不走注单缓存链路。
type GameStorageRecord struct {
	ID             uint64 `gorm:"column:id;primaryKey;autoIncrement;"`
	UserId         uint32 `gorm:"column:user_id;not null;uniqueIndex:uk_game_storage;"`
	GameId         uint32 `gorm:"column:game_id;not null;uniqueIndex:uk_game_storage;"`
	CurrencySymbol string `gorm:"column:currency_symbol;size:32;not null;uniqueIndex:uk_game_storage;"`
	StorageKey     string `gorm:"column:storage_key;size:128;not null;uniqueIndex:uk_game_storage;"`
	Payload        string `gorm:"column:payload;type:longtext;"`
	ExpireAt       int64  `gorm:"column:expire_at;index;"`
	UpdatedAt      int64  `gorm:"column:updated_at;"`
}

func (GameStorageRecord) TableName() string {
	return "fg_game_storage"
}

var gameStorageMigrated bool

func (dd *DBDao) ensureGameStorageTable() error {
	if gameStorageMigrated {
		return nil
	}
	if dd.player == nil {
		return errors.New("player db is nil")
	}
	if err := dd.player.AutoMigrate(&GameStorageRecord{}); err != nil {
		return err
	}
	gameStorageMigrated = true
	return nil
}

func (dd *DBDao) SaveGameStorage(userId, gameId uint32, currencySymbol, storageKey, payload string, expireSeconds int64) error {
	if err := dd.ensureGameStorageTable(); err != nil {
		return err
	}
	currencySymbol = strings.TrimSpace(currencySymbol)
	storageKey = strings.TrimSpace(storageKey)
	if userId == 0 || gameId == 0 || currencySymbol == "" || storageKey == "" {
		return errors.New("invalid game storage key")
	}

	now := time.Now().Unix()
	expireAt := int64(0)
	if expireSeconds > 0 {
		expireAt = now + expireSeconds
	}

	rec := &GameStorageRecord{
		UserId:         userId,
		GameId:         gameId,
		CurrencySymbol: currencySymbol,
		StorageKey:     storageKey,
		Payload:        payload,
		ExpireAt:       expireAt,
		UpdatedAt:      now,
	}
	return dd.player.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "game_id"},
			{Name: "currency_symbol"},
			{Name: "storage_key"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"payload", "expire_at", "updated_at"}),
	}).Create(rec).Error
}

func (dd *DBDao) LoadGameStorage(userId, gameId uint32, currencySymbol, storageKey string) (string, bool, error) {
	if err := dd.ensureGameStorageTable(); err != nil {
		return "", false, err
	}
	currencySymbol = strings.TrimSpace(currencySymbol)
	storageKey = strings.TrimSpace(storageKey)
	if userId == 0 || gameId == 0 || currencySymbol == "" || storageKey == "" {
		return "", false, errors.New("invalid game storage key")
	}

	var rec GameStorageRecord
	err := dd.player.Where(
		"user_id = ? AND game_id = ? AND currency_symbol = ? AND storage_key = ?",
		userId, gameId, currencySymbol, storageKey,
	).Take(&rec).Error
	if err == gorm.ErrRecordNotFound {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if rec.ExpireAt > 0 && rec.ExpireAt < time.Now().Unix() {
		_ = dd.DeleteGameStorage(userId, gameId, currencySymbol, storageKey)
		return "", false, nil
	}
	return rec.Payload, true, nil
}

func (dd *DBDao) DeleteGameStorage(userId, gameId uint32, currencySymbol, storageKey string) error {
	if err := dd.ensureGameStorageTable(); err != nil {
		return err
	}
	currencySymbol = strings.TrimSpace(currencySymbol)
	storageKey = strings.TrimSpace(storageKey)
	if userId == 0 || gameId == 0 || currencySymbol == "" || storageKey == "" {
		return errors.New("invalid game storage key")
	}
	return dd.player.Where(
		"user_id = ? AND game_id = ? AND currency_symbol = ? AND storage_key = ?",
		userId, gameId, currencySymbol, storageKey,
	).Delete(&GameStorageRecord{}).Error
}
