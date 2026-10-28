package gogame

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func ManagerModels() []any {
	return []any{
		&AwardTemplate{},
		&Blinds{},
		&Ticket{},
		&TicketCurrencyRate{},
		&Match{},
		&RoomConfig{},
	}
}

func PlayerModels() []any {
	return []any{
		&TicketHold{},
		&MatchPlayer{},
		&MatchRoom{},
	}
}

func SeedIfEmpty(m, p *gorm.DB) {
	_ = p
	now := time.Now().Unix()
	seedAwards(m, now)
	seedBlinds(m, now)
	seedTickets(m, now)
	seedRates(m, now)
	seedRooms(m, now)
	seedNoShowIfMissing(m, now)
	seedHoldemCatalogIfMissing(m, now)
	seedZjhIfMissing(m, now)
	seedSgayIfMissing(m, now)
	seedMatches(m, now)
}

func seedAwards(m *gorm.DB, now int64) {
	var n int64
	m.Model(&AwardTemplate{}).Count(&n)
	if n > 0 {
		return
	}
	free := mustJSON([]AwardRank{
		{Rank: 1, Items: []AwardItem{{Kind: RewardKindMoney, Currency: DefaultCurrency, Amount: "50000"}}},
		{Rank: 2, Items: []AwardItem{{Kind: RewardKindMoney, Currency: DefaultCurrency, Amount: "30000"}}},
		{Rank: 3, Items: []AwardItem{{Kind: RewardKindMoney, Currency: DefaultCurrency, Amount: "20000"}}},
		{Rank: 4, Items: []AwardItem{{Kind: RewardKindTicket, TicketID: 1, Amount: "5"}}},
	})
	paid := mustJSON([]AwardRank{
		{Rank: 1, Items: []AwardItem{
			{Kind: RewardKindMoney, Currency: DefaultCurrency, Amount: "100000"},
			{Kind: RewardKindTicket, TicketID: 2, Amount: "10"},
		}},
		{Rank: 2, Items: []AwardItem{{Kind: RewardKindMoney, Currency: DefaultCurrency, Amount: "50000"}}},
		{Rank: 3, Items: []AwardItem{{Kind: RewardKindMoney, Currency: DefaultCurrency, Amount: "20000"}}},
	})
	m.Create([]*AwardTemplate{
		{Id: 1, Name: "德州每日免费赛奖励", Detail: free, CreateTime: now, UpdateTime: now},
		{Id: 2, Name: "德州付费赛奖励", Detail: paid, CreateTime: now, UpdateTime: now},
	})
}

func seedBlinds(m *gorm.DB, now int64) {
	var n int64
	m.Model(&Blinds{}).Count(&n)
	if n > 0 {
		return
	}
	levels := []struct {
		lv     int32
		sb, bb int64
		delay  bool
	}{
		{1, 50, 100, true},
		{2, 75, 150, true},
		{3, 100, 200, true},
		{4, 150, 300, true},
		{5, 200, 400, true},
		{6, 300, 600, true},
		{7, 400, 800, true},
		{8, 500, 1000, true},
		{9, 600, 1200, true},
		{10, 800, 1600, true},
		{11, 1000, 2000, true},
		{12, 1500, 3000, true},
		{13, 2000, 4000, true},
		{14, 2500, 5000, false},
		{15, 3000, 6000, false},
		{16, 4000, 8000, false},
		{17, 5000, 10000, false},
		{18, 8000, 16000, false},
		{19, 10000, 20000, false},
		{20, 15000, 30000, false},
	}
	items := make([]BlindItem, 0, len(levels))
	for _, lv := range levels {
		item := BlindItem{
			Level:    lv.lv,
			Min:      decimal.NewFromInt(lv.sb),
			Max:      decimal.NewFromInt(lv.bb),
			Delay:    lv.delay,
			Bet:      decimal.Zero,
			Interval: 180,
			Type:     1,
		}
		if lv.lv == 10 {
			item.Type = 3
			item.Interval = 60
		}
		items = append(items, item)
	}
	m.Create(&Blinds{Id: 1, Name: "德州MTT标准盲注", Data: mustJSON(items), CreateTime: now, UpdateTime: now})
}

