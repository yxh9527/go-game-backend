package view

import "github.com/shopspring/decimal"

type AggsSumItem struct {
	Value decimal.Decimal `json:"value"`
}

type UserInfoAggsItem struct {
	EffectiveBetsTotal *AggsSumItem `json:"effectiveBetsTotal"`
	ProfitLossTotal    *AggsSumItem `json:"profitLossTotal"`
	PumpTotal          *AggsSumItem `json:"pumpTotal"`
}

type UserInfoAggs struct {
	Aggs     *UserInfoAggsItem `json:"Aggregations"`
	DocCount int               `json:"DocCount"`
	Key      int64             `json:"Key"`
}

type AgentInfoAggsItem struct {
	EffectiveBetsTotal *AggsSumItem `json:"effectiveBetsTotal"`
	ProfitLossTotal    *AggsSumItem `json:"profitLossTotal"`
	PumpTotal          *AggsSumItem `json:"pumpTotal"`
	UserTotal          *AggsSumItem `json:"userTotal"`
	UserBets           *AggsSumItem `json:"userBetsTotal"`
	RevenueTotal       *AggsSumItem `json:"revenueTotal"`
	ChipsTotal         *AggsSumItem `json:"chipsTotal"`
	TotalRegUser       int64        `json:"totalRegUser"`
	RangeRegUser       int64        `json:"rangeRegUser"`
	ScoreUp            float64      `json:"score_up"`
	ScoreDown          float64      `json:"score_down"`
}

type AgentInfoAggs struct {
	Aggs     *AgentInfoAggsItem `json:"Aggregations"`
	DocCount int                `json:"DocCount"`
	Key      int64              `json:"Key"`
}

type AgentTotalCount struct {
	UserNum int64 `gorm:"column:userNum" json:"userNum"`
	AgentId int64 `gorm:"column:agentId" json:"agentId"`
}

func (ac *AgentTotalCount) TableName() string {
	return "gp_user"
}

type UserScoreSum struct {
	ScoreSum float64 `gorm:"column:scoreSum" json:"scoreSum"`
	AgentId  int64   `gorm:"column:agentId" json:"agentId"`
	LogType  int     `gorm:"column:logType" json:"logType"`
}

func (uss *UserScoreSum) TableName() string {
	return "gp_user_score_log"
}

type UserGameDataByHourItem struct {
	RecordTime int64   `gorm:"column:record_time;" json:"record_time"`
	Value      float64 `gorm:"column:value;" json:"value"`
}

func (t *UserGameDataByHourItem) TableName() string {
	return "gp_game_data_hour"
}

type UserGameDataByHourEff struct {
	RecordTime int64   `gorm:"column:record_time;" json:"record_time"`
	Value      float64 `gorm:"column:value;" json:"eff"`
}

func (t *UserGameDataByHourEff) TableName() string {
	return "gp_game_data_hour"
}

type UserGameDataByHourPro struct {
	RecordTime int64   `gorm:"column:record_time;" json:"record_time"`
	Value      float64 `gorm:"column:value;" json:"pro"`
}

func (t *UserGameDataByHourPro) TableName() string {
	return "gp_game_data_hour"
}

type UserGameDataByHourCnt struct {
	RecordTime int64   `gorm:"column:record_time;" json:"record_time"`
	Value      float64 `gorm:"column:value;" json:"cnt"`
}

func (t *UserGameDataByHourCnt) TableName() string {
	return "gp_game_data_hour"
}

type UserGameDataByHourActive struct {
	RecordTime int64   `gorm:"column:record_time;" json:"record_time"`
	Value      float64 `gorm:"column:value;" json:"active"`
}

func (t *UserGameDataByHourActive) TableName() string {
	return "gp_game_data_hour"
}

//按天打点数据项定义
type DataAnalysisItem struct {
	WebId              int     `json:"webId"`
	AgentId            int     `json:"agentId"`
	GameId             int64   `json:"gameId"`
	Symbol             string  `json:"symbol"`
	DocCount           int     `json:"doc_count"`
	ProfitLossTotal    float64 `json:"profitLossTotal"`
	UserTotal          int     `json:"userTotal"`
	EffectiveBetsTotal float64 `json:"effectiveBetsTotal"`
	ChipsTotal         float64 `json:"chipsTotal"`
	RevenueTotal       float64 `json:"revenueTotal"`
	PumpTotal          float64 `json:"pumpTotal"`
	UserBetsTotal      float64 `json:"userBetsTotal"`
	Date               string  `json:"date"`
	StartTime          int64   `json:"startTime"`
	EndTime            int64   `json:"endTime"`
	CreateAt           int64   `json:"createAt"`
	GameName           string  `json:"gameName"`
}

//玩家游戏数据
type UserGameDataStatisItem struct {
	UserId        uint32  `json:"id"`
	EffectiveBets float64 `json:"totalEffBet"`
	Chips         float64 `json:"totalChips"`
	ProfitLoss    float64 `json:"totalProfLoss"`
	Pump          float64 `json:"totalPump"`
	Count         int     `json:"totalNumber"`
	PlayTimes     int     `json:"times"`
	UpdateTime    int64   `json:"updateTime"`
}
