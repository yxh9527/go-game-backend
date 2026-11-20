package dao

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"github.com/shopspring/decimal"
	"strings"
	"testing"
)

func TestTicketCreditAllocationUsesTicketsPerCurrencyUnitAndNeverOverpays(t *testing.T) {
	for _, v := range []struct {
		total, available, capacity int64
		rate                       string
		count, credit              int64
	}{
		{110, 20, 1000, "100", 20, 20}, {110, 500, 1000, "100", 110, 110}, {11, 10, 1000, "25", 2, 8},
		{1, 1, 1000, "200", 0, 0}, {1, 2, 1000, "200", 2, 1}, {110, 500, 0, "100", 0, 0},
		{9007199254740991, 1000, 1000, "0.00000001", 1000, 10000000000000},
	} {
		count, credit, e := ticketCreditAllocation(v.total, v.available, v.capacity, decimal.RequireFromString(v.rate))
		if e != nil || count != v.count || credit != v.credit {
			t.Fatal(v, count, credit, e)
		}
	}
	for _, rate := range []string{"0", "-1", "0.000000001", "9007199254740992"} {
		if _, _, e := ticketCreditAllocation(110, 20, 1000, decimal.RequireFromString(rate)); e == nil {
			t.Fatal("invalid rate accepted", rate)
		}
	}
}

