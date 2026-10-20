package entity

//作为表结构 pb里面的对象 0值会被忽略不能用做表结构

type RecordLog struct {
	CR      string `json:"cr"`
	SR      string `json:"sr"`
	RoundId string `json:"roundId"`
}

type CacheRecordsReq struct {
	UserId         uint32      `json:"userId"`
	AgentId        uint32      `json:"agentId"`
	GameId         uint32      `json:"gameId"`
	Account        string      `json:"account"`
	NickName       string      `json:"nickName"`
	Bet            float64     `json:"bet"`
	ExBet          float64     `json:"exBet"`
	Currency       string      `json:"currency"`
	CurrencySymbol string      `json:"currencySymbol"`
	BaseBet        float64     `json:"base_bet"`
	Win            float64     `json:"win"`
	ExWin          float64     `json:"exWin"`
	Rtp            float64     `json:"rtp"`
	PlayedDate     int64       `json:"playedDate"`
	RoundID        string      `json:"roundID"`
	Init           string      `json:"init"`
	Log            []RecordLog `json:"log"`
	Symbol         string      `json:"symbol"`
	Hash           string      `json:"hash"`
	Balance        float64     `json:"balance"`
	BalanceCash    float64     `json:"balance_cash"`
	BalanceBonus   float64     `json:"balance_bonus"`
	RowVersion     int64       `json:"rowVersion"`
	Revenue        float64     `json:"revenue"`
	ExRevenue      float64     `json:"exRevenue"`
	Pump           float64     `json:"pmup"`
	GameName       string      `json:"gameName"`
}

type CacheBillsReq struct {
	UserId         uint32  `json:"userId"`
	AgentId        uint32  `json:"agentId"`
	GameId         uint32  `json:"gameId"`
	Symbol         string  `json:"symbol"`
	Bet            float64 `json:"bet"`
	CurrentScore   float64 `json:"currentScore"`
	Currency       string  `json:"currency"`
	CurrencySymbol string  `json:"currencySymbol"`
	CreateTime     int64   `json:"createTime"`
	RoundID        string  `json:"roundId"`
	FlowingWaterOn string  `json:"flowingWaterOn"`
	Desc           string  `json:"desc"`
}

type LastPlay struct {
	UserId         uint32  `json:"userId"`
	AgentId        uint32  `json:"agentId"`
	Symbol         string  `json:"symbol"`
	Bet            float64 `json:"bet"`
	CurrentScore   float64 `json:"currentScore"`
	Currency       string  `json:"currency"`
	CurrencySymbol string  `json:"currencySymbol"`
	CreateTime     int64   `json:"createTime"`
	RoundID        string  `json:"roundId"`
	FlowingWaterOn string  `json:"flowingWaterOn"`
	Desc           string  `json:"desc"`
}
