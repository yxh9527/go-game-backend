package dao

import (
	"context"
	"fmt"
	"time"
)

// The existing finance-round snapshot and debit are written in one Lua call.
// No separate journal, wallet marker or startup replay store is introduced.
const financeBetScript = `
local player,rounds,dirty=KEYS[1],KEYS[2],KEYS[3]
local roundId,betId,raw=ARGV[1],ARGV[2],ARGV[3]
local draft=cjson.decode(raw)
local bet=draft.bets[betId]
local roundType=redis.call('TYPE',rounds).ok
local dirtyType=redis.call('TYPE',dirty).ok
if roundType~='none' and roundType~='hash' then return redis.error_reply('FINANCE_ROUND_TYPE') end
if dirtyType~='none' and dirtyType~='set' then return redis.error_reply('FINANCE_DIRTY_TYPE') end
local prior=redis.call('HGET',rounds,roundId)
if prior then
 local old=cjson.decode(prior)
 if old.agent~=draft.agent or old.gameId~=draft.gameId or old.level~=draft.level then return redis.error_reply('FINANCE_ROUTE_CONFLICT') end
 if old.bets and old.bets[betId] then
  local b=old.bets[betId]
  if b.userId~=bet.userId or b.currencyType~=bet.currencyType or b.amount~=bet.amount or (b.areaId or '')~=(bet.areaId or '') then return redis.error_reply('FINANCE_BET_CONFLICT') end
  return {1,prior}
 end
 if old.betRevision~=draft.betRevision then return redis.error_reply('FINANCE_REVISION_CONFLICT') end
 if old.state~='BETTING' and not (old.state=='RESERVED' and old.reservation and old.reservation.mode=='LIABILITY_CAP' and old.reservation.expiresAt>tonumber(ARGV[8])) then return redis.error_reply('FINANCE_ROUND_CLOSED') end
 for id,b in pairs(old.bets or {}) do
  if not draft.bets[id] then return redis.error_reply('FINANCE_SNAPSHOT_CONFLICT') end
  for field,value in pairs(b) do if draft.bets[id][field]~=value then return redis.error_reply('FINANCE_SNAPSHOT_CONFLICT') end end
 end
 for id,b in pairs(draft.bets) do if id~=betId and not (old.bets and old.bets[id]) then return redis.error_reply('FINANCE_SNAPSHOT_CONFLICT') end end
else
 for id,b in pairs(draft.bets) do if id~=betId then return redis.error_reply('FINANCE_ROUND_MISSING') end end
end
if redis.call('HGET',player,'id')~=ARGV[4] then return redis.error_reply('FINANCE_PLAYER_MISSING') end
if redis.call('HGET',player,'agent_id')~=ARGV[5] or redis.call('HGET',player,'currency_type')~=ARGV[6] then return redis.error_reply('FINANCE_ACCOUNT_MISMATCH') end
local balance=tonumber(redis.call('HGET',player,'currency'))
local revision=tonumber(redis.call('HGET',player,'cash_revision') or '0')
local amount=tonumber(ARGV[7]);local limit=9007199254740991
if not balance or balance<0 or balance%1~=0 or balance>limit or not amount or amount<0 or amount%1~=0 or amount>limit then return redis.error_reply('FINANCE_BALANCE_INVALID') end
if not revision or revision<0 or revision%1~=0 or revision>=limit then return redis.error_reply('FINANCE_REVISION_INVALID') end
local reserved=redis.call('GET',KEYS[4]);local carry=reserved and cjson.decode(reserved).amount or 0
if type(carry)~='number' or carry<0 or carry%1~=0 or carry>balance then return redis.error_reply('FINANCE_CARRY_INVALID') end
if amount>balance-carry then return redis.error_reply('FINANCE_INSUFFICIENT') end
local updated=balance-amount
bet.currency=string.format('%.0f',math.floor(updated/100))..'.'..string.format('%02d',updated%100)
local encoded=cjson.encode(draft)
if amount>0 then redis.call('HINCRBY',player,'currency','-'..ARGV[7]);redis.call('HINCRBY',player,'cash_revision',1) end
redis.call('HSET',rounds,roundId,encoded)
redis.call('EXPIRE',player,1200)
redis.pcall('SADD',dirty,ARGV[4])
return {0,encoded}
`

func (rd *RedisDao) ApplyFinanceBet(roundID, betID, raw string, userID, agentID uint32, currency string, amount int64) (string, bool, error) {
	if rd == nil || rd.cli == nil || amount < 0 || amount > 9007199254740991 {
		return "", false, fmt.Errorf("invalid finance bet storage")
	}
	keys := []string{fmt.Sprintf("player_%d", userID), "lottery_finance_rounds", "dirty_list_imp", fmt.Sprintf("gogame:cash-carry:%d:%d:%s", agentID, userID, currency)}
	values, err := rd.cli.Eval(context.Background(), financeBetScript, keys, roundID, betID, raw, userID, agentID, currency, amount, time.Now().Unix()).Slice()
	if err != nil {
		return "", false, err
	}
	if len(values) != 2 {
		return "", false, fmt.Errorf("invalid finance commit response")
	}
	flag, ok := values[0].(int64)
	rawResult, valid := values[1].(string)
	if !ok || !valid {
		return "", false, fmt.Errorf("invalid finance commit payload")
	}
	return rawResult, flag == 1, nil
}
