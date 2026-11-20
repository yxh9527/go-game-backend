package dao

import (
	"encoding/json"
	"fmt"
	"testing"
)

func seatNetFixture(t *testing.T) (*RedisDao, SeatNetBatch, string, string) {
	t.Helper()
	rd, mr := currencyTestRedis(t)
	for _, uid := range []uint32{42, 43} {
		mr.HSet(fmt.Sprintf("player_%d", uid), "id", fmt.Sprint(uid), "agent_id", "7", "currency_type", "CNY", "currency", "100000", "cash_revision", "3")
		mr.Set(fmt.Sprintf("gogame:cash-carry:7:%d:CNY", uid), fmt.Sprintf(`{"amount":10000,"token":"seat-%d","tableKey":"7:niuniu:1","revision":5}`, uid))
	}
	before := `{"state":"BETTING"}`
	mr.HSet("lottery_finance_rounds", "net-round", before)
	batch := SeatNetBatch{ID: "net-settlement", Accounts: []SeatNetAccount{{42, 7, "CNY", 3}, {43, 7, "CNY", 3}}, Steps: []SeatNetStep{{"a", 1, "carryDelta", -123, "seat-42", "7:niuniu:1", 5}, {"b", 2, "carryDelta", 123, "seat-43", "7:niuniu:1", 5}}}
	terminal := `{"state":"SETTLED","settlement":{"results":[{"userId":42},{"userId":43}]}}`
	return rd, batch, before, terminal
}
func TestSeatNetAtomicBatchOnlyTransfersTheNetAmountOnce(t *testing.T) {
	rd, batch, before, terminal := seatNetFixture(t)
	first, err := rd.SettleSeatNet("net-round", "fingerprint", before, terminal, batch)
	if err != nil {
		t.Fatal(err)
	}
	second, err := rd.SettleSeatNet("net-round", "fingerprint", before, terminal, batch)
	if err != nil || string(second) != string(first) {
		t.Fatalf("retry: %s %v", second, err)
	}
	for i, uid := range []uint32{42, 43} {
		balance, _ := rd.GetPlayerCurrency(uid)
		expected := int64(99877)
		if i == 1 {
			expected = 100123
		}
		if balance != expected {
			t.Fatalf("user %d balance=%d", uid, balance)
		}
		revision, _ := rd.HGet(fmt.Sprintf("player_%d", uid), "cash_revision")
		if revision != "4" {
			t.Fatal("revision advanced more than once")
		}
		raw, _ := rd.Get(fmt.Sprintf("gogame:cash-carry:7:%d:CNY", uid))
		var carry struct{ Amount, Revision int64 }
		if json.Unmarshal([]byte(raw), &carry) != nil || carry.Revision != 6 || carry.Amount != expected-90000 {
			t.Fatalf("carry %s", raw)
		}
	}
	raw, _ := rd.HGet("lottery_finance_rounds", "net-round")
	var round struct {
		State      string
		Settlement struct{ Results []struct{ Currency string } }
	}
	if json.Unmarshal([]byte(raw), &round) != nil || round.State != "SETTLED" || round.Settlement.Results[0].Currency != "998.77" || round.Settlement.Results[1].Currency != "1001.23" {
		t.Fatalf("terminal=%s", raw)
	}
	if _, err = rd.SettleSeatNet("net-round", "changed", before, terminal, batch); err == nil {
		t.Fatal("changed settlement accepted")
	}
}
func TestSeatNetLastAccountFailureDoesNotCreditOrDebitAnyOtherPlayer(t *testing.T) {
	for _, failure := range []string{"cash revision", "carry revision", "negative stack", "wrong token"} {
		t.Run(failure, func(t *testing.T) {
			rd, batch, before, terminal := seatNetFixture(t)
			switch failure {
			case "cash revision":
				batch.Accounts[1].Revision = 2
			case "carry revision":
				batch.Steps[1].Revision = 4
			case "negative stack":
				batch.Steps[1].Amount = -10001
			case "wrong token":
				batch.Steps[1].Token = "wrong"
			}
			if _, err := rd.SettleSeatNet("net-round", "failure", before, terminal, batch); err == nil {
				t.Fatal("invalid last account accepted")
			}
			for _, uid := range []uint32{42, 43} {
				balance, _ := rd.GetPlayerCurrency(uid)
				if balance != 100000 {
					t.Fatalf("partial transfer to %d", uid)
				}
			}
			raw, _ := rd.HGet("lottery_finance_rounds", "net-round")
			if raw != before {
				t.Fatal("failed settlement changed terminal state")
			}
		})
	}
}
