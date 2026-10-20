package util

import (
	"fmt"

	"app/config"
	"app/tables/player"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB(sysConfig *config.RunConfig) (*gorm.DB, *gorm.DB, error) {
	m := sysConfig.Mysql["player"]
	playerSource := fmt.Sprintf("%s:%s@(%s:%d)/%s?charset=utf8", m.User, m.Password, m.Host, m.Port, m.Database)
	playerDB, err := gorm.Open(mysql.Open(playerSource), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}
	manager := sysConfig.Mysql["manager"]
	managerSource := fmt.Sprintf("%s:%s@(%s:%d)/%s?charset=utf8", manager.User, manager.Password, manager.Host, manager.Port, manager.Database)
	mdb, me := gorm.Open(mysql.Open(managerSource), &gorm.Config{})
	if me != nil {
		return nil, nil, me
	}
	if !playerDB.Migrator().HasColumn(&player.Player{}, "ex_data") {
		if err := playerDB.Migrator().AddColumn(&player.Player{}, "ExData"); err != nil {
			return nil, nil, err
		}
	}
	return playerDB, mdb, nil
}
