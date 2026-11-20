package unified

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"lottery/dao"
	"testing"
	"time"
)

func TestScratchExpiredSoldKeyNeverAllowsResaleToAnyPlayerOrCurrency(t *testing.T) {
	f := newScratchFixture(t)
	input, serial := f.purchase(false)
	if _, response := f.call("issue", input); !response.Success {
		t.Fatal(response)
	}
	claim := f.input
	claim.Serial = serial
	if _, response := f.call("reveal", claim); !response.Success {
		t.Fatal(response)
	}
	id, _ := f.rd.ScratchSerialRound(input.ScratchScope, serial)
	round := f.svc.rounds[id]
	round.Settlement.CompletedAt = time.Now().Unix() - 3601
	// Age the isolated stored fixture directly: the production save path
	// intentionally rejects rewriting an old financial completion timestamp.
	aged, _ := json.Marshal(round)
	if err := f.client.HSet(context.Background(), "lottery_finance_rounds", id, aged).Err(); err != nil {
		t.Fatal(err)
	}
	if err := f.rd.FinishScratch(input.ScratchScope, id, serial, "GGL1", round.Scratch.Win); err != nil {
		t.Fatal(err)
	}
	soldKey := "gogame:scratch:sold:" + input.ScratchScope.Market() + ":" + serial
	if f.client.Exists(context.Background(), soldKey).Val() != 0 {
		t.Fatal("old sold key was not deleted")
	}
	f.client.HDel(context.Background(), "lottery_finance_rounds", id)
	delete(f.svc.rounds, id)
	for _, scope := range []dao.ScratchScope{input.ScratchScope, {Agent: 7, GameID: 91061, UserID: 43, Currency: "CNY"}, {Agent: 7, GameID: 91061, UserID: 42, Currency: "USD"}} {
		raw, _ := json.Marshal(input)
		var retry scratchInput
		json.Unmarshal(raw, &retry)
		retry.ScratchScope = scope
		retry.RequestID = fmt.Sprintf("new-resale-%d-%s", scope.UserID, scope.Currency)
		for i := range retry.Candidates {
			for n := range retry.Candidates[i].Tickets {
				retry.Candidates[i].Tickets[n].RoundID += "-resale"
			}
		}
		if err := f.rd.SaveScratchOffers(scope, map[string]dao.ScratchOffer{retry.Refs[0]: {Serials: []string{serial}, Expires: time.Now().Unix() + 60}}); err != nil {
			t.Fatal(err)
		}
		before42, before43 := f.balance(42), f.balance(43)
		if _, response := f.call("issue", retry); response.Success {
			t.Fatal("paid serial resold", scope)
		}
		if f.balance(42) != before42 || f.balance(43) != before43 || len(f.svc.rounds) != 0 {
			t.Fatal("resale guard changed money or recreated rounds")
		}
	}
}

func TestScratchAllocationAndBookSearchExcludeDurableSalesWhenRedisMappingIsMissing(t *testing.T) {
	f := newScratchFixture(t)
	input, serial := f.purchase(false)
	if _, response := f.call("issue", input); !response.Success {
		t.Fatal(response)
	}
	f.client.Del(context.Background(), "gogame:scratch:sold:"+input.ScratchScope.Market()+":"+serial)
	f.client.Set(context.Background(), "gogame:scratch:sequence:"+input.ScratchScope.Market(), 0, 0)
	query := f.input
	query.UserID = 43
	data, response := f.call("allocate", query)
	if !response.Success || len(data["matchedSerials"].([]any)) != 0 {
		t.Fatal("allocation offered a paid serial", response, data)
	}

	f = newScratchFixture(t)
	bookInput, first := f.purchase(true)
	// One durable sold serial is enough to exclude the whole book, even for
	// a different player. No visible book or prior display offer proves stock.
	f.sold = map[string]bool{bookInput.ScratchScope.Market() + ":" + first: true}
	f.client.Set(context.Background(), "gogame:scratch:sequence:"+bookInput.ScratchScope.Market(), 0, 0)
	query = f.input
	query.UserID = 43
	query.SearchType = "BOOK_TAIL"
	query.BookTail = "789"
	data, response = f.call("search", query)
	if !response.Success {
		t.Fatal(response)
	}
	for _, ref := range data["matchedSerials"].([]any) {
		if ref == bookInput.Refs[0] {
			t.Fatal("partly sold book offered again")
		}
	}
}

