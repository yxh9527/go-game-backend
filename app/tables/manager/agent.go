package manager

// 代理表
type Agent struct {
	Id              int64   `gorm:"column:id;primaryKey;autoIncrement;not null;" json:"id"`
	AesKey          string  `gorm:"column:aesKey;size:40;" json:"aesKey"`
	Md5Key          string  `gorm:"column:md5Key;size:40;" json:"md5Key"`
	AgentKey        string  `gorm:"column:agentKey;size:50;" json:"agentKey"`
	NickName        string  `gorm:"column:nickName;size:50;index:agent_name;" json:"nickName"`
	Email           string  `gorm:"column:email;size:100;" json:"email"`
	WebId           int64   `gorm:"column:webId;default:0;index:agent_webId;" json:"webId"`
	UpperLevel      int64   `gorm:"column:upperLevel;default:0;comment:上级代理ID;" json:"upperLevel"`
	Level           int32   `gorm:"column:level;default:1;comment:代理级别 1一级代理 2二级代理 3三级代理;" json:"level"`
	Remarks         string  `gorm:"column:remarks;size:1000;comment:备注;" json:"remarks"`
	Contacts        string  `gorm:"column:contacts;size:50;comment:负责人;" json:"contacts"`
	Phone           string  `gorm:"column:phone;size:50;comment:负责人联系方式;" json:"phone"`
	StartTime       int32   `gorm:"column:startTime;index:agent_time;comment:生效开始时间" json:"startTime"`
	EndTime         int32   `gorm:"column:endTime;index:agent_time;comment:生效结束时间" json:"endTime"`
	IsPermanent     int32   `gorm:"column:isPermanent;default:1;comment:是否永久  1是 0否" json:"isPermanent"`
	IsFrozen        int32   `gorm:"column:isFrozen;default:0;comment:是否冻结 1是 0否" json:"isFrozen"`
	State           int32   `gorm:"column:state;default:1;comment:状态 1正常 2维护" json:"state"`
	CreateTime      int32   `gorm:"column:createTime;index:agent_createTime;comment:创建时间" json:"createTime"`
	UpdateTime      int32   `gorm:"column:updateTime;comment:更新时间" json:"updateTime"`
	Yushe           float64 `gorm:"column:yushe;default:0;comment:预设" json:"yushe"`
	Point           float64 `gorm:"column:point;default:0;comment:点数" json:"point"`
	Stock           float64 `gorm:"column:stock;default:0;comment:库存" json:"stock"`
	IsDel           byte    `gorm:"column:isDel;default:0;comment:是否删除 1是 0否" json:"isDel"`
	GameIds         string  `gorm:"column:gameIds;comment:代理游戏" json:"gameIds"`
	TotalProfLoss   float64 `gorm:"column:totalProfLoss;default:0" json:"totalProfLoss"`
	TotalUserNumber float64 `gorm:"column:totalUserNumber;default:0" json:"totalUserNumber"`
	TotalBet        float64 `gorm:"column:totalBet;default:0" json:"totalBet"`
	TotalEffBet     float64 `gorm:"column:totalEffBet;default:0" json:"totalEffBet"`
	TotalPump       float64 `gorm:"column:totalPump;default:0" json:"totalPump"`
}

func (t *Agent) TableName() string {
	return "gp_agent"
}
