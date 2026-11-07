package dao

import (
	"app/cashwallet"
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"regexp"
	"strings"
)

const cashMax int64 = 9007199254740991

var cashID = regexp.MustCompile(`^[a-zA-Z0-9_.:/-]{1,200}$`)
var cashCurrency = regexp.MustCompile(`^[A-Z0-9]{1,12}$`)

type CashAccount struct {
	UserID   uint32 `json:"userId"`
	AgentID  uint32 `json:"agentId"`
	Currency string `json:"currency"`
	Revision int64  `json:"revision"`
}
type CashStep struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Account   int    `json:"account"`
	Amount    int64  `json:"amount"`
	Token     string `json:"token,omitempty"`
	TableKey  string `json:"tableKey,omitempty"`
	Revision  int64  `json:"revision,omitempty"`
	KeepOwner bool   `json:"keepOwner,omitempty"`
}
type CashRequest struct {
	ID       string        `json:"id"`
	Accounts []CashAccount `json:"accounts"`
	Steps    []CashStep    `json:"steps"`
}

func cashKeys(a CashAccount) []string {
	suffix := fmt.Sprintf("%d:%d:%s", a.AgentID, a.UserID, a.Currency)
	return []string{fmt.Sprintf("player_%d", a.UserID), "gogame:cash-carry:" + suffix}
}
func validateCashAccount(a CashAccount) error {
	if a.UserID == 0 || a.AgentID == 0 || !cashCurrency.MatchString(a.Currency) {
		return status.Error(codes.InvalidArgument, "invalid cash account")
	}
	return nil
}
func cashError(err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	switch {
	case strings.Contains(message, "CASH_ACCOUNT_MISMATCH"):
		return status.Error(codes.PermissionDenied, "cash account agent or currency mismatch")
	case strings.Contains(message, "CASH_PLAYER_MISSING"):
		return status.Error(codes.FailedPrecondition, "authoritative player cache unavailable")
	case strings.Contains(message, "CASH_INSUFFICIENT"):
		return status.Error(codes.FailedPrecondition, "insufficient available balance")
	case strings.Contains(message, "CASH_CARRY_"), strings.Contains(message, "CASH_REVISION"), strings.Contains(message, "CASH_OPERATION_CONFLICT"), strings.Contains(message, "CASH_ID_CONFLICT"):
		return status.Error(codes.FailedPrecondition, "cash reservation or operation conflict")
	case strings.Contains(message, "CASH_"):
		return status.Error(codes.FailedPrecondition, "invalid authoritative cash state")
	default:
		return err
	}
}

const cashReadScript = `
if redis.call('HGET',KEYS[1],'id') ~= ARGV[1] then return redis.error_reply('CASH_PLAYER_MISSING') end
if redis.call('HGET',KEYS[1],'agent_id') ~= ARGV[2] or redis.call('HGET',KEYS[1],'currency_type') ~= ARGV[3] then return redis.error_reply('CASH_ACCOUNT_MISMATCH') end
local balance = tonumber(redis.call('HGET',KEYS[1],'currency'))
if not balance or balance < 0 or balance % 1 ~= 0 or balance > 9007199254740991 then return redis.error_reply('CASH_BALANCE_INVALID') end
local raw = redis.call('GET',KEYS[2])
local carry = raw and cjson.decode(raw) or cjson.null
return cjson.encode({balance=balance,revision=tonumber(redis.call('HGET',KEYS[1],'cash_revision') or '0'),carry=carry,ttlMs=raw and redis.call('PTTL',KEYS[2]) or 0})
`

func CashRead(ctx context.Context, client redis.UniversalClient, a CashAccount) (json.RawMessage, error) {
	if err := validateCashAccount(a); err != nil {
		return nil, err
	}
	if client == nil {
		return nil, status.Error(codes.Unavailable, "cash Redis unavailable")
	}
	raw, err := client.Eval(ctx, cashReadScript, cashKeys(a)[:2], a.UserID, a.AgentID, a.Currency).Text()
	return json.RawMessage(raw), cashError(err)
}
func CashApply(ctx context.Context, client redis.UniversalClient, request CashRequest) (json.RawMessage, error) {
	if !cashID.MatchString(request.ID) || len(request.Accounts) == 0 || len(request.Accounts) > 64 || len(request.Steps) == 0 || len(request.Steps) > 1024 {
		return nil, status.Error(codes.InvalidArgument, "invalid cash transaction size or id")
	}
	if client == nil {
		return nil, status.Error(codes.Unavailable, "cash Redis unavailable")
	}
	keys := []string{"dirty_list_imp"}
	seen := map[uint32]bool{}
	for _, a := range request.Accounts {
		if err := validateCashAccount(a); err != nil {
			return nil, err
		}
		if seen[a.UserID] {
			return nil, status.Error(codes.InvalidArgument, "duplicate cash account")
		}
		seen[a.UserID] = true
		keys = append(keys, cashKeys(a)...)
	}
	for _, op := range request.Steps {
		if !cashID.MatchString(op.ID) || op.Account < 1 || op.Account > len(request.Accounts) || op.Amount > cashMax || op.Amount < -cashMax || op.Revision < 0 || op.Revision > cashMax {
			return nil, status.Error(codes.InvalidArgument, "invalid cash operation")
		}
		switch op.Kind {
		case "reserve", "carryDelta", "release", "renew":
			if !cashID.MatchString(op.Token) || !cashID.MatchString(op.TableKey) {
				return nil, status.Error(codes.InvalidArgument, "cash token and table required")
			}
		case "debit", "credit":
		default:
			return nil, status.Error(codes.InvalidArgument, "unknown cash operation")
		}
	}
	encoded, _ := json.Marshal(request)
	result, err := client.Eval(ctx, cashwallet.Apply, keys, encoded).Text()
	return json.RawMessage(result), cashError(err)
}
func (dd *DBDao) gogameCash(ctx context.Context, operation, payload string) (any, error) {
	rd := RedisIns()
	if rd == nil || rd.Client == nil {
		return nil, status.Error(codes.Unavailable, "cash Redis unavailable")
	}
	if operation == "cashApply" {
		var request CashRequest
		if err := DecodeGogameJSON(payload, &request); err != nil {
			return nil, err
		}
		return CashApply(ctx, rd.Client, request)
	}
	var query CashAccount
	if err := DecodeGogameJSON(payload, &query); err != nil {
		return nil, err
	}
	if operation != "cashRead" {
		return nil, status.Error(codes.InvalidArgument, "unknown cash operation")
	}
	return CashRead(ctx, rd.Client, query)
}