func seedTickets(m *gorm.DB, now int64) {
	var n int64
	m.Model(&Ticket{}).Count(&n)
	if n > 0 {
		return
	}
	m.Create([]*Ticket{
		{Id: 1, Name: "通用门票", Price: decimal.NewFromInt(100), Currency: DefaultCurrency, MatchId: 0, Num: 100000, Remain: 100000, CreateTime: now, UpdateTime: now},
		{Id: 2, Name: "德州每日赛门票", Price: decimal.NewFromInt(100), Currency: DefaultCurrency, MatchId: 0, Num: 50000, Remain: 50000, CreateTime: now, UpdateTime: now},
	})
}

func seedRates(m *gorm.DB, now int64) {
	var n int64
	m.Model(&TicketCurrencyRate{}).Count(&n)
	if n > 0 {
		return
	}
	rates := []struct {
		cur  string
		rate string
	}{
		{"CNY", "100"},
		{"USD", "14"},
		{"EUR", "13"},
		{"USDT", "14"},
		{"HK", "109"},
		{"JPY", "2222"},
		{"BRL", "70"},
		{"INR", "1176"},
		{"MXN", "233"},
		{"IDR", "200000"},
		{"MYR", "67"},
		{"PHP", "769"},
		{"SGD", "19"},
		{"VND", "333333"},
		{"THB", "500"},
		{"KRW", "20000"},
	}
	rows := make([]*TicketCurrencyRate, 0, len(rates))
	for i, r := range rates {
		rate, err := decimal.NewFromString(r.rate)
		if err != nil {
			panic(err)
		}
		rows = append(rows, &TicketCurrencyRate{
			Id:         int64(i + 1),
			Currency:   r.cur,
			Rate:       rate,
			CreateTime: now,
			UpdateTime: now,
		})
	}
	m.Create(rows)
}

func seedRooms(m *gorm.DB, now int64) {
	var n int64
	m.Model(&RoomConfig{}).Count(&n)
	if n > 0 {
		return
	}
	rooms := []*RoomConfig{
		texasRoom(now, 70, 10, 20, 500, 0, 2, 20, 3, true, "德州 10/20"),
		texasRoom(now, 1, 100, 200, 5000, 0, 2, 200, 3, true, "德州 100/200"),
		texasRoom(now, 2, 300, 600, 30000, 0, 2, 600, 3, true, "德州 300/600"),
		baccaratRoom(now, 5803, 0, 1, "经典百家乐 1"),
		baccaratRoom(now, 5804, 0, 2, "经典百家乐 2"),
		baccaratRoom(now, 5805, 0, 3, "经典百家乐 3"),
		scratchRoom(now, ScratchMahjong, 6101, 26, "麻将胡了"),
		scratchRoom(now, ScratchQueen, 6301, 26, "赏金女王"),
		zjhRoom(now, 2501, 10, 100, 3, "炸金花 底注10"),
	}
	m.Create(rooms)
}

func seedZjhIfMissing(m *gorm.DB, now int64) {
	var n int64
	m.Model(&RoomConfig{}).Where("room_type = ?", ZjhRoomType).Count(&n)
	if n > 0 {
		return
	}
	m.Create(zjhRoom(now, 2501, 10, 100, 3, "炸金花 底注10"))
}

func seedNoShowIfMissing(m *gorm.DB, now int64) {
	rooms := []*RoomConfig{
		texasRoom(now, 1924, 10, 20, 500, 0, 2, 20, 3, false, "德州不秀牌 10/20"),
		texasRoom(now, 1925, 100, 200, 5000, 0, 2, 200, 3, false, "德州不秀牌 100/200"),
		texasRoom(now, 1926, 300, 600, 30000, 0, 2, 600, 3, false, "德州不秀牌 300/600"),
	}
	for _, room := range rooms {
		var count int64
		result := m.Model(&RoomConfig{}).Where("game_room_config_id = ? AND room_type = ?", room.GameRoomConfigId, room.RoomType).Count(&count)
		if result.Error != nil || count > 0 {
			continue
		}
		m.Create(room)
	}
}

