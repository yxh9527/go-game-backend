package gogame

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
)

// Copy into app/tables/gogame and call after seedNoShowIfMissing in SeedIfEmpty.
// Inserts missing catalog rows only; never overwrites operator configuration.
// Listing special modes does not enable an unimplemented game engine.
func seedHoldemCatalogIfMissing(m *gorm.DB, now int64) {
	tiers := []struct {
		id, sb, bb, carry int64
		mode, class       int32
		initial           int64
		straddle          bool
		aof               int64
		show              bool
		name              string
	}{
		{70, 10, 20, 500, 0, 2, 20, false, 0, true, "德州秀牌 10/20"},
		{1921, 10, 20, 500, 2, 2, 20, false, 0, true, "暴击德州秀牌 10/20"},
		{1, 100, 200, 5000, 0, 2, 200, false, 0, true, "德州秀牌 100/200"},
		{1922, 100, 200, 5000, 2, 2, 200, false, 0, true, "暴击德州秀牌 100/200"},
		{2, 300, 600, 30000, 0, 2, 600, false, 0, true, "德州秀牌 300/600"},
		{1923, 300, 600, 30000, 2, 2, 600, false, 0, true, "暴击德州秀牌 300/600"},
		{3, 500, 1000, 50000, 0, 4, 1000, false, 0, true, "德州秀牌 500/1000"},
		{4, 1000, 2000, 100000, 0, 4, 2000, false, 0, true, "德州秀牌 1000/2000"},
		{5, 2000, 4000, 200000, 0, 4, 4000, false, 0, true, "德州秀牌 2000/4000"},
		{1927, 100, 200, 100, 8, 2, 0, true, 100, true, "AOF必推秀牌 100/200"},
		{121, 500, 1000, 500, 8, 2, 0, true, 500, true, "AOF必推秀牌 500/1000"},
		{123, 5000, 10000, 5000, 8, 4, 0, true, 5000, true, "AOF必推秀牌 5000/10000"},
		{1920, 30000, 60000, 30000, 8, 4, 0, true, 30000, true, "AOF必推秀牌 30000/60000"},
		{1924, 10, 20, 500, 0, 2, 20, false, 0, false, "德州不秀牌 10/20"},
		{1928, 10, 20, 500, 2, 2, 20, false, 0, false, "暴击德州不秀牌 10/20"},
		{1925, 100, 200, 5000, 0, 2, 200, false, 0, false, "德州不秀牌 100/200"},
		{1929, 100, 200, 5000, 2, 2, 200, false, 0, false, "暴击德州不秀牌 100/200"},
		{1926, 300, 600, 30000, 0, 2, 600, false, 0, false, "德州不秀牌 300/600"},
		{1930, 300, 600, 30000, 2, 2, 600, false, 0, false, "暴击德州不秀牌 300/600"},
	}
	for index, t := range tiers {
		var count int64
		if err := m.Model(&RoomConfig{}).Where("room_type = ? AND game_room_config_id = ?", TexasRoomType, t.id).Count(&count).Error; err != nil {
			panic(err)
		}
		if count > 0 {
			continue
		}
		row := texasRoom(now, t.id, t.sb, t.bb, t.carry, t.mode, t.class, t.initial, 3, t.show, t.name)
		var fields map[string]any
		if err := json.Unmarshal([]byte(row.Payload), &fields); err != nil {
			panic(err)
		}
		fields["straddle"], fields["aofBet"] = t.straddle, t.aof
		if t.mode == 2 {
			// Captured baseline; explicit room configuration, never a blind multiplier.
			fields["criticalFlag"], fields["criticalAnte"], fields["criticalInterval"] = true, 100, "5-8"
		}
		fields["key"] = fmt.Sprintf("19/%d/%d/%v/%d/%d/%d/%d/%d/%v", t.sb, t.bb, t.straddle, t.mode, t.initial, t.class, t.carry, t.aof, t.show)
		row.Payload = mustJSON(fields)
		row.SubGameType = 0
		if t.show {
			row.SubGameType = 1
		}
		row.SortIndex = int32(1900 + index)
		if err := m.Create(row).Error; err != nil {
			panic(err)
		}
	}
}
