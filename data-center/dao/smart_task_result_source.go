package dao

import (
	"app/esindex"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/olivere/elastic/v7"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

// Keep this fixed operation on data-center, where the existing ES source is
// available. No new protocol fields, cache, financial journal or money replay.
func (dd *DBDao) ResolveSmartTask(ctx context.Context, es *ESDao, payload string) (any, error) {
	var input smartTaskInput
	if err := DecodeGogameJSON(payload, &input); err != nil {
		return nil, err
	}
	return dd.resolveSmartTask(ctx, input, es)
}

func smartTaskCurrentSnapshot(ctx context.Context, task map[string]any) (map[string]any, error) {
	rd := RedisIns()
	if rd == nil || rd.Client == nil {
		return nil, status.Error(codes.Unavailable, "financial snapshot query unavailable")
	}
	attempt := paymentMap(task["activeAttempt"])
	raw, err := rd.Client.HGet(ctx, "lottery_finance_rounds", fmt.Sprint(attempt["roundId"])).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, status.Error(codes.Unavailable, "financial snapshot query failed")
	}
	var round map[string]any
	if err = DecodeGogameJSON(raw, &round); err != nil {
		return nil, err
	}
	state := round["state"]
	if state == "SETTLED" || state == "VOIDED" || state == "EXPIRED" {
		at := int64(0)
		if state == "EXPIRED" {
			at = gogameNumber(paymentMap(round["reservation"])["expiresAt"])
		} else {
			at = gogameNumber(paymentMap(round["settlement"])["completedAt"])
		}
		if at <= 0 {
			at = gogameNumber(round["updatedAt"])
		}
		now := time.Now().Unix()
		if at <= 0 || at > now {
			return nil, paymentInvalid("financial completion time is not confirmed; responsibility retained")
		}
		if now-at >= 3600 {
			return nil, nil
		}
	}
	return round, nil
}

// Older nonzero net proves this was not a full refund. Zero-net games require
// the original financial service's normal-settlement marker in the same log.
func (es *ESDao) smartTaskSettledNet(ctx context.Context, task map[string]any) (int64, error) {
	if es == nil || es.es == nil {
		return 0, paymentInvalid("original financial result is unavailable; responsibility retained")
	}
	attempt := paymentMap(task["activeAttempt"])
	id := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%d|%d|%s", gogameNumber(task["agentId"]), gogameNumber(task["userId"]), attempt["roundId"]))))
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	result, err := es.es.Get().Index(esindex.Settlement()).Id(id).Do(ctx)
	if elastic.IsNotFound(err) || err == nil && (result == nil || !result.Found) {
		return 0, paymentInvalid("original settlement is unavailable; responsibility retained")
	}
	if err != nil {
		return 0, status.Error(codes.Unavailable, "original settlement query failed")
	}
	if result.Id != id || len(result.Source) == 0 || len(result.Source) > 2*1024*1024 {
		return 0, paymentInvalid("original settlement identity or size invalid")
	}
	var report map[string]any
	if err = DecodeGogameJSON(string(result.Source), &report); err != nil {
		return 0, err
	}
	return smartTaskDocumentNet(task, report)
}

func smartTaskDocumentNet(task, report map[string]any) (int64, error) {
	attempt := paymentMap(task["activeAttempt"])
	if !validSmartAttempt(smartTaskInput{AgentID: gogameNumber(task["agentId"]), Attempt: attempt}) || report["level"] == nil || report["complete"] != true || report["roundID"] != attempt["roundId"] || report["currency"] != task["currency"] ||
		gogameNumber(report["agentId"]) != gogameNumber(task["agentId"]) || gogameNumber(report["userId"]) != gogameNumber(task["userId"]) ||
		gogameNumber(report["gameId"]) != gogameNumber(attempt["gameId"]) || gogameNumber(report["level"]) != gogameNumber(attempt["level"]) {
		return 0, paymentInvalid("original settlement task identity mismatch")
	}
	bet, e1 := taskMoney(report["bet"])
	win, e2 := taskMoney(report["win"])
	_, e3 := taskMoney(report["balance"])
	if e1 != nil || e2 != nil || e3 != nil || !bet.Shift(2).Equal(decimal.NewFromInt(gogameNumber(task["betAmount"]))) {
		return 0, paymentInvalid("original settlement money invalid")
	}
	net := win.Sub(bet).Shift(2)
	if net.Abs().GreaterThan(decimal.NewFromInt(cashMax)) {
		return 0, paymentInvalid("original settlement net exceeds safe range")
	}
	var envelope map[string]any
	if err := DecodeGogameJSON(fmt.Sprint(report["log"]), &envelope); err != nil {
		return 0, err
	}
	if proof, present := envelope["smartTaskSettlement"]; net.IsZero() || present {
		if err := smartTaskNormalSettlementProof(task, paymentMap(proof)); err != nil {
			return 0, err
		}
	}
	display := paymentMap(envelope["completedDisplayReport"])
	reference := paymentMap(display["smartTask"])
	if reference["taskId"] != task["taskId"] || reference["attemptId"] != attempt["attemptId"] || gogameNumber(reference["inning"]) != gogameNumber(attempt["inning"]) ||
		gogameNumber(display["inning"]) != gogameNumber(attempt["inning"]) || gogameNumber(display["userId"]) != gogameNumber(task["userId"]) ||
		gogameNumber(display["agentId"]) != gogameNumber(task["agentId"]) || display["currency"] != task["currency"] ||
		gogameNumber(display["roomId"]) != gogameNumber(task["roomId"]) || gogameNumber(display["roomType"]) != gogameNumber(task["roomType"]) ||
		gogameNumber(display["net"]) != net.IntPart() || gogameNumber(display["effectiveBet"]) != gogameNumber(task["betAmount"]) {
		return 0, paymentInvalid("original settlement log does not confirm this task execution")
	}
	return net.IntPart(), nil
}

func smartTaskNormalSettlementProof(task, proof map[string]any) error {
	attempt := paymentMap(task["activeAttempt"])
	id, ok := proof["settlementId"].(string)
	completed := gogameNumber(proof["completedAt"])
	if gogameNumber(proof["version"]) != 1 || proof["state"] != "SETTLED" || proof["mode"] != "NORMAL" || proof["roundId"] != attempt["roundId"] ||
		!ok || !cashID.MatchString(id) || completed <= 0 || completed > time.Now().Unix() || proof["taskId"] != task["taskId"] ||
		proof["attemptId"] != attempt["attemptId"] || gogameNumber(proof["inning"]) != gogameNumber(attempt["inning"]) {
		return paymentInvalid("original normal-settlement proof is missing or mismatched; responsibility retained")
	}
	return nil
}
