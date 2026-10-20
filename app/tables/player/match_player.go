package player

import "github.com/shopspring/decimal"

type MatchPlayer struct {
	Id             int64           `gorm:"column:id;primaryKey;" json:"id"`                                   //id  playerId
	MatchId        int64           `gorm:"column:match_id;primaryKey;" json:"match_id"`                       //比赛id
	Name           string          `gorm:"column:name;size:128;" json:"name"`                                 //name
	Avatar         string          `gorm:"column:avatar;size:256;" json:"avatar"`                             //头像
	Points         decimal.Decimal `gorm:"column:points;type:decimal(10,2);default:0;" json:"points"`         //计分牌
	Rank           int             `gorm:"column:rank;default:0;" json:"rank"`                                //排行
	DisuseTime     int64           `gorm:"column:disuse_time;default:0;" json:"disuse_time"`                  //淘汰时间
	PrvPoints      decimal.Decimal `gorm:"column:prv_points;type:decimal(10,2);default:0;" json:"prv_points"` //进入比赛游戏时的计分牌
	BuyPointsCount int32           `gorm:"column:buy_points_count;default:0;" json:"buy_points_count"`        //购买计分牌次数
	Value          decimal.Decimal `gorm:"column:value;type:decimal(10,2);default:0;" json:"value"`           //身价
	Reward         decimal.Decimal `gorm:"column:reward;type:decimal(10,2);default:0;" json:"reward"`         //赏金
	ServerId       string          `gorm:"column:server_id;size:256" json:"server_id"`                        //所在服务器id
	RoomId         int64           `gorm:"column:room_id;default:0" json:"room_id"`                           //所在房间id
}

func (p *MatchPlayer) TableName() string {
	return "match_player"
}
