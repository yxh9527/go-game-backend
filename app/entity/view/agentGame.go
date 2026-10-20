package view

import "app/tables/manager"

type AgentGame struct {
	*manager.AgentGame
	AgentName string `gorm:"column:agentName" json:"agentName"`
	GameName  string `gorm:"column:gameName" json:"gameName"`
}
