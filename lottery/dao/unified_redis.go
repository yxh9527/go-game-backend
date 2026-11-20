package dao

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"strconv"
	"time"
)

var (
	// ErrPlayerNotCached 表示玩家完整资料尚未加载到 Redis。
	ErrPlayerNotCached = errors.New("player is not cached")
	// ErrInsufficientFunds 表示本次扣款会使玩家余额小于零。
	ErrInsufficientFunds = errors.New("insufficient player funds")
	// updateCurrencyScript 将存在性检查、余额非负校验、增减款和脏数据标记合并为一次原子操作。
	updateCurrencyScript = redis.NewScript(`
local playerKey = KEYS[1]
if redis.call("HEXISTS", playerKey, "id") == 0 then
    return {0, 0}
end
local current = tonumber(redis.call("HGET", playerKey, "currency") or "0")
local updated = current + tonumber(ARGV[1])
if updated < 0 then
    return {1, current}
end
redis.call("HINCRBY", playerKey, "currency", ARGV[1])
redis.call("HINCRBY", playerKey, "exp", ARGV[2])
redis.call("EXPIRE", playerKey, ARGV[3])
redis.call("SADD", KEYS[2], ARGV[4])
return {2, updated}
`)
	// refundWithCancelMarkerScript：先入账再写 marker。
	// marker 仅表示钱包已实际入账；若 marker 已存在则跳过入账（幂等重试）。
	// 返回：{0,0} 未缓存；{1,bal} 余额不足；{2,bal} 本次新入账；{3,bal} 已入账过。
	refundWithCancelMarkerScript = redis.NewScript(`
local playerKey = KEYS[1]
local dirtyKey = KEYS[2]
local markerKey = KEYS[3]
local delta = tonumber(ARGV[1])
local exp = tonumber(ARGV[2])
local ttl = tonumber(ARGV[3])
local playerId = ARGV[4]
local markerValue = ARGV[5]
local markerTTL = tonumber(ARGV[6])
if redis.call("EXISTS", markerKey) == 1 then
    if redis.call("GET",markerKey) ~= markerValue then return redis.error_reply('CASH_MARKER_CONFLICT') end
    if redis.call("HEXISTS",playerKey,"id")==0 then return {0,0} end
    local current = tonumber(redis.call("HGET", playerKey, "currency") or "0")
    return {3, current}
end
if redis.call("HEXISTS", playerKey, "id") == 0 then
    return {0, 0}
end
local current = tonumber(redis.call("HGET", playerKey, "currency") or "0")
local revision = tonumber(redis.call('HGET',playerKey,'cash_revision') or '0')
local limit=9007199254740991
if not current or current<0 or current%1~=0 or current>limit or not delta or delta<0 or delta%1~=0 or delta>limit-current then return redis.error_reply('CASH_BALANCE_INVALID') end
if not revision or revision<0 or revision%1~=0 or revision>=limit then return redis.error_reply('CASH_REVISION') end
local dirtyType=redis.call('TYPE',dirtyKey).ok
if dirtyType~='none' and dirtyType~='set' then return redis.error_reply('CASH_DIRTY_TYPE') end
local updated = current + delta
local carryEncoded=nil
if ARGV[7] and ARGV[7]~='' then
 local binding=cjson.decode(ARGV[7])
 if redis.call('HGET',playerKey,'agent_id')~=tostring(binding.agent) or redis.call('HGET',playerKey,'currency_type')~=binding.currency then return redis.error_reply('CASH_ACCOUNT_MISMATCH') end
 local raw=redis.call('GET',KEYS[4]);local carry=raw and cjson.decode(raw) or nil
 if not carry or carry.token~=binding.token or carry.tableKey~=binding.tableKey or carry.revision~=binding.revision then return redis.error_reply('CASH_CARRY_CONFLICT') end
 if type(carry.amount)~='number' or carry.amount<0 or carry.amount%1~=0 or carry.amount>current or carry.amount+delta>updated or carry.revision>=limit then return redis.error_reply('CASH_CARRY_INVALID') end
 carry.amount=carry.amount+delta;carry.revision=carry.revision+1;carryEncoded=cjson.encode(carry)
end
if updated < 0 then
    return {1, current}
end
redis.call("HINCRBY", playerKey, "currency", ARGV[1])
redis.call('HINCRBY',playerKey,'cash_revision',1)
if carryEncoded then redis.call('SET',KEYS[4],carryEncoded,'PX',300000) end
if exp ~= 0 then
    redis.call("HINCRBY", playerKey, "exp", exp)
end
redis.call("EXPIRE", playerKey, ttl)
redis.call("SADD", dirtyKey, playerId)
redis.call("SET", markerKey, markerValue, "EX", markerTTL)
return {2, updated}
`)
	// tryReservePoolScript 原子检查可用奖池并增加预留，避免并发预赔透支。
	tryReservePoolScript = redis.NewScript(`
local current = tonumber(redis.call("ZSCORE", KEYS[1], ARGV[1]) or "0")
local basePool = tonumber(ARGV[2])
local requested = tonumber(ARGV[3])
if basePool - current < requested then
    return 0
end
redis.call("ZINCRBY", KEYS[1], requested, ARGV[1])
return 1
`)
	// releasePoolReservationScript 校验预留余额后再释放，防止重复释放影响其他牌局。
	releasePoolReservationScript = redis.NewScript(`
local current = tonumber(redis.call("ZSCORE", KEYS[1], ARGV[1]) or "0")
local released = tonumber(ARGV[2])
if current + 0.0000001 < released then
    return 0
end
local updated = current - released
if updated <= 0.0000001 then
    redis.call("ZREM", KEYS[1], ARGV[1])
else
    redis.call("ZADD", KEYS[1], updated, ARGV[1])
end
return 1
`)
	// reserveWithLedgerScript：汇总 Q + 按 reservationId 建 ledger。
	// KEYS[1]=agent_reserved_data KEYS[2]=lottery:reservation_ledger
	// ARGV: poolMember, basePool, amount, reservationId, ledgerValue(ACTIVE|...)
	// 返回：0 不足；1 新建成功；2 已是 ACTIVE（幂等，不加 Q）；3 已 RELEASED 拒绝复用。
	reserveWithLedgerScript = redis.NewScript(`
local reservationId = ARGV[4]
local existing = redis.call("HGET", KEYS[2], reservationId)
if existing then
  local status = string.match(existing, "^([^|]+)")
  if status == "ACTIVE" then
    return 2
  end
  return 3
end
local current = tonumber(redis.call("ZSCORE", KEYS[1], ARGV[1]) or "0")
local basePool = tonumber(ARGV[2])
local requested = tonumber(ARGV[3])
if basePool - current < requested then
  return 0
end
redis.call("ZINCRBY", KEYS[1], requested, ARGV[1])
redis.call("HSET", KEYS[2], reservationId, ARGV[5])
return 1
`)
	// releaseWithLedgerScript：校验 reservationId ledger 后释放 Q。
	// ARGV: poolMember, amount, reservationId, releaseId, releasedLedgerValue, allowMigrate(0/1), migrateLedgerBefore
	// 返回：0 失败；1 本次新释放；2 已释放（幂等，不改 Q）。
	releaseWithLedgerScript = redis.NewScript(reservationMigrationProof + `
local reservationId = ARGV[3]
local releaseId = ARGV[4]
local existing = redis.call("HGET", KEYS[2], reservationId)
if not existing then
  if tonumber(ARGV[6]) ~= 1 then
    return 0
  end
  if not originalReservation(ARGV[8],reservationId,ARGV[2],ARGV[1]) then return 0 end
  -- 升级兼容：无 ledger 的旧预留，允许一次性补建并释放。
  existing = ARGV[7]
  redis.call("HSET", KEYS[2], reservationId, existing)
end
local status = string.match(existing, "^([^|]+)")
local savedStatus,savedAmount,savedRelease,savedRound,savedAgent,savedPool=string.match(existing,'^([^|]*)|([^|]*)|([^|]*)|([^|]*)|([^|]*)|([^|]*)|')
if not savedStatus or savedRound~=ARGV[8] or savedAgent..'_'..savedPool~=ARGV[1] or tonumber(savedAmount)~=tonumber(ARGV[2]) then return 0 end
if status == "RELEASED" then
  return 2
end
if status ~= "ACTIVE" then
  return 0
end
local current = tonumber(redis.call("ZSCORE", KEYS[1], ARGV[1]) or "0")
local released = tonumber(ARGV[2])
if current + 0.0000001 < released then
  return 0
end
local updated = current - released
if updated <= 0.0000001 then
  redis.call("ZREM", KEYS[1], ARGV[1])
else
  redis.call("ZADD", KEYS[1], updated, ARGV[1])
end
redis.call("HSET", KEYS[2], reservationId, ARGV[5])
return 1
`)
	// adjustReservationLedgerScript：LIABILITY_CAP 差额调整，保持 ACTIVE。
	// ARGV: poolMember, basePool, reservationId, oldAmount, newAmount, activeLedgerValue
	// 返回：0 失败；1 成功；2 ledger 缺失且不允许（由 Go 侧决定是否 migrate 后重试）。
	adjustReservationLedgerScript = redis.NewScript(`
local reservationId = ARGV[3]
local existing = redis.call("HGET", KEYS[2], reservationId)
if not existing then
  return 2
end
local status = string.match(existing, "^([^|]+)")
if status ~= "ACTIVE" then
  return 0
end
local oldAmount = tonumber(ARGV[4])
local newAmount = tonumber(ARGV[5])
local delta = newAmount - oldAmount
local current = tonumber(redis.call("ZSCORE", KEYS[1], ARGV[1]) or "0")
if delta > 0 then
  local basePool = tonumber(ARGV[2])
  if basePool - current < delta then
    return 0
  end
  redis.call("ZINCRBY", KEYS[1], delta, ARGV[1])
elseif delta < 0 then
  local released = -delta
  if current + 0.0000001 < released then
    return 0
  end
  local updated = current - released
  if updated <= 0.0000001 then
    redis.call("ZREM", KEYS[1], ARGV[1])
  else
    redis.call("ZADD", KEYS[1], updated, ARGV[1])
  end
end
redis.call("HSET", KEYS[2], reservationId, ARGV[6])
return 1
`)
)

