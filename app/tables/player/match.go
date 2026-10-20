package player

import "github.com/shopspring/decimal"

type Match struct {
	Id             int64           `gorm:"column:id;primaryKey;autoIncrement;" json:"id"`       //id
	Name           string          `gorm:"column:name;size:128;" json:"name"`                   //name
	AgentId        int64           `gorm:"column:agent_id" json:"agent_id"`                     //代理id
	Type           string          `gorm:"column:type;size:128;" json:"type"`                   //赛事类型  MatchType 赛事类型以逗号分隔
	GameType       int32           `gorm:"column:game_type" json:"game_type"`                   //游戏类型
	Level          int             `gorm:"column:level;" json:"level"`                          //当前级别
	Interval       int             `gorm:"column:interval;" json:"interval"`                    //中场休息
	Sign           int             `gorm:"column:sign;" json:"sign"`                            //已报名
	Pot            decimal.Decimal `gorm:"column:pot;type:decimal(10,2);" json:"pot"`           //总彩池
	Award          int64           `gorm:"column:award" json:"award"`                           //奖励
	MaxBuy         int             `gorm:"column:max_buy" json:"max_buy"`                       //买入上限
	GameId         int             `gorm:"column:game_id" json:"game_id"`                       //游戏
	State          int             `gorm:"column:state" json:"state"`                           //比赛状态
	Fee            decimal.Decimal `gorm:"column:fee;type:decimal(10,2);" json:"fee"`           //报名费
	Base           int             `gorm:"column:base;default:0;" json:"base"`                  //基础报名人数
	RoomFee        decimal.Decimal `gorm:"column:room_fee;type:decimal(10,2);" json:"room_fee"` //房费
	BlindsId       int64           `gorm:"column:blinds_id" json:"blinds_id"`                   //盲注表id
	Score          decimal.Decimal `gorm:"column:score;type:decimal(10,2);" json:"score"`       //积分
	Desc           string          `gorm:"column:desc" json:"desc"`                             //说明
	News           string          `gorm:"column:news" json:"news"`                             //资讯
	Tag            string          `gorm:"column:tag" json:"tag"`                               //描述
	SettlementType int             `gorm:"column:settlement_type" json:"settlement_type"`       //结算类型 0 普通结算方式 1 猎人结算方式
	Price          decimal.Decimal `gorm:"column:price;type:decimal(10,2);" json:"price"`       //购买一次计分牌所花费的金币(CNY)
	Num            decimal.Decimal `gorm:"column:num;type:decimal(10,2);" json:"num"`           //购买一次计分牌所的数量
	Delay          int             `gorm:"column:delay;default:0;" json:"delay"`                //延迟报名
	StartTime      int64           `gorm:"column:start_time" json:"start_time"`                 //开始时间
	EndTime        int64           `gorm:"column:end_time;default:0;" json:"end_time"`          //结束时间
	CreateTime     int64           `gorm:"column:create_time" json:"create_time"`               //创建时间
}

func (p *Match) TableName() string {
	return "match"
}
