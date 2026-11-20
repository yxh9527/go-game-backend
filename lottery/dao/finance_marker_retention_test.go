package dao

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func completedReceiptRound(mode string, at int64) map[string]any {
	state := "SETTLED"
	if mode == "VOID_REFUND" {
		state = "VOIDED"
	}
	return map[string]any{"roundId": "original-round", "state": state, "updatedAt": at, "settlement": map[string]any{"settlementId": "original-settle", "mode": mode, "status": "COMPLETED", "requestHash": "original-fingerprint", "completedAt": at, "results": []any{map[string]any{"userId": 42, "code": 0, "currency": "100.00"}}}}
}

func TestCompletedFinanceReceiptsUseOriginalOneHourDeadlineAndNeverRenew(t *testing.T) {
	for _, mode := range []string{"NORMAL", "VOID_REFUND", "SEAT_NET", "BANKER_NET"} {
		t.Run(mode, func(t *testing.T) {
			rd, client := retentionRedis(t)
			ctx := context.Background()
			now := int64(1800000000)
			round := completedReceiptRound(mode, now-300)
			raw, _ := json.Marshal(round)
			client.HSet(ctx, "lottery_finance_rounds", "original-round", raw)
			key, value := SettleWalletMarkerKey("original-settle", 42), "original-settle"
			if mode == "SEAT_NET" || mode == "BANKER_NET" {
				kind := "seat-net"
				if mode == "BANKER_NET" {
					kind = "banker"
				}
				key = fmt.Sprintf("lottery:settle_wallet:%s:%x", kind, sha256.Sum256([]byte("original-settle")))
				value = `{"fingerprint":"original-fingerprint","result":"original-result"}`
			}
			client.Set(ctx, key, value, 7*24*time.Hour)
			client.HSet(ctx, "player_42", "currency", 12345, "cash_revision", 8)
			if n, err := rd.RetainCompletedFinanceReceipts(ctx, "original-round", now); err != nil || n != 1 || client.PTTL(ctx, key).Val() != 3300*time.Second {
				t.Fatal(n, err, client.PTTL(ctx, key).Val())
			}
			client.PExpire(ctx, key, time.Minute)
			if n, err := rd.RetainCompletedFinanceReceipts(ctx, "original-round", now+10); err != nil || n != 0 || client.PTTL(ctx, key).Val() != time.Minute {
				t.Fatal("repeated confirmation renewed shorter expiry", n, err)
			}
			if client.Get(ctx, key).Val() != value || client.HGet(ctx, "lottery_finance_rounds", "original-round").Val() != string(raw) || client.HGet(ctx, "player_42", "currency").Val() != "12345" || client.HGet(ctx, "player_42", "cash_revision").Val() != "8" {
				t.Fatal("retention modified original money or result")
			}
		})
	}
}

func TestClaimedMoneyAndUnconfirmedCancelRemainProtected(t *testing.T) {
	for _, variant := range []string{"claimed", "active-state", "failed-account", "missing-account-code", "future-time", "no-time", "cancel-pending", "cancel-unpaid", "cancel-incomplete"} {
		t.Run(variant, func(t *testing.T) {
			rd, client := retentionRedis(t)
			ctx := context.Background()
			now := int64(1800000000)
			round := completedReceiptRound("NORMAL", now-300)
			settlement := round["settlement"].(map[string]any)
			switch variant {
			case "claimed":
				settlement["status"] = "CLAIMED"
			case "active-state":
				round["state"] = "BETTING"
			case "failed-account":
				settlement["results"].([]any)[0].(map[string]any)["code"] = 1
			case "missing-account-code":
				delete(settlement["results"].([]any)[0].(map[string]any), "code")
			case "future-time":
				settlement["completedAt"] = now + 1
			case "no-time":
				settlement["completedAt"] = 0
			default:
				delete(round, "settlement")
				round["state"] = "BETTING"
				claim := map[string]any{"requestId": "cancel-original", "userId": 42, "status": "COMPLETED", "success": true, "walletRefunded": true, "refundAmount": "1.00", "completedAt": now - 300}
				if variant == "cancel-pending" {
					claim["status"] = "CLAIMED"
				}
				if variant == "cancel-unpaid" {
					claim["walletRefunded"] = false
				}
				if variant == "cancel-incomplete" {
					claim["success"] = false
				}
				round["cancelRequests"] = map[string]any{"cancel-original": claim}
			}
			raw, _ := json.Marshal(round)
			client.HSet(ctx, "lottery_finance_rounds", "original-round", raw)
			key := SettleWalletMarkerKey("original-settle", 42)
			if len(variant) >= 6 && variant[:6] == "cancel" {
				key = "lottery:cancel_wallet:cancel-original"
			}
			client.Set(ctx, key, "original-settle", 7*24*time.Hour)
			if n, err := rd.RetainCompletedFinanceReceipts(ctx, "original-round", now); err != nil || n != 0 || client.PTTL(ctx, key).Val() != 7*24*time.Hour {
				t.Fatal("unfinished money lost its confirmation protection", n, err)
			}
		})
	}
}

