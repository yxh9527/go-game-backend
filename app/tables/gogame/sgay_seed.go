package gogame

import (
	"fmt"

	"gorm.io/gorm"
)

// Call after seedZjhIfMissing in SeedIfEmpty. Insert only missing tiers;
// do not update or reset operator configuration or running rooms.
func seedSgayIfMissing(m *gorm.DB, now int64) {
	tiers := []struct {
		id, ante, carry int64
		class           int32
	}{
		{3203, 3000, 30000, 4},
		{3204, 5000, 50000, 4},
		{3201, 500, 5000, 2},
		{3202, 1000, 10000, 2},
	}
	for index, t := range tiers {
		var count int64
		if err := m.Model(&RoomConfig{}).Where("room_type = ? AND game_room_config_id = ?", 32, t.id).Count(&count).Error; err != nil {
			panic(err)
		}
		if count > 0 {
			continue
		}
		name := fmt.Sprintf("三公暗一(底注%d)", t.ante/100)
		row := &RoomConfig{
			GameType: 32, SubGameType: 0, RoomType: 32, GameRoomConfigId: t.id,
			PlayMode: 0, CarryMin: t.carry, Category: "normal", Name: name, State: 1,
			SortIndex: int32(3200 + index), CreateTime: now, UpdateTime: now,
			Payload: mustJSON(map[string]any{
				"roomType": 32, "gameRoomConfigId": t.id, "playMode": 0,
				"ante": t.ante, "carryMin": t.carry, "carryMax": -1,
				"smallBlind": 0, "bigBlind": 0, "initialBet": 0,
				"straddle": false, "forceShowCard": false, "forceShowType": 0,
				"seeCardNo": 0, "maxPeople": 7, "gameStartNum": 2,
				"roomClassification": t.class, "gameName": name,
				"randomSit": true, "banTalk": true, "observed": true,
				"openInsurance": false, "allowCarryOut": true,
				"taxType": 2, "taxPercent": 500, "maxTax": 0,
				"key": fmt.Sprintf("32/0/%d/%d/%d", t.ante, t.class, t.carry),
			}),
		}
		if err := m.Create(row).Error; err != nil {
			panic(err)
		}
	}
}
