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
	"time"
)

func NewFinanceRedis(client redis.UniversalClient) *RedisDao { return &RedisDao{cli: client} }

type ScratchScope struct {
	Agent    uint32 `json:"agent"`
	GameID   uint32 `json:"gameId"`
	UserID   uint32 `json:"userId"`
	Currency string `json:"currency"`
}

func (s ScratchScope) Key() string {
	return fmt.Sprintf("%d:%d:%d:%s", s.Agent, s.GameID, s.UserID, s.Currency)
}
func (s ScratchScope) Market() string { return fmt.Sprintf("%d:%d", s.Agent, s.GameID) }

type ScratchOffer struct {
	Serials []string `json:"serials"`
	Book    bool     `json:"book"`
	Expires int64    `json:"expires"`
}

func (rd *RedisDao) ScratchOfferMap(s ScratchScope) (map[string]ScratchOffer, error) {
	raw, err := rd.cli.HGetAll(context.Background(), "gogame:scratch:offers:"+s.Key()).Result()
	if err != nil {
		return nil, err
	}
	result := map[string]ScratchOffer{}
	for ref, value := range raw {
		var offer ScratchOffer
		if json.Unmarshal([]byte(value), &offer) != nil {
			return nil, fmt.Errorf("invalid scratch offer")
		}
		if offer.Expires > time.Now().Unix() {
			result[ref] = offer
		}
	}
	return result, nil
}
func (rd *RedisDao) SaveScratchOffers(s ScratchScope, offers map[string]ScratchOffer) error {
	key := "gogame:scratch:offers:" + s.Key()
	pipe := rd.cli.TxPipeline()
	pipe.Del(context.Background(), key)
	for ref, offer := range offers {
		raw, _ := json.Marshal(offer)
		pipe.HSet(context.Background(), key, ref, raw)
	}
	pipe.Expire(context.Background(), key, 60*time.Second)
	_, err := pipe.Exec(context.Background())
	return err
}
func (rd *RedisDao) NextScratchBook(s ScratchScope, kind, tail string) (string, error) {
	seq, err := rd.cli.Incr(context.Background(), "gogame:scratch:sequence:"+s.Market()).Result()
	if err != nil {
		return "", err
	}
	width := 7 - len(tail)
	prefix := ""
	if width > 0 {
		power := int64(1)
		for i := 0; i < width; i++ {
			power *= 10
		}
		prefix = fmt.Sprintf("%0*d", width, seq%power)
	}
	return kind + "-" + time.Now().In(time.FixedZone("CST", 8*3600)).Format("060102") + "-" + prefix + tail, nil
}
func ScratchSerial(book string, index int) string {
	prefix := fmt.Sprintf("%s-%02d", book, index)
	sum := sha256.Sum256([]byte(prefix))
	return fmt.Sprintf("%s-%d", prefix, sum[0]%10)
}
func (rd *RedisDao) ScratchSerialRound(s ScratchScope, serial string) (string, error) {
	return rd.cli.Get(context.Background(), "gogame:scratch:sold:"+s.Market()+":"+serial).Result()
}
func (rd *RedisDao) ScratchPending(s ScratchScope) ([]string, error) {
	count, err := rd.cli.SCard(context.Background(), "gogame:scratch:pending:"+s.Key()).Result()
	if err != nil {
		return nil, err
	}
	if count > 100 {
		return nil, fmt.Errorf("invalid scratch pending count")
	}
	return rd.cli.SMembers(context.Background(), "gogame:scratch:pending:"+s.Key()).Result()
}
func (rd *RedisDao) DropScratchPending(s ScratchScope, roundID, serial string) error {
	return rd.cli.Eval(context.Background(), dropScratchPendingScript, []string{"gogame:scratch:pending:" + s.Key(), "gogame:scratch:sold:" + s.Market() + ":" + serial, "lottery_finance_rounds"}, roundID, "", "", s.Agent, s.GameID, s.UserID, s.Currency, serial, time.Now().Unix()).Err()
}
func (rd *RedisDao) ScratchReceipt(s ScratchScope, requestID string) (string, error) {
	return rd.cli.Get(context.Background(), scratchBuyMarker(s, requestID)).Result()
}
func scratchBuyMarker(s ScratchScope, id string) string {
	return fmt.Sprintf("lottery:settle_wallet:scratch-buy:%s:%x", s.Key(), sha256.Sum256([]byte(id)))
}

