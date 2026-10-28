package gogame

import "github.com/shopspring/decimal"

type BlindItem struct {
	Level    int32           `json:"level"`
	Min      decimal.Decimal `json:"min"`
	Max      decimal.Decimal `json:"max"`
	Delay    bool            `json:"delay"`
	Bet      decimal.Decimal `json:"bet"`
	Interval int32           `json:"interval"`
	Type     int32           `json:"type"` // 1 普通 2 准备暂停 3 中场休息
}

type Blinds struct {
	Id         int64  `gorm:"column:id;primaryKey;autoIncrement;" json:"id"`
	Name       string `gorm:"column:name;size:128;" json:"name"`
	Data       string `gorm:"column:data;type:text;" json:"data"`
	CreateTime int64  `gorm:"column:create_time;default:0;" json:"create_time"`
	UpdateTime int64  `gorm:"column:update_time;default:0;" json:"update_time"`
}

func (t *Blinds) TableName() string {
	return "gg_blinds"
}
