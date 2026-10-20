package dao

import (
	"app/config"
	"app/tables/manager"
	"app/tables/player"
	"fmt"
	"math/rand"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type DbDao struct {
	Player  *gorm.DB
	Manager *gorm.DB
}

var dbIns *DbDao = nil

func InitDB(c *config.RunConfig) error {
	p := c.Mysql["player"]
	playerSource := fmt.Sprintf("%s:%s@(%s:%d)/%s?charset=utf8", p.User, p.Password, p.Host, p.Port, p.Database)
	player, err := gorm.Open(mysql.Open(playerSource), &gorm.Config{})
	if err != nil {
		return err
	}
	mm := c.Mysql["manager"]
	managerSource := fmt.Sprintf("%s:%s@(%s:%d)/%s?charset=utf8", mm.User, mm.Password, mm.Host, mm.Port, mm.Database)
	manager, err := gorm.Open(mysql.Open(managerSource), &gorm.Config{})
	if err != nil {
		return err
	}
	dbIns = &DbDao{Player: player, Manager: manager}
	return nil
}

func Mysql() *DbDao {
	return dbIns
}

func (d *DbDao) LoadAgents() []*manager.Agent {
	agents := make([]*manager.Agent, 0, 64)
	d.Manager.Model(manager.Agent{}).Debug().Find(&agents)
	return agents
}

func (d *DbDao) LoadApiConfigs() []*manager.ApiConfig {
	apiConfigs := make([]*manager.ApiConfig, 0, 20)
	d.Manager.Model(manager.ApiConfig{}).Debug().Find(&apiConfigs)
	return apiConfigs
}

// 获取代理库中的玩家信息
func (d *DbDao) GetAgentPlayerInfoByAgentIdAndAcc(agentId int64, acc string) *manager.User {
	aPlayer := &manager.User{}
	if d.Manager.Model(manager.User{}).Debug().Where("agentId=? and userId = ?", agentId, acc).Take(aPlayer).RowsAffected <= 0 {
		return nil
	}
	return aPlayer
}

// 获取玩家游戏对象
func (d *DbDao) GetGamePlayer(agentId, id int64) *player.Player {
	p := &player.Player{}
	if d.Player.Model(player.Player{}).Debug().Where("proxy_id=? and user_id = ?", agentId, id).Take(p).RowsAffected <= 0 {
		return nil
	}
	return p
}

func (d *DbDao) AddNewPlayer(agentId, webId int64, money float64, acc, nickName, ip, currencyType string, isTourist int32) (*manager.User, *player.Player) {
	txAgent := d.Manager.Begin()
	n := time.Now().Unix()
	u := &manager.User{
		UserId:       acc,
		NickName:     nickName,
		AgentId:      agentId,
		WebId:        webId,
		LogIp:        ip,
		LogTime:      n,
		CreateTime:   int32(n),
		UpdateTime:   int32(n),
		CurrencyType: currencyType,
		State:        1,
		IsTourist:    isTourist,
	}
	rAgent := txAgent.Create(u)
	if e := rAgent.Error; e != nil {
		if e := txAgent.Rollback(); e != nil {
			zap.L().Error("增加代理库玩家信息失败，回滚异常", zap.Any("err", e), zap.Any("acc", acc))
		}
		return nil, nil
	}
	txPlayer := d.Player.Begin()
	ui := &player.Player{
		UserId:       u.Id,
		NickName:     nickName,
		Account:      acc,
		Score:        decimal.NewFromFloat(money),
		Sex:          rand.Intn(2) + 1,
		Pic:          fmt.Sprintf("%d", rand.Intn(25)+1),
		ProxyId:      int(agentId),
		WebsiteId:    int(webId),
		LoginIp:      ip,
		LoginTime:    time.Now().Unix(),
		CurrencyType: currencyType,
		IsTourist:    isTourist,
	}
	if err := d.Player.Create(ui).Error; err != nil {
		zap.L().Error("插入新玩家数据失败", zap.Any("err", err))
		_ = txPlayer.Rollback()
		_ = txAgent.Rollback()
		return nil, nil
	}
	_ = txAgent.Commit()
	_ = txPlayer.Commit()
	return u, ui
}

func (d *DbDao) AddScoreLog(userId int, agentId int64, acc string, score float64, logType int, userNewScore decimal.Decimal) bool {
	nt := time.Now()
	logOn := fmt.Sprintf("%d", nt.UnixNano())
	err := d.Manager.Debug().Exec("insert into gp_user_score_log values(?,?,?,?,?,?,?,?,?)", 0, logOn, agentId, userId, acc, score, logType, userNewScore.InexactFloat64(), nt.Unix()).Error
	zap.L().Debug("AddScoreLog", zap.Any("err", err))
	return true
}

func (d *DbDao) GetAgentPlayersByAcc(accs []string) []*player.Player {
	result := make([]*player.Player, 0, 128)
	d.Player.Model(&player.Player{}).Debug().Where("account in ? ", accs).Find(&result)
	return result
}