func creditPlan(t *testing.T) (MatchPaymentPlan, []int64, [][]driver.Value) {
	t.Helper()
	p := MatchPaymentPlan{Key: "match:7:9:42:1", UserID: 42, AgentID: 7, MatchID: 9, Gold: 100, Service: 10, Total: 110, Currency: "CNY", TicketID: 5, TicketCount: 1, TicketCredit: true}
	ids := []int64{}
	rows := [][]driver.Value{}
	for i := int64(1); i <= 21; i++ {
		ids = append(ids, i)
		rows = append(rows, []driver.Value{i})
	}
	return p, ids, rows
}
func creditPreparation(t *testing.T, p MatchPaymentPlan, rows [][]driver.Value, save func(string)) []gogameSQLStep {
	document := matchPaymentDocument(p, nil, "pending")
	base := matchPaymentSQL(p, document, 1, save, nil)
	steps := append(matchTicketMetadata(), base[:2]...)
	steps = append(steps, gogameSQLStep{contains: "SELECT rate FROM gg_ticket_currency_rate", columns: []string{"rate"}, values: [][]driver.Value{{"100"}}},
		gogameSQLStep{contains: "LIMIT 1000 FOR UPDATE", columns: []string{"id"}, values: rows},
		gogameSQLStep{contains: "SELECT match_id FROM gg_ticket", columns: []string{"match_id"}, values: [][]driver.Value{{int64(0)}}},
		gogameSQLStep{contains: "LIMIT ? FOR UPDATE", columns: []string{"id"}, values: rows})
	for range rows {
		steps = append(steps, gogameSQLStep{contains: "SET is_use=1"})
	}
	return append(steps, base[2])
}
func creditPrepared(p MatchPaymentPlan, ids []int64) map[string]any {
	return matchPaymentDocument(p, &matchPaymentDetail{Fingerprint: paymentHash(p), Status: "prepared", TicketIDs: ids, CreditLocked: true, CreditTickets: 20, CreditAmount: 20, TicketRate: "100", CashGold: 80, CashService: 10}, "pending")
}
func ticketRestoreSteps(ids []int64) []gogameSQLStep {
	steps := []gogameSQLStep{}
	for range ids {
		steps = append(steps, gogameSQLStep{contains: "SELECT owner,is_use,use_match_id", columns: []string{"owner", "is_use", "use_match_id"}, values: [][]driver.Value{{int64(42), int64(1), int64(9)}}}, gogameSQLStep{contains: "SET is_use=0"})
	}
	return steps
}
func TestTicketCreditLocksOriginalRateSurvivesLostReceiptAndRestoresOriginalTickets(t *testing.T) {
	client, _ := matchPaymentRedis(t)
	p, ids, rows := creditPlan(t)
	ctx := context.Background()
	var prepared, charged string
	steps := append(creditPreparation(t, p, rows, func(s string) { prepared = s }), matchPaymentSQL(p, creditPrepared(p, ids), 1, nil, errors.New("SQL save failed"))...)
	db, _ := gogameTestDB(t, steps...)
	if _, e := (&DBDao{manager: db, db: db}).matchPayment(ctx, client, MatchPaymentInput{Operation: "charge", Plan: p}); e == nil {
		t.Fatal("lost SQL receipt reported as success")
	}
	if client.HGet(ctx, "player_42", "currency").Val() != "910" || !strings.Contains(prepared, `"ticketRate":"100"`) {
		t.Fatal("cash debit or locked ticket rate incorrect")
	}
	// Retrying a prepared credit plan performs no rate or inventory selection.
	steps = append(matchTicketMetadata(), matchPaymentSQL(p, creditPrepared(p, ids), 1, nil, nil)...)
	steps = append(steps, matchPaymentSQL(p, creditPrepared(p, ids), 1, func(s string) { charged = s }, nil)...)
	db, _ = gogameTestDB(t, steps...)
	value, e := (&DBDao{manager: db, db: db}).matchPayment(ctx, client, MatchPaymentInput{Operation: "charge", Plan: p})
	if e != nil {
		t.Fatal(e)
	}
	receipt := value.(map[string]any)
	if receipt["paidGold"] != float64(80) || receipt["paidService"] != float64(10) || receipt["paidTicketCount"] != float64(21) || client.HGet(ctx, "player_42", "currency").Val() != "910" {
		t.Fatal("retry redrew or recharged the original credit", value)
	}
	var document map[string]any
	if e = DecodeGogameJSON(charged, &document); e != nil {
		t.Fatal(e)
	}
	paymentMap(paymentMap(document["registrations"])["42"])["status"] = "refund_pending"
	base := matchPaymentSQL(p, document, 4, func(s string) { charged = s }, nil)
	steps = append(matchTicketMetadata(), base[:2]...)
	steps = append(steps, ticketRestoreSteps(ids)...)
	steps = append(steps, base[2])
	db, script := gogameTestDB(t, steps...)
	value, e = (&DBDao{manager: db, db: db}).matchPayment(ctx, client, MatchPaymentInput{Operation: "refund", Plan: p, IncludeService: true})
	if e != nil || len(script.steps) != 0 {
		t.Fatal("original ticket refund failed", e)
	}
	if gogameNumber(value.(map[string]any)["refunded"]) != 90 || client.HGet(ctx, "player_42", "currency").Val() != "1000" {
		t.Fatal("refund cashed out ticket value or omitted paid cash", value)
	}
	DecodeGogameJSON(charged, &document)
	steps = append(matchTicketMetadata(), matchPaymentSQL(p, document, 4, nil, nil)...)
	db, _ = gogameTestDB(t, steps...)
	if _, e = (&DBDao{manager: db, db: db}).matchPayment(ctx, client, MatchPaymentInput{Operation: "refund", Plan: p, IncludeService: true}); e != nil || client.HGet(ctx, "player_42", "currency").Val() != "1000" {
		t.Fatal("repeated refund changed money or restored tickets twice", e)
	}
}
func TestTicketCreditInsufficientCashRestoresPreparedTicketsWithoutDebit(t *testing.T) {
	client, _ := matchPaymentRedis(t)
	p, ids, rows := creditPlan(t)
	ctx := context.Background()
	client.HSet(ctx, "player_42", "currency", 80)
	base := matchPaymentSQL(p, creditPrepared(p, ids), 1, func(s string) {
		if !strings.Contains(s, `"status":"declined"`) {
			t.Fatal("failed cash payment did not close original ticket preparation")
		}
	}, nil)
	steps := creditPreparation(t, p, rows, nil)
	steps = append(steps, base[:2]...)
	steps = append(steps, ticketRestoreSteps(ids)...)
	steps = append(steps, base[2])
	db, script := gogameTestDB(t, steps...)
	if _, e := (&DBDao{manager: db, db: db}).matchPayment(ctx, client, MatchPaymentInput{Operation: "charge", Plan: p}); e == nil {
		t.Fatal("insufficient cash accepted")
	}
	if len(script.steps) != 0 || client.HGet(ctx, "player_42", "currency").Val() != "80" || client.Exists(ctx, paymentMarker(p, "charge")).Val() != 0 {
		t.Fatal("cash decline changed balance or skipped ticket restoration")
	}
}
func TestTournamentTicketsReadVerifiesOwnerAndUsesTheConfiguredRate(t *testing.T) {
	client, _ := matchPaymentRedis(t)
	ctx := context.Background()
	steps := append(matchTicketMetadata(), gogameSQLStep{contains: "SELECT rate FROM gg_ticket_currency_rate", columns: []string{"rate"}, values: [][]driver.Value{{"100"}}}, gogameSQLStep{contains: "COUNT(h.id)", columns: []string{"id", "name", "num"}, values: [][]driver.Value{{int64(5), "ticket", int64(20)}}})
	db, _ := gogameTestDB(t, steps...)
	value, e := (&DBDao{manager: db, db: db}).tournamentTickets(ctx, client, CashAccount{UserID: 42, AgentID: 7, Currency: "CNY"})
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(value)
	if !strings.Contains(string(raw), `"ticketValue":1`) {
		t.Fatal("one ticket was not valued at one score", value)
	}
	db, _ = gogameTestDB(t)
	if _, e = (&DBDao{manager: db, db: db}).tournamentTickets(ctx, client, CashAccount{UserID: 42, AgentID: 8, Currency: "CNY"}); e == nil {
		t.Fatal("foreign agent queried ticket inventory")
	}
}
