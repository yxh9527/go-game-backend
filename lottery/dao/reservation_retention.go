package dao

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"strconv"
	"strings"
	"time"
)

// Missing legacy ledgers may be reconstructed only from the original current
// reservation, never solely from a delayed caller's amount and old round ID.
const reservationMigrationProof = `
local function originalReservation(roundId,reservationId,amount,poolMember)
 local raw=redis.call('HGET',KEYS[3],roundId)
 if not raw then return false end
 local ok,r=pcall(cjson.decode,raw)
 if not ok or type(r)~='table' or type(r.reservation)~='table' then return false end
 local v=r.reservation
 if r.state~='RESERVED' or v.status~='RESERVED' or v.reservationId~=reservationId then return false end
 if tostring(r.agent)..'_'..tostring(r.poolSymbol)~=poolMember then return false end
 local stored=tonumber(v.totalPayoutCny)
 return stored and stored==tonumber(amount)
end
`

var seedReservationFromRoundScript = redis.NewScript(reservationMigrationProof + `
local existing=redis.call('HGET',KEYS[2],ARGV[1])
if existing then return 1 end
if not originalReservation(ARGV[2],ARGV[1],ARGV[3],ARGV[4]) then return 0 end
redis.call('HSET',KEYS[2],ARGV[1],ARGV[5])
return 1
`)

func (rd *RedisDao) SeedReservationLedgerFromRound(agentID int64, symbol string, amount decimal.Decimal, reservationID, roundID string) error {
	if reservationID == "" || roundID == "" || amount.IsNegative() {
		return fmt.Errorf("invalid reservation migration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	value := EncodeReservationLedger(ReservationLedgerActive, amount.String(), "", roundID, agentID, symbol, "", time.Now().Unix())
	n, err := seedReservationFromRoundScript.Run(ctx, rd.cli, []string{"agent_reserved_data", reservationLedgerRedisKey, "lottery_finance_rounds"}, reservationID, roundID, amount.String(), fmt.Sprintf("%d_%s", agentID, symbol), value).Int()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("original active reservation required for ledger migration")
	}
	return nil
}

var removeReleasedReservationsScript = redis.NewScript(`
local kind=redis.call('TYPE',KEYS[1]).ok
if kind~='none' and kind~='hash' then return redis.error_reply('INVALID_RESERVATION_LEDGER_TYPE') end
local rounds=redis.call('TYPE',KEYS[2]).ok
if rounds~='none' and rounds~='hash' then return redis.error_reply('INVALID_FINANCE_ROUND_TYPE') end
local count=0
for i=1,#ARGV,3 do
 local id=ARGV[i];local original=ARGV[i+1];local roundId=ARGV[i+2]
 if redis.call('HGET',KEYS[1],id)==original and redis.call('HEXISTS',KEYS[2],roundId)==0 then
  redis.call('HDEL',KEYS[1],id);count=count+1
 end
end
return count
`)

// HSCAN's COUNT is a hint; deletion remains limited to 100 exact original
// released fields. A later scan cycle revisits any excess candidates.
func (rd *RedisDao) RemoveReleasedReservationLedgers(cursor uint64, now int64) (uint64, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	entries, next, err := rd.cli.HScan(ctx, reservationLedgerRedisKey, cursor, "*", 100).Result()
	if err != nil {
		return cursor, 0, err
	}
	args := make([]interface{}, 0, 300)
	for i := 0; i+1 < len(entries) && len(args) < 300; i += 2 {
		fields := strings.Split(entries[i+1], "|")
		if len(fields) != 8 || fields[0] != ReservationLedgerReleased || fields[2] == "" || fields[3] == "" || fields[5] == "" {
			continue
		}
		at, e := strconv.ParseInt(fields[7], 10, 64)
		if e != nil || at <= 0 || now < at || now-at < 3600 {
			continue
		}
		amount, e := decimal.NewFromString(fields[1])
		if e != nil || amount.IsNegative() {
			continue
		}
		agent, e := strconv.ParseUint(fields[4], 10, 32)
		if e != nil || agent > uint64(^uint32(0)) {
			continue
		}
		args = append(args, entries[i], entries[i+1], fields[3])
	}
	if len(args) == 0 {
		return next, 0, nil
	}
	n, err := removeReleasedReservationsScript.Run(ctx, rd.cli, []string{reservationLedgerRedisKey, "lottery_finance_rounds"}, args...).Int()
	if err != nil {
		return cursor, 0, err
	}
	return next, n, nil
}
