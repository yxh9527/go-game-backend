package player

import "github.com/shopspring/decimal"

type BlindItem struct {
	Level    int32           `json:"level"`
	Min      decimal.Decimal `json:"min"`
	Max      decimal.Decimal `json:"max"`
	Delay    bool            `json:"delay"`    //是否可以报名
	Bet      decimal.Decimal `json:"bet"`      //
	Interval int32           `json:"interval"` //张盲时间
	Type     int32           `json:"type"`     //涨盲类型 1:普通类型 2:准备暂停 3:中场休息暂停
}

type BlindData struct {
	Items []*BlindItem `json:"items"`
}

// 盲注结构表
type Blinds struct {
	Id         int64  `gorm:"column:id;primaryKey;autoIncrement;" json:"id"` //id
	Name       string `gorm:"column:name" json:"name"`
	Data       string `gorm:"column:data" json:"data"`
	Createtime int64  `gorm:"column:create_time" json:"create_time"`
}

func (p *Blinds) TableName() string {
	return "blinds"
}
