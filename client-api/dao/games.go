package dao

import (
	"strconv"
	"strings"
	"sync"
)

// GameMap gameId -> symbol，用于兼容只有 symbol 的老注单。
type GameMap struct {
	mu   sync.RWMutex
	byId map[uint32]string
}

func NewGameMap(raw map[string]string) *GameMap {
	m := &GameMap{byId: make(map[uint32]string)}
	for k, v := range raw {
		id64, err := strconv.ParseUint(strings.TrimSpace(k), 10, 32)
		if err != nil || id64 == 0 {
			continue
		}
		symbol := strings.TrimSpace(v)
		if symbol == "" {
			continue
		}
		m.byId[uint32(id64)] = symbol
	}
	return m
}

func (m *GameMap) Symbol(gameId uint32) string {
	if m == nil {
		return ""
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.byId[gameId]
}
