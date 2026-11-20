package dao

import (
	"app/cashwallet"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// Payment details remain in the existing registration/prize runtime document.
// The existing wallet-marker namespace protects the Redis/MySQL commit boundary;
// no extra payment journal, replay queue or startup reconciliation is introduced.
type MatchPaymentPlan struct {
	Key          string `json:"key"`
	UserID       uint32 `json:"userId"`
	AgentID      uint32 `json:"agentId"`
	MatchID      int64  `json:"matchId"`
	Gold         int64  `json:"gold"`
	Service      int64  `json:"service"`
	Total        int64  `json:"total"`
	Currency     string `json:"currency"`
	TicketID     int64  `json:"ticketId"`
	TicketCount  int64  `json:"ticketCount"`
	TicketCredit bool   `json:"ticketCredit,omitempty"`
	Kind         string `json:"kind"`
	Amount       int64  `json:"amount"`
}
type MatchPaymentInput struct {
	Operation      string           `json:"operation"`
	Plan           MatchPaymentPlan `json:"plan"`
	IncludeService bool             `json:"includeService"`
}
type paymentCount int64

func (n *paymentCount) UnmarshalJSON(raw []byte) error {
	value, e := strconv.ParseInt(strings.Trim(string(raw), "\""), 10, 64)
	if e != nil || value < 0 || value > cashMax {
		return fmt.Errorf("invalid original award count")
	}
	*n = paymentCount(value)
	return nil
}

type matchPaymentDetail struct {
	Fingerprint   string         `json:"fingerprint"`
	Status        string         `json:"status"`
	TicketIDs     []int64        `json:"ticketIds"`
	Receipt       map[string]any `json:"receipt,omitempty"`
	Refund        map[string]any `json:"refund,omitempty"`
	CreditLocked  bool           `json:"creditLocked,omitempty"`
	CreditTickets int64          `json:"creditTickets,omitempty"`
	CreditAmount  int64          `json:"creditAmount,omitempty"`
	TicketRate    string         `json:"ticketRate,omitempty"`
	CashGold      int64          `json:"cashGold,omitempty"`
	CashService   int64          `json:"cashService,omitempty"`
}

func paymentHash(v any) string {
	raw, _ := json.Marshal(v)
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}
func paymentDecode(value any, target any) error {
	raw, e := json.Marshal(value)
	if e != nil {
		return e
	}
	return json.Unmarshal(raw, target)
}
func paymentMap(value any) map[string]any { v, _ := value.(map[string]any); return v }
func paymentInvalid(message string) error { return status.Error(codes.FailedPrecondition, message) }
func paymentMarker(p MatchPaymentPlan, operation string) string {
	return "lottery:settle_wallet:match:" + paymentHash([]string{p.Key, operation})
}

// Marker validation and cashwallet preflight happen before any mutation. A
// marker has no TTL until its authoritative SQL receipt has committed.
const matchCashPrefix = `
local old=redis.call('GET',KEYS[4]);local seal=cjson.decode(ARGV[2])
if old then local receipt=cjson.decode(old);if receipt.fingerprint~=seal.fingerprint then return redis.error_reply('MATCH_PAYMENT_CONFLICT')end;return old end
local encoded=cjson.encode(seal)
local function apply()
`
const matchCashSuffix = `
end
local result=apply();if type(result)~='string' then return result end
redis.call('SET',KEYS[4],encoded)
return encoded
`

func MatchPaymentCash(ctx context.Context, client redis.UniversalClient, input MatchPaymentInput, operation, kind string, amount int64, receipt map[string]any) (map[string]any, error) {
	if client == nil {
		return nil, status.Error(codes.Unavailable, "match wallet unavailable")
	}
	p := input.Plan
	fingerprint := paymentHash(input)
	marker := paymentMarker(p, operation)
	// Read marker before the revision: retries remain valid after unrelated play.
	if raw, e := client.Get(ctx, marker).Result(); e == nil {
		return decodeMatchMarker(raw, fingerprint)
	} else if e != redis.Nil {
		return nil, e
	}
	var snapshot struct {
		Revision int64 `json:"revision"`
	}
	raw, e := CashRead(ctx, client, CashAccount{UserID: p.UserID, AgentID: p.AgentID, Currency: p.Currency})
	if e != nil {
		return nil, e
	}
	if e = json.Unmarshal(raw, &snapshot); e != nil {
		return nil, e
	}
	cash := CashRequest{ID: p.Key + ":" + operation, Accounts: []CashAccount{{UserID: p.UserID, AgentID: p.AgentID, Currency: p.Currency, Revision: snapshot.Revision}}, Steps: []CashStep{{ID: p.Key + ":" + operation, Account: 1, Kind: kind, Amount: amount}}}
	a, _ := json.Marshal(cash)
	b, _ := json.Marshal(map[string]any{"fingerprint": fingerprint, "receipt": receipt})
	keys := append([]string{"dirty_list_imp"}, cashKeys(cash.Accounts[0])...)
	keys = append(keys, marker)
	result, e := client.Eval(ctx, matchCashPrefix+cashwallet.Apply+matchCashSuffix, keys, string(a), string(b)).Text()
	if e != nil {
		if strings.Contains(e.Error(), "CASH_INSUFFICIENT") {
			return nil, paymentInvalid("insufficient available balance")
		}
		return nil, status.Error(codes.Unavailable, "match wallet result requires confirmation")
	}
	return decodeMatchMarker(result, fingerprint)
}
func decodeMatchMarker(raw, fingerprint string) (map[string]any, error) {
	var value struct {
		Fingerprint string         `json:"fingerprint"`
		Receipt     map[string]any `json:"receipt"`
	}
	if json.Unmarshal([]byte(raw), &value) != nil || value.Fingerprint != fingerprint || value.Receipt == nil {
		return nil, paymentInvalid("payment request conflict")
	}
	return value.Receipt, nil
}

var paymentSchema = regexp.MustCompile(`^[a-zA-Z0-9_]{1,64}$`)

// Qualified identifiers originate only from verified server metadata. Both DBs
// must be reachable from the manager connection on the very same MySQL server.
func (dd *DBDao) matchTicketTable(ctx context.Context) (string, error) {
	if dd.db == nil || dd.manager == nil {
		return "", status.Error(codes.Unavailable, "ticket database unavailable")
	}
	a, e := gogameRows(ctx, dd.manager, "SELECT @@server_uuid AS server_id,DATABASE() AS db_name")
	if e != nil {
		return "", e
	}
	b, e := gogameRows(ctx, dd.db, "SELECT @@server_uuid AS server_id,DATABASE() AS db_name")
	if e != nil {
		return "", e
	}
	if len(a) != 1 || len(b) != 1 || a[0]["server_id"] == nil || b[0]["server_id"] == nil || len(fmt.Sprint(a[0]["server_id"])) != 36 || fmt.Sprint(a[0]["server_id"]) != fmt.Sprint(b[0]["server_id"]) {
		return "", status.Error(codes.Unavailable, "atomic tournament tickets require one MySQL server")
	}
	schema := fmt.Sprint(b[0]["db_name"])
	if !paymentSchema.MatchString(schema) {
		return "", status.Error(codes.Unavailable, "invalid ticket database identity")
	}
	return "`" + schema + "`.`gg_ticket_hold`", nil
}
func validPaymentPlan(p MatchPaymentPlan) bool {
	if p.UserID == 0 || p.AgentID == 0 || p.MatchID <= 0 || !cashCurrency.MatchString(p.Currency) || !cashID.MatchString(p.Key) {
		return false
	}
	for _, n := range []int64{p.Gold, p.Service, p.Total, p.Amount, p.TicketID, p.TicketCount} {
		if n < 0 || n > cashMax {
			return false
		}
	}
	return p.Gold <= cashMax-p.Service && p.Total == p.Gold+p.Service && p.TicketCount <= 1000 && p.Amount <= cashMax
}
func findMatchPayment(row, state map[string]any, input MatchPaymentInput) (map[string]any, error) {
	p := input.Plan
	if gogameNumber(row["agent_id"]) != int64(p.AgentID) || gogameNumber(row["id"]) != p.MatchID {
		return nil, status.Error(codes.PermissionDenied, "payment match owner mismatch")
	}
	if input.Operation == "award" {
		if gogameNumber(row["state"]) != 3 {
			return nil, paymentInvalid("match has not finished")
		}
		participant := paymentMap(paymentMap(state["registrations"])[strconv.FormatUint(uint64(p.UserID), 10)])
		if participant == nil || (participant["status"] != "winner" && participant["status"] != "eliminated") || gogameNumber(participant["agentId"]) != int64(p.AgentID) {
			return nil, paymentInvalid("award recipient is not a final participant")
		}
		var config []struct {
			Rank  paymentCount `json:"rank"`
			Items []struct {
				Kind     string       `json:"kind"`
				Currency string       `json:"currency"`
				Amount   paymentCount `json:"amount"`
				TicketID paymentCount `json:"ticketId"`
			} `json:"items"`
		}
		if json.Unmarshal([]byte(fmt.Sprint(row["award_json"])), &config) != nil {
			return nil, paymentInvalid("invalid original award configuration")
		}
		configured := false
		for _, rank := range config {
			if int64(rank.Rank) != gogameNumber(participant["rank"]) {
				continue
			}
			for index, item := range rank.Items {
				currency := item.Currency
				if currency == "" {
					currency = fmt.Sprint(row["currency"])
				}
				if p.Key == fmt.Sprintf("match:%d:%d:award:%d:%d", p.AgentID, p.MatchID, p.UserID, index) && p.Kind == item.Kind && p.Currency == currency && p.Amount == int64(item.Amount) && p.TicketID == int64(item.TicketID) {
					configured = true
				}
			}
		}
		if !configured {
			return nil, paymentInvalid("award differs from the locked rank configuration")
		}
		prizes, _ := state["prizes"].([]any)
		for _, v := range prizes {
			entry := paymentMap(v)
			var saved MatchPaymentPlan
			if paymentDecode(entry["plan"], &saved) == nil && saved == p {
				return entry, nil
			}
		}
		return nil, paymentInvalid("award plan does not match final standings")
	}
	entry := paymentMap(paymentMap(state["registrations"])[strconv.FormatUint(uint64(p.UserID), 10)])
	parent := entry
	if entry != nil && paymentMap(entry["plan"])["key"] != p.Key {
		entry = nil
		requests, _ := parent["reentries"].([]any)
		if len(requests) > 200 {
			return nil, paymentInvalid("too many tournament reentry requests")
		}
		for _, request := range requests {
			candidate := paymentMap(request)
			if paymentMap(candidate["plan"])["key"] == p.Key {
				entry = candidate
				break
			}
		}
		if entry != nil && entry["reentry"] != true {
			return nil, paymentInvalid("invalid original reentry plan")
		}
	}
	var saved MatchPaymentPlan
	if entry == nil || paymentDecode(entry["plan"], &saved) != nil || saved != p || p.Key != fmt.Sprintf("match:%d:%d:%d:%d", p.AgentID, p.MatchID, p.UserID, gogameNumber(entry["attempt"])) {
		return nil, paymentInvalid("registration payment plan mismatch")
	}
	return entry, nil
}
func paymentDetail(entry map[string]any, p MatchPaymentPlan) (*matchPaymentDetail, error) {
	d := &matchPaymentDetail{Fingerprint: paymentHash(p), Status: "prepared", TicketIDs: []int64{}}
	if entry["payment"] != nil {
		if paymentDecode(entry["payment"], d) != nil || d.Fingerprint != paymentHash(p) {
			return nil, paymentInvalid("original payment record mismatch")
		}
	}
	return d, nil
}
func paymentSet(entry map[string]any, d *matchPaymentDetail) {
	raw, _ := json.Marshal(d)
	var value map[string]any
	_ = DecodeGogameJSON(string(raw), &value)
	entry["payment"] = value
}
func paymentRegistrationOpen(row, entry, state map[string]any) bool {
	if entry["status"] != "pending" {
		return false
	}
	if entry["late"] != true {
		return gogameNumber(row["state"]) == 1 && gogameNumber(row["start_time"]) > time.Now().UnixMilli()
	}
	if gogameNumber(row["state"]) != 2 || state["invalidated"] != nil {
		return false
	}
	if entry["reentry"] == true {
		parent := paymentMap(paymentMap(state["registrations"])[fmt.Sprint(gogameNumber(entry["userId"]))])
		limit, number := gogameNumber(row["rebuy_time"]), gogameNumber(entry["reentryNumber"])
		if parent["status"] != "eliminated" || gogameNumber(parent["points"]) != 0 || limit <= 0 || limit > 100 || number <= 0 || number > limit {
			return false
		}
		requests, _ := parent["reentries"].([]any)
		used := int64(0)
		for _, request := range requests {
			if paymentMap(request)["status"] == "admitted" {
				used++
			}
		}
		if number != used+1 {
			return false
		}
	}
	level, seconds, start := gogameNumber(row["delay_register_level"]), gogameNumber(row["delay_register_time"]), gogameNumber(row["start_time"])
	if level < 0 || seconds < 0 || time.Now().UnixMilli() < start || (level == 0 && seconds == 0) || seconds > (cashMax-start)/1000 {
		return false
	}
	if seconds > 0 && time.Now().UnixMilli() >= start+seconds*1000 {
		return false
	}
	clock := paymentMap(state["clock"])
	current := gogameNumber(clock["currentBlindLv"])
	if level > 0 {
		deadline, exists := clock["nextBlindAt"]
		if !exists || gogameNumber(deadline) < 0 || (gogameNumber(deadline) > 0 && time.Now().UnixMilli() >= gogameNumber(deadline)) {
			return false
		}
	}
	return level == 0 || (current > 0 && current <= level)
}

func (dd *DBDao) consumeMatchTickets(ctx context.Context, tx *gorm.DB, table string, p MatchPaymentPlan) ([]int64, error) {
	if p.TicketCount == 0 {
		return []int64{}, nil
	}
	var ticket struct {
		MatchID int64 `gorm:"column:match_id"`
	}
	rows, e := gogameRows(ctx, tx, "SELECT match_id FROM gg_ticket WHERE id=? FOR UPDATE", p.TicketID)
	if e != nil {
		return nil, e
	}
	if len(rows) != 1 {
		return nil, paymentInvalid("ticket type unavailable")
	}
	ticket.MatchID = gogameNumber(rows[0]["match_id"])
	if ticket.MatchID != 0 && ticket.MatchID != p.MatchID {
		return nil, paymentInvalid("ticket is bound to another match")
	}
	holds, e := gogameRows(ctx, tx, "SELECT id FROM "+table+" WHERE owner=? AND ticket_id=? AND is_use=0 ORDER BY id LIMIT ? FOR UPDATE", p.UserID, p.TicketID, p.TicketCount)
	if e != nil {
		return nil, e
	}
	if int64(len(holds)) != p.TicketCount {
		return nil, paymentInvalid("insufficient eligible tickets")
	}
	ids := []int64{}
	for _, hold := range holds {
		id := gogameNumber(hold["id"])
		if id <= 0 {
			return nil, paymentInvalid("invalid ticket ownership")
		}
		ids = append(ids, id)
		if _, e = tx.Statement.ConnPool.ExecContext(ctx, "UPDATE "+table+" SET is_use=1,use_match_id=?,use_time=? WHERE id=? AND owner=? AND is_use=0", p.MatchID, time.Now().Unix(), id, p.UserID); e != nil {
			return nil, e
		}
	}
	return ids, nil
}
func restoreMatchTickets(ctx context.Context, tx *gorm.DB, table string, p MatchPaymentPlan, ids []int64) error {
	for _, id := range ids {
		rows, e := gogameRows(ctx, tx, "SELECT owner,is_use,use_match_id FROM "+table+" WHERE id=? FOR UPDATE", id)
		if e != nil {
			return e
		}
		if len(rows) != 1 || gogameNumber(rows[0]["owner"]) != int64(p.UserID) || gogameNumber(rows[0]["is_use"]) != 1 || gogameNumber(rows[0]["use_match_id"]) != p.MatchID {
			return paymentInvalid("original ticket claim changed")
		}
		if _, e = tx.Statement.ConnPool.ExecContext(ctx, "UPDATE "+table+" SET is_use=0,use_match_id=0,use_time=0 WHERE id=?", id); e != nil {
			return e
		}
	}
	return nil
}

func (dd *DBDao) gogameMatchPayment(ctx context.Context, payload string) (any, error) {
	var input MatchPaymentInput
	if e := DecodeGogameJSON(payload, &input); e != nil {
		return nil, e
	}
	if !validPaymentPlan(input.Plan) || (input.Operation != "charge" && input.Operation != "refund" && input.Operation != "award" && input.Operation != "query") || (input.Operation != "refund" && input.IncludeService) {
		return nil, status.Error(codes.InvalidArgument, "invalid tournament payment")
	}
	if input.Operation == "query" {
		return dd.matchPayment(ctx, nil, input)
	}
	rd := RedisIns()
	if rd == nil || rd.Client == nil {
		return nil, status.Error(codes.Unavailable, "tournament wallet unavailable")
	}
	return dd.matchPayment(ctx, rd.Client, input)
}
func (dd *DBDao) matchPayment(ctx context.Context, client redis.UniversalClient, input MatchPaymentInput) (any, error) {
	p := input.Plan
	if input.Operation == "query" {
		var result map[string]any
		found, e := dd.withGogameMatchSQL(ctx, p.MatchID, func(_ *gorm.DB, row, state map[string]any) error {
			lookup := input
			if p.Kind != "" {
				lookup.Operation = "award"
			}
			entry, e := findMatchPayment(row, state, lookup)
			if e != nil {
				return e
			}
			if p.Kind != "" {
				status := "pending"
				if entry["authorityReceipt"] != nil {
					status = "paid"
				}
				result = map[string]any{"key": p.Key, "status": status, "receipt": entry["authorityReceipt"]}
				return nil
			}
			d, e := paymentDetail(entry, p)
			if e != nil {
				return e
			}
			status := d.Status
			if entry["payment"] == nil {
				status = "uncharged"
			}
			result = map[string]any{"key": p.Key, "status": status, "receipt": d.Receipt, "refund": d.Refund}
			return nil
		})
		if e != nil {
			return nil, e
		}
		if !found {
			return nil, status.Error(codes.NotFound, "match not found")
		}
		return result, nil
	}
	table := ""
	if p.TicketCount > 0 || p.TicketCredit || p.Kind == "ticket" {
		var e error
		table, e = dd.matchTicketTable(ctx)
		if e != nil {
			return nil, e
		}
	}
	if input.Operation == "charge" {
		found, e := dd.withGogameMatchSQL(ctx, p.MatchID, func(tx *gorm.DB, row, state map[string]any) error {
			entry, e := findMatchPayment(row, state, input)
			if e != nil {
				return e
			}
			d, e := paymentDetail(entry, p)
			if e != nil {
				return e
			}
			if entry["payment"] != nil {
				return nil
			}
			if !paymentRegistrationOpen(row, entry, state) {
				return paymentInvalid("registration payment is closed")
			}
			if p.Gold != gogameNumber(row["register_gold"]) || p.Service != gogameNumber(row["service_gold"]) || p.Currency != row["currency"] || p.TicketCount != gogameNumber(row["register_ticket"]) || (p.TicketCount > 0 && p.TicketID != gogameNumber(row["ticket_id"])) {
				return paymentInvalid("registration fees changed")
			}
			d.TicketIDs, e = dd.prepareCreditTickets(ctx, tx, table, p, d, row)
			if e != nil {
				return e
			}
			paymentSet(entry, d)
			return nil
		})
		if e != nil {
			return nil, e
		}
		if !found {
			return nil, status.Error(codes.NotFound, "match not found")
		}
	}
	var result map[string]any
	var decline error
	found, e := dd.withGogameMatchSQL(ctx, p.MatchID, func(tx *gorm.DB, row, state map[string]any) error {
		entry, e := findMatchPayment(row, state, input)
		if e != nil {
			return e
		}
		if input.Operation == "award" {
			result, e = dd.matchAward(ctx, tx, client, table, entry, p, input)
			return e
		}
		d, e := paymentDetail(entry, p)
		if e != nil {
			return e
		}
		switch input.Operation {
		case "charge":
			if d.Status == "charged" {
				result = d.Receipt
				return nil
			}
			if d.Status != "prepared" {
				return paymentInvalid("payment attempt is closed")
			}
			if !paymentRegistrationOpen(row, entry, state) {
				raw, err := client.Get(ctx, paymentMarker(p, "charge")).Result()
				if err != nil && err != redis.Nil {
					return err
				}
				if err == redis.Nil {
					if e = restoreMatchTickets(ctx, tx, table, p, d.TicketIDs); e != nil {
						return e
					}
					d.Status = "declined"
					paymentSet(entry, d)
					decline = paymentInvalid("registration payment is closed")
					return nil
				}
				if _, e = decodeMatchMarker(raw, paymentHash(MatchPaymentInput{Operation: "charge", Plan: p})); e != nil {
					return e
				}
			}
			gold, service := p.Gold, p.Service
			if p.TicketCredit {
				if !d.CreditLocked {
					return paymentInvalid("ticket credit preparation missing")
				}
				gold, service = d.CashGold, d.CashService
			}
			receipt := map[string]any{"key": p.Key, "paidGold": gold, "paidService": service, "currency": p.Currency, "paidTicketCount": p.TicketCount + d.CreditTickets, "ticketId": p.TicketID}
			if p.TicketCredit {
				receipt["grossGold"] = p.Gold
				receipt["grossService"] = p.Service
				receipt["ticketCreditAmount"] = d.CreditAmount
				receipt["creditTicketCount"] = d.CreditTickets
				receipt["ticketRate"] = d.TicketRate
			}
			result, e = MatchPaymentCash(ctx, client, input, "charge", "debit", gold+service, receipt)
			if e != nil {
				if status.Code(e) == codes.FailedPrecondition && status.Convert(e).Message() == "insufficient available balance" {
					if e2 := restoreMatchTickets(ctx, tx, table, p, d.TicketIDs); e2 != nil {
						return e2
					}
					d.Status = "declined"
					paymentSet(entry, d)
					decline = e
					return nil
				}
				return e
			}
			d.Status = "charged"
			d.Receipt = result
			paymentSet(entry, d)
			return nil
		case "refund":
			if !input.IncludeService {
				return paymentInvalid("refund policy mismatch")
			}
			if d.Status == "refunded" {
				result = d.Refund
				return nil
			}
			if entry["status"] != "refund_pending" && gogameNumber(row["state"]) != 4 {
				return paymentInvalid("registration is not refundable")
			}
			if d.Status == "prepared" {
				raw, err := client.Get(ctx, paymentMarker(p, "charge")).Result()
				if err != nil && err != redis.Nil {
					return err
				}
				if err == nil {
					d.Receipt, e = decodeMatchMarker(raw, paymentHash(MatchPaymentInput{Operation: "charge", Plan: p}))
					if e != nil {
						return e
					}
					d.Status = "charged"
				}
			}
			charged := d.Status == "charged"
			refund := int64(0)
			if charged {
				refund = p.Gold + p.Service
				if p.TicketCredit {
					refund = gogameNumber(d.Receipt["paidGold"]) + gogameNumber(d.Receipt["paidService"])
				}
			}
			result = map[string]any{"key": p.Key, "currency": p.Currency, "refunded": refund, "restoredTickets": int64(len(d.TicketIDs)), "uncharged": !charged}
			if p.TicketCredit && charged {
				result["chargeReceipt"] = d.Receipt
			}
			if charged {
				result, e = MatchPaymentCash(ctx, client, input, "refund", "credit", refund, result)
				if e != nil {
					return e
				}
			}
			if d.Status != "declined" {
				if e = restoreMatchTickets(ctx, tx, table, p, d.TicketIDs); e != nil {
					return e
				}
			}
			d.Status = "refunded"
			d.Refund = result
			paymentSet(entry, d)
			return nil
		}
		return status.Error(codes.InvalidArgument, "unknown payment")
	})
	if e != nil {
		return nil, e
	}
	if !found {
		return nil, status.Error(codes.NotFound, "match not found")
	}
	if decline != nil {
		return nil, decline
	}
	// SQL now contains the durable receipt. Seven-day marker expiry cannot turn
	// a late retry into another charge: all retries first read that SQL receipt.
	if result != nil {
		_ = client.Expire(ctx, paymentMarker(p, input.Operation), 7*24*time.Hour).Err()
	}
	return result, nil
}
func (dd *DBDao) matchAward(ctx context.Context, tx *gorm.DB, client redis.UniversalClient, table string, entry map[string]any, p MatchPaymentPlan, input MatchPaymentInput) (map[string]any, error) {
	if entry["authorityReceipt"] != nil {
		return paymentMap(entry["authorityReceipt"]), nil
	}
	if p.Amount <= 0 || !strings.HasPrefix(p.Key, fmt.Sprintf("match:%d:%d:award:%d:", p.AgentID, p.MatchID, p.UserID)) {
		return nil, paymentInvalid("invalid original award")
	}
	receipt := map[string]any{"key": p.Key, "amount": p.Amount, "currency": p.Currency, "kind": p.Kind, "ticketId": p.TicketID}
	switch p.Kind {
	case "money":
		var e error
		receipt, e = MatchPaymentCash(ctx, client, input, "award", "credit", p.Amount, receipt)
		if e != nil {
			return nil, e
		}
	case "ticket":
		if p.Amount > 1000 {
			return nil, paymentInvalid("ticket award exceeds batch limit")
		}
		rows, e := gogameRows(ctx, tx, "SELECT remain FROM gg_ticket WHERE id=? FOR UPDATE", p.TicketID)
		if e != nil {
			return nil, e
		}
		if len(rows) != 1 || gogameNumber(rows[0]["remain"]) < p.Amount {
			return nil, paymentInvalid("ticket issuance inventory insufficient")
		}
		if _, e = tx.Statement.ConnPool.ExecContext(ctx, "UPDATE gg_ticket SET remain=remain-? WHERE id=?", p.Amount, p.TicketID); e != nil {
			return nil, e
		}
		for i := int64(0); i < p.Amount; i++ {
			if _, e = tx.Statement.ConnPool.ExecContext(ctx, "INSERT INTO "+table+" (ticket_id,owner,is_use,use_time,use_match_id,create_time) VALUES (?,?,0,0,0,?)", p.TicketID, p.UserID, time.Now().Unix()); e != nil {
				return nil, e
			}
		}
	default:
		return nil, paymentInvalid("unsupported award kind")
	}
	entry["authorityReceipt"] = receipt
	return receipt, nil
}
