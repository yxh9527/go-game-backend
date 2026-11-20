package gogame

import (
	"app/tables/manager"
	"fmt"
	"gorm.io/gorm"
	"strings"
)

// GameRoute is the trusted projection consumed by Node. GameType comes from
// ClientGameType, not the existing gp_game_type foreign key.
type GameRoute struct {
	GameType uint32 `json:"gameType"`
	RoomType uint32 `json:"roomType"`
	Number   uint32 `json:"number"`
	Symbol   string `json:"symbol"`
	State    int16  `json:"state"`
	IsFrozen int16  `json:"isFrozen"`
}

func ProjectGameRoutes(games []*manager.Game) ([]GameRoute, error) {
	result := make([]GameRoute, 0)
	numbers := make(map[int]int)
	for _, game := range games {
		if game != nil {
			numbers[game.Number]++
			if numbers[game.Number] > 1 {
				return nil, fmt.Errorf("duplicate financial game number %d", game.Number)
			}
		}
	}
	pairs := make(map[string]bool)
	symbols := make(map[string]bool)
	for _, game := range games {
		if game == nil {
			return nil, fmt.Errorf("nil game in routing catalogue")
		}
		if game.ClientGameType == nil && game.RoomType == nil {
			continue
		}
		if game.ClientGameType == nil || game.RoomType == nil || *game.ClientGameType == 0 || *game.RoomType == 0 {
			return nil, fmt.Errorf("game %d has an incomplete client route", game.Number)
		}
		if game.Number <= 0 || uint64(game.Number) > uint64(^uint32(0)) || numbers[game.Number] != 1 {
			return nil, fmt.Errorf("financial game number %d is invalid or duplicated", game.Number)
		}
		key := fmt.Sprintf("%d:%d", *game.ClientGameType, *game.RoomType)
		if pairs[key] {
			return nil, fmt.Errorf("duplicate client game route %s", key)
		}
		pairs[key] = true
		symbol := strings.TrimSpace(game.ConfName)
		if symbol == "" || game.ConfName != symbol || (game.State != 1 && game.State != 2) || (game.IsFrozen != 0 && game.IsFrozen != 1) {
			return nil, fmt.Errorf("game %d has invalid routing metadata", game.Number)
		}
		if symbols[symbol] {
			return nil, fmt.Errorf("duplicate client game pool symbol %s", symbol)
		}
		symbols[symbol] = true
		result = append(result, GameRoute{GameType: *game.ClientGameType, RoomType: *game.RoomType, Number: uint32(game.Number), Symbol: symbol, State: game.State, IsFrozen: game.IsFrozen})
	}
	return result, nil
}

// Existing games keep NULL/NULL. No legacy number/category is backfilled or
// renumbered. MySQL permits multiple NULLs in this composite unique index.
func MigrateGameRoutes(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("game route migration requires manager database")
	}
	for _, field := range []string{"ClientGameType", "RoomType"} {
		if !db.Migrator().HasColumn(&manager.Game{}, field) {
			if err := db.Migrator().AddColumn(&manager.Game{}, field); err != nil {
				return err
			}
		}
	}
	var games []*manager.Game
	if err := db.Find(&games).Error; err != nil {
		return err
	}
	if _, err := ProjectGameRoutes(games); err != nil {
		return err
	}
	for _, name := range []string{"uq_gp_game_client_room", "uq_gp_game_number"} {
		if !db.Migrator().HasIndex(&manager.Game{}, name) {
			if err := db.Migrator().CreateIndex(&manager.Game{}, name); err != nil {
				return err
			}
		}
	}
	return nil
}
