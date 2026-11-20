package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

// DDL belongs to InitMysqlDb/deployment. A player purchase must not migrate a
// production table after its wallet has already been debited.
func (dd *DBDao) ScratchStorageReady(parent context.Context) error {
	if dd == nil || dd.player == nil {
		return fmt.Errorf("scratch ownership database unavailable")
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	m := dd.player.WithContext(ctx).Migrator()
	model := &GameStorageRecord{}
	if !m.HasTable(model) || !m.HasIndex(model, "uk_game_storage") {
		return fmt.Errorf("scratch ownership schema is not initialized")
	}
	for _, field := range []string{"UserId", "GameId", "CurrencySymbol", "StorageKey", "Payload", "ExpireAt", "UpdatedAt"} {
		if !m.HasColumn(model, field) {
			return fmt.Errorf("scratch ownership schema incomplete")
		}
	}
	return nil
}
func (dd *DBDao) SaveScratchGameStorage(parent context.Context, agentID, userID, gameID uint32, currency, key, payload string) error {
	if dd == nil || dd.player == nil || userID == 0 || gameID == 0 || currency == "" || key == "" || len(key) > 128 || len(payload) > 128*1024 {
		return fmt.Errorf("invalid scratch ownership record")
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	row := &GameStorageRecord{UserId: userID, GameId: gameID, CurrencySymbol: currency, StorageKey: key, Payload: payload, ExpireAt: 0, UpdatedAt: time.Now().Unix()}
	var ticket struct {
		Serial   string `json:"serialFull"`
		RoundID  string `json:"roundId"`
		UserID   uint32 `json:"userId"`
		Currency string `json:"currency"`
	}
	if json.Unmarshal([]byte(payload), &ticket) != nil || key != "scratch:"+ticket.Serial || ticket.UserID != userID || ticket.Currency != currency {
		return fmt.Errorf("invalid scratch ownership payload")
	}
	return dd.player.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := saveScratchSold(tx, ScratchScope{Agent: agentID, GameID: gameID, UserID: userID, Currency: currency}, ticket.Serial, ticket.RoundID); err != nil {
			return err
		}
		return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "game_id"}, {Name: "currency_symbol"}, {Name: "storage_key"}}, DoUpdates: clause.AssignmentColumns([]string{"payload", "expire_at", "updated_at"})}).Create(row).Error
	})
}
