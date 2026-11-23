package tables

// GameSeedItem seeds one game-server room route into gp_game.
type GameSeedItem struct {
	Number         int
	NameZH         string
	Symbol         string
	TypeName       string
	ClientGameType uint32
	RoomType       uint32
}

// GameServerSeeds mirrors the modules and room types registered in
// game-server/src/games/catalog.js. Keep one row per routable room type so
// gp_game can publish the same routing catalogue consumed by the server.
func GameServerSeeds() []GameSeedItem {
	return []GameSeedItem{
		{Number: 91019, NameZH: "德州扑克", Symbol: "dzpk", TypeName: "normal", ClientGameType: 19, RoomType: 19},
		{Number: 91101, NameZH: "德州比赛", Symbol: "dzpk_mtt", TypeName: "normal", ClientGameType: 19, RoomType: 101},
		{Number: 91024, NameZH: "棋牌牛牛", Symbol: "qznn", TypeName: "normal", ClientGameType: 24, RoomType: 24},
		{Number: 91025, NameZH: "炸金花", Symbol: "ctjh", TypeName: "normal", ClientGameType: 21, RoomType: 25},
		{Number: 91026, NameZH: "麻将", Symbol: "jsmj", TypeName: "normal", ClientGameType: 26, RoomType: 26},
		{Number: 91028, NameZH: "麻将", Symbol: "csmj", TypeName: "normal", ClientGameType: 26, RoomType: 28},
		{Number: 91030, NameZH: "麻将", Symbol: "hzxl", TypeName: "normal", ClientGameType: 26, RoomType: 30},
		{Number: 91027, NameZH: "十三水", Symbol: "sss", TypeName: "normal", ClientGameType: 27, RoomType: 27},
		{Number: 91031, NameZH: "常规二十一点", Symbol: "bj21", TypeName: "normal", ClientGameType: 31, RoomType: 31},
		{Number: 91032, NameZH: "三公暗一", Symbol: "sgya", TypeName: "normal", ClientGameType: 32, RoomType: 32},
		{Number: 92050, NameZH: "百人德州", Symbol: "hundred_holdem", TypeName: "hundred", ClientGameType: 1002, RoomType: 50},
		{Number: 92051, NameZH: "百人金花", Symbol: "hundred_zjh", TypeName: "hundred", ClientGameType: 1002, RoomType: 51},
		{Number: 92052, NameZH: "百人牛牛", Symbol: "hundred_niuniu", TypeName: "hundred", ClientGameType: 1002, RoomType: 52},
		{Number: 92053, NameZH: "奔驰宝马", Symbol: "arcade_bcbm", TypeName: "hundred", ClientGameType: 1002, RoomType: 53},
		{Number: 92054, NameZH: "森林舞会", Symbol: "arcade_slwh", TypeName: "hundred", ClientGameType: 1002, RoomType: 54},
		{Number: 92055, NameZH: "飞禽走兽", Symbol: "arcade_fqzs", TypeName: "hundred", ClientGameType: 1002, RoomType: 55},
		{Number: 92056, NameZH: "麻将转了", Symbol: "arcade_mjzl", TypeName: "hundred", ClientGameType: 1002, RoomType: 56},
		{Number: 92057, NameZH: "趣味捕鱼", Symbol: "arcade_qwby", TypeName: "hundred", ClientGameType: 1002, RoomType: 57},
		{Number: 92058, NameZH: "百人百家乐", Symbol: "brbjl", TypeName: "hundred", ClientGameType: 1002, RoomType: 58},
		{Number: 92059, NameZH: "百人龙虎斗", Symbol: "brlhd", TypeName: "hundred", ClientGameType: 1002, RoomType: 59},
		{Number: 92060, NameZH: "百人二十一点", Symbol: "hundred_blackjack", TypeName: "hundred", ClientGameType: 1002, RoomType: 60},
		{Number: 91061, NameZH: "麻将胡了", Symbol: "mjhl", TypeName: "slots", ClientGameType: 1004, RoomType: 61},
		{Number: 92062, NameZH: "百人麻将", Symbol: "hundred_mahjong", TypeName: "hundred", ClientGameType: 1002, RoomType: 62},
		{Number: 91063, NameZH: "赏金女王", Symbol: "sjnv", TypeName: "slots", ClientGameType: 1004, RoomType: 63},
	}
}
