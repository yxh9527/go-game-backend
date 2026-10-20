package manager

type User struct {
	Id            int64   `gorm:"column:id;primaryKey;autoIncrement;not null;" json:"id"`
	UserId        string  `gorm:"column:userId;size:100;index:user_userId;" json:"userId"`
	NickName      string  `gorm:"column:nickName;size:100;index:user_nickName;" json:"nickName"`
	AgentId       int64   `gorm:"column:agentId;index:user_agentId;" json:"agentId"`
	WebId         int64   `gorm:"column:webId;index:agent_webId;" json:"webId"`
	TotalNumber   int64   `gorm:"column:totalNumber;comment:场次;" json:"totalNumber"`
	Times         int64   `gorm:"column:times;comment:时间;" json:"times"`
	ControlNumber int64   `gorm:"column:controlNumber;comment:控制次数;" json:"controlNumber"`
	State         int     `gorm:"column:state;comment:状态 1正常 2冻结;" json:"state"`
	LogTime       int64   `gorm:"column:logTime;comment:最近一次登录时间;" json:"logTime"`
	LogIp         string  `gorm:"column:logIp;comment:登陆ip" json:"logIp"`
	CreateTime    int32   `gorm:"column:createTime;index:user_createTime;comment:注册时间" json:"createTime"`
	UpdateTime    int32   `gorm:"column:updateTime;comment:更新时间" json:"updateTime"`
	TotalProfLoss float64 `gorm:"column:totalProfLoss;default:0" json:"totalProfLoss"`
	TotalEffBet   float64 `gorm:"column:totalEffBet;default:0" json:"totalEffBet"`
	TotalPump     float64 `gorm:"column:totalPump;default:0" json:"totalPumpv"`
	TotalBet      float64 `gorm:"column:totalBet;default:0" json:"totalBet"`
	CurrencyType  string  `gorm:"column:currencyType;size:30;" json:"currencyType"`
	Revenue       float64 `gorm:"column:revenue;default:0" json:"revenue"`
	InCtl         byte    `gorm:"column:isCtl;default:0;comment:是否在控制中 0 否  1 是" json:"isCtl"`
	IsTourist     int32   `gorm:"column:isTourist;default:0;comment:是否是游客账户 0 否  1 是" json:"isTourist"`
}

func (t *User) TableName() string {
	return "gp_user"
}
