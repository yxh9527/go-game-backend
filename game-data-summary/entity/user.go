package entity

import "github.com/shopspring/decimal"

type User struct {
	UserId       int64           `gorm:"column:user_id;primaryKey;"`
	NickName     string          `gorm:"column:nick_name;size:32;"`
	Account      string          `gorm:"column:account;size:32;"`
	Score        decimal.Decimal `gorm:"column:score;default:0"`
	Pwd          string          `gorm:"column:pwd;default:'';size:64;"`
	Sex          int             `gorm:"column:sex;default:0"`
	Pic          string          `gorm:"column:pic;size:256;"`
	Exp          uint32          `gorm:"column:exp;default:0;size:256;"`
	ProxyId      int             `gorm:"column:proxy_id;default:0"`
	State        int             `gorm:"column:state;default:0"`
	WebsiteId    int             `gorm:"column:website_id;default:0"`
	LoginTime    int64           `gorm:"column:login_time;default:0"`
	LoginIp      string          `gorm:"column:login_ip;size:32;"`
	MoneyLimit   decimal.Decimal `gorm:"column:money_limit;default:0"`
	CurrencyType string          `gorm:"column:currency_type;size:32;"`
	Revenue      decimal.Decimal `gorm:"column:revenue;default:0"`
	AllTimes     int32           `gorm:"column:all_times;default:0;"`
}

func (p *User) TableName() string {
	return "user_info"
}
