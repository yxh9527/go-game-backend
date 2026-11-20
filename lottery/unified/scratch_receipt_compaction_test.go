package unified

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestWholeScratchBookCompactsOnlyAfterEveryTicketOwnershipMirrorConfirms(t *testing.T) {
	f := newScratchFixture(t)
	input, _ := f.purchase(true)
	f.mirrorFailure = true
	_, response := f.call("issue", input)
	if response.Success || f.balance(42) != 98000 {
		t.Fatal("expected original committed purchase before mirror fault", response)
	}
	before, err := f.rd.ScratchReceipt(input.ScratchScope, input.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(before, `"rounds"`) || strings.Contains(before, `"deliveryConfirmed":true`) {
		t.Fatal("unconfirmed batch discarded")
	}
	f.mirrorFailure = false
	first, response := f.call("issue", input)
	if !response.Success || len(f.mirrors) != 20 {
		t.Fatal(response, len(f.mirrors))
	}
	after, err := f.rd.ScratchReceipt(input.ScratchScope, input.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(after, `"rounds"`) || !strings.Contains(after, `"deliveryConfirmed":true`) || len(after) >= len(before)/2 {
		t.Fatal("delivered book kept duplicate private snapshots", len(before), len(after))
	}
	f.svc.testScratchMirror = nil
	f.svc.rounds = map[string]*financeRound{}
	second, response := f.call("issue", input)
	one, _ := json.Marshal(first)
	two, _ := json.Marshal(second)
	if !response.Success || string(one) != string(two) || f.balance(42) != 98000 || len(f.svc.rounds) != 0 {
		t.Fatal("delivered receipt retry restored state or repeated money", response)
	}
}

func TestDeliveredPurchaseRetryCannotResurrectAClaimedRoundAfterOneHourCleanup(t *testing.T) {
	f := newScratchFixture(t)
	input, serial := f.purchase(false)
	original, response := f.call("issue", input)
	if !response.Success {
		t.Fatal(response)
	}
	claim := f.input
	claim.Serial = serial
	_, response = f.call("reveal", claim)
	if !response.Success {
		t.Fatal(response)
	}
	id, _ := f.rd.ScratchSerialRound(input.ScratchScope, serial)
	round := f.svc.rounds[id]
	balance := f.balance(42)
	now := time.Now().Unix()
	round.Settlement.CompletedAt = now - 3600
	raw, _ := json.Marshal(round)
	f.client.HSet(context.Background(), financeRoundHash, id, string(raw))
	if err := f.svc.purgeRoundLocked(round, now); err != nil {
		t.Fatal(err)
	}
	f.svc.testScratchMirror = nil
	retry, response := f.call("issue", input)
	one, _ := json.Marshal(original)
	two, _ := json.Marshal(retry)
	if !response.Success || string(one) != string(two) || len(f.svc.rounds) != 0 || f.client.HExists(context.Background(), financeRoundHash, id).Val() || f.balance(42) != balance {
		t.Fatal("old purchase replay recreated cleared financial state", response)
	}
	input.Face = 200
	_, response = f.call("issue", input)
	if response.Success || f.balance(42) != balance {
		t.Fatal("changed original request was accepted", response)
	}
}

func TestOldFullReceiptNeverRestoresMissingFinancialStateAndPartialSQLMirrorRetainsOriginalRecoveryData(t *testing.T) {
	f := newScratchFixture(t)
	input, _ := f.purchase(true)
	mirror := f.svc.testScratchMirror
	calls := 0
	f.svc.testScratchMirror = func(r *financeRound) error {
		calls++
		if calls == 2 {
			return context.DeadlineExceeded
		}
		return mirror(r)
	}
	_, response := f.call("issue", input)
	if response.Success || len(f.mirrors) != 1 {
		t.Fatal(response, len(f.mirrors))
	}
	original, err := f.rd.ScratchReceipt(input.ScratchScope, input.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(original, `"rounds"`) || strings.Contains(original, `"deliveryConfirmed":true`) {
		t.Fatal("partial SQL delivery compacted all ticket snapshots")
	}
	ids, _ := f.rd.ScratchPending(input.ScratchScope)
	f.svc.rounds = map[string]*financeRound{}
	f.client.HDel(context.Background(), financeRoundHash, ids[0])
	f.svc.testScratchMirror = mirror
	_, response = f.call("issue", input)
	if response.Success || f.balance(42) != 98000 || f.client.HExists(context.Background(), financeRoundHash, ids[0]).Val() {
		t.Fatal("old unconfirmed purchase copy restored missing money state", response)
	}
	current, _ := f.rd.ScratchReceipt(input.ScratchScope, input.RequestID)
	if current != original {
		t.Fatal("failed recovery destroyed original receipt")
	}
}

func TestReceiptCompactionConflictDoesNotReportAlreadyDeliveredPurchaseAsFailed(t *testing.T) {
	f := newScratchFixture(t)
	input, _ := f.purchase(false)
	mirror := f.svc.testScratchMirror
	f.svc.testScratchMirror = func(round *financeRound) error {
		if err := mirror(round); err != nil {
			return err
		}
		keys := f.client.Keys(context.Background(), "lottery:settle_wallet:scratch-buy:*").Val()
		if len(keys) != 1 {
			t.Fatal(keys)
		}
		f.client.Set(context.Background(), keys[0], "concurrent receipt replacement", time.Hour)
		return nil
	}
	_, response := f.call("issue", input)
	if !response.Success || f.balance(42) != 99900 || len(f.mirrors) != 1 {
		t.Fatal("cache compaction error turned confirmed money and SQL delivery into failure", response)
	}
	original, _ := f.rd.ScratchReceipt(input.ScratchScope, input.RequestID)
	if original != "concurrent receipt replacement" {
		t.Fatal("compaction overwrote a concurrent change")
	}
}
