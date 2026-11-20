package gogame

import "github.com/shopspring/decimal"

// Ticket 门票种类。MatchId=0 为通用门票。
type Ticket struct {
	Id         int64           `gorm:"column:id;primaryKey;autoIncrement;" json:"id"`
	Name       string          `gorm:"column:name;size:128;" json:"name"`
	Price      decimal.Decimal `gorm:"column:price;type:decimal(20,4);" json:"price"`
	Currency   string          `gorm:"column:currency;size:16;default:CNY;" json:"currency"`
	MatchId    int64           `gorm:"column:match_id;index;default:0;" json:"match_id"`
	Num        int64           `gorm:"column:num;default:0;" json:"num"`
	Remain     int64           `gorm:"column:remain;default:0;" json:"remain"`
	CreateTime int64           `gorm:"column:create_time;default:0;" json:"create_time"`
	UpdateTime int64           `gorm:"column:update_time;default:0;" json:"update_time"`
}

func (t *Ticket) TableName() string {
	return "gg_ticket"
}

// TicketCurrencyRate 门票兑货币比例，按币种配置，报名扣费必须读当前比例。
type TicketCurrencyRate struct {
	Id         int64           `gorm:"column:id;primaryKey;autoIncrement;" json:"id"`
	Currency   string          `gorm:"column:currency;size:16;uniqueIndex;" json:"currency"`
	Rate       decimal.Decimal `gorm:"column:rate;type:decimal(20,8);" json:"rate"`
	CreateTime int64           `gorm:"column:create_time;default:0;" json:"create_time"`
	UpdateTime int64           `gorm:"column:update_time;default:0;" json:"update_time"`
}

func (t *TicketCurrencyRate) TableName() string {
	return "gg_ticket_currency_rate"
}

// TicketHold 门票持有记录。奖励发门票写入本表，报名抵扣将 IsUse 标为已用。
type TicketHold struct {
	Id         int64  `gorm:"column:id;primaryKey;autoIncrement;" json:"id"`
	TicketId   int64  `gorm:"column:ticket_id;index;default:0;" json:"ticket_id"`
	Owner      int64  `gorm:"column:owner;index;default:0;" json:"owner"`
	IsUse      int32  `gorm:"column:is_use;index;default:0;" json:"is_use"`
	UseTime    int64  `gorm:"column:use_time;default:0;" json:"use_time"`
	UseMatchId int64  `gorm:"column:use_match_id;index;default:0;" json:"use_match_id"`
	CreateTime int64  `gorm:"column:create_time;default:0;" json:"create_time"`
}

func (t *TicketHold) TableName() string {
	return "gg_ticket_hold"
}
