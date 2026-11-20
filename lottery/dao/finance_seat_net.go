package dao

import (
	"app/cashwallet"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"strconv"
	"strings"
	"time"
)

type SeatNetAccount struct {
	UserID   uint32 `json:"userId"`
	AgentID  uint32 `json:"agentId"`
	Currency string `json:"currency"`
	Revision int64  `json:"revision"`
}
type SeatNetStep struct {
	ID       string `json:"id"`
	Account  int    `json:"account"`
	Kind     string `json:"kind"`
	Amount   int64  `json:"amount"`
	Token    string `json:"token"`
	TableKey string `json:"tableKey"`
	Revision int64  `json:"revision"`
}
type SeatNetBatch struct {
	ID       string           `json:"id"`
	Accounts []SeatNetAccount `json:"accounts"`
	Steps    []SeatNetStep    `json:"steps"`
}

// Reuse the same cash/carry preflight and mutation rules. The existing terminal
// round and existing settlement-marker namespace join this one atomic write.
const seatNetScriptPrefix = `
local cash=cjson.decode(ARGV[1]);local roundId=ARGV[2];local fingerprint=ARGV[3]
local rounds=KEYS[#KEYS-1];local marker=KEYS[#KEYS]
local kind=redis.call('TYPE',rounds).ok
if kind~='none' and kind~='hash' then return redis.error_reply('FINANCE_ROUND_TYPE') end
local prior=redis.call('GET',marker)
if prior then local receipt=cjson.decode(prior);if receipt.fingerprint~=fingerprint then return redis.error_reply('FINANCE_SETTLEMENT_CONFLICT') end;return receipt.result end
if redis.call('HGET',rounds,roundId)~=ARGV[4] then return redis.error_reply('FINANCE_ROUND_CONFLICT') end
local final=cjson.decode(ARGV[5]);local results=final.settlement.results
for i,a in ipairs(cash.accounts) do
 local current=tonumber(redis.call('HGET',KEYS[2*i],'currency'))
 local delta=cash.steps[i].amount
 if not current or type(delta)~='number' then return redis.error_reply('CASH_BALANCE_INVALID') end
 local balance=current+delta
 local cents=string.format('%.0f',balance)
 local whole=#cents<=2 and '0' or cents:sub(1,-3)
 local fraction=#cents==1 and ('0'..cents) or cents:sub(-2)
 results[i].currency=whole..'.'..fraction
end
local terminal=cjson.encode(final)
local function applyCash()
`
const seatNetScriptSuffix = `
end
local result=applyCash()
if type(result)~='string' then return result end
redis.call('HSET',rounds,roundId,terminal)
redis.call('SET',marker,cjson.encode({fingerprint=fingerprint,result=result}),'EX',3600)
return result
`

// Banker batch joins ordinary bettor credits, banker carry deltas, terminal
// round and the existing reservation ledger release in one Lua transaction.
const bankerBatchPrefix = `
local extra=cjson.decode(ARGV[6]);local pool=KEYS[#KEYS-1];local ledger=KEYS[#KEYS]
local rounds=KEYS[#KEYS-3];local marker=KEYS[#KEYS-2]
local old=redis.call('GET',marker)
if old then local r=cjson.decode(old);if r.fingerprint~=ARGV[3] then return redis.error_reply('FINANCE_SETTLEMENT_CONFLICT')end;return r.result end
local function typ(k,t)local a=redis.call('TYPE',k).ok;if a~='none' and a~=t then error('BANKER_KEY_TYPE')end end
typ(rounds,'hash');typ(pool,'zset');typ(ledger,'hash')
if redis.call('HGET',rounds,ARGV[2])~=ARGV[4] then return redis.error_reply('FINANCE_ROUND_CONFLICT')end
local original=redis.call('HGET',ledger,extra.reservationId)
if original~=extra.originalLedger then return redis.error_reply('BANKER_RESERVATION_CONFLICT')end
local q=tonumber(redis.call('ZSCORE',pool,extra.poolMember)or'0');local release=tonumber(extra.amount)
if not q or not release or release<0 or q<release then return redis.error_reply('BANKER_POOL_INVALID')end
local cash=cjson.decode(ARGV[1]);local final=cjson.decode(ARGV[5]);local results=final.settlement.results
for i,a in ipairs(cash.accounts)do
 local balance=tonumber(redis.call('HGET',KEYS[2*i],'currency'));local op=cash.steps[i]
 if not balance then return redis.error_reply('CASH_BALANCE_INVALID')end
 balance=balance+op.amount
 local cents=string.format('%.0f',balance);local whole=#cents<=2 and'0'or cents:sub(1,-3);local fraction=#cents==1 and('0'..cents)or cents:sub(-2)
 results[i].currency=whole..'.'..fraction
end
local terminal=cjson.encode(final)
local function applyCash()
`
const bankerBatchSuffix = `
end
local result=applyCash();if type(result)~='string' then return result end
redis.call('HSET',rounds,ARGV[2],terminal)
if release~=0 then redis.call('ZINCRBY',pool,-release,extra.poolMember)end
redis.call('HSET',ledger,extra.reservationId,extra.releasedLedger)
redis.call('SET',marker,cjson.encode({fingerprint=ARGV[3],result=result}),'EX',3600)
return result
`

