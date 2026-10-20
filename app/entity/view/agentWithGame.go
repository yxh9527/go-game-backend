package view

import "app/tables/manager"

type AgentListItem struct {
	*manager.Agent
	WebName string `gorm:"column:webName;size:50;" json:"webName"`
}
