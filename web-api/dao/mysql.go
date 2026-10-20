package dao

import (
	"app/config"
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type DbDao struct {
	Player  *gorm.DB
	Manager *gorm.DB
}

var dbIns *DbDao = nil

func InitDB(globalConfig *config.RunConfig) error {
	m := globalConfig.Mysql
	playerSource := fmt.Sprintf("%s:%s@(%s:%d)/%s?charset=utf8", m["player"].User, m["player"].Password, m["player"].Host, m["player"].Port, m["player"].Database)
	player, err := gorm.Open(mysql.Open(playerSource), &gorm.Config{})
	if err != nil {
		return err
	}
	managerSource := fmt.Sprintf("%s:%s@(%s:%d)/%s?charset=utf8", m["manager"].User, m["manager"].Password, m["manager"].Host, m["manager"].Port, m["manager"].Database)
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
