package dao

import (
	"app/tables/manager"
	"sync"
)

var GameCacheIns *GameCacheMgr

type GameCacheMgr struct {
	lock  *sync.Mutex
	Games map[string]*manager.Game
}

func loadGameData() {
	GameCacheIns.lock.Lock()
	defer GameCacheIns.lock.Unlock()
	games := make([]*manager.Game, 0, 128)
	Mysql().Manager.Model(manager.Game{}).Debug().Find(&games)
	for _, v := range games {
		GameCacheIns.Games[v.ConfName] = v
	}
}

// 加载游戏缓存列表
func InitGameCacheMgr() {
	if GameCacheIns == nil {
		GameCacheIns = &GameCacheMgr{
			lock:  &sync.Mutex{},
			Games: make(map[string]*manager.Game),
		}
		loadGameData()
	}
}

func (gcm *GameCacheMgr) GetGame(symbol string) *manager.Game {
	gcm.lock.Lock()
	defer gcm.lock.Unlock()

	game := gcm.Games[symbol]
	if game == nil {
		game = &manager.Game{}
		Mysql().Manager.Model(manager.Game{}).Debug().Where("confName = ?", symbol).Take(game)
		gcm.Games[symbol] = game
	}
	return game
}