func texasRoom(now int64, cfgID, sb, bb, carryMin int64, playMode, class int32, initialBet, seeCardNo int64, showCard bool, name string) *RoomConfig {
	show := int32(0)
	if showCard {
		show = 1
	}
	key := fmt.Sprintf("%d/%d/%d/%v/%d/%d/%d/%d/0/%v", TexasRoomType, sb, bb, false, playMode, initialBet, class, carryMin, showCard)
	payload := mustJSON(map[string]any{
		"roomType":            TexasRoomType,
		"gameRoomConfigId":    cfgID,
		"totalPeople":         0,
		"seatInfoList":        nil,
		"winStreakList":       nil,
		"gameStatus":          0,
		"dealerBgNo":          0,
		"lobbyThumbNo":        0,
		"gameStatusStartTime": 0,
		"gameStatusDuration":  0,
		"serverNow":           0,
		"smallBlind":          sb,
		"bigBlind":            bb,
		"straddle":            false,
		"carryMin":            carryMin,
		"playMode":            playMode,
		"roomClassification":  class,
		"ante":                0,
		"scoreMode":           nil,
		"initialBet":          initialBet,
		"seeCardNo":           seeCardNo,
		"key":                 key,
		"aofBet":              nil,
		"hundredGame":         0,
		"hundredMaxBet":       0,
		"openHundredGame":     false,
		"hundredMinBet":       0,
		"waysJson":            nil,
		"extendInfo":          nil,
		"roomIndex":           0,
		"vipTableData":        nil,
		"domain":              nil,
		"serviceId":           nil,
		"shield":              nil,
		"privateRoom":         false,
		"roomId":              0,
		"forceShowCard":       showCard,
	})
	return &RoomConfig{
		GameType:         TexasGameType,
		SubGameType:      1,
		RoomType:         TexasRoomType,
		GameRoomConfigId: cfgID,
		PlayMode:         playMode,
		RoomIndex:        0,
		SmallBlind:       sb,
		BigBlind:         bb,
		CarryMin:         carryMin,
		ForceShowCard:    show,
		Category:         RoomCategoryNormal,
		Name:             name,
		Payload:          payload,
		State:            1,
		SortIndex:        int32(cfgID),
		CreateTime:       now,
		UpdateTime:       now,
	}
}

func baccaratRoom(now int64, cfgID int64, playMode, roomIndex int32, name string) *RoomConfig {
	key := fmt.Sprintf("%d/0/0/false/%d/0/0/0/0/%d", BaccaratRoom, playMode, roomIndex)
	payload := mustJSON(map[string]any{
		"roomType":            BaccaratRoom,
		"gameRoomConfigId":    cfgID,
		"totalPeople":         0,
		"seatInfoList":        nil,
		"winStreakList":       nil,
		"gameStatus":          0,
		"dealerBgNo":          0,
		"lobbyThumbNo":        0,
		"gameStatusStartTime": 0,
		"gameStatusDuration":  0,
		"serverNow":           0,
		"smallBlind":          0,
		"bigBlind":            0,
		"straddle":            false,
		"carryMin":            0,
		"playMode":            playMode,
		"roomClassification":  0,
		"ante":                0,
		"scoreMode":           nil,
		"initialBet":          0,
		"seeCardNo":           0,
		"key":                 key,
		"aofBet":              nil,
		"hundredGame":         1,
		"hundredMaxBet":       100000000,
		"openHundredGame":     false,
		"hundredMinBet":       100,
		"waysJson":            nil,
		"extendInfo":          nil,
		"roomIndex":           roomIndex,
		"vipTableData":        nil,
		"domain":              nil,
		"serviceId":           nil,
		"shield":              nil,
		"privateRoom":         false,
		"roomId":              0,
		"forceShowCard":       false,
	})
	return &RoomConfig{
		GameType:         BaccaratGame,
		SubGameType:      0,
		RoomType:         BaccaratRoom,
		GameRoomConfigId: cfgID,
		PlayMode:         playMode,
		RoomIndex:        roomIndex,
		HundredMinBet:    100,
		HundredMaxBet:    100000000,
		Category:         RoomCategoryHundred,
		Name:             name,
		Payload:          payload,
		State:            1,
		SortIndex:        roomIndex,
		CreateTime:       now,
		UpdateTime:       now,
	}
}

