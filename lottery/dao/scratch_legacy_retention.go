package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"strconv"
	"strings"
	"time"
)

// Minimal proof from the existing immutable market owner and claimed ticket.
// Missing/inconsistent evidence is retained, never treated as an unpaid serial.
type ScratchRetiredOwner struct {
	UserID   uint32
	Currency string
}

func (dd *DBDao) ScratchClaimedOwner(ctx context.Context, scope ScratchScope, serial, roundID string, now int64) (*ScratchRetiredOwner, error) {
	if dd == nil || dd.player == nil {
		return nil, fmt.Errorf("scratch ownership database unavailable")
	}
	key, err := scratchSoldKey(scope, serial)
	if err != nil {
		return nil, err
	}
	var owners []GameStorageRecord
	if err = dd.player.WithContext(ctx).Select("payload,expire_at").Where("user_id=? AND game_id=? AND currency_symbol=? AND storage_key=?", 0, scope.GameID, "*", key).Limit(1).Find(&owners).Error; err != nil {
		return nil, err
	}
	if len(owners) != 1 || owners[0].ExpireAt != 0 || len(owners[0].Payload) > 128*1024 {
		return nil, nil
	}
	var owner scratchSoldOwner
	if json.Unmarshal([]byte(owners[0].Payload), &owner) != nil || owner.Version != 1 || owner.Agent != scope.Agent || owner.GameID != scope.GameID || owner.Serial != serial || owner.RoundID != roundID || owner.UserID == 0 || len(owner.Currency) == 0 || len(owner.Currency) > 12 {
		return nil, nil
	}
	var rows []GameStorageRecord
	if err = dd.player.WithContext(ctx).Select("payload,expire_at").Where("user_id=? AND game_id=? AND currency_symbol=? AND storage_key=?", owner.UserID, scope.GameID, owner.Currency, "scratch:"+serial).Limit(1).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) != 1 || rows[0].ExpireAt != 0 || len(rows[0].Payload) > 128*1024 {
		return nil, nil
	}
	var ticket struct {
		UserID    uint32 `json:"userId"`
		Currency  string `json:"currency"`
		Serial    string `json:"serialFull"`
		RoundID   string `json:"roundId"`
		Claimed   bool   `json:"claimed"`
		ClaimedAt int64  `json:"claimedAt"`
	}
	if json.Unmarshal([]byte(rows[0].Payload), &ticket) != nil || !ticket.Claimed || ticket.ClaimedAt <= 0 || ticket.ClaimedAt > now*1000 || now-ticket.ClaimedAt/1000 < 3600 || ticket.UserID != owner.UserID || ticket.Currency != owner.Currency || ticket.Serial != serial || ticket.RoundID != roundID {
		return nil, nil
	}
	return &ScratchRetiredOwner{owner.UserID, owner.Currency}, nil
}

var retireLegacyScratchSoldScript = redis.NewScript(`
if redis.call('GET',KEYS[1])~=ARGV[1] then return 0 end
if redis.call('HEXISTS','lottery_finance_rounds',ARGV[1])~=0 then return 0 end
redis.call('DEL',KEYS[1]);redis.call('SREM',KEYS[2],ARGV[1]);return 1
`)

// Bounded maintenance on the existing five-second loop. It never changes SQL,
// rebuilds a round, issues a ticket or invokes a financial operation.
func (rd *RedisDao) RemoveProvenLegacyScratchSold(cursor uint64, proof func(context.Context, ScratchScope, string, string, int64) (*ScratchRetiredOwner, error)) (uint64, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	keys, next, err := rd.cli.Scan(ctx, cursor, "gogame:scratch:sold:*", 100).Result()
	if err != nil {
		return cursor, 0, err
	}
	if len(keys) > 100 {
		keys = keys[:100]
	}
	changed := 0
	now := time.Now().Unix()
	for _, key := range keys {
		if err = ctx.Err(); err != nil {
			return next, changed, err
		}
		parts := strings.SplitN(strings.TrimPrefix(key, "gogame:scratch:sold:"), ":", 3)
		if len(parts) != 3 || len(parts[2]) == 0 || len(parts[2]) > 80 {
			continue
		}
		agent, e1 := strconv.ParseUint(parts[0], 10, 32)
		game, e2 := strconv.ParseUint(parts[1], 10, 32)
		if e1 != nil || e2 != nil || game == 0 {
			continue
		}
		scope := ScratchScope{Agent: uint32(agent), GameID: uint32(game)}
		if scope.Market()+":"+parts[2] != strings.TrimPrefix(key, "gogame:scratch:sold:") {
			continue
		}
		length, err := rd.cli.StrLen(ctx, key).Result()
		if err != nil {
			return next, changed, err
		}
		if length < 1 || length > 160 {
			continue
		}
		id, err := rd.cli.Get(ctx, key).Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return next, changed, err
		}
		present, err := rd.cli.HExists(ctx, "lottery_finance_rounds", id).Result()
		if err != nil {
			return next, changed, err
		}
		if present {
			continue
		}
		owner, err := proof(ctx, scope, parts[2], id, now)
		if err != nil {
			return next, changed, err
		}
		if owner == nil {
			continue
		}
		if owner.UserID == 0 || len(owner.Currency) == 0 || len(owner.Currency) > 12 {
			continue
		}
		valid := true
		for _, c := range owner.Currency {
			if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
				valid = false
			}
		}
		if !valid {
			continue
		}
		scope.UserID = owner.UserID
		scope.Currency = owner.Currency
		n, err := retireLegacyScratchSoldScript.Run(ctx, rd.cli, []string{key, "gogame:scratch:pending:" + scope.Key()}, id).Int()
		if err != nil {
			return next, changed, err
		}
		changed += n
	}
	return next, changed, nil
}
