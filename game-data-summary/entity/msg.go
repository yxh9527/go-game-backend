package entity

import "github.com/shopspring/decimal"

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

//移除单控
type DelPlayerSingleCtrl struct {
	Uid uint32 `json:"uid"`
}