func scratchRoom(now int64, roomType int32, cfgID int64, roomIndex int32, name string) *RoomConfig {
	carryMin := int64(100000000)
	key := fmt.Sprintf("%d/0/0/false/0/0/0/%d/0", roomType, carryMin)
	payload := mustJSON(map[string]any{
		"roomType":            roomType,
		"gameRoomConfigId":    cfgID,
		"totalPeople":         0,
		"seatInfoList":        nil,
		"winStreakList":       nil,
		"gameStatus":          0,
		"dealerBgNo":          0,
		"lobbyThumbNo":        0,
		"gameStatusStartTime": 0,
		"gameStatusDuration":  0,
		"serverNow":           0,
		"smallBlind":          0,
		"bigBlind":            0,
		"straddle":            false,
		"carryMin":            carryMin,
		"playMode":            0,
		"roomClassification":  0,
		"ante":                0,
		"scoreMode":           nil,
		"initialBet":          0,
		"seeCardNo":           0,
		"key":                 key,
		"aofBet":              nil,
		"hundredGame":         1,
		"hundredMaxBet":       100000000,
		"openHundredGame":     false,
		"hundredMinBet":       100,
		"waysJson":            nil,
		"extendInfo":          nil,
		"roomIndex":           roomIndex,
		"vipTableData":        nil,
		"domain":              nil,
		"serviceId":           nil,
		"shield":              nil,
		"privateRoom":         false,
		"roomId":              0,
		"forceShowCard":       false,
	})
	return &RoomConfig{
		GameType:         ScratchGame,
		SubGameType:      0,
		RoomType:         roomType,
		GameRoomConfigId: cfgID,
		PlayMode:         0,
		RoomIndex:        roomIndex,
		CarryMin:         carryMin,
		HundredMinBet:    100,
		HundredMaxBet:    100000000,
		Category:         RoomCategorySlots,
		Name:             name,
		Payload:          payload,
		State:            1,
		SortIndex:        int32(cfgID),
		CreateTime:       now,
		UpdateTime:       now,
	}
}

func zjhRoom(now int64, cfgID, ante, carryMin int64, playMode int32, name string) *RoomConfig {
	key := fmt.Sprintf("%d/0/0/false/%d/%d/2/%d/0", ZjhRoomType, playMode, ante, carryMin)
	payload := mustJSON(map[string]any{
		"roomType":            ZjhRoomType,
		"gameRoomConfigId":    cfgID,
		"totalPeople":         0,
		"seatInfoList":        nil,
		"winStreakList":       nil,
		"gameStatus":          0,
		"dealerBgNo":          0,
		"lobbyThumbNo":        0,
		"gameStatusStartTime": 0,
		"gameStatusDuration":  0,
		"serverNow":           0,
		"smallBlind":          0,
		"bigBlind":            0,
		"straddle":            false,
		"carryMin":            carryMin,
		"playMode":            playMode,
		"roomClassification":  2,
		"ante":                ante,
		"scoreMode":           nil,
		"initialBet":          ante,
		"seeCardNo":           0,
		"key":                 key,
		"aofBet":              nil,
		"hundredGame":         0,
		"hundredMaxBet":       0,
		"openHundredGame":     false,
		"hundredMinBet":       0,
		"waysJson":            nil,
		"extendInfo":          nil,
		"roomIndex":           0,
		"vipTableData":        nil,
		"domain":              nil,
		"serviceId":           nil,
		"shield":              nil,
		"privateRoom":         false,
		"roomId":              0,
		"forceShowCard":       true,
	})
	return &RoomConfig{
		GameType:         ZjhLobbyType,
		SubGameType:      0,
		RoomType:         ZjhRoomType,
		GameRoomConfigId: cfgID,
		PlayMode:         playMode,
		RoomIndex:        0,
		CarryMin:         carryMin,
		ForceShowCard:    1,
		Category:         RoomCategoryNormal,
		Name:             name,
		Payload:          payload,
		State:            1,
		SortIndex:        int32(cfgID),
		CreateTime:       now,
		UpdateTime:       now,
	}
}

