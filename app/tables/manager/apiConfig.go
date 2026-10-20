package manager

//
type ApiConfig struct {
	Id            int64  `gorm:"column:id;primaryKey;autoIncrement;" json:"id"`
	Name          string `gorm:"column:name;size:255;" json:"name"`
	Min           int    `gorm:"column:min_score" json:"min_score"`
	Max           int    `gorm:"column:max_score" json:"max_score"`
	ClientApiUrls string `gorm:"column:client_api_urls" json:"client_api_urls"`
	HallUrls      string `gorm:"column:hall_urls" json:"hall_urls"`
}

func (t *ApiConfig) TableName() string {
	return "gp_api_config"
}
