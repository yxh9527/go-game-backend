package tables

import (
	"app/esindex"
	"app/tables/manager"
	"app/tables/player"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type poolConfigItem struct {
	Min        string `json:"min"`
	Normal     string `json:"normal"`
	Max        string `json:"max"`
	NormalRate string `json:"normalRate"`
	MinRate    string `json:"minRate"`
	MaxRate    string `json:"maxRate"`
	Control    string `json:"control"`
	Revenue    string `json:"revenue"`
	Base       string `json:"base"`
}

type poolConfigValue struct {
	Symbol string                    `json:"symbol"`
	Name   string                    `json:"name"`
	NameZH string                    `json:"nameZH"`
	GameID int                       `json:"gameId"`
	Pool   map[string]poolConfigItem `json:"pool"`
}

// 初始化数据库结构
func InitMysqlDb(m, p *gorm.DB) {
	if !m.Migrator().HasTable(&manager.MsgType{}) {
		m.AutoMigrate(&manager.MsgType{})
		mt := []*manager.MsgType{
			{Id: 1, Title: "游戏公告", Class: 1},
			{Id: 2, Title: "维护公告", Class: 1},
			{Id: 3, Title: "管理消息", Class: 2},
		}
		m.Create(mt)
	}
	if !m.Migrator().HasTable(&manager.SystemConf{}) {
		m.AutoMigrate(&manager.SystemConf{})
		sc := []*manager.SystemConf{
			{Id: 1, SystemState: 1},
		}
		m.Create(sc)
	}

	if !m.Migrator().HasTable(&manager.SystemUser{}) {
		m.AutoMigrate(&manager.SystemUser{})
		su := []*manager.SystemUser{
			{Id: 1, Account: "admin1030", Password: "e10adc3949ba59abbe56e057f20f883e", UType: 1, AgentId: 0, UName: "", CreateTime: time.Now().Unix(), IsForzen: 0},
		}
		m.Create(su)
	}
	if !m.Migrator().HasTable(&manager.GameType{}) {
		m.AutoMigrate(&manager.GameType{})
	}
	{
		var gameTypeCount int64
		m.Model(&manager.GameType{}).Count(&gameTypeCount)
		if gameTypeCount == 0 {
			m.Create([]*manager.GameType{
				{Name: "slots"},
				{Name: "fruits"},
				{Name: "ug5"},
			})
		}
	}
	if !m.Migrator().HasTable(&manager.Game{}) {
		m.AutoMigrate(&manager.Game{})
	}
	{
		var gameCount int64
		m.Model(&manager.Game{}).Count(&gameCount)
		if gameCount == 0 {
			now := int(time.Now().Unix())
			typeByName := map[string]int{}
			var gameTypes []manager.GameType
			m.Find(&gameTypes)
			for _, gt := range gameTypes {
				typeByName[gt.Name] = int(gt.Id)
			}
			seeds := DefaultGameSeeds()
			games := make([]*manager.Game, 0, len(seeds))
			for _, seed := range seeds {
				gameTypeId := typeByName[seed.TypeName]
				if gameTypeId == 0 {
					continue
				}
				games = append(games, &manager.Game{
					Number:     seed.Number,
					Name:       seed.Symbol,
					NameZH:     seed.NameZH,
					ConfName:   seed.Symbol,
					GameClass:  1,
					GameType:   gameTypeId,
					LimitTime:  "10",
					IsFrozen:   0,
					State:      1,
					CreateTime: now,
					UpdateTime: now,
					Weight:     0,
					ShowType:   1,
					IsShow:     1,
				})
			}
			if len(games) > 0 {
				m.Create(games)
			}
		}
	}
	if !m.Migrator().HasTable(&manager.PoolConfig{}) {
		m.AutoMigrate(&manager.PoolConfig{})
		var games []manager.Game
		m.Find(&games)

		poolConfigs := make([]*manager.PoolConfig, 0, len(games))
		for _, game := range games {
			value, err := json.Marshal(poolConfigValue{
				Symbol: game.ConfName,
				Name:   game.ConfName,
				NameZH: game.NameZH,
				GameID: game.Number,
				Pool: map[string]poolConfigItem{
					"1": {
						Min:        "300",
						Normal:     "900",
						Max:        "3000",
						NormalRate: "0.99",
						MinRate:    "0.99",
						MaxRate:    "0.99",
						Control:    "1",
						Revenue:    "0.03",
						Base:       "0",
					},
				},
			})
			if err != nil {
				panic(err)
			}

			poolConfigs = append(poolConfigs, &manager.PoolConfig{
				Key:   esindex.ConfigKey("pool", game.ConfName),
				Value: string(value),
			})
		}
		poolConfigs = append(poolConfigs, &manager.PoolConfig{
			Key:   esindex.ConfigKey("currency"),
			Value: "{\"currency\":{\"CNY\":1,\"USD\":7.2,\"EUR\":7.8,\"INR\":0.085,\"USDT\":7.2,\"BRL\":1.42,\"HK\":0.92,\"JPY\":0.045,\"MXN\":0.43,\"IDR\":0.0005,\"MYR\":1.5,\"PHP\":0.13,\"SGD\":5.4,\"VND\":0.0003,\"THB\":0.2,\"KRW\":0.005}}",
		})
		poolConfigs = append(poolConfigs, &manager.PoolConfig{
			Key:   esindex.ConfigKey("ctrl", "default"),
			Value: "{\"award_config\":[{\"id\":1,\"name\":\"single\",\"min\":\"0\",\"max\":\"0\",\"pool_odds\":[{\"odds\":\"45\",\"multiple\":\"10\"},{\"odds\":\"50\",\"multiple\":\"20\"},{\"odds\":\"60\",\"multiple\":\"40\"}]},{\"id\":2,\"name\":\"default\",\"min\":\"2\",\"max\":\"5\",\"pool_odds\":[{\"odds\":\"95\",\"multiple\":\"80\"},{\"odds\":\"97\",\"multiple\":\"100\"},{\"odds\":\"98\",\"multiple\":\"120\"}]},{\"id\":3,\"name\":\"0-10\",\"min\":\"0\",\"max\":\"10\",\"pool_odds\":[{\"odds\":\"95\",\"multiple\":\"90\"},{\"odds\":\"95\",\"multiple\":\"120\"},{\"odds\":\"97\",\"multiple\":\"150\"}]},{\"id\":4,\"name\":\"10-35\",\"min\":\"10\",\"max\":\"35\",\"pool_odds\":[{\"odds\":\"65\",\"multiple\":\"50\"},{\"odds\":\"80\",\"multiple\":\"90\"},{\"odds\":\"88\",\"multiple\":\"130\"}]},{\"id\":5,\"name\":\"35-100\",\"min\":\"35\",\"max\":\"100\",\"pool_odds\":[{\"odds\":\"60\",\"multiple\":\"50\"},{\"odds\":\"80\",\"multiple\":\"70\"},{\"odds\":\"90\",\"multiple\":\"120\"}]},{\"id\":6,\"name\":\"-25-0\",\"min\":\"-25\",\"max\":\"0\",\"pool_odds\":[{\"odds\":\"90\",\"multiple\":\"50\"},{\"odds\":\"90\",\"multiple\":\"90\"},{\"odds\":\"95\",\"multiple\":\"150\"}]},{\"id\":7,\"name\":\"-50--25\",\"min\":\"-50\",\"max\":\"-25\",\"pool_odds\":[{\"odds\":\"90\",\"multiple\":\"50\"},{\"odds\":\"95\",\"multiple\":\"120\"},{\"odds\":\"98\",\"multiple\":\"150\"}]},{\"id\":8,\"name\":\"-100--50\",\"min\":\"-100\",\"max\":\"-50\",\"pool_odds\":[{\"odds\":\"90\",\"multiple\":\"120\"},{\"odds\":\"95\",\"multiple\":\"150\"},{\"odds\":\"99\",\"multiple\":\"180\"}]}],\"gameId\":0,\"symbol\":\"\"}",
		})
		poolConfigs = append(poolConfigs, &manager.PoolConfig{
			Key:   esindex.ConfigKey("autoCtrl"),
			Value: "[{\"totalEffect\":\"999999\",\"totalProfLoss\":\"500000\",\"totalProfLossRate\":\"60\",\"controlRate\":\"50\",\"score\":\"100000\"}]",
		})
		poolConfigs = append(poolConfigs, &manager.PoolConfig{
			Key:   esindex.ConfigKey("system"),
			Value: "{\"game_url\":[\"http://127.0.0.1:9700\"],\"replays\":[\"\"]}",
		})
		m.Create(poolConfigs)
	}
	//manager
	m.AutoMigrate(&manager.Agent{},
		&manager.AgentConfig{},
		&manager.AgentGame{},
		&manager.AgentGameConf{},
		&manager.ApiConfig{},
		&manager.Feedback{},
		&manager.GameDataHour{},
		&manager.GameDataSummary{},
		&manager.Log{},
		&manager.Msg{},
		&manager.MsgType{},
		&manager.PoolConfig{},
		&manager.PlayerDataHour{},
		&manager.PlayerDataSummary{},
		&manager.PlayerProRank{},
		&manager.ProfitLoos{},
		&manager.Statistics{},
		&manager.SystemUserMsg{},
		&manager.User{},
		&manager.Web{},
		&manager.UserScoreLog{},
	)
	//player
	p.AutoMigrate(
		&player.Player{},
	)
}
