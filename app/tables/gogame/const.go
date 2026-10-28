package gogame

// 比赛状态：草稿 → 报名中 → 进行中 → 结束 / 取消
const (
	MatchStateDraft       int32 = 0
	MatchStateRegistering int32 = 1
	MatchStateRunning     int32 = 2
	MatchStateFinished    int32 = 3
	MatchStateCancelled   int32 = 4
)

const (
	MatchPlayerRegistered int32 = 1
	MatchPlayerPlaying    int32 = 2
	MatchPlayerEliminated int32 = 3
	MatchPlayerCancelled  int32 = 4
)

const (
	RoomCategoryNormal  = "normal"
	RoomCategoryHundred = "hundred"
	RoomCategorySlots   = "slots"
)

const (
	RewardKindMoney  = "money"
	RewardKindTicket = "ticket"
)

// RedisChannelMatch 比赛状态广播频道，不得占用配置同步用的 message。
const RedisChannelMatch = "match"

const (
	EventMatchUpsert = "match_upsert"
	EventMatchState  = "match_state"
	EventMatchSign   = "match_sign"
)

func CanTransitMatch(from, to int32) bool {
	switch from {
	case MatchStateDraft:
		return to == MatchStateRegistering || to == MatchStateCancelled
	case MatchStateRegistering:
		return to == MatchStateRunning || to == MatchStateCancelled
	case MatchStateRunning:
		return to == MatchStateFinished || to == MatchStateCancelled
	default:
		return false
	}
}

const (
	DefaultCurrency = "CNY"
	TexasGameType   = 19
	TexasRoomType   = 19
	TexasMatchRoom  = 101
	BaccaratGame    = 1002
	BaccaratRoom    = 58
	ScratchGame     = 1004
	ScratchMahjong  = 61
	ScratchQueen    = 63
	ZjhLobbyType    = 21
	ZjhRoomType     = 25
)