func TestScratchOwnershipLookupFailureStopsBeforeChargingAndDoesNotMakeOffers(t *testing.T) {
	f := newScratchFixture(t)
	input, _ := f.purchase(false)
	f.svc.testScratchSold = func(context.Context, dao.ScratchScope, []string) (map[string]bool, error) {
		return nil, errors.New("ownership SQL offline")
	}
	if _, response := f.call("issue", input); response.Success || f.balance(42) != 100000 || len(f.svc.rounds) != 0 {
		t.Fatal("lookup failure charged player", response)
	}
	query := f.input
	query.UserID = 43
	if _, response := f.call("allocate", query); response.Success {
		t.Fatal("lookup failure returned usable offer")
	}
	offers, err := f.rd.ScratchOfferMap(query.ScratchScope)
	if err != nil || len(offers) != 0 {
		t.Fatal(offers, err)
	}
}

func TestScratchLegacyTerminalTicketRequiresOwnershipConfirmationBeforeShortening(t *testing.T) {
	for _, offline := range []bool{false, true} {
		f := newScratchFixture(t)
		input, serial := f.purchase(false)
		if _, response := f.call("issue", input); !response.Success {
			t.Fatal(response)
		}
		claim := f.input
		claim.Serial = serial
		if _, response := f.call("reveal", claim); !response.Success {
			t.Fatal(response)
		}
		id, _ := f.rd.ScratchSerialRound(input.ScratchScope, serial)
		round := f.svc.rounds[id]
		round.Scratch.SoldMirrored = false
		delete(f.sold, input.ScratchScope.Market()+":"+serial)
		f.mirrorFailure = offline
		if err := f.svc.saveRoundLocked(round); err != nil {
			t.Fatal(err)
		}
		soldKey := "gogame:scratch:sold:" + input.ScratchScope.Market() + ":" + serial
		pendingKey := "gogame:scratch:pending:" + input.ScratchScope.Key()
		f.client.Set(context.Background(), soldKey, id, 0)
		f.client.SAdd(context.Background(), pendingKey, id)
		before := f.balance(42)
		_, response := f.call("state", f.input)
		if offline {
			if response.Success || f.client.PTTL(context.Background(), soldKey).Val() != -1 || !f.client.SIsMember(context.Background(), pendingKey, id).Val() || round.Scratch.SoldMirrored {
				t.Fatal("unconfirmed legacy ownership retired", response)
			}
		} else if !response.Success || !round.Scratch.SoldMirrored || !f.sold[input.ScratchScope.Market()+":"+serial] || f.client.PTTL(context.Background(), soldKey).Val() > time.Hour {
			t.Fatal("confirmed legacy ownership not upgraded", response)
		}
		if f.balance(42) != before {
			t.Fatal("ownership upgrade replayed payout")
		}
	}
}

func TestScratchMirrorSaveFailureDoesNotLeaveFalseConfirmedFlags(t *testing.T) {
	f := newScratchFixture(t)
	input, serial := f.purchase(false)
	if _, response := f.call("issue", input); !response.Success {
		t.Fatal(response)
	}
	id, _ := f.rd.ScratchSerialRound(input.ScratchScope, serial)
	round := f.svc.rounds[id]
	round.Scratch.Mirrored = false
	round.Scratch.SoldMirrored = false
	f.client.Del(context.Background(), "lottery_finance_rounds")
	f.client.Set(context.Background(), "lottery_finance_rounds", "wrong Redis type", 0)
	if err := f.svc.mirrorScratch(context.Background(), round); err == nil || round.Scratch.Mirrored || round.Scratch.SoldMirrored {
		t.Fatal("failed snapshot save left false delivery confirmation", err)
	}
}