const (
	reservationLedgerRedisKey = "lottery:reservation_ledger"
	financeReconcileSetKey    = "lottery:finance_reconcile"
	settleWalletMarkerPrefix  = "lottery:settle_wallet:"
	SettleWalletMarkerTTLSec  = int32(7 * 24 * 3600)
)

// ReservationLedger 状态。
const (
	ReservationLedgerActive   = "ACTIVE"
	ReservationLedgerReleased = "RELEASED"
)

// EncodeReservationLedger 使用 Lua 友好的管道分隔格式。
// status|amount|releaseId|roundId|agent|poolSymbol|reason|updatedAt
type CancelWalletRefundResult struct {
	// AlreadyDone 为 true 表示 marker 已存在，本次未再次入账。
	AlreadyDone bool
	NewCurrency int64
}

type ReserveWithLedgerResult struct {
	// Ok 表示可用（新建或幂等命中 ACTIVE）。
	Ok bool
	// AlreadyActive 为 true 时未再次增加 Q。
	AlreadyActive bool
	// RejectedReleased 表示 reservationId 已 RELEASED，禁止复用。
	RejectedReleased bool
}

type ReleaseWithLedgerResult struct {
	Ok              bool
	AlreadyReleased bool
}

func EncodeReservationLedger(
	status, amount, releaseID, roundID string,
	agent int64,
	poolSymbol, reason string,
	updatedAt int64,
) string {
	return fmt.Sprintf("%s|%s|%s|%s|%d|%s|%s|%d",
		status, amount, releaseID, roundID, agent, poolSymbol, reason, updatedAt)
}

