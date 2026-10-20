package player

type TiketDetail struct {
	Id         int64  `gorm:"column:id;primaryKey;autoIncrement;" json:"id"`
	TicektId   int64  `gorm:"column:ticket_id;default:0;" json:"ticket_id"`
	MatchId    int64  `gorm:"column:match_id;default:0;" json:"match_id"`    //绑定的比赛id
	MatchName  string `gorm:"column:match_name;size:128;" json:"match_name"` //比赛名字
	Name       string `gorm:"column:name;size:128;" json:"name"`             //name
	CreateTime int64  `gorm:"column:create_time;default:0;" json:"create_time"`
	Owner      int64  `gorm:"column:owner;default:0;" json:"owner"`       //拥有着
	IsUse      bool   `gorm:"column:is_use;default:false;" json:"is_use"` //是否使用
	UseTime    int64  `gorm:"column:use_time;default:0;" json:"use_time"` //使用时间
}

func (p *TiketDetail) TableName() string {
	return "tiket_detail"
}
