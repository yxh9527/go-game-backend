package dao

import (
	"errors"
	"fmt"

	"app/tables/manager"
	"app/tables/player"
	"client-api/config"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DBDao struct {
	player  *gorm.DB
	manager *gorm.DB
}

var dbIns *DBDao

func InitDB(c *config.RunConfig) error {
	if c == nil || c.Mysql == nil {
		return errors.New("mysql config missing")
	}
	playerCfg, ok1 := c.Mysql["player"]
	managerCfg, ok2 := c.Mysql["manager"]
	if !ok1 || !ok2 {
		return errors.New("mysql.player/manager required")
	}
	open := func(cfg config.MysqlItem) (*gorm.DB, error) {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
		return gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	}
	pdb, err := open(playerCfg)
	if err != nil {
		return err
	}
	mdb, err := open(managerCfg)
	if err != nil {
		return err
	}
	dbIns = &DBDao{player: pdb, manager: mdb}
	return nil
}

func DB() *DBDao {
	return dbIns
}

func (d *DBDao) Ping() error {
	if d == nil {
		return errors.New("db not initialized")
	}
	sqlDB, err := d.player.DB()
	if err != nil {
		return err
	}
	if err := sqlDB.Ping(); err != nil {
		return err
	}
	sqlDB2, err := d.manager.DB()
	if err != nil {
		return err
	}
	return sqlDB2.Ping()
}

func (d *DBDao) GetGameByNumber(number int64) *manager.Game {
	if d == nil || number <= 0 {
		return nil
	}
	g := &manager.Game{}
	if err := d.manager.Where("number = ?", number).Take(g).Error; err != nil {
		return nil
	}
	return g
}

func (d *DBDao) GetUserTotalEffBet(userId uint32) float64 {
	if d == nil || userId == 0 {
		return 0
	}
	u := &manager.User{}
	if err := d.manager.Select("totalEffBet").Where("id = ?", userId).Take(u).Error; err != nil {
		return 0
	}
	return u.TotalEffBet
}

func (d *DBDao) GetPlayer(userId uint32) (*player.Player, error) {
	if d == nil || userId == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	p := &player.Player{}
	err := d.player.Where("user_id = ?", userId).Take(p).Error
	return p, err
}

func (d *DBDao) GetAgent(agentId int64) (*manager.Agent, error) {
	if d == nil || agentId <= 0 {
		return nil, gorm.ErrRecordNotFound
	}
	a := &manager.Agent{}
	err := d.manager.Where("id=? AND isDel=0", agentId).Take(a).Error
	return a, err
}

func (d *DBDao) ResolveTopAgentId(agentId int64) uint32 {
	current := agentId
	for i := 0; i < 8 && current > 0; i++ {
		agent, err := d.GetAgent(current)
		if err != nil || agent == nil {
			break
		}
		if agent.Level <= 1 || agent.UpperLevel <= 0 {
			return uint32(agent.Id)
		}
		current = agent.UpperLevel
	}
	if agentId > 0 {
		return uint32(agentId)
	}
	return 0
}

func (d *DBDao) LoadGameMap() map[string]string {
	out := map[string]string{}
	if d == nil {
		return out
	}
	var games []manager.Game
	if err := d.manager.Find(&games).Error; err != nil {
		zap.L().Error("load games failed", zap.Error(err))
		return out
	}
	for _, g := range games {
		if g.Number > 0 && g.ConfName != "" {
			out[fmt.Sprintf("%d", g.Number)] = g.ConfName
		}
	}
	return out
}
