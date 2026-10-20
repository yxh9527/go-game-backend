package common

type Session struct {
	AgentId      int64  `json:"agentId"`
	UserId       int64  `json:"userId"`
	Symbol       string `json:"symbol"`
	NickName     string `json:"nickName"`
	AuthToken    string `json:"authToken"`
	Mgckey       string `json:"mgckey"`
	Lang         string `json:"lang"`
	GameId       int64  `json:"gameId"`
	Account      string `json:"account"`
	LastAuthTime int64  `json:"lastAuthTime"`
	AuthCount    int64  `json:"authCount"`
	CurrencyType string `json:"currencyType"`
	IsTourist    int32  `json:"isTourist"`
}
