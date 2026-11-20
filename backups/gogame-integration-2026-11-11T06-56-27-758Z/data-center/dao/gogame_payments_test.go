package dao

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func matchPaymentRedis(t *testing.T) (*redis.Client, MatchPaymentPlan) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	p := MatchPaymentPlan{Key: "match:7:9:42:1", UserID: 42, AgentID: 7, MatchID: 9, Gold: 100, Service: 10, Total: 110, Currency: "CNY"}
	if e := client.HSet(context.Background(), "player_42", "id", 42, "agent_id", 7, "currency_type", "CNY", "currency", 1000, "cash_revision", 0).Err(); e != nil {
		t.Fatal(e)
	}
	return client, p
}
func matchPaymentDocument(p MatchPaymentPlan, detail *matchPaymentDetail, state string) map[string]any {
	entry := map[string]any{"plan": p, "attempt": 1, "agentId": p.AgentID, "status": state}
	if detail != nil {
		paymentSet(entry, detail)
	}
	return map[string]any{"registrations": map[string]any{"42": entry}, "outbox": []any{}}
}
func matchPaymentSQL(p MatchPaymentPlan, document map[string]any, matchState int64, update func(string), failure error) []gogameSQLStep {
	raw, _ := json.Marshal(document)
	award := `[{"rank":1,"items":[{"kind":"money","amount":"50","currency":"CNY"}]}]`
	if p.Kind == "ticket" {
		award = fmt.Sprintf(`[{"rank":"1","items":[{"kind":"ticket","amount":"%d","ticketId":"%d"}]}]`, p.Amount, p.TicketID)
	}
	start := time.Now().Add(time.Hour).UnixMilli()
	if matchState == 2 {
		start = time.Now().Add(-10 * time.Second).UnixMilli()
	}
	return []gogameSQLStep{
		{contains: "SELECT * FROM gg_match WHERE id=? FOR UPDATE", columns: []string{"id", "agent_id", "state", "start_time", "currency", "register_gold", "service_gold", "register_ticket", "ticket_id", "refund_service_fee", "award_json", "delay_register_level", "delay_register_time"}, values: [][]driver.Value{{p.MatchID, int64(p.AgentID), matchState, start, p.Currency, p.Gold, p.Service, p.TicketCount, p.TicketID, int64(0), award, int64(2), int64(120)}}},
		{contains: "SELECT payload FROM gg_match_runtime", columns: []string{"payload"}, values: [][]driver.Value{{string(raw)}}},
		{contains: "INSERT INTO gg_match_runtime", err: failure, check: func(args []driver.NamedValue) {
			if update != nil {
				update(args[1].Value.(string))
			}
		}},
	}
}

