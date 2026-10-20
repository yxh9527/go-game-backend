package dao

import (
	"app/tables/manager"
	"app/tables/player"
	"fmt"
	"sync"
	"time"

	"app/config"

	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var playerDB, managerDB *gorm.DB = nil, nil
var gameMamanger *GamesManager = nil
var agentManager *AgentsManager = nil

func InitDB(sysConfig *config.RunConfig) error {
	pc := sysConfig.Mysql["player"]
	playerSource := fmt.Sprintf("%s:%s@(%s:%d)/%s?charset=utf8", pc.User, pc.Password, pc.Host, pc.Port, pc.Database)
	p, err := gorm.Open(mysql.Open(playerSource), &gorm.Config{})
	if err != nil {
		return err
	}
	mc := sysConfig.Mysql["manager"]
	managerSource := fmt.Sprintf("%s:%s@(%s:%d)/%s?charset=utf8", mc.User, mc.Password, mc.Host, mc.Port, mc.Database)
	m, me := gorm.Open(mysql.Open(managerSource), &gorm.Config{})
	if me != nil {
		return me
	}
	playerDB, managerDB = p, m
	if !p.Migrator().HasColumn(&player.Player{}, "ex_data") {
		if err := p.Migrator().AddColumn(&player.Player{}, "ExData"); err != nil {
			return err
		}
	}
	return nil
}

type DBDao struct {
	player  *gorm.DB
	manager *gorm.DB
}

func NewDBDao() *DBDao {
	return &DBDao{player: playerDB, manager: managerDB}
}

func PDIns() *gorm.DB {
	return playerDB
}

func MDIns() *gorm.DB {
	return managerDB
}

func (dd *DBDao) GetPlayer(playerId uint32) (*player.Player, error) {
	var user player.Player
	err := dd.player.Model(user).Where("user_id=?", playerId).Take(&user).Error
	return &user, err
}

type GamesManager struct {
	lock       *sync.RWMutex
	games      map[string]*manager.Game
	gamesIdMap map[int64]*manager.Game
	db         *gorm.DB
}

func (gm *GamesManager) load(db *gorm.DB) {
	gm.lock.Lock()
	defer gm.lock.Unlock()
	gm.db = db
	tmp := make([]*manager.Game, 0, 128)
	db.Model(manager.Game{}).Debug().Find(&tmp)
	for _, item := range tmp {
		gm.games[item.ConfName] = item
		gm.gamesIdMap[int64(item.Number)] = item
	}
}

func (gm *GamesManager) Get(symbol string) *manager.Game {
	gm.lock.Lock()
	tmp := &manager.Game{}
	g := gm.games[symbol]
	gm.lock.Unlock()
	if g == nil {
		gm.db.Model(manager.Game{}).Debug().Where("confName = ?", symbol).Take(tmp)
		gm.lock.Lock()
		gm.games[tmp.ConfName] = tmp
		gm.gamesIdMap[int64(tmp.Number)] = tmp
		gm.lock.Unlock()
		return tmp
	} else {
		return g
	}
}

// id = number
func (gm *GamesManager) GetById(number int64) *manager.Game {
	gm.lock.Lock()
	tmp := &manager.Game{}
	g := gm.gamesIdMap[number]
	gm.lock.Unlock()
	if g == nil {
		gm.db.Model(manager.Game{}).Debug().Where("number = ?", number).Take(tmp)
		gm.lock.Lock()
		gm.games[tmp.ConfName] = tmp
		gm.gamesIdMap[int64(tmp.Number)] = tmp
		gm.lock.Unlock()
		return tmp
	} else {
		return g
	}
}

func (gm *GamesManager) Add(g *manager.Game) {
	gm.lock.Lock()
	defer gm.lock.Unlock()

	gm.gamesIdMap[int64(g.Number)] = g
	gm.games[g.ConfName] = g
}

func (gm *GamesManager) ChangeStatus(number, status uint32) {
	gm.lock.Lock()
	defer gm.lock.Unlock()

	game := gm.gamesIdMap[int64(number)]
	if game != nil {
		game.IsFrozen = int16(status)
		if game.IsFrozen == 1 {
			game.State = 2
		} else {
			game.State = 1
		}
	}
}

func InitGameManager() {
	if gameMamanger == nil {
		tmp := &GamesManager{lock: &sync.RWMutex{}, games: make(map[string]*manager.Game), gamesIdMap: make(map[int64]*manager.Game)}
		tmp.load(managerDB)
		gameMamanger = tmp
	}
}

func GamesManagerIns() *GamesManager {
	return gameMamanger
}

func InitAgentManager() {
	if agentManager == nil {
		tmp := &AgentsManager{lock: &sync.RWMutex{}, agents: make(map[int64]*manager.Agent)}
		tmp.load(managerDB)
		agentManager = tmp
		go func() {
			defer func() {
				if err := recover(); err != nil {
					zap.L().Error("panic", zap.Any("err", err))
				}
			}()
			ticker := time.NewTicker(300 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				tmp.load(managerDB)
			}
		}()
	}
}

func AgentManagerIns() *AgentsManager {
	return agentManager
}

type AgentsManager struct {
	lock   *sync.RWMutex
	agents map[int64]*manager.Agent
}

func (gm *AgentsManager) load(db *gorm.DB) {
	tmp := make([]*manager.Agent, 0, 128)
	db.Model(manager.Agent{}).Debug().Find(&tmp)
	gm.lock.Lock()
	defer gm.lock.Unlock()
	for _, item := range tmp {
		gm.agents[item.Id] = item
	}
}

func (gm *AgentsManager) Get(id int64) *manager.Agent {
	gm.lock.RLock()
	defer gm.lock.RUnlock()

	return gm.agents[id]
}

func (gm *AgentsManager) Add(a *manager.Agent) {
	gm.lock.Lock()
	defer gm.lock.Unlock()

	gm.agents[a.Id] = a
}

func (gm *AgentsManager) ChangeStatus(id, status uint32) {
	gm.lock.Lock()
	defer gm.lock.Unlock()

	agent := gm.agents[int64(id)]
	if agent != nil {
		agent.IsFrozen = int32(status)
		if agent.IsFrozen == 1 {
			agent.State = 2
		} else {
			agent.State = 1
		}
	}
}
