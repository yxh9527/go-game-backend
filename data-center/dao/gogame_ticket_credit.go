package dao

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// The configured rate is tickets per one display currency unit (100 score).
// At rate=100, one ticket offsets exactly one score: 100 tickets offset 1.00.
// Keep decimal arithmetic exact until the final whole-score truncation.
func ticketCreditAllocation(total, available, capacity int64, rate decimal.Decimal) (int64, int64, error) {
	if total < 0 || total > cashMax || available < 0 || capacity < 0 || capacity > 1000 || !rate.IsPositive() || rate.Exponent() < -8 || rate.GreaterThan(decimal.NewFromInt(cashMax)) {
		return 0, 0, paymentInvalid("invalid ticket credit configuration")
	}
	cap := decimal.NewFromInt(total).Mul(rate).Div(decimal.NewFromInt(100)).Floor()
	count := available
	if count > capacity {
		count = capacity
	}
	if cap.LessThan(decimal.NewFromInt(count)) {
		count = cap.IntPart()
	}
	if count <= 0 {
		return 0, 0, nil
	}
	credit := decimal.NewFromInt(count).Mul(decimal.NewFromInt(100)).Div(rate).Floor()
	if !credit.IsPositive() {
		return 0, 0, nil
	}
	if credit.GreaterThan(decimal.NewFromInt(total)) {
		return 0, 0, paymentInvalid("ticket credit exceeds fees")
	}
	return count, credit.IntPart(), nil
}

func (dd *DBDao) gogameTournamentTickets(ctx context.Context, payload string) (any, error) {
	var account CashAccount
	if err := DecodeGogameJSON(payload, &account); err != nil {
		return nil, err
	}
	rd := RedisIns()
	if rd == nil || rd.Client == nil {
		return nil, status.Error(codes.Unavailable, "ticket account verification unavailable")
	}
	return dd.tournamentTickets(ctx, rd.Client, account)
}
func (dd *DBDao) tournamentTickets(ctx context.Context, client redis.UniversalClient, account CashAccount) (any, error) {
	if _, err := CashRead(ctx, client, account); err != nil {
		return nil, err
	}
	table, err := dd.matchTicketTable(ctx)
	if err != nil {
		return nil, err
	}
	rates, err := gogameRows(ctx, dd.manager, "SELECT rate FROM gg_ticket_currency_rate WHERE currency=?", account.Currency)
	if err != nil {
		return nil, err
	}
	if len(rates) != 1 {
		return nil, paymentInvalid("ticket exchange rate missing")
	}
	rate, err := decimal.NewFromString(fmt.Sprint(rates[0]["rate"]))
	if err != nil {
		return nil, paymentInvalid("invalid ticket exchange rate")
	}
	if _, _, err = ticketCreditAllocation(0, 0, 0, rate); err != nil {
		return nil, err
	}
	rows, err := gogameRows(ctx, dd.manager, "SELECT t.id,t.name,COUNT(h.id) AS num FROM gg_ticket t JOIN "+table+" h ON h.ticket_id=t.id WHERE h.owner=? AND h.is_use=0 AND t.match_id=0 GROUP BY t.id,t.name ORDER BY t.id LIMIT 1001", account.UserID)
	if err != nil {
		return nil, err
	}
	if len(rows) > 1000 {
		return nil, status.Error(codes.ResourceExhausted, "too many ticket types")
	}
	result := []map[string]any{}
	for _, row := range rows {
		count := gogameNumber(row["num"])
		if count < 0 || count > cashMax {
			return nil, paymentInvalid("invalid ticket inventory")
		}
		result = append(result, map[string]any{"id": gogameNumber(row["id"]), "name": row["name"], "num": count, "ticketValue": decimal.NewFromInt(100).Div(rate).InexactFloat64(), "ticketRate": rate.String(), "currency": account.Currency})
	}
	return result, nil
}
func (dd *DBDao) prepareCreditTickets(ctx context.Context, tx *gorm.DB, table string, p MatchPaymentPlan, d *matchPaymentDetail, row map[string]any) ([]int64, error) {
	if !p.TicketCredit {
		return dd.consumeMatchTickets(ctx, tx, table, p)
	}
	if gogameNumber(row["allow_ticket_pay"]) != 1 || p.TicketID <= 0 || p.TicketID != gogameNumber(row["ticket_id"]) || p.TicketCount > 1000 {
		return nil, paymentInvalid("ticket credit is not configured for this match")
	}
	rates, e := gogameRows(ctx, tx, "SELECT rate FROM gg_ticket_currency_rate WHERE currency=? FOR UPDATE", p.Currency)
	if e != nil {
		return nil, e
	}
	if len(rates) != 1 {
		return nil, paymentInvalid("ticket exchange rate missing")
	}
	rate, e := decimal.NewFromString(fmt.Sprint(rates[0]["rate"]))
	if e != nil {
		return nil, paymentInvalid("invalid ticket exchange rate")
	}
	holds, e := gogameRows(ctx, tx, "SELECT id FROM "+table+" WHERE owner=? AND ticket_id=? AND is_use=0 ORDER BY id LIMIT 1000 FOR UPDATE", p.UserID, p.TicketID)
	if e != nil {
		return nil, e
	}
	if int64(len(holds)) < p.TicketCount {
		return nil, paymentInvalid("insufficient required tickets")
	}
	count, credit, e := ticketCreditAllocation(p.Total, int64(len(holds))-p.TicketCount, 1000-p.TicketCount, rate)
	if e != nil {
		return nil, e
	}
	selected := p
	selected.TicketCount += count
	ids, e := dd.consumeMatchTickets(ctx, tx, table, selected)
	if e != nil {
		return nil, e
	}
	d.CreditLocked = true
	d.CreditTickets = count
	d.CreditAmount = credit
	d.TicketRate = rate.String()
	goldCredit := credit
	if goldCredit > p.Gold {
		goldCredit = p.Gold
	}
	d.CashGold = p.Gold - goldCredit
	d.CashService = p.Service - (credit - goldCredit)
	return ids, nil
}