func ParseReservationLedgerStatus(raw string) string {
	if raw == "" {
		return ""
	}
	for i := 0; i < len(raw); i++ {
		if raw[i] == '|' {
			return raw[:i]
		}
	}
	return raw
}

func SettleWalletMarkerKey(settlementID string, userID uint32) string {
	return fmt.Sprintf("%s%s:%d", settleWalletMarkerPrefix, settlementID, userID)
}

func (rd *RedisDao) RefundPlayerCurrencyWithCancelMarker(
	playerId uint32,
	currencyDelta int64,
	markerKey string,
	markerValue string,
	markerTTLSeconds int32,
	carry ...FinanceCarryCredit,
) (CancelWalletRefundResult, error) {
	if currencyDelta < 0 || currencyDelta > 9007199254740991 || markerKey == "" || markerValue == "" || markerTTLSeconds <= 0 {
		return CancelWalletRefundResult{}, fmt.Errorf("invalid finance credit")
	}
	pID := "player_" + strconv.FormatUint(uint64(playerId), 10)
	carryKey, carryRaw, bindingErr := FinanceCarryArgs(playerId, carry)
	if bindingErr != nil {
		return CancelWalletRefundResult{}, bindingErr
	}
	result, err := refundWithCancelMarkerScript.Run(
		context.Background(),
		rd.cli,
		[]string{pID, "dirty_list_imp", markerKey, carryKey},
		currencyDelta,
		0,
		int64((20*time.Minute)/time.Second),
		playerId,
		markerValue,
		int64(markerTTLSeconds),
		carryRaw,
	).Slice()
	if err != nil {
		return CancelWalletRefundResult{}, err
	}
	if len(result) != 2 {
		return CancelWalletRefundResult{}, fmt.Errorf("unexpected cancel refund script result")
	}
	status, ok := result[0].(int64)
	if !ok {
		return CancelWalletRefundResult{}, fmt.Errorf("unexpected cancel refund script status")
	}
	value, ok := result[1].(int64)
	if !ok {
		return CancelWalletRefundResult{}, fmt.Errorf("unexpected cancel refund script balance")
	}
	switch status {
	case 0:
		return CancelWalletRefundResult{}, ErrPlayerNotCached
	case 1:
		return CancelWalletRefundResult{NewCurrency: value}, ErrInsufficientFunds
	case 2:
		return CancelWalletRefundResult{AlreadyDone: false, NewCurrency: value}, nil
	case 3:
		return CancelWalletRefundResult{AlreadyDone: true, NewCurrency: value}, nil
	default:
		return CancelWalletRefundResult{}, fmt.Errorf("unknown cancel refund script status: %d", status)
	}
}

