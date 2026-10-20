package entity

import (
	"app/entity/view"
	"app/tables/manager"

	"github.com/shopspring/decimal"
)

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type LoginResultUser struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	UserType int    `json:"userType"`
}

type LoginResult struct {
	Token string           `json:"token"`
	User  *LoginResultUser `json:"user"`
}

type GameListItem struct {
	Id       int    `json:"id"`
	Number   int    `json:"number"`
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	NameZH   string `json:"nameZH"`
	GameType int    `json:"gameType"`
	Status   int    `json:"status"`
}

type LinkAgentData struct {
	Id      int    `json:"id"`
	Name    string `json:"name"`
	GameIds string `json:"gameIds"`
}

type LinkAgentResult struct {
	Id     int              `json:"id"`
	Name   string           `json:"name"`
	Agents []*LinkAgentData `json:"agentList"`
}

type WebListResult struct {
	Data     []*manager.Web `json:"data"`
	LastPage int64          `json:"last_page"`
	Total    int64          `json:"total"`
	Page     int64          `json:"current_page"`
	PerPage  int64          `json:"per_page"`
}

type GamerListResult struct {
	Id   int             `json:"id"`
	Name string          `json:"name"`
	Data []*manager.Game `json:"gameList"`
}

type AgentList struct {
	Data     []*view.AgentListItem `json:"data"`
	LastPage int64                 `json:"last_page"`
	Total    int64                 `json:"total"`
	Page     int64                 `json:"current_page"`
	PerPage  int64                 `json:"per_page"`
}

type AgentSelectItem struct {
	Id      int              `json:"id"`
	Name    string           `json:"name"`
	Level   int              `json:"level"`
	SubList []*manager.Agent `json:"subList"`
}

type AddAgentGameIdsItem struct {
	Id  int   `json:"id"`
	Val []int `json:"val"`
}

type AgentInfoGameList struct {
	Id       int      `json:"id"`
	Val      []int    `json:"val"`
	NameList []string `json:"nameList"`
}

type AgentInfo struct {
	*manager.Agent
	GameList []*AgentInfoGameList `json:"gameList"`
}

type ApiConfigList struct {
	Data     []*manager.ApiConfig `json:"data"`
	LastPage int64                `json:"last_page"`
	Total    int64                `json:"total"`
	Page     int64                `json:"current_page"`
	PerPage  int64                `json:"per_page"`
}

type UserListItem struct {
	*manager.User
	CycleProfitLoss    float64 `json:"cycle_profitLoss"`
	CycleEffectiveBets float64 `json:"cycle_effectiveBets"`
	MonthEffectiveBets float64 `json:"month_effectiveBets"`
	MonthProfitLoss    float64 `json:"month_profitLoss"`
	MonthDocCount      int     `json:"month_docCount"`
	Rate               float64 `json:"rate"`
	RateScore          float64 `json:"rate_score"`
	LeftScore          float64 `json:"left_score"`
}

type UserList struct {
	Data     []*UserListItem `json:"data"`
	LastPage int64           `json:"last_page"`
	Total    int64           `json:"total"`
	Page     int64           `json:"current_page"`
	PerPage  int64           `json:"per_page"`
}

type BillList struct {
	Data     []*view.Bill `json:"data"`
	LastPage int64        `json:"last_page"`
	Total    int64        `json:"total"`
	Page     int64        `json:"current_page"`
	PerPage  int64        `json:"per_page"`
}

type SettlementList struct {
	Data     []*view.Settlement `json:"data"`
	Total    int64              `json:"total"`
	Page     int64              `json:"page"`
	PageSize int64              `json:"pageSize"`
}

type GovernUserCtl struct {
	T int64 `json:"t"`
	E int64 `json:"e"`
}

type ManagerAccountList struct {
	Data     []*manager.SystemUser `json:"data"`
	LastPage int64                 `json:"last_page"`
	Total    int64                 `json:"total"`
	Page     int64                 `json:"current_page"`
	PerPage  int64                 `json:"per_page"`
}

