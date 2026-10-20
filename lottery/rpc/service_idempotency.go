package rpc

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"micro_service/services"
)

const idempotencyTTLSeconds int32 = 60 * 60 // 1 hour

type idempotencyEntry struct {
	Signature string `json:"s"`
	Done      bool   `json:"d"`
	Payload   string `json:"p"`
}

func idempotencySignature(parts ...string) string {
	h := sha256.New()
	for i, p := range parts {
		if i > 0 {
			h.Write([]byte{0})
		}
		h.Write([]byte(p))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func buildIdempotencyKey(op string, parts ...string) string {
	b := strings.Builder{}
	b.WriteString("idem:facade:")
	b.WriteString(op)
	for _, p := range parts {
		b.WriteByte(':')
		b.WriteString(strings.TrimSpace(p))
	}
	return b.String()
}

// beginIdempotency 查重并占位。
// hit=true 表示命中已完成结果，直接返回 payload；
// code!=OK 表示冲突/系统错误，应直接失败。
func (d *LotteryService) beginIdempotency(key, signature string) (hit bool, payload string, code services.ErrorCode) {
	raw, err := d.rds.Get(key)
	if err != nil && err != redis.Nil {
		zap.L().Error("idempotency get failed", zap.String("key", key), zap.Error(err))
		return false, "", services.ErrorCode_SYSTEM_ERROR
	}
	if err == nil && raw != "" {
		entry := &idempotencyEntry{}
		if uErr := jsoniter.UnmarshalFromString(raw, entry); uErr != nil {
			zap.L().Error("idempotency parse failed", zap.String("key", key), zap.Error(uErr))
			return false, "", services.ErrorCode_SYSTEM_ERROR
		}
		if entry.Signature != signature {
			zap.L().Error("idempotency signature conflict",
				zap.String("key", key),
				zap.String("expect", entry.Signature),
				zap.String("got", signature))
			return false, "", services.ErrorCode_PARAMS_INVALID
		}
		if entry.Done {
			return true, entry.Payload, services.ErrorCode_OK
		}
		// 进行中：并发重放，先按系统繁忙处理
		return false, "", services.ErrorCode_SYSTEM_ERROR
	}

	pending, _ := jsoniter.MarshalToString(&idempotencyEntry{Signature: signature, Done: false})
	ok, err := d.rds.SetNX(key, pending, idempotencyTTLSeconds)
	if err != nil {
		zap.L().Error("idempotency setnx failed", zap.String("key", key), zap.Error(err))
		return false, "", services.ErrorCode_SYSTEM_ERROR
	}
	if !ok {
		// 并发抢占失败，再读一次
		raw2, gErr := d.rds.Get(key)
		if gErr != nil && gErr != redis.Nil {
			return false, "", services.ErrorCode_SYSTEM_ERROR
		}
		if raw2 == "" {
			return false, "", services.ErrorCode_SYSTEM_ERROR
		}
		entry := &idempotencyEntry{}
		if uErr := jsoniter.UnmarshalFromString(raw2, entry); uErr != nil {
			return false, "", services.ErrorCode_SYSTEM_ERROR
		}
		if entry.Signature != signature {
			return false, "", services.ErrorCode_PARAMS_INVALID
		}
		if entry.Done {
			return true, entry.Payload, services.ErrorCode_OK
		}
		return false, "", services.ErrorCode_SYSTEM_ERROR
	}
	return false, "", services.ErrorCode_OK
}

func (d *LotteryService) commitIdempotency(key, signature, payload string) {
	done, err := jsoniter.MarshalToString(&idempotencyEntry{
		Signature: signature,
		Done:      true,
		Payload:   payload,
	})
	if err != nil {
		zap.L().Error("idempotency marshal commit failed", zap.String("key", key), zap.Error(err))
		_ = d.rds.Del(key)
		return
	}
	if err := d.rds.Set(key, done, idempotencyTTLSeconds); err != nil {
		zap.L().Error("idempotency commit failed", zap.String("key", key), zap.Error(err))
	}
}

func (d *LotteryService) abortIdempotency(key string) {
	if strings.TrimSpace(key) == "" {
		return
	}
	if err := d.rds.Del(key); err != nil {
		zap.L().Error("idempotency abort failed", zap.String("key", key), zap.Error(err))
	}
}

// 百人加注/退注每次都是独立扣退。游戏服 roundId 常为 mp-{gameId}-{userId}，
// 同一玩家同一游戏会连续不同金额、不同区域加注，不能按 roundId 做完成态幂等。
// 这里只做短占位，挡住进行中的并发重入。
const multiOpLockTTLSeconds int32 = 5

func (d *LotteryService) beginMultiOpLock(key string) services.ErrorCode {
	if strings.TrimSpace(key) == "" {
		return services.ErrorCode_SYSTEM_ERROR
	}
	ok, err := d.rds.SetNX(key, "1", multiOpLockTTLSeconds)
	if err != nil {
		zap.L().Error("multi-op lock failed", zap.String("key", key), zap.Error(err))
		return services.ErrorCode_SYSTEM_ERROR
	}
	if !ok {
		return services.ErrorCode_SYSTEM_ERROR
	}
	return services.ErrorCode_OK
}

func (d *LotteryService) endMultiOpLock(key string) {
	d.abortIdempotency(key)
}

func restoreSlotsDoBetResp(payload string) (*services.SlotsDoBetResp, services.ErrorCode) {
	resp := &services.SlotsDoBetResp{}
	if err := jsoniter.UnmarshalFromString(payload, resp); err != nil {
		zap.L().Error("restore SlotsDoBetResp failed", zap.Error(err))
		return nil, services.ErrorCode_SYSTEM_ERROR
	}
	return resp, services.ErrorCode_OK
}

func restoreFruitDoBetResp(payload string) (*services.FruitDoBetResp, services.ErrorCode) {
	resp := &services.FruitDoBetResp{}
	if err := jsoniter.UnmarshalFromString(payload, resp); err != nil {
		return nil, services.ErrorCode_SYSTEM_ERROR
	}
	return resp, services.ErrorCode_OK
}

func restoreFruitDoBetMultiResp(payload string) (*services.FruitDoBetMultiResp, services.ErrorCode) {
	resp := &services.FruitDoBetMultiResp{}
	if err := jsoniter.UnmarshalFromString(payload, resp); err != nil {
		return nil, services.ErrorCode_SYSTEM_ERROR
	}
	return resp, services.ErrorCode_OK
}

func restoreFruitRefundMultiResp(payload string) (*services.FruitRefundMultiResp, services.ErrorCode) {
	resp := &services.FruitRefundMultiResp{}
	if err := jsoniter.UnmarshalFromString(payload, resp); err != nil {
		return nil, services.ErrorCode_SYSTEM_ERROR
	}
	return resp, services.ErrorCode_OK
}

func restoreFruitSettleRoundResp(payload string) (*services.FruitSettleRoundResp, services.ErrorCode) {
	resp := &services.FruitSettleRoundResp{}
	if err := jsoniter.UnmarshalFromString(payload, resp); err != nil {
		return nil, services.ErrorCode_SYSTEM_ERROR
	}
	return resp, services.ErrorCode_OK
}

func restoreUg5GuessResp(payload string) (*services.Ug5GuessResp, services.ErrorCode) {
	resp := &services.Ug5GuessResp{}
	if err := jsoniter.UnmarshalFromString(payload, resp); err != nil {
		return nil, services.ErrorCode_SYSTEM_ERROR
	}
	return resp, services.ErrorCode_OK
}

func u32Str(v uint32) string { return fmt.Sprintf("%d", v) }
func i64Str(v int64) string  { return fmt.Sprintf("%d", v) }
