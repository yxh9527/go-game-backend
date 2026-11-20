package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// A minimal immutable ownership row in the existing ticket storage table.
// This namespace has no cards, payout data, report body or financial replay.
type scratchSoldOwner struct {
	Version  int    `json:"version"`
	Agent    uint32 `json:"agent"`
	GameID   uint32 `json:"gameId"`
	UserID   uint32 `json:"userId"`
	Currency string `json:"currency"`
	Serial   string `json:"serialFull"`
	RoundID  string `json:"roundId"`
}

func scratchSoldKey(scope ScratchScope, serial string) (string, error) {
	if scope.GameID == 0 || len(serial) == 0 || len(serial) > 80 || strings.ContainsAny(serial, "\x00\n\r") {
		return "", fmt.Errorf("invalid scratch serial scope")
	}
	return fmt.Sprintf("scratch-sold:%d:%s", scope.Agent, serial), nil
}

func saveScratchSold(tx *gorm.DB, scope ScratchScope, serial, roundID string) error {
	key, err := scratchSoldKey(scope, serial)
	if err != nil {
		return err
	}
	if scope.UserID == 0 || scope.Currency == "" || roundID == "" {
		return fmt.Errorf("invalid scratch serial owner")
	}
	owner := scratchSoldOwner{Version: 1, Agent: scope.Agent, GameID: scope.GameID, UserID: scope.UserID, Currency: scope.Currency, Serial: serial, RoundID: roundID}
	raw, err := json.Marshal(owner)
	if err != nil {
		return err
	}
	// User 0 / currency * are reserved solely for market-level ticket ownership.
	// Existing uk_game_storage indexes the complete lookup; no DDL is needed.
	row := GameStorageRecord{GameId: scope.GameID, CurrencySymbol: "*", StorageKey: key, Payload: string(raw), ExpireAt: 0, UpdatedAt: time.Now().Unix()}
	if err = tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
		return err
	}
	var existing GameStorageRecord
	if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id=? AND game_id=? AND currency_symbol=? AND storage_key=?", 0, scope.GameID, "*", key).Take(&existing).Error; err != nil {
		return err
	}
	var actual scratchSoldOwner
	if existing.ExpireAt != 0 || json.Unmarshal([]byte(existing.Payload), &actual) != nil || actual != owner {
		return fmt.Errorf("SCRATCH_SERIAL_OWNER_CONFLICT")
	}
	return nil
}

// Query every owner and currency in this market, not just the requesting player.
// Redis misses are never interpreted as an available paid serial on their own.
func (dd *DBDao) ScratchSoldSerials(parent context.Context, scope ScratchScope, serials []string) (map[string]bool, error) {
	if dd == nil || dd.player == nil {
		return nil, fmt.Errorf("scratch ownership database unavailable")
	}
	if len(serials) < 1 || len(serials) > 100 {
		return nil, fmt.Errorf("invalid scratch serial batch")
	}
	keys := make([]string, 0, len(serials))
	byKey := map[string]string{}
	for _, serial := range serials {
		key, err := scratchSoldKey(scope, serial)
		if err != nil {
			return nil, err
		}
		if _, seen := byKey[key]; !seen {
			keys = append(keys, key)
			byKey[key] = serial
		}
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	var rows []GameStorageRecord
	if err := dd.player.WithContext(ctx).Select("storage_key,payload,expire_at").Where("user_id=? AND game_id=? AND currency_symbol=? AND storage_key IN ?", 0, scope.GameID, "*", keys).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := map[string]bool{}
	for _, row := range rows {
		serial, ok := byKey[row.StorageKey]
		var owner scratchSoldOwner
		if !ok || row.ExpireAt != 0 || json.Unmarshal([]byte(row.Payload), &owner) != nil || owner.Version != 1 || owner.Agent != scope.Agent || owner.GameID != scope.GameID || owner.UserID == 0 || owner.Currency == "" || owner.Serial != serial || owner.RoundID == "" {
			return nil, fmt.Errorf("invalid saved scratch serial owner")
		}
		result[serial] = true
	}
	return result, nil
}