type ScratchIssuedRound struct {
	RoundID       string `json:"roundId"`
	Serial        string `json:"serial"`
	Raw           string `json:"raw"`
	ReservationID string `json:"reservationId"`
	Ledger        string `json:"ledger"`
	PayoutCNY     string `json:"payoutCny"`
}
type ScratchIssue struct {
	Fingerprint      string               `json:"fingerprint"`
	BasePoolAfterBet string               `json:"basePoolAfterBet"`
	PoolMember       string               `json:"poolMember"`
	Rounds           []ScratchIssuedRound `json:"rounds"`
	Result           json.RawMessage      `json:"result"`
}

const scratchIssuePrefix = `
local issue=cjson.decode(ARGV[2]);local old=redis.call('GET',KEYS[9])
if old then local receipt=cjson.decode(old);if receipt.fingerprint~=issue.fingerprint then return redis.error_reply('SCRATCH_REQUEST_CONFLICT') end;return old end
local function typ(key,expected)local t=redis.call('TYPE',key).ok;if t~='none' and t~=expected then error('SCRATCH_KEY_TYPE') end end
typ(KEYS[4],'hash');typ(KEYS[5],'zset');typ(KEYS[6],'hash');typ(KEYS[7],'set');typ(KEYS[8],'hash')
if redis.call('SCARD',KEYS[7])+#issue.rounds>100 then return redis.error_reply('SCRATCH_PENDING_LIMIT') end
local total=0;local seen={};local encoded={}
local cash=cjson.decode(ARGV[1]);local current=tonumber(redis.call('HGET',KEYS[2],'currency'))
if not current then return redis.error_reply('CASH_PLAYER_MISSING') end
local balance=current-cash.steps[1].amount
local cents=string.format('%.0f',balance);local whole=#cents<=2 and '0' or cents:sub(1,-3);local fraction=#cents==1 and ('0'..cents) or cents:sub(-2)
for i,r in ipairs(issue.rounds)do
 if seen[r.serial] or redis.call('EXISTS',KEYS[9+i])~=0 or redis.call('HEXISTS',KEYS[4],r.roundId)==1 or redis.call('HEXISTS',KEYS[6],r.reservationId)==1 then return redis.error_reply('SCRATCH_ALREADY_SOLD') end
 seen[r.serial]=true
  local payout=tonumber(r.payoutCny);if not payout or payout<0 or payout>9007199254740991-total then return redis.error_reply('SCRATCH_PRIZE_INVALID') end;total=total+payout
 local round=cjson.decode(r.raw);for _,bet in pairs(round.bets)do bet.currency=whole..'.'..fraction end
 round.scratch.purchaseBalance=balance;encoded[i]=cjson.encode(round)
end
local reserved=tonumber(redis.call('ZSCORE',KEYS[5],issue.poolMember) or '0')
local budget=tonumber(issue.basePoolAfterBet)
if not budget or budget>9007199254740991 or reserved<0 or reserved>9007199254740991-total then return redis.error_reply('SCRATCH_POOL_INVALID') end
if tonumber(issue.basePoolAfterBet)-reserved<total then return redis.error_reply('SCRATCH_POOL_INSUFFICIENT') end
local result=issue.result;result.walletAmount=balance
local receipt=cjson.encode({fingerprint=issue.fingerprint,result=result,rounds=encoded})
local function applyCash()
`
const scratchIssueSuffix = `
end
local cashResult=applyCash();if type(cashResult)~='string' then return cashResult end
for i,r in ipairs(issue.rounds)do redis.call('HSET',KEYS[4],r.roundId,encoded[i]);redis.call('HSET',KEYS[6],r.reservationId,r.ledger);redis.call('SADD',KEYS[7],r.roundId);redis.call('SET',KEYS[9+i],r.roundId) end
if total~=0 then redis.call('ZINCRBY',KEYS[5],total,issue.poolMember)end
redis.call('SET',KEYS[9],receipt,'EX',604800)
return receipt
`

