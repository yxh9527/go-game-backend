package dao

import (
	"context"
	"runtime"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DataSyncer struct {
	rds    redis.UniversalClient
	db     *gorm.DB
	rdsDao *RedisDao
	closeC chan struct{}
}

func NewDataSyncer(db *gorm.DB) *DataSyncer {
	return &DataSyncer{
		rds:    RedisIns().Client,
		db:     db,
		rdsDao: RedisIns(),
		closeC: make(chan struct{}),
	}
}

func (syncer *DataSyncer) Start() {
	for i := 0; i < runtime.NumCPU(); i++ {
		go syncer.syncRedis2Mysql()
		go syncer.syncRedis2MysqlImp()
	}
}

func (syncer *DataSyncer) Stop() {
	close(syncer.closeC)
}

func (syncer *DataSyncer) syncRedis2Mysql() {
	defer func() {
		if e := recover(); e != nil {
			zap.L().Error("数据落地协程panic", zap.Any("recover", e))
		}
	}()

	t := time.NewTimer(time.Second * 300)
	for {
		select {
		case <-syncer.closeC:
			zap.L().Info("DataSyncer收到退出通知，已退出协程")
			return
		default:
			playerId, err := syncer.rds.SPop(context.Background(), "dirty_list").Uint64()
			if err != nil {
				if err != redis.Nil {
					zap.L().Error("数据落地：redis spop dirty_list", zap.Error(err))
				}

				if !t.Reset(time.Second * 2) {
					t = time.NewTimer(time.Second * 2)
				}
				<-t.C

				continue
			}

			if playerId == 0 {
				zap.L().Error("数据落地：redis spop dirty_list 有 playerId==0", zap.Uint64("player_id", playerId))
				continue
			}

			p, _, err := syncer.rdsDao.GetPlayerOrigin(uint32(playerId))
			if err == redis.Nil {
				zap.L().Warn("数据落地：redis hget not exist playerId", zap.Uint64("playerId", playerId), zap.Error(err))
				continue
			} else if err != nil {
				zap.L().Error("数据落地：redis hget", zap.Error(err))
				continue
			}

			if p == nil {
				zap.L().Error("数据落地：p == nil", zap.Uint64("player_id", playerId))
				continue
			}

			res := syncer.db.Exec(`
UPDATE user_info
SET sex         = ?,
    pic         = ?,
    login_time  = ?,
    login_ip    = ?,
    money_limit = ?,
	all_times   = ?,
	ex_data     = ?
WHERE user_id = ?
`, p.Gender, p.Avatar, p.LoginTimeStamp, p.LoginIP, p.CurrencyLimit, p.AllTimes, p.ExData, p.Id)

			if res.Error != nil {
				zap.L().Error("数据落地：写入数据库出错", zap.Stringer("data", p), zap.Error(err))
				continue
			}

			zap.L().Info("数据落地，玩家数据入库成功", zap.Stringer("data", p), zap.Time("t", time.Now()))
		}
	}
}

func (syncer *DataSyncer) syncRedis2MysqlImp() {
	defer func() {
		if e := recover(); e != nil {
			zap.L().Error("数据落地协程panic", zap.Any("recover", e))
		}
	}()

	t := time.NewTimer(time.Second * 300)
	for {
		select {
		case <-syncer.closeC:
			zap.L().Info("DataSyncer收到退出通知，已退出协程")
			return
		default:
			playerId, err := syncer.rds.SPop(context.Background(), "dirty_list_imp").Uint64()
			if err != nil {
				if err != redis.Nil {
					zap.L().Error("数据落地：redis spop dirty_list", zap.Error(err))
				}

				if !t.Reset(time.Second * 2) {
					t = time.NewTimer(time.Second * 2)
				}
				<-t.C

				continue
			}

			if playerId == 0 {
				zap.L().Error("数据落地：redis spop dirty_list 有 playerId==0", zap.Uint64("player_id", playerId))
				continue
			}

			p, _, err := syncer.rdsDao.GetPlayerOrigin(uint32(playerId))
			if err == redis.Nil {
				zap.L().Warn("数据落地：redis hget not exist playerId", zap.Uint64("playerId", playerId), zap.Error(err))
				continue
			} else if err != nil {
				zap.L().Error("数据落地：redis hget", zap.Error(err))
				continue
			}

			if p == nil {
				zap.L().Error("数据落地：p == nil", zap.Uint64("player_id", playerId))
				continue
			}

			res := syncer.db.Exec(`
			UPDATE user_info
			SET score       = ?
			WHERE user_id = ?
			`, p.GameCurrency, p.Id)

			if res.Error != nil {
				zap.L().Error("数据落地：写入数据库出错", zap.Stringer("data", p), zap.Error(err))
				continue
			}

			zap.L().Info("数据落地imp，玩家数据入库成功", zap.Stringer("data", p), zap.Time("t", time.Now()))
		}
	}
}
