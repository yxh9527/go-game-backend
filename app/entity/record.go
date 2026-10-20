package entity

//作为表结构 pb里面的对象 0值会被忽略不能用做表结构

type RecordLog struct {
	CR      string `json:"cr"`
	SR      string `json:"sr"`
	RoundId string `json:"roundId"`
}

type CacheRecordsReq struct {
	WebId          uint32  `json:"webId"`
	UserId         uint32  `json:"userId"`
	AgentId        uint32  `json:"agentId"`
	GameId         uint32  `json:"gameId"`
	Account        string  `json:"account"`
	NickName       string  `json:"nickName"`
	Bet            float64 `json:"bet"`
	ExBet          float64 `json:"exBet"`
	Currency       string  `json:"currency"`
	CurrencySymbol string  `json:"currencySymbol"`
	BaseBet        float64 `json:"base_bet"`
	Win            float64 `json:"win"`
	ExWin          float64 `json:"exWin"`
	Rtp            float64 `json:"rtp"`
	PlayedDate     int64   `json:"playedDate"`
	RoundID        string  `json:"roundID"`
	Init           string  `json:"init"`
	Log            string  `json:"log"`
	Symbol         string  `json:"symbol"`
	Hash           string  `json:"hash"`
	Balance        float64 `json:"balance"`
	BalanceCash    float64 `json:"balance_cash"`
	BalanceBonus   float64 `json:"balance_bonus"`
	RowVersion     int64   `json:"rowVersion"`
	Revenue        float64 `json:"revenue"`
	ExRevenue      float64 `json:"exRevenue"`
	Pump           float64 `json:"pmup"`
	GameName       string  `json:"gameName"`
	Chips          float64 `json:"chips"`
	Complete       bool    `json:"complete"`
	IsTourist      int32   `json:"isTourist"`
}

type CommonRecord struct {
	RecordId           string  `json:"recordId"`
	DispatchRewardGold float64 `json:"dispatchRewardGold"`
}

type ConnectionRecord struct {
	BetGold     float64 `json:"betGold"`
	WinLoseGold float64 `json:"winLoseGold"`
}

type BetRecord struct {
	TotalBetGold float64 `json:"totalBetGold"`
}

type UserRecordInfo struct {
	Common     *CommonRecord     `json:"commonRecord"`
	Connection *ConnectionRecord `json:"connectionRecord"`
	BetRecord  *BetRecord        `json:"betRecord"`
}

type ClientRecordsReq struct {
	UserId         uint32 `json:"userId"`
	AgentId        uint32 `json:"agentId"`
	GameId         uint32 `json:"gameId"`
	Account        string `json:"account"`
	NickName       string `json:"nickName"`
	Bet            string `json:"bet"`
	Currency       string `json:"currency"`
	CurrencySymbol string `json:"currencySymbol"`
	BaseBet        string `json:"base_bet"`
	Win            string `json:"win"`
	Rtp            string `json:"rtp"`
	PlayedDate     int64  `json:"playedDate"`
	RoundID        string `json:"roundID"`
	Init           string `json:"init"`
	Log            string `json:"log"`
	Symbol         string `json:"symbol"`
	Hash           string `json:"hash"`
	Balance        string `json:"balance"`
	BalanceCash    string `json:"balance_cash"`
	BalanceBonus   string `json:"balance_bonus"`
	RowVersion     int64  `json:"rowVersion"`
	IsTourist      int32  `json:"isTourist"`
}

type CacheBillsReq struct {
	UserId         uint32  `json:"userId"`
	AgentId        uint32  `json:"agentId"`
	GameId         uint32  `json:"gameId"`
	Symbol         string  `json:"symbol"`
	Bet            float64 `json:"bet"`   // 下注金额，下注为负
	Award          float64 `json:"award"` // 返奖金额，>=0
	CurrentScore   float64 `json:"currentScore"`
	Currency       string  `json:"currency"`
	CurrencySymbol string  `json:"currencySymbol"`
	CreateTime     int64   `json:"createTime"`
	RoundID        string  `json:"roundId"`
	FlowingWaterOn string  `json:"flowingWaterOn"`
	Desc           string  `json:"desc"`
}