func (rd *RedisDao) GetAgentReserved(agentID int64, symbol string) decimal.Decimal {
	key := fmt.Sprintf("%d_%s", agentID, symbol)
	value, err := rd.cli.ZScore(context.Background(), "agent_reserved_data", key).Result()
	if err == redis.Nil {
		return decimal.Zero
	}
	if err != nil {
		zap.L().Error("读取奖池有效预留失败", zap.Int64("agentId", agentID), zap.String("symbol", symbol), zap.Error(err))
		return decimal.Zero
	}
	return decimal.NewFromFloat(value)
}

func (rd *RedisDao) TryReserveAgentPool(agentID int64, symbol string, basePool, amount decimal.Decimal) (bool, error) {
	key := fmt.Sprintf("%d_%s", agentID, symbol)
	result, err := tryReservePoolScript.Run(
		context.Background(),
		rd.cli,
		[]string{"agent_reserved_data"},
		key,
		basePool.String(),
		amount.String(),
	).Int64()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (rd *RedisDao) ReleaseAgentReserved(agentID int64, symbol string, amount decimal.Decimal) (bool, error) {
	key := fmt.Sprintf("%d_%s", agentID, symbol)
	result, err := releasePoolReservationScript.Run(
		context.Background(),
		rd.cli,
		[]string{"agent_reserved_data"},
		key,
		amount.String(),
	).Int64()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (rd *RedisDao) TryReserveAgentPoolWithLedger(
	agentID int64,
	symbol string,
	basePool, amount decimal.Decimal,
	reservationID, roundID string,
) (ReserveWithLedgerResult, error) {
	if reservationID == "" {
		return ReserveWithLedgerResult{}, fmt.Errorf("reservationId required")
	}
	key := fmt.Sprintf("%d_%s", agentID, symbol)
	ledger := EncodeReservationLedger(
		ReservationLedgerActive,
		amount.String(),
		"",
		roundID,
		agentID,
		symbol,
		"",
		time.Now().Unix(),
	)
	code, err := reserveWithLedgerScript.Run(
		context.Background(),
		rd.cli,
		[]string{"agent_reserved_data", reservationLedgerRedisKey},
		key,
		basePool.String(),
		amount.String(),
		reservationID,
		ledger,
	).Int64()
	if err != nil {
		return ReserveWithLedgerResult{}, err
	}
	switch code {
	case 0:
		return ReserveWithLedgerResult{Ok: false}, nil
	case 1:
		return ReserveWithLedgerResult{Ok: true}, nil
	case 2:
		return ReserveWithLedgerResult{Ok: true, AlreadyActive: true}, nil
	case 3:
		return ReserveWithLedgerResult{RejectedReleased: true}, nil
	default:
		return ReserveWithLedgerResult{}, fmt.Errorf("unknown reserveWithLedger status: %d", code)
	}
}

func (rd *RedisDao) ReleaseAgentReservedWithLedger(
	agentID int64,
	symbol string,
	amount decimal.Decimal,
	reservationID, roundID, releaseID, reason string,
	allowMigrate bool,
) (ReleaseWithLedgerResult, error) {
	if reservationID == "" {
		return ReleaseWithLedgerResult{}, fmt.Errorf("reservationId required")
	}
	key := fmt.Sprintf("%d_%s", agentID, symbol)
	now := time.Now().Unix()
	releasedLedger := EncodeReservationLedger(
		ReservationLedgerReleased,
		amount.String(),
		releaseID,
		roundID,
		agentID,
		symbol,
		reason,
		now,
	)
	migrateLedger := EncodeReservationLedger(
		ReservationLedgerActive,
		amount.String(),
		"",
		roundID,
		agentID,
		symbol,
		"",
		now,
	)
	migrateFlag := 0
	if allowMigrate {
		migrateFlag = 1
	}
	code, err := releaseWithLedgerScript.Run(
		context.Background(),
		rd.cli,
		[]string{"agent_reserved_data", reservationLedgerRedisKey, "lottery_finance_rounds"},
		key,
		amount.String(),
		reservationID,
		releaseID,
		releasedLedger,
		migrateFlag,
		migrateLedger,
		roundID,
	).Int64()
	if err != nil {
		return ReleaseWithLedgerResult{}, err
	}
	switch code {
	case 0:
		return ReleaseWithLedgerResult{Ok: false}, nil
	case 1:
		return ReleaseWithLedgerResult{Ok: true}, nil
	case 2:
		return ReleaseWithLedgerResult{Ok: true, AlreadyReleased: true}, nil
	default:
		return ReleaseWithLedgerResult{}, fmt.Errorf("unknown releaseWithLedger status: %d", code)
	}
}

func (rd *RedisDao) AdjustAgentReservedWithLedger(
	agentID int64,
	symbol string,
	basePool, oldAmount, newAmount decimal.Decimal,
	reservationID, roundID string,
) (bool, error) {
	if reservationID == "" {
		return false, fmt.Errorf("reservationId required")
	}
	key := fmt.Sprintf("%d_%s", agentID, symbol)
	ledger := EncodeReservationLedger(
		ReservationLedgerActive,
		newAmount.String(),
		"",
		roundID,
		agentID,
		symbol,
		"",
		time.Now().Unix(),
	)
	code, err := adjustReservationLedgerScript.Run(
		context.Background(),
		rd.cli,
		[]string{"agent_reserved_data", reservationLedgerRedisKey},
		key,
		basePool.String(),
		reservationID,
		oldAmount.String(),
		newAmount.String(),
		ledger,
	).Int64()
	if err != nil {
		return false, err
	}
	switch code {
	case 1:
		return true, nil
	case 0, 2:
		return false, nil
	default:
		return false, fmt.Errorf("unknown adjustReservationLedger status: %d", code)
	}
}

func (rd *RedisDao) GetReservationLedgerRaw(reservationID string) (string, error) {
	val, err := rd.cli.HGet(context.Background(), reservationLedgerRedisKey, reservationID).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

func (rd *RedisDao) SeedReservationLedgerActive(
	agentID int64,
	symbol string,
	amount decimal.Decimal,
	reservationID, roundID string,
) error {
	if reservationID == "" {
		return fmt.Errorf("reservationId required")
	}
	ledger := EncodeReservationLedger(
		ReservationLedgerActive,
		amount.String(),
		"",
		roundID,
		agentID,
		symbol,
		"",
		time.Now().Unix(),
	)
	ok, err := rd.cli.HSetNX(context.Background(), reservationLedgerRedisKey, reservationID, ledger).Result()
	if err != nil {
		return err
	}
	if !ok {
		// 已存在则视为成功（可能并发补建）。
		return nil
	}
	return nil
}

func (rd *RedisDao) AddFinanceReconcileRound(roundID string) error {
	if roundID == "" {
		return nil
	}
	return rd.cli.SAdd(context.Background(), financeReconcileSetKey, roundID).Err()
}

func (rd *RedisDao) PopFinanceReconcileRounds(limit int64) ([]string, error) {
	if limit <= 0 {
		limit = 50
	}
	ids, err := rd.cli.SRandMemberN(context.Background(), financeReconcileSetKey, limit).Result()
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	members := make([]interface{}, len(ids))
	for i, id := range ids {
		members[i] = id
	}
	_ = rd.cli.SRem(context.Background(), financeReconcileSetKey, members...).Err()
	return ids, nil
}

func (rd *RedisDao) CreditPlayerCurrencyWithSettleMarker(
	playerID uint32,
	currencyDelta int64,
	settlementID string,
	carry ...FinanceCarryCredit,
) (CancelWalletRefundResult, error) {
	return rd.RefundPlayerCurrencyWithCancelMarker(
		playerID,
		currencyDelta,
		SettleWalletMarkerKey(settlementID, playerID),
		settlementID,
		SettleWalletMarkerTTLSec,
		carry...,
	)
}

func (rd *RedisDao) HGetAll(key string) (map[string]string, error) {
	return rd.cli.HGetAll(context.Background(), key).Result()
}
