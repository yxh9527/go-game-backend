package dao

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

// Old archives were created only after seven/thirty days, so every legacy
// archive entry already exceeds the new one-hour lifetime. UNLINK avoids a
// blocking synchronous free of a potentially large old hash.
func (rd *RedisDao) RemoveLegacyFinanceArchive() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return rd.cli.Unlink(ctx, "lottery_finance_rounds_archive").Err()
}

var deleteExpiredFinanceRoundScript = redis.NewScript(`
local q=redis.call('TYPE',KEYS[2]).ok
if q~='none' and q~='set' then return redis.error_reply('INVALID_FINANCE_RECONCILE_TYPE') end
local raw=redis.call('HGET',KEYS[1],ARGV[1])
if not raw then redis.call('SREM',KEYS[2],ARGV[1]);return 1 end
local ok,r=pcall(cjson.decode,raw)
if not ok or type(r)~='table' then return redis.error_reply('INVALID_FINANCE_ROUND') end
if r.state~='SETTLED' and r.state~='VOIDED' and r.state~='EXPIRED' then return 0 end
local ref=tonumber(r.updatedAt) or 0
if (r.state=='SETTLED' or r.state=='VOIDED') and type(r.settlement)=='table' and tonumber(r.settlement.completedAt) and tonumber(r.settlement.completedAt)>0 then
 ref=tonumber(r.settlement.completedAt)
elseif r.state=='EXPIRED' and type(r.reservation)=='table' and tonumber(r.reservation.expiresAt) and tonumber(r.reservation.expiresAt)>0 then
 ref=tonumber(r.reservation.expiresAt)
end
local now=tonumber(ARGV[2]);local lifetime=tonumber(ARGV[3])
if ref<=0 or now-ref<lifetime then return 0 end
redis.call('HDEL',KEYS[1],ARGV[1])
redis.call('SREM',KEYS[2],ARGV[1])
return 1
`)

func (rd *RedisDao) DeleteExpiredFinanceRound(parent context.Context, roundID string, now, lifetime int64) (bool, error) {
	if roundID == "" || now <= 0 || lifetime != 3600 {
		return false, fmt.Errorf("invalid finance snapshot expiry")
	}
	ctx, cancel := context.WithTimeout(parent, time.Second)
	defer cancel()
	n, err := deleteExpiredFinanceRoundScript.Run(ctx, rd.cli,
		[]string{"lottery_finance_rounds", financeReconcileSetKey}, roundID, now, lifetime).Int()
	return n == 1, err
}
