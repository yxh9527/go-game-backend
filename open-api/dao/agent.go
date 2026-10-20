package dao

import (
	"app/tables/manager"
	"sync"
	"time"

	"go.uber.org/zap"
)

var agentMgr *AgentMgr

type AgentMgr struct {
	L      sync.RWMutex
	Agents map[int64]*manager.Agent
}

func InitAgentMgr() {
	if agentMgr == nil {
		a := &AgentMgr{
			L:      sync.RWMutex{},
			Agents: make(map[int64]*manager.Agent),
		}
		a.load()
		a.sync()
		agentMgr = a
	}
}

func GetAgentMgr() *AgentMgr {
	return agentMgr
}

func (a *AgentMgr) Get(id int64) *manager.Agent {
	a.L.RLock()
	defer a.L.RUnlock()

	if agent, ok := a.Agents[id]; ok {
		return agent
	}
	return nil
}

// 启动加载代理所有数据
func (a *AgentMgr) load() {
	a.L.Lock()
	defer a.L.Unlock()

	if agents := Mysql().LoadAgents(); len(agents) > 0 {
		for _, v := range agents {
			a.Agents[v.Id] = v
		}
	}
}

// 定时对比差异，新增代理，同步数据
func (a *AgentMgr) sync() {
	go func() {

		ticker := time.NewTicker(time.Second * 60)
		defer ticker.Stop()

		defer func() {
			if err := recover(); err != nil {
				zap.L().Error("panic", zap.Any("err", err))
			}
		}()

		for range ticker.C {
			a.load()
		}
	}()
}