func seedMatches(m *gorm.DB, now int64) {
	var n int64
	m.Model(&Match{}).Count(&n)
	if n > 0 {
		return
	}
	var freeAward, paidAward AwardTemplate
	m.First(&freeAward, 1)
	m.First(&paidAward, 2)
	msNow := time.Now().UnixMilli()
	draftStart := msNow + 24*60*60*1000
	freeStart := msNow + 2*60*60*1000
	paidStart := msNow + 6*60*60*1000
	m.Create([]*Match{
		{
			Id: 1, ChampionshipName: langName("德州草稿赛", "Draft Tournament", "Torneio Rascunho"),
			GameType: TexasGameType, RoomType: TexasMatchRoom, ChampionshipType: 1, RegisterType: 4,
			Currency: DefaultCurrency, InitPoint: 500000, MinPlayer: 2, MaxPlayer: 1000,
			DelayRegisterLevel: 13, DelayRegisterTime: 2340, RegisterTimeLimit: 1800,
			ForceShowCard: 1, FixPot: 1, TotalPot: 100000, BlindsId: 1, BlindType: 2,
			AwardId: 1, AwardJSON: freeAward.Detail, State: MatchStateDraft,
			StartTime: draftStart, SortIndex: 99, BigBlind: 100, SmallBlind: 50,
			AllowTicketPay: 1, CreateTime: now, UpdateTime: now,
		},
		{
			Id: 2, ChampionshipName: langName("德州每日免费赛", "Daily Free Tournament", "NLH Daily Freeroll"),
			GameType: TexasGameType, RoomType: TexasMatchRoom, ChampionshipType: 1, RegisterType: 4,
			Currency: DefaultCurrency, InitPoint: 500000, MinPlayer: 2, MaxPlayer: 5000,
			DelayRegisterLevel: 13, DelayRegisterTime: 2340, RegisterTimeLimit: 1800,
			ForceShowCard: 1, FixPot: 1, TotalPot: 100000, FreeTypePot: 100000,
			BlindsId: 1, BlindType: 2, AwardId: 1, AwardJSON: freeAward.Detail,
			State: MatchStateRegistering, StartTime: freeStart, SortIndex: 1,
			BigBlind: 100, SmallBlind: 50, IncreaseBindTime: 3, AllowTicketPay: 1,
			Domain: "127.0.0.1:11041", Shield: "jt-room1", CreateTime: now, UpdateTime: now,
		},
		{
			Id: 3, ChampionshipName: langName("德州每日付费赛", "Daily Paid Tournament", "NLH Daily Buy-in"),
			GameType: TexasGameType, RoomType: TexasMatchRoom, ChampionshipType: 1, RegisterType: 3,
			RegisterGold: 1500, ServiceGold: 500, Currency: DefaultCurrency,
			RegisterTicket: 0, TicketId: 2, AllowTicketPay: 1, TicketValue: 100, TicketName: "德州每日赛门票",
			InitPoint: 1000000, MinPlayer: 2, MaxPlayer: 2000,
			DelayRegisterLevel: 13, DelayRegisterTime: 2340, RegisterTimeLimit: 1800,
			ForceShowCard: 1, FixPot: 0, TotalPot: 400000,
			BlindsId: 1, BlindType: 2, AwardId: 2, AwardJSON: paidAward.Detail,
			State: MatchStateRegistering, StartTime: paidStart, SortIndex: 2,
			BigBlind: 100, SmallBlind: 50, IncreaseBindTime: 3,
			Domain: "127.0.0.1:11041", Shield: "jt-room1", CreateTime: now, UpdateTime: now,
		},
	})
}

func langName(zh, en, pt string) string {
	return mustJSON([]map[string]string{
		{"lang": "zh_cn", "championshipName": zh},
		{"lang": "en", "championshipName": en},
		{"lang": "pt", "championshipName": pt},
	})
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}
