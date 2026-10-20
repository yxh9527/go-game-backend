package dao

import (
	"app/config"
	"context"
	"fmt"
	"game-data-summary/entity"
	"runtime"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

var GamesMgr *GamesManager = nil
var DB *DbInfo = nil

// 玩家信息
type AgentInfo struct {
	AgentId  uint32 `db:"id"` //玩家id
	WebId    uint32 `db:"webId"`
	NickName string `db:"nickName"`
}

type UserInfo struct {
	Id uint32 `db:"id"`
}

type GamesManager struct {
	lock *sync.RWMutex

	games map[int]*entity.Game
}

func InitGamesManager() {
	if GamesMgr == nil {
		GamesMgr = &GamesManager{
			lock:  &sync.RWMutex{},
			games: make(map[int]*entity.Game),
		}
		GamesMgr.load()
		go func() {
			if err := recover(); err != nil {
				zap.L().Error("panic", zap.Any("err", err))
			}
			ticker := time.NewTicker(200 * time.Second)
			for range ticker.C {
				GamesMgr.load()
			}
		}()
	}
}

func (g *GamesManager) load() {
	rows, err := DB.client.QueryContext(context.Background(), `select id,number,name,nameZH,confName from gp_game `)
	if err != nil {
		zap.L().Debug("err", zap.Error(err))
		return
	}
	result := make(map[int]*entity.Game)
	for rows.Next() {
		id, number, name, nameZH, confName := 0, 0, "", "", ""
		if err := rows.Scan(&id, &number, &name, &nameZH, &confName); err != nil {
			zap.L().Debug("err", zap.Error(err))
			continue
		}
		result[number] = &entity.Game{
			Id:       int64(id),
			Number:   number,
			Name:     name,
			NameZH:   nameZH,
			ConfName: confName,
		}
	}
	_ = rows.Close()
	g.lock.Lock()
	g.games = result
	g.lock.Unlock()
}

func (g *GamesManager) List() []*entity.Game {
	games := make([]*entity.Game, 0, 128)
	g.lock.Lock()
	for _, v := range g.games {
		games = append(games, v)
	}
	g.lock.Unlock()
	return games
}

type DbInfo struct {
	client *sqlx.DB
}

func NewDbInfo(cfg *config.RunConfig) {
	if DB == nil {
		DB = &DbInfo{
			client: &sqlx.DB{},
		}
		if c, err := initDB(cfg); err == nil {
			DB.client = c
		} else {
			zap.L().Error("创建mysql链接失败", zap.Error(err))
		}
	}
}

func initDB(cfg *config.RunConfig) (*sqlx.DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	source := "%s:%s@(%s:%d)/%s?charset=utf8"
	c := cfg.Mysql["manager"]
	db, err := sqlx.ConnectContext(ctx, "mysql", fmt.Sprintf(source, c.User, c.Password, c.Host, c.Port, c.Database))
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(runtime.NumCPU())
	db.SetMaxIdleConns(runtime.NumCPU())
	db.SetConnMaxLifetime(10 * time.Second)

	return db, nil
}

func (ud *DbInfo) GetAgentList() []*AgentInfo {
	result := make([]*AgentInfo, 0, 128)
	rows, err := ud.client.QueryContext(context.Background(), `select id,nickName,webId from gp_agent `)
	defer func() {
		_ = rows.Close()
	}()
	if err != nil {
		zap.L().Debug("err", zap.Error(err))
		return nil
	}
	for rows.Next() {
		id, name, webId := 0, "", 0
		if err := rows.Scan(&id, &name, &webId); err != nil {
			zap.L().Debug("err", zap.Error(err))
			continue
		}
		agent := &AgentInfo{
			AgentId:  uint32(id),
			NickName: name,
			WebId:    uint32(webId),
		}
		result = append(result, agent)
	}
	return result
}

// 获取新注册的用户数量
func (ud *DbInfo) GetNewRegisterPlayerCount(agentId, webId, bTime, eTime uint32) int {
	cnt := 0
	err := ud.client.GetContext(context.Background(), &cnt, `
select count(id) from gp_user where createTime <? and createTime>? and agentId=? and webId=?`,
		eTime, bTime, agentId, webId)
	if err != nil {
		zap.L().Error("获取新注册玩家数量失败", zap.Any("bTime", bTime), zap.Any("eTime", eTime), zap.Error(err))
		return 0
	}
	return cnt
}

func (ud *DbInfo) GetNewPlayersList(agentId, webId, bTime, eTime uint32) map[int]bool {
	result := make(map[int]bool)
	rows, err := ud.client.QueryContext(context.Background(), `
	select id  from gp_user where agentId=? and webId=? and createTime<? and createTime>? `, agentId, webId, eTime, bTime)
	if err != nil {
		zap.L().Debug("err", zap.Error(err))
		return nil
	}
	for rows.Next() {
		id := 0
		if err := rows.Scan(&id); err != nil {
			zap.L().Debug("err", zap.Error(err))
			continue
		}
		result[id] = true
	}
	_ = rows.Close()
	return result
}

// 玩家活跃相关统计入库
func (ud *DbInfo) AddPlayerSummaryData(agentId int, recordTime, webId, newRegister, allActive, newRegisterActive uint32) {
	_, _ = ud.client.ExecContext(context.Background(), `insert into gp_player_summary values(?,?,?,?,?,?)`,
		recordTime, newRegister, allActive, newRegisterActive, agentId, webId)
}

// 游戏数据相关统计入库
func (ud *DbInfo) AddGameSummaryData(agentId int, recordTime, webId uint32, gameId int, eff, pro, ss decimal.Decimal, keepAlive uint32, notKeepAlive uint32, acitveCount uint32, betsCount uint32) {
	_, _ = ud.client.ExecContext(context.Background(), `insert into gp_game_data_summary values(?,?,?,?,?,?,?,?,?,?,?)`,
		recordTime, agentId, gameId, webId, eff, pro, ss, keepAlive, notKeepAlive, acitveCount, betsCount)
}

// 游戏周活跃人数信息
func (ud *DbInfo) AddGameWeekActive(agentId int, gameId, weekActive, bTime, eTime uint32) {
	_, _ = ud.client.ExecContext(context.Background(), `insert into gp_game_week_active values(?,?,?,?,?)`,
		agentId, gameId, weekActive, bTime, eTime)
}

// 玩家盈亏排行数据
func (ud *DbInfo) AddPlayerProRank(agentId int, userId, recordDate, recordType, level, isWin uint32, eff, pro decimal.Decimal) {
	_, _ = ud.client.ExecContext(context.Background(), `insert into gp_player_pro_rank values(?,?,?,?,?,?,?,?)`,
		agentId, userId, recordDate, recordType, level, isWin, eff, pro)
}

///////////////////////////////////////////////每小时打点

func (ud *DbInfo) AddPlayerActiveCountByHour(recordTime, activeCount uint32) {
	_, _ = ud.client.ExecContext(context.Background(), `insert into gp_player_data_hour values(?,?)`,
		recordTime, activeCount)
}

func (ud *DbInfo) AddGameDataByHour(recordTime, cnt, gameId, active uint32, eff, pro decimal.Decimal) {
	_, _ = ud.client.ExecContext(context.Background(), `insert into gp_game_data_hour values(?,?,?,?,?,?)`,
		recordTime, gameId, eff, pro, cnt, active)
}
