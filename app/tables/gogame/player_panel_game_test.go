package gogame

import (
	"gorm.io/gorm/schema"
	"sync"
	"testing"
)

func TestPlayerPanelGameMigrationKeepsLegacyRowsHoldemAndIndexesGameQueries(t *testing.T) {
	model, err := schema.Parse(&PlayerPanelHand{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	field := model.LookUpField("RoomType")
	if field == nil || field.DBName != "room_type" || field.DefaultValue != "19" || !field.NotNull {
		t.Fatal("existing Holdem rows would not retain their game", field)
	}
	indexes := map[string]*schema.Index{}
	for _, index := range model.ParseIndexes() {
		indexes[index.Name] = index
	}
	newIndex, ok := indexes["idx_panel_game_account"]
	if !ok || len(newIndex.Fields) != 5 {
		t.Fatal("game statistics index missing", indexes)
	}
	for i, want := range []string{"agent_id", "player_id", "currency", "room_type", "ended_at"} {
		if newIndex.Fields[i].DBName != want {
			t.Fatal(newIndex)
		}
	}
	if original, ok := indexes["idx_panel_account"]; !ok || len(original.Fields) != 4 {
		t.Fatal("original index changed", indexes)
	}
}