func (rd *RedisDao) SettleBankerBatch(roundID, fingerprint, before, terminal string, batch SeatNetBatch, reservationID, poolMember, amount string) (json.RawMessage, error) {
	if rd == nil || rd.cli == nil || len(batch.Accounts) == 0 || len(batch.Accounts) > 1024 || len(batch.Accounts) != len(batch.Steps) {
		return nil, fmt.Errorf("invalid banker financial batch")
	}
	marker := fmt.Sprintf("lottery:settle_wallet:banker:%x", sha256.Sum256([]byte(batch.ID)))
	if raw, e := rd.cli.Get(context.Background(), marker).Result(); e == nil {
		var saved struct {
			Fingerprint string `json:"fingerprint"`
			Result      string `json:"result"`
		}
		if json.Unmarshal([]byte(raw), &saved) != nil || saved.Fingerprint != fingerprint {
			return nil, fmt.Errorf("banker settlement conflict")
		}
		return json.RawMessage(saved.Result), nil
	} else if e != redis.Nil {
		return nil, e
	}
	keys := []string{"dirty_list_imp"}
	seen := map[uint32]bool{}
	for i, a := range batch.Accounts {
		op := batch.Steps[i]
		if a.UserID == 0 || seen[a.UserID] || op.Account != i+1 || (op.Kind != "credit" && op.Kind != "carryDelta") {
			return nil, fmt.Errorf("invalid banker account")
		}
		seen[a.UserID] = true
		keys = append(keys, fmt.Sprintf("player_%d", a.UserID), fmt.Sprintf("gogame:cash-carry:%d:%d:%s", a.AgentID, a.UserID, a.Currency))
	}
	ledger, err := rd.cli.HGet(context.Background(), "lottery:reservation_ledger", reservationID).Result()
	if err != nil {
		return nil, err
	}
	fields := strings.Split(ledger, "|")
	if len(fields) != 8 || fields[0] != ReservationLedgerActive || fields[3] != roundID || fields[4]+"_"+fields[5] != poolMember {
		return nil, fmt.Errorf("banker reservation not active or mismatched")
	}
	originalAmount, e := decimal.NewFromString(fields[1])
	requested, e2 := decimal.NewFromString(amount)
	if e != nil || e2 != nil || !originalAmount.Equal(requested) {
		return nil, fmt.Errorf("banker reservation amount mismatch")
	}
	fields[0] = ReservationLedgerReleased
	fields[2] = "settle:" + batch.ID
	fields[6] = "banker_net"
	fields[7] = strconv.FormatInt(time.Now().Unix(), 10)
	extra, _ := json.Marshal(map[string]any{"reservationId": reservationID, "poolMember": poolMember, "amount": amount, "originalLedger": ledger, "releasedLedger": strings.Join(fields, "|")})
	keys = append(keys, "lottery_finance_rounds", fmt.Sprintf("lottery:settle_wallet:banker:%x", sha256.Sum256([]byte(batch.ID))), "agent_reserved_data", "lottery:reservation_ledger")
	raw, _ := json.Marshal(batch)
	result, err := rd.cli.Eval(context.Background(), bankerBatchPrefix+cashwallet.Apply+bankerBatchSuffix, keys, string(raw), roundID, fingerprint, before, terminal, string(extra)).Text()
	return json.RawMessage(result), err
}

func (rd *RedisDao) SettleSeatNet(roundID, fingerprint, before, terminal string, batch SeatNetBatch) (json.RawMessage, error) {
	if rd == nil || rd.cli == nil || len(batch.Accounts) == 0 || len(batch.Accounts) > 64 || len(batch.Steps) != len(batch.Accounts) {
		return nil, fmt.Errorf("invalid seat financial batch")
	}
	keys := []string{"dirty_list_imp"}
	seen := map[uint32]bool{}
	for i, a := range batch.Accounts {
		step := batch.Steps[i]
		if a.UserID == 0 || seen[a.UserID] || step.Account != i+1 || step.Kind != "carryDelta" || step.Revision < 0 {
			return nil, fmt.Errorf("invalid seat account or operation")
		}
		seen[a.UserID] = true
		keys = append(keys, fmt.Sprintf("player_%d", a.UserID), fmt.Sprintf("gogame:cash-carry:%d:%d:%s", a.AgentID, a.UserID, a.Currency))
	}
	keys = append(keys, "lottery_finance_rounds", fmt.Sprintf("lottery:settle_wallet:seat-net:%x", sha256.Sum256([]byte(batch.ID))))
	raw, err := json.Marshal(batch)
	if err != nil {
		return nil, err
	}
	saved, err := rd.cli.Eval(context.Background(), seatNetScriptPrefix+cashwallet.Apply+seatNetScriptSuffix, keys, string(raw), roundID, fingerprint, before, terminal).Text()
	return json.RawMessage(saved), err
}

// Called under the finance service's per-agent write ownership and round lock.
// Unlike the display reader, Redis errors cannot turn reserved funds into zero.
func (gcm *GameCacheMgr) CanFundSeatNet(agentID int64, symbol string, base, net decimal.Decimal) (bool, error) {
	if !net.IsPositive() {
		return true, nil
	}
	agent := gcm.GetAgent(agentID)
	agent.lock.Lock()
	defer agent.lock.Unlock()
	reserved, err := RedisIns().cli.ZScore(context.Background(), "agent_reserved_data", fmt.Sprintf("%d_%s", agentID, symbol)).Result()
	if err != nil && err != redis.Nil {
		return false, err
	}
	available := poolValue(agent.GetGame(symbol)).Add(base).Sub(decimal.NewFromFloat(reserved))
	return available.GreaterThanOrEqual(net), nil
}