func matchTicketMetadata() []gogameSQLStep {
	step := gogameSQLStep{contains: "SELECT @@server_uuid", columns: []string{"server_id", "db_name"}, values: [][]driver.Value{{"11111111-1111-1111-1111-111111111111", "players"}}}
	return []gogameSQLStep{step, step}
}
func TestMatchTicketPreparationAndDeclineRestoreAreAtomicWithRuntime(t *testing.T) {
	client, p := matchPaymentRedis(t)
	p.TicketCount = 1
	p.TicketID = 5
	ctx := context.Background()
	client.Set(ctx, "gogame:cash-carry:7:42:CNY", `{"token":"seat","tableKey":"7:table:1","revision":1,"amount":950}`, time.Minute)
	prepared := &matchPaymentDetail{Fingerprint: paymentHash(p), Status: "prepared", TicketIDs: []int64{81}}
	first := matchPaymentSQL(p, matchPaymentDocument(p, nil, "pending"), 1, nil, nil)
	second := matchPaymentSQL(p, matchPaymentDocument(p, prepared, "pending"), 1, func(s string) {
		if !strings.Contains(s, `"status":"declined"`) {
			t.Fatal("definitive decline was not recorded")
		}
	}, nil)
	steps := append(matchTicketMetadata(), first[:2]...)
	steps = append(steps, gogameSQLStep{contains: "SELECT match_id FROM gg_ticket", columns: []string{"match_id"}, values: [][]driver.Value{{int64(0)}}},
		gogameSQLStep{contains: "SELECT id FROM `players`.`gg_ticket_hold`", columns: []string{"id"}, values: [][]driver.Value{{int64(81)}}}, gogameSQLStep{contains: "SET is_use=1"})
	steps = append(steps, first[2])
	steps = append(steps, second[:2]...)
	steps = append(steps, gogameSQLStep{contains: "SELECT owner,is_use,use_match_id", columns: []string{"owner", "is_use", "use_match_id"}, values: [][]driver.Value{{int64(42), int64(1), int64(9)}}}, gogameSQLStep{contains: "SET is_use=0"}, second[2])
	db, script := gogameTestDB(t, steps...)
	if _, e := (&DBDao{db: db, manager: db}).matchPayment(ctx, client, MatchPaymentInput{Operation: "charge", Plan: p}); status.Code(e) != codes.FailedPrecondition {
		t.Fatal("expected definitive decline", e)
	}
	if len(script.steps) != 0 || script.log[len(script.log)-1] != "commit" || client.HGet(ctx, "player_42", "currency").Val() != "1000" {
		t.Fatal("ticket decline did not commit the restoration with unchanged money")
	}
}
func TestMatchTicketsRejectDifferentServersAndAwardInventoryIsNotDuplicated(t *testing.T) {
	client, p := matchPaymentRedis(t)
	p.Key = "match:7:9:award:42:0"
	p.Kind = "ticket"
	p.Amount = 2
	p.TicketID = 5
	p.Gold = 0
	p.Service = 0
	p.Total = 0
	metadata := matchTicketMetadata()
	metadata[1].values = [][]driver.Value{{"22222222-2222-2222-2222-222222222222", "players"}}
	db, _ := gogameTestDB(t, metadata...)
	if _, e := (&DBDao{db: db, manager: db}).matchPayment(context.Background(), client, MatchPaymentInput{Operation: "award", Plan: p}); status.Code(e) != codes.Unavailable {
		t.Fatal("different MySQL servers were treated as one atomic transaction", e)
	}
	document := map[string]any{"registrations": map[string]any{"42": map[string]any{"status": "winner", "rank": 1, "agentId": 7}}, "outbox": []any{}, "prizes": []any{map[string]any{"plan": p, "status": "pending"}}}
	var saved string
	steps := matchPaymentSQL(p, document, 3, func(s string) { saved = s }, nil)
	all := append(matchTicketMetadata(), steps[:2]...)
	all = append(all, gogameSQLStep{contains: "SELECT remain FROM gg_ticket", columns: []string{"remain"}, values: [][]driver.Value{{int64(2)}}}, gogameSQLStep{contains: "SET remain=remain-?"}, gogameSQLStep{contains: "INSERT INTO `players`.`gg_ticket_hold`"}, gogameSQLStep{contains: "INSERT INTO `players`.`gg_ticket_hold`"}, steps[2])
	db, script := gogameTestDB(t, all...)
	if _, e := (&DBDao{db: db, manager: db}).matchPayment(context.Background(), client, MatchPaymentInput{Operation: "award", Plan: p}); e != nil {
		t.Fatal(e)
	}
	if len(script.steps) != 0 || script.log[len(script.log)-1] != "commit" {
		t.Fatal("ticket issuance and original prize receipt were not committed together")
	}
	if e := DecodeGogameJSON(saved, &document); e != nil {
		t.Fatal(e)
	}
	all = append(matchTicketMetadata(), matchPaymentSQL(p, document, 3, nil, nil)...)
	db, script = gogameTestDB(t, all...)
	if _, e := (&DBDao{db: db, manager: db}).matchPayment(context.Background(), client, MatchPaymentInput{Operation: "award", Plan: p}); e != nil {
		t.Fatal(e)
	}
	if len(script.steps) != 0 || client.HGet(context.Background(), "player_42", "currency").Val() != "1000" {
		t.Fatal("repeated ticket award changed inventory or cash")
	}
}
func TestMatchPaymentCashIsAtomicBoundedAndIdempotent(t *testing.T) {
	client, p := matchPaymentRedis(t)
	ctx := context.Background()
	input := MatchPaymentInput{Operation: "charge", Plan: p}
	receipt := map[string]any{"key": p.Key, "paidGold": p.Gold, "paidService": p.Service, "currency": p.Currency}
	client.Set(ctx, "gogame:cash-carry:7:42:CNY", `{"token":"seat","tableKey":"7:table:1","revision":1,"amount":950}`, time.Minute)
	if _, e := MatchPaymentCash(ctx, client, input, "charge", "debit", 110, receipt); status.Code(e) != codes.FailedPrecondition {
		t.Fatal("spent reserved seat chips", e)
	}
	if client.HGet(ctx, "player_42", "currency").Val() != "1000" || client.Exists(ctx, paymentMarker(p, "charge")).Val() != 0 {
		t.Fatal("rejected debit wrote money or receipt")
	}
	client.Del(ctx, "gogame:cash-carry:7:42:CNY")
	if _, e := MatchPaymentCash(ctx, client, input, "charge", "debit", 110, receipt); e != nil {
		t.Fatal(e)
	}
	client.HIncrBy(ctx, "player_42", "cash_revision", 3)
	if _, e := MatchPaymentCash(ctx, client, input, "charge", "debit", 110, receipt); e != nil {
		t.Fatal(e)
	}
	if client.HGet(ctx, "player_42", "currency").Val() != "890" {
		t.Fatal("repeated payment debited twice")
	}
	if client.TTL(ctx, paymentMarker(p, "charge")).Val() != -1 {
		t.Fatal("unconfirmed SQL payment marker can expire")
	}
	changed := input
	changed.Plan.Gold = 101
	changed.Plan.Total = 111
	if _, e := MatchPaymentCash(ctx, client, changed, "charge", "debit", 111, receipt); status.Code(e) != codes.FailedPrecondition {
		t.Fatal("changed plan reused marker", e)
	}
}
func TestMatchPaymentRecoversDebitAfterSQLFailureAndRefundsBothFees(t *testing.T) {
	client, p := matchPaymentRedis(t)
	input := MatchPaymentInput{Operation: "charge", Plan: p}
	ctx := context.Background()
	prepared := &matchPaymentDetail{Fingerprint: paymentHash(p), Status: "prepared", TicketIDs: []int64{}}
	initial := matchPaymentDocument(p, nil, "pending")
	pending := matchPaymentDocument(p, prepared, "pending")
	steps := append(matchPaymentSQL(p, initial, 1, nil, nil), matchPaymentSQL(p, pending, 1, nil, errors.New("mysql commit failed"))...)
	db, script := gogameTestDB(t, steps...)
	if _, e := (&DBDao{manager: db}).matchPayment(ctx, client, input); e == nil {
		t.Fatal("SQL failure reported success")
	}
	if client.HGet(ctx, "player_42", "currency").Val() != "890" || script.log[len(script.log)-1] != "rollback" {
		t.Fatal("fault did not leave the confirmed debit and a rolled-back receipt")
	}
	var final string
	steps = append(matchPaymentSQL(p, pending, 1, nil, nil), matchPaymentSQL(p, pending, 1, func(s string) { final = s }, nil)...)
	db, _ = gogameTestDB(t, steps...)
	value, e := (&DBDao{manager: db}).matchPayment(ctx, client, input)
	if e != nil {
		t.Fatal(e)
	}
	if value.(map[string]any)["paidGold"] != float64(100) || client.HGet(ctx, "player_42", "currency").Val() != "890" {
		t.Fatal("retry recharged instead of confirming the original debit")
	}
	var committed map[string]any
	if e = DecodeGogameJSON(final, &committed); e != nil {
		t.Fatal(e)
	}
	paymentMap(paymentMap(committed["registrations"])["42"])["status"] = "refund_pending"
	db, _ = gogameTestDB(t, matchPaymentSQL(p, committed, 4, func(s string) { final = s }, nil)...)
	refund := MatchPaymentInput{Operation: "refund", Plan: p, IncludeService: true}
	value, e = (&DBDao{manager: db}).matchPayment(ctx, client, refund)
	if e != nil {
		t.Fatal(e)
	}
	if fmt.Sprint(value.(map[string]any)["refunded"]) != "110" || client.HGet(ctx, "player_42", "currency").Val() != "1000" {
		t.Fatal("refund did not return both fees")
	}
	if e = DecodeGogameJSON(final, &committed); e != nil {
		t.Fatal(e)
	}
	db, _ = gogameTestDB(t, matchPaymentSQL(p, committed, 4, nil, nil)...)
	if _, e = (&DBDao{manager: db}).matchPayment(ctx, client, refund); e != nil || client.HGet(ctx, "player_42", "currency").Val() != "1000" {
		t.Fatal("refund repeated", e)
	}
}
func TestMatchPaymentRefusesForeignPlanBeforeCashAndClosesUnchargedRefund(t *testing.T) {
	client, p := matchPaymentRedis(t)
	original := matchPaymentDocument(p, nil, "refund_pending")
	bad := p
	bad.Gold = 200
	bad.Total = 210
	steps := matchPaymentSQL(p, original, 4, nil, nil)[:2]
	db, _ := gogameTestDB(t, steps...)
	if _, e := (&DBDao{manager: db}).matchPayment(context.Background(), client, MatchPaymentInput{Operation: "charge", Plan: bad}); e == nil {
		t.Fatal("foreign stored plan accepted")
	}
	var saved string
	db, _ = gogameTestDB(t, matchPaymentSQL(p, original, 4, func(s string) { saved = s }, nil)...)
	value, e := (&DBDao{manager: db}).matchPayment(context.Background(), client, MatchPaymentInput{Operation: "refund", Plan: p, IncludeService: true})
	if e != nil || value.(map[string]any)["uncharged"] != true || client.HGet(context.Background(), "player_42", "currency").Val() != "1000" {
		t.Fatal(value, e)
	}
	if !strings.Contains(saved, `"status":"refunded"`) {
		t.Fatal("uncharged refund did not close the payment attempt")
	}
}
func TestMatchMoneyAwardChecksLockedStandingsAndSurvivesLostSQLReceipt(t *testing.T) {
	client, p := matchPaymentRedis(t)
	p.Key = "match:7:9:award:42:0"
	p.Gold = 0
	p.Service = 0
	p.Total = 0
	p.Kind = "money"
	p.Amount = 50
	document := map[string]any{"registrations": map[string]any{"42": map[string]any{"status": "winner", "rank": 1, "agentId": 7}}, "outbox": []any{}, "prizes": []any{map[string]any{"plan": p, "status": "pending"}}}
	db, _ := gogameTestDB(t, matchPaymentSQL(p, document, 3, nil, errors.New("lost SQL receipt"))...)
	input := MatchPaymentInput{Operation: "award", Plan: p}
	if _, e := (&DBDao{manager: db}).matchPayment(context.Background(), client, input); e == nil {
		t.Fatal("SQL failure reported as delivered")
	}
	var saved string
	db, _ = gogameTestDB(t, matchPaymentSQL(p, document, 3, func(s string) { saved = s }, nil)...)
	if _, e := (&DBDao{manager: db}).matchPayment(context.Background(), client, input); e != nil {
		t.Fatal(e)
	}
	if client.HGet(context.Background(), "player_42", "currency").Val() != "1050" || !strings.Contains(saved, "authorityReceipt") {
		t.Fatal("award repeated or lacked a durable receipt")
	}
	// Matching a forged pending plan is insufficient: the locked award config wins.
	wrong := p
	wrong.Amount = 51
	document["prizes"] = []any{map[string]any{"plan": wrong, "status": "pending"}}
	db, _ = gogameTestDB(t, matchPaymentSQL(p, document, 3, nil, nil)[:2]...)
	if _, e := (&DBDao{manager: db}).matchPayment(context.Background(), client, MatchPaymentInput{Operation: "award", Plan: wrong}); e == nil {
		t.Fatal("pending prize overrode locked rank reward")
	}
}

