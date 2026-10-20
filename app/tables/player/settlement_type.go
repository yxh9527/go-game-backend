package player

type SettlementType struct {
	Id         int64  `gorm:"column:id;primaryKey;autoIncrement;" json:"id"` //id
	Name       string `gorm:"column:name;size:128;" json:"name"`             //name
	CreateTime int64  `gorm:"column:create_time"`
}

func (p *SettlementType) TableName() string {
	return "settlement_type"
}
