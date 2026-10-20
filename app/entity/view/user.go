package view

import "app/tables/manager"

type UserInfo struct {
	*manager.User
	WebName   string `gorm:"column:webName" json:"webName"`
	AgentName string `gorm:"column:agentName" json:"agentName"`
}
