package player

import "github.com/shopspring/decimal"

type Tiket struct {
	Id         int64           `gorm:"column:id;primaryKey;autoIncrement;" json:"id"`
	MatchId    int64           `gorm:"column:match_id;default:0;" json:"match_id"`    //绑定的比赛id
	MatchName  string          `gorm:"column:match_name;size:128;" json:"match_name"` //比赛名字
	Name       string          `gorm:"column:name;size:128;" json:"name"`             //name
	CreateTime int64           `gorm:"column:create_time;default:0;" json:"create_time"`
	Price      decimal.Decimal `gorm:"column:price;type:decimal(10,2);" json:"price"` //价值
	Num        int64           `gorm:"column:num;default:0;" json:"num"`              //数量
	Remain     int64           `gorm:"column:remain;default:0;" json:"remain"`        //剩余数量
}

func (p *Tiket) TableName() string {
	return "tiket"
}
