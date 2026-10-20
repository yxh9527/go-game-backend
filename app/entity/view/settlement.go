package view

type Settlement struct {
	AgentId         int64   `json:"agentId"`
	UserId          int64   `json:"userId"`
	GameId          int64   `json:"gameId"`
	Account         string  `json:"account"`
	NickName        string  `json:"nickName"`
	OfficeNumber    string  `json:"roundID"`
	BeginTime       int64   `json:"playedDate"`
	EffectiveBets   float64 `json:"bet"`
	ProfitLoss      float64 `json:"win"`
	UserScore       float64 `json:"balance"`
	RowVersion      int64   `json:"rowVersion"`
	CurrencyType    string  `json:"currency"`
	Revenue         float64 `json:"revenue"`
	ExEffectiveBets float64 `json:"exBet"`
	ExProfitLoss    float64 `json:"exWin"`
	ExRevenue       float64 `json:"exRevenue"`
	GameName        string  `json:"gameName"`
	Hash            string  `json:"hash"`
	Complete        bool    `json:"complete"`
	Symbol          string  `json:"symbol"`
	IsTourist       int32   `json:"isTourist"`
	Log             string  `json:"log"`
}