type UserAgentGameDataByHourData struct {
	Effects []*view.UserGameDataByHourEff    `json:"eff"`
	Pros    []*view.UserGameDataByHourPro    `json:"pro"`
	Counts  []*view.UserGameDataByHourCnt    `json:"cnt"`
	Active  []*view.UserGameDataByHourActive `json:"active"`
}

type AgGroupItem struct {
	DocCount           int         `json:"docCount"`
	DocCount2          int         `json:"doc_count"`
	EffectiveBetsTotal float64     `json:"effectiveBetsTotal"`
	Key                int64       `json:"key"`
	ProfitLossTotal    float64     `json:"profitLossTotal"`
	PumpTotal          float64     `json:"pumpTotal"`
	UserTotal          float64     `json:"userTotal"`
	Games              interface{} `json:"games"`
}

type AgGroupData struct {
	AgentId  int          `json:"agentId"`
	NickName string       `json:"nickName"`
	Now      *AgGroupItem `json:"now"`
	Last     *AgGroupItem `json:"last"`
}

type ServerTime struct {
	Time int64 `json:"timestamp"`
}

type ReportFormItem struct {
	AgentId            int64   `json:"agentId"`
	GameId             int     `json:"gameId"`
	DocCount           int     `json:"doc_count"`
	ProfitLossTotal    float64 `json:"profitLossTotal"`
	ChipsTotal         float64 `json:"chipsTotal"`
	UserTotal          int     `json:"userTotal"`
	Doc2Count          int     `json:"docCount"`
	UserBetsTotal      float64 `json:"userBetsTotal"`
	EffectiveBetsTotal float64 `json:"effectiveBetsTotal"`
	RevenueTotal       float64 `json:"revenueTotal"`
	PumpTotal          float64 `json:"pumpTotal"`
	AgentName          string  `json:"agentName"`
	GameName           string  `json:"gameName"`
	Symbol             string  `json:"symbol"`
	PlayType           int32   `json:"playType"`
}

type ReportForm struct {
	Data               []*ReportFormItem `json:"data"`
	DocCount           int64             `json:"docCount"`
	UserBetsTotal      float64           `json:"userBetsTotal"`
	ProfitLossTotal    float64           `json:"profitLossTotal"`
	EffectiveBetsTotal float64           `json:"effectiveBetsTotal"`
	RevenueTotal       float64           `json:"revenueTotal"`
	PumpTotal          float64           `json:"pumpTotal"`
	ChipsTotal         float64           `json:"chipsTotal"`
	Total              int64             `json:"total"`
	Page               int64             `json:"current_page"`
	PerPage            int64             `json:"per_page"`
}

type AgentGameAggs struct {
	Data []*ReportFormItem `json:"data"`
}

type ReportFormHistory struct {
	Data    []interface{} `json:"data"`
	Total   int64         `json:"total"`
	Page    int64         `json:"current_page"`
	PerPage int64         `json:"per_page"`
}

type GovernUserRecord struct {
	Data     []interface{} `json:"data"`
	Total    int64         `json:"total"`
	Page     int64         `json:"page"`
	PageSize int64         `json:"pageSize"`
}

type Msg struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

// 配置相关消息
type ConfigMsg struct {
	Key  string `json:"key"`
	Data string `json:"data"`
}

// 添加单控
type AddPlayerSingleCtrl struct {
	Uid         uint32          `json:"uid"`    //玩家id
	RemainScore decimal.Decimal `json:"remain"` //剩余积分
	CtrlScore   decimal.Decimal `json:"ctrl"`   //控制积分
	UpdateTime  int64           `json:"time"`   //修改时间
	Rate        decimal.Decimal `json:"rate"`   //进入单控的概率
}

// 移除单控
type DelPlayerSingleCtrl struct {
	Uid uint32 `json:"uid"`
}

type AddGame struct {
	*manager.Game
}

type GameStatuChange struct {
	Symbol   string `json:"symbol"`
	Status   uint32 `json:"status"`
	ShowType uint32 `json:"showType"`
}

type AddAgent struct {
	*manager.Agent
}

type AgentStatuChange struct {
	Id     uint32 `json:"id"`
	Status uint32 `json:"status"`
}

type ResetPool struct {
}