func TestMatchPaymentQueryUsesSQLReceiptWithoutWalletOrTicketDatabase(t *testing.T) {
	_, p := matchPaymentRedis(t)
	p.TicketID = 5
	p.TicketCount = 1
	receipt := map[string]any{"key": p.Key, "paidGold": 100, "paidService": 10, "currency": "CNY", "paidTicketCount": 1, "ticketId": 5}
	detail := &matchPaymentDetail{Fingerprint: paymentHash(p), Status: "charged", TicketIDs: []int64{81}, Receipt: receipt}
	db, script := gogameTestDB(t, matchPaymentSQL(p, matchPaymentDocument(p, detail, "registered"), 1, nil, nil)...)
	value, e := (&DBDao{manager: db}).matchPayment(context.Background(), nil, MatchPaymentInput{Operation: "query", Plan: p})
	if e != nil || value.(map[string]any)["status"] != "charged" || len(script.steps) != 0 {
		t.Fatal("query did not return the existing SQL receipt", value, e)
	}
}

func TestLateRegistrationFundsEnforceConfiguredWindowAndClock(t *testing.T) {
	now := time.Now().UnixMilli()
	row := map[string]any{"state": int64(2), "start_time": now - 10000, "delay_register_level": int64(2), "delay_register_time": int64(120)}
	entry := map[string]any{"status": "pending", "late": true}
	state := map[string]any{"clock": map[string]any{"currentBlindLv": int64(2), "nextBlindAt": now + 10000}}
	if !paymentRegistrationOpen(row, entry, state) {
		t.Fatal("eligible late payment denied")
	}
	state["clock"] = map[string]any{"currentBlindLv": int64(1), "nextBlindAt": now - 1}
	if paymentRegistrationOpen(row, entry, state) {
		t.Fatal("stale level snapshot authorized a late payment")
	}
	state["clock"] = map[string]any{"currentBlindLv": int64(3), "nextBlindAt": now + 10000}
	if paymentRegistrationOpen(row, entry, state) {
		t.Fatal("payment passed its level cutoff")
	}
	state["clock"] = map[string]any{"currentBlindLv": int64(2), "nextBlindAt": now + 10000}
	row["start_time"] = now - 120001
	if paymentRegistrationOpen(row, entry, state) {
		t.Fatal("payment passed its time cutoff")
	}
	row["start_time"] = now - 10000
	entry["late"] = false
	if paymentRegistrationOpen(row, entry, state) {
		t.Fatal("ordinary registration charged after start")
	}
	entry["late"] = true
	state["invalidated"] = map[string]any{"reason": "GAME_STATE_LOST"}
	if paymentRegistrationOpen(row, entry, state) {
		t.Fatal("invalid match accepted a late payment")
	}
}