func (rd *RedisDao) IssueScratch(s ScratchScope, requestID string, issue ScratchIssue, cash SeatNetBatch) (json.RawMessage, error) {
	if len(issue.Rounds) < 1 || len(issue.Rounds) > 100 || len(cash.Accounts) != 1 || len(cash.Steps) != 1 || cash.Accounts[0].UserID != s.UserID || cash.Accounts[0].AgentID != s.Agent || cash.Accounts[0].Currency != s.Currency || cash.Steps[0].Kind != "debit" {
		return nil, fmt.Errorf("invalid scratch cash batch")
	}
	keys := []string{"dirty_list_imp", fmt.Sprintf("player_%d", s.UserID), fmt.Sprintf("gogame:cash-carry:%d:%d:%s", s.Agent, s.UserID, s.Currency), "lottery_finance_rounds", "agent_reserved_data", "lottery:reservation_ledger", "gogame:scratch:pending:" + s.Key(), "gogame:scratch:offers:" + s.Key(), scratchBuyMarker(s, requestID)}
	for _, round := range issue.Rounds {
		keys = append(keys, "gogame:scratch:sold:"+s.Market()+":"+round.Serial)
	}
	a, _ := json.Marshal(cash)
	b, _ := json.Marshal(issue)
	result, err := rd.cli.Eval(context.Background(), scratchIssuePrefix+cashwallet.Apply+scratchIssueSuffix, keys, string(a), string(b)).Text()
	return json.RawMessage(result), err
}
func (gcm *GameCacheMgr) FinanceBasePool(agentID int64, symbol string, base decimal.Decimal) decimal.Decimal {
	a := gcm.GetAgent(agentID)
	a.lock.Lock()
	defer a.lock.Unlock()
	return poolValue(a.GetGame(symbol)).Add(base)
}

const scratchFinishScript = scratchRetirementScript + `
local function typ(k,t)local a=redis.call('TYPE',k).ok;if a~='none' and a~=t then error('SCRATCH_FINALIZE_TYPE')end end
typ(KEYS[1],'set');typ(KEYS[2],'string');typ(KEYS[3],'hash');typ(KEYS[4],'hash')
local round,ticket,remaining=scratchTerminal('SETTLED')
if ticket.claimed~=true or string.upper(ticket.kind)~=ARGV[2] or tonumber(ticket.winAmount)~=tonumber(ARGV[3]) then return redis.error_reply('SCRATCH_CLAIM_MISMATCH') end
if ticket.historyRecorded then redis.call('SREM',KEYS[1],ARGV[1]);retireScratchSold(remaining);return 1 end
local current=tonumber(redis.call('HGET',KEYS[4],ARGV[2]) or '0');local prize=tonumber(ARGV[3])
if not current or current<0 or not prize or prize<0 or current>9007199254740991-prize then return redis.error_reply('SCRATCH_TOTAL_INVALID')end
round.scratch.historyRecorded=true;round.scratch.claimed=true;round.scratch.mirrored=true;local updated=cjson.encode(round)
redis.call('SREM',KEYS[1],ARGV[1]);retireScratchSold(remaining)
redis.call('HSET',KEYS[3],ARGV[1],updated);redis.call('HSET',KEYS[4],ARGV[2],string.format('%.0f',current+prize))
return 1
`

func (rd *RedisDao) FinishScratch(s ScratchScope, roundID, serial, kind string, prize int64) error {
	return rd.cli.Eval(context.Background(), scratchFinishScript, []string{"gogame:scratch:pending:" + s.Key(), "gogame:scratch:sold:" + s.Market() + ":" + serial, "lottery_finance_rounds", fmt.Sprintf("gogame:scratch:accumulated:%d:%d:%s", s.Agent, s.UserID, s.Currency)}, roundID, kind, prize, s.Agent, s.GameID, s.UserID, s.Currency, serial, time.Now().Unix()).Err()
}
func (rd *RedisDao) ScratchTotals(s ScratchScope) (map[string]int64, error) {
	raw, err := rd.cli.HGetAll(context.Background(), fmt.Sprintf("gogame:scratch:accumulated:%d:%d:%s", s.Agent, s.UserID, s.Currency)).Result()
	if err != nil {
		return nil, err
	}
	totals := map[string]int64{"GGL1": 0, "GGL2": 0}
	for _, kind := range []string{"GGL1", "GGL2"} {
		if raw[kind] != "" {
			n, e := strconv.ParseInt(raw[kind], 10, 64)
			if e != nil || n < 0 {
				return nil, fmt.Errorf("invalid scratch totals")
			}
			totals[kind] = n
		}
	}
	return totals, nil
}
func (rd *RedisDao) ScratchRevision(userID uint32) (int64, error) {
	raw, err := rd.cli.HGet(context.Background(), fmt.Sprintf("player_%d", userID), "cash_revision").Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(raw, 10, 64)
}
