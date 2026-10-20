package manager

type PoolConfig struct {
	Id    int64  `gorm:"column:id;primaryKey;autoIncrement;" json:"id"`
	Key   string `gorm:"column:key;size:255;" json:"key"`
	Value string `gorm:"column:value;type:text" json:"value"`
}

func (t *PoolConfig) TableName() string {
	return "pool_config"
}