func TestCompletedCancelInActiveRoomExpiresFromItsOwnOriginalCompletion(t *testing.T) {
	rd, client := retentionRedis(t)
	ctx := context.Background()
	now := int64(1800000000)
	round := map[string]any{"roundId": "original-round", "state": "BETTING", "cancelRequests": map[string]any{"cancel-original": map[string]any{"requestId": "cancel-original", "userId": 42, "status": "COMPLETED", "success": true, "walletRefunded": true, "refundAmount": "1.00", "completedAt": now - 3600}}}
	raw, _ := json.Marshal(round)
	client.HSet(ctx, "lottery_finance_rounds", "original-round", raw)
	client.Set(ctx, "lottery:cancel_wallet:cancel-original", "1.00", 7*24*time.Hour)
	if n, err := rd.RetainCompletedFinanceReceipts(ctx, "original-round", now); err != nil || n != 1 || client.Exists(ctx, "lottery:cancel_wallet:cancel-original").Val() != 0 {
		t.Fatal(n, err)
	}
	if client.HGet(ctx, "lottery_finance_rounds", "original-round").Val() != string(raw) {
		t.Fatal("active original round was removed")
	}
	if n, err := rd.RetainCompletedFinanceReceipts(ctx, "original-round", now+10); err != nil || n != 0 || client.Exists(ctx, "lottery:cancel_wallet:cancel-original").Val() != 0 {
		t.Fatal("expired confirmation was recreated", n, err)
	}
}

func TestConflictingReceiptsAndChangedSnapshotsCannotBeShortened(t *testing.T) {
	rd, client := retentionRedis(t)
	ctx := context.Background()
	now := int64(1800000000)
	round := completedReceiptRound("NORMAL", now-100)
	round["settlement"].(map[string]any)["results"] = []any{map[string]any{"userId": 42, "code": 0}, map[string]any{"userId": 43, "code": 0}}
	raw, _ := json.Marshal(round)
	client.HSet(ctx, "lottery_finance_rounds", "original-round", raw)
	client.Set(ctx, SettleWalletMarkerKey("original-settle", 42), "original-settle", 7*24*time.Hour)
	client.Set(ctx, SettleWalletMarkerKey("original-settle", 43), "different-receipt", 7*24*time.Hour)
	if _, err := rd.RetainCompletedFinanceReceipts(ctx, "original-round", now); err == nil {
		t.Fatal("mismatched original confirmation was shortened")
	}
	if client.PTTL(ctx, SettleWalletMarkerKey("original-settle", 42)).Val() != 7*24*time.Hour {
		t.Fatal("conflicting batch partially changed expiry")
	}
	refs, _ := json.Marshal([]completedFinanceReceipt{{Key: SettleWalletMarkerKey("original-settle", 42), Value: "original-settle", CompletedAt: now - 100}})
	client.HSet(ctx, "lottery_finance_rounds", "original-round", `{"roundId":"original-round","state":"BETTING"}`)
	n, err := retainCompletedFinanceReceiptsScript.Run(ctx, client, []string{"lottery_finance_rounds", SettleWalletMarkerKey("original-settle", 42)}, "original-round", string(raw), string(refs), now).Int()
	if err != nil || n != 0 || client.PTTL(ctx, SettleWalletMarkerKey("original-settle", 42)).Val() != 7*24*time.Hour {
		t.Fatal("stale snapshot changed current protection", n, err)
	}
}

func TestReceiptRetentionChunksLargeCompletedRoundsWithoutExtendingShortTTL(t *testing.T) {
	rd, client := retentionRedis(t)
	ctx := context.Background()
	now := int64(1800000000)
	round := completedReceiptRound("NORMAL", now-300)
	results := []any{}
	for uid := uint32(1); uid <= 145; uid++ {
		results = append(results, map[string]any{"userId": uid, "code": 0})
		client.Set(ctx, SettleWalletMarkerKey("original-settle", uid), "original-settle", 7*24*time.Hour)
	}
	round["settlement"].(map[string]any)["results"] = results
	raw, _ := json.Marshal(round)
	client.HSet(ctx, "lottery_finance_rounds", "original-round", raw)
	if n, err := rd.RetainCompletedFinanceReceipts(ctx, "original-round", now); err != nil || n != 145 {
		t.Fatal(n, err)
	}
	for uid := uint32(1); uid <= 145; uid++ {
		if ttl := client.PTTL(ctx, SettleWalletMarkerKey("original-settle", uid)).Val(); ttl != 3300*time.Second {
			t.Fatal(uid, ttl)
		}
	}
}

func TestSeatNetNewAtomicReceiptHasOneHourTTLAndRetryDoesNotRenew(t *testing.T) {
	rd, batch, before, terminal := seatNetFixture(t)
	ctx := context.Background()
	if _, err := rd.SettleSeatNet("net-round", "original-fingerprint", before, terminal, batch); err != nil {
		t.Fatal(err)
	}
	key := fmt.Sprintf("lottery:settle_wallet:seat-net:%x", sha256.Sum256([]byte(batch.ID)))
	if ttl := rd.cli.PTTL(ctx, key).Val(); ttl > time.Hour || ttl < 59*time.Minute {
		t.Fatal("atomic receipt did not use one hour", ttl)
	}
	rd.cli.PExpire(ctx, key, time.Minute)
	if _, err := rd.SettleSeatNet("net-round", "original-fingerprint", before, terminal, batch); err != nil || rd.cli.PTTL(ctx, key).Val() > time.Minute {
		t.Fatal("net settlement retry renewed original receipt", err)
	}
}