func TestLateRegistrationDebitRechecksWindowAfterPreparation(t *testing.T) {
	for _, expired := range []bool{false, true} {
		t.Run(fmt.Sprint(expired), func(t *testing.T) {
			client, p := matchPaymentRedis(t)
			ctx := context.Background()
			initial := matchPaymentDocument(p, nil, "pending")
			paymentMap(paymentMap(initial["registrations"])["42"])["late"] = true
			initial["clock"] = map[string]any{"currentBlindLv": int64(1), "nextBlindAt": time.Now().Add(time.Minute).UnixMilli()}
			prepared := &matchPaymentDetail{Fingerprint: paymentHash(p), Status: "prepared", TicketIDs: []int64{}}
			pending := matchPaymentDocument(p, prepared, "pending")
			paymentMap(paymentMap(pending["registrations"])["42"])["late"] = true
			pending["clock"] = initial["clock"]
			second := matchPaymentSQL(p, pending, 2, nil, nil)
			if expired {
				second[0].values[0][3] = time.Now().Add(-121 * time.Second).UnixMilli()
			}
			steps := append(matchPaymentSQL(p, initial, 2, nil, nil), second...)
			db, _ := gogameTestDB(t, steps...)
			_, err := (&DBDao{manager: db}).matchPayment(ctx, client, MatchPaymentInput{Operation: "charge", Plan: p})
			if expired {
				if status.Code(err) != codes.FailedPrecondition || client.HGet(ctx, "player_42", "currency").Val() != "1000" {
					t.Fatal("expired late registration spent money", err)
				}
			} else if err != nil || client.HGet(ctx, "player_42", "currency").Val() != "890" {
				t.Fatal("eligible late registration did not debit exactly once", err)
			}
		})
	}
}
