package dao

import (
	"context"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCashLargeAtomicBatchAndCapacity(t *testing.T) {
	for _, count := range []int{65, 1000, cashMaxAccounts} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			client, server, _ := cashFixture(t)
			req := CashRequest{ID: "large-atomic"}
			for i := 1; i <= count; i++ {
				uid := uint32(10000 + i)
				key := fmt.Sprintf("player_%d", uid)
				server.HSet(key, "id", fmt.Sprint(uid), "agent_id", "7", "currency_type", "CNY", "currency", "1000")
				req.Accounts = append(req.Accounts, CashAccount{UserID: uid, AgentID: 7, Currency: "CNY"})
				req.Steps = append(req.Steps, CashStep{ID: fmt.Sprintf("credit-%d", i), Kind: "credit", Account: i, Amount: 200})
			}
			bad := req
			bad.Accounts = append([]CashAccount(nil), req.Accounts...)
			bad.Accounts[len(bad.Accounts)-1].Revision = 1
			if _, err := CashApply(context.Background(), client, bad); status.Code(err) != codes.FailedPrecondition {
				t.Fatal("bad last account was not rejected", err)
			}
			for _, a := range req.Accounts {
				if server.HGet(fmt.Sprintf("player_%d", a.UserID), "currency") != "1000" {
					t.Fatal("partial commit on rejected batch", a.UserID)
				}
			}
			if _, err := CashApply(context.Background(), client, req); err != nil {
				t.Fatal(err)
			}
			for _, a := range req.Accounts {
				key := fmt.Sprintf("player_%d", a.UserID)
				if server.HGet(key, "currency") != "1200" || server.HGet(key, "cash_revision") != "1" {
					t.Fatal("missing or duplicate payout", a.UserID)
				}
			}
			if _, err := CashApply(context.Background(), client, req); status.Code(err) != codes.FailedPrecondition {
				t.Fatal("duplicate old-version batch accepted", err)
			}
		})
	}
}

func TestCashOversizedBatchIsRejectedBeforeMutation(t *testing.T) {
	client, _, a := cashFixture(t)
	accounts := make([]CashAccount, cashMaxAccounts+1)
	for i := range accounts {
		accounts[i] = CashAccount{UserID: uint32(i + 1000), AgentID: 7, Currency: "CNY"}
	}
	req := CashRequest{ID: "too-many-accounts", Accounts: accounts, Steps: []CashStep{{ID: "step", Account: 1, Kind: "credit", Amount: 1}}}
	if _, err := CashApply(context.Background(), client, req); status.Code(err) != codes.InvalidArgument {
		t.Fatal(err)
	}
	req.Accounts = []CashAccount{a}
	req.Steps = make([]CashStep, cashMaxSteps+1)
	for i := range req.Steps {
		req.Steps[i] = CashStep{ID: fmt.Sprintf("step-%d", i), Account: 1, Kind: "credit", Amount: 1}
	}
	if _, err := CashApply(context.Background(), client, req); status.Code(err) != codes.InvalidArgument {
		t.Fatal(err)
	}
}

func TestCashMaximumStepBatchRemainsAtomic(t *testing.T) {
	client, server, a := cashFixture(t)
	req := CashRequest{ID: "maximum-steps", Accounts: []CashAccount{a}}
	for i := 0; i < cashMaxSteps; i++ {
		req.Steps = append(req.Steps, CashStep{ID: fmt.Sprintf("step-%d", i), Account: 1, Kind: "credit", Amount: 1})
	}
	if _, err := CashApply(context.Background(), client, req); err != nil {
		t.Fatal(err)
	}
	if got := server.HGet("player_42", "currency"); got != fmt.Sprint(1000+cashMaxSteps) {
		t.Fatal("incorrect maximum-step batch payout", got)
	}
}
