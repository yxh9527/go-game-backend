package common

import "github.com/shopspring/decimal"

type LoginResp struct {
	Code int    `json:"code"`
	Url  string `json:"url"`
}

type SimpleResp struct {
	Code int `json:"code"`
}

type ScoreResp struct {
	Code  int     `json:"code"`
	Money float64 `json:"money"`
}

type DataResp struct {
	Code int           `json:"code"`
	List []interface{} `json:"list"`
}

type Response struct {
	S string      `json:"s"`
	M string      `json:"m"`
	D interface{} `json:"d"`
}

type DataCenterParams struct {
	PlayerId      string `json:"player_id"`
	CurrencyDelta string `json:"currency_delta"`
}

type DataCenterResponse struct {
	Code  int             `json:"code"`
	Score decimal.Decimal `json:"new_currency"`
}
