package entity

var AesKey = "7z2q$l^Jw^DM3*Ol"
var AesIV = "tBu&NN$qQb9bi5L*"

var TokenTimeOut int64 = 30 * 24 * 60 * 60

type LoginInfo struct {
	Id        int32  `json:"id"`
	Name      string `json:"name"`
	LoginTime int64  `json:"logon"`
	UserType  int32  `json:"UT"`
	AgentId   int64  `json:"agentId"`
	Rand      int64  `json:"rand"`
}

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
