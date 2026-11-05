package dao

import (
	"app/tables/gogame"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

const GogamePayloadLimit = 24 * 1024 * 1024

type GogameArgs struct {
	ID       int64                      `json:"id"`
	UserID   uint32                     `json:"userId"`
	Score    json.RawMessage            `json:"score"`
	Row      map[string]json.RawMessage `json:"row"`
	Patch    map[string]json.RawMessage `json:"patch"`
	Schedule *gogame.MatchSchedule      `json:"schedule"`
	Slot     struct {
		Key       string `json:"key"`
		StartTime int64  `json:"startTime"`
	} `json:"slot"`
}

func DecodeGogameJSON(raw string, target any) error {
	if len(raw) == 0 || len(raw) > GogamePayloadLimit {
		return status.Error(codes.InvalidArgument, "invalid gogame payload size")
	}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	dec.UseNumber()
	if err := dec.Decode(target); err != nil {
		return status.Error(codes.InvalidArgument, "invalid gogame JSON")
	}
	if dec.Decode(new(any)) != io.EOF {
		return status.Error(codes.InvalidArgument, "trailing gogame JSON")
	}
	return nil
}

// The SQL identifiers below come only from fixed operations and Go table models.
// Caller-supplied SQL, table names and unknown model fields are rejected.
var gogameSchemas sync.Map

func gogameValues(model any, raw map[string]json.RawMessage) (map[string]any, error) {
	if raw == nil {
		return nil, status.Error(codes.InvalidArgument, "row is required")
	}
	parsed, err := schema.Parse(model, &gogameSchemas, schema.NamingStrategy{})
	if err != nil {
		return nil, err
	}
	values := make(map[string]any, len(raw))
	for name, b := range raw {
		field := parsed.FieldsByDBName[name]
		if field == nil || field.PrimaryKey {
			return nil, status.Error(codes.InvalidArgument, "unknown or immutable field: "+name)
		}
		if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
			return nil, status.Error(codes.InvalidArgument, "null model field: "+name)
		}
		value := reflect.New(field.FieldType)
		if err := json.Unmarshal(b, value.Interface()); err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid model field: "+name)
		}
		values[name] = value.Elem().Interface()
	}
	return values, nil
}

func gogameRows(ctx context.Context, db *gorm.DB, query string, args ...any) ([]map[string]any, error) {
	if db == nil {
		return nil, status.Error(codes.Unavailable, "gogame database unavailable")
	}
	rows, err := db.Statement.ConnPool.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		refs := make([]any, len(columns))
		for i := range values {
			refs[i] = &values[i]
		}
		if err := rows.Scan(refs...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(columns))
		for i, key := range columns {
			value := values[i]
			if b, ok := value.([]byte); ok {
				value = string(b)
			}
			row[key] = value
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
func gogameOne(ctx context.Context, db *gorm.DB, table string, id int64) (map[string]any, error) {
	if id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be positive")
	}
	rows, err := gogameRows(ctx, db, "SELECT * FROM "+table+" WHERE id=?", id)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0], nil
}
func gogameInsert(ctx context.Context, db *gorm.DB, table string, values map[string]any) (map[string]any, error) {
	if len(values) == 0 {
		return nil, status.Error(codes.InvalidArgument, "empty insert")
	}
	names := make([]string, 0, len(values))
	for k := range values {
		names = append(names, k)
	}
	sort.Strings(names)
	quoted, args, marks := make([]string, len(names)), make([]any, len(names)), make([]string, len(names))
	for i, k := range names {
		quoted[i] = "\x60" + k + "\x60"
		args[i] = values[k]
		marks[i] = "?"
	}
	result, err := db.Statement.ConnPool.ExecContext(ctx, "INSERT INTO "+table+" ("+strings.Join(quoted, ",")+") VALUES ("+strings.Join(marks, ",")+")", args...)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return gogameOne(ctx, db, table, id)
}
func gogameUpdate(ctx context.Context, db *gorm.DB, table string, id int64, values map[string]any) error {
	if len(values) == 0 {
		return nil
	}
	names := make([]string, 0, len(values))
	for k := range values {
		names = append(names, k)
	}
	sort.Strings(names)
	parts, args := make([]string, len(names)), make([]any, 0, len(names)+1)
	for i, k := range names {
		parts[i] = "\x60" + k + "\x60=?"
		args = append(args, values[k])
	}
	args = append(args, id)
	_, err := db.Statement.ConnPool.ExecContext(ctx, "UPDATE "+table+" SET "+strings.Join(parts, ",")+" WHERE id=?", args...)
	return err
}
func gogameNumber(value any) int64 {
	switch n := value.(type) {
	case int64:
		return n
	case int32:
		return int64(n)
	case uint64:
		return int64(n)
	case json.Number:
		i, _ := n.Int64()
		return i
	case float64:
		return int64(n)
	}
	return 0
}

func (dd *DBDao) GogameCall(ctx context.Context, operation, payload string) (any, error) {
	var args GogameArgs
	if err := DecodeGogameJSON(payload, &args); err != nil {
		return nil, err
	}
	lists := map[string][2]string{
		"listRooms": {"gg_room_config", " WHERE state=1"}, "listMatches": {"gg_match", " ORDER BY sort_index ASC,id DESC"},
		"listTickets": {"gg_ticket", " ORDER BY id"}, "listRates": {"gg_ticket_currency_rate", " ORDER BY id"},
		"listAwards": {"gg_award_template", " ORDER BY id"}, "listBlinds": {"gg_blinds", " ORDER BY id"}, "listSchedules": {"gg_match_schedule", " ORDER BY id"},
	}
	if list, ok := lists[operation]; ok {
		return gogameRows(ctx, dd.manager, "SELECT * FROM "+list[0]+list[1])
	}
	switch operation {
	case "getMatch":
		return gogameOne(ctx, dd.manager, "gg_match", args.ID)
	case "getAward":
		return gogameOne(ctx, dd.manager, "gg_award_template", args.ID)
	case "listMatchPlayers", "listMatchRooms":
		return dd.gogameParticipants(ctx, operation)
	case "countUnusedTickets":
		if args.UserID == 0 {
			return 0, nil
		}
		rows, err := gogameRows(ctx, dd.db, "SELECT COUNT(*) AS n FROM gg_ticket_hold WHERE owner=? AND is_use=0", args.UserID)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			return 0, nil
		}
		return rows[0]["n"], nil
	case "saveSchedule":
		return dd.gogameSchedule(ctx, args)
	case "ensureOccurrence":
		return dd.gogameOccurrence(ctx, args)
	case "persistScore":
		return nil, status.Error(codes.FailedPrecondition, "score snapshots are not an authoritative wallet operation; use existing financial service")
	}
	type description struct {
		table  string
		model  any
		update bool
	}
	entities := map[string]description{
		"insertMatch": {"gg_match", &gogame.Match{}, false}, "updateMatch": {"gg_match", &gogame.Match{}, true},
		"insertTicket": {"gg_ticket", &gogame.Ticket{}, false}, "updateTicket": {"gg_ticket", &gogame.Ticket{}, true},
		"insertAward": {"gg_award_template", &gogame.AwardTemplate{}, false}, "updateAward": {"gg_award_template", &gogame.AwardTemplate{}, true},
		"insertBlinds": {"gg_blinds", &gogame.Blinds{}, false}, "updateBlinds": {"gg_blinds", &gogame.Blinds{}, true},
	}
	if operation == "upsertRate" {
		values, err := gogameValues(&gogame.TicketCurrencyRate{}, args.Row)
		if err != nil {
			return nil, err
		}
		if values["currency"] == nil || values["rate"] == nil {
			return nil, status.Error(codes.InvalidArgument, "currency/rate required")
		}
		var saved map[string]any
		err = dd.manager.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			_, e := tx.Statement.ConnPool.ExecContext(ctx, "INSERT INTO gg_ticket_currency_rate (currency,rate,create_time,update_time) VALUES (?,?,?,?) ON DUPLICATE KEY UPDATE rate=VALUES(rate),update_time=VALUES(update_time)", values["currency"], values["rate"], values["create_time"], values["update_time"])
			if e != nil {
				return e
			}
			rows, e := gogameRows(ctx, tx, "SELECT * FROM gg_ticket_currency_rate WHERE currency=?", values["currency"])
			if e != nil {
				return e
			}
			if len(rows) == 0 {
				return status.Error(codes.Internal, "rate not saved")
			}
			saved = rows[0]
			return nil
		})
		return saved, err
	}
	entity, ok := entities[operation]
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "unsupported gogame storage operation")
	}
	raw := args.Row
	if entity.update {
		raw = args.Patch
	}
	values, err := gogameValues(entity.model, raw)
	if err != nil {
		return nil, err
	}
	if !entity.update {
		if operation == "insertMatch" && gogameNumber(values["agent_id"]) <= 0 {
			return nil, status.Error(codes.InvalidArgument, "agent_id required")
		}
		return gogameInsert(ctx, dd.manager, entity.table, values)
	}
	if args.ID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	var saved map[string]any
	err = dd.manager.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rows, e := gogameRows(ctx, tx, "SELECT * FROM "+entity.table+" WHERE id=? FOR UPDATE", args.ID)
		if e != nil {
			return e
		}
		if len(rows) == 0 {
			return status.Error(codes.NotFound, "entity not found")
		}
		if operation == "updateMatch" && values["agent_id"] != nil && gogameNumber(values["agent_id"]) != gogameNumber(rows[0]["agent_id"]) {
			return status.Error(codes.PermissionDenied, "match agent is immutable")
		}
		if e = gogameUpdate(ctx, tx, entity.table, args.ID, values); e != nil {
			return e
		}
		saved, e = gogameOne(ctx, tx, entity.table, args.ID)
		return e
	})
	return saved, err
}

func (dd *DBDao) gogameSchedule(ctx context.Context, args GogameArgs) (any, error) {
	values, err := gogameValues(&gogame.MatchSchedule{}, args.Row)
	if err != nil {
		return nil, err
	}
	agent := gogameNumber(values["agent_id"])
	if agent <= 0 {
		return nil, status.Error(codes.InvalidArgument, "schedule agent required")
	}
	if args.ID == 0 {
		return gogameInsert(ctx, dd.manager, "gg_match_schedule", values)
	}
	var saved map[string]any
	err = dd.manager.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rows, e := gogameRows(ctx, tx, "SELECT * FROM gg_match_schedule WHERE id=? FOR UPDATE", args.ID)
		if e != nil {
			return e
		}
		if len(rows) == 0 {
			return status.Error(codes.NotFound, "schedule not found")
		}
		if gogameNumber(rows[0]["agent_id"]) != agent {
			return status.Error(codes.PermissionDenied, "schedule agent is immutable")
		}
		if e = gogameUpdate(ctx, tx, "gg_match_schedule", args.ID, values); e != nil {
			return e
		}
		saved, e = gogameOne(ctx, tx, "gg_match_schedule", args.ID)
		return e
	})
	return saved, err
}

func (dd *DBDao) gogameOccurrence(ctx context.Context, args GogameArgs) (any, error) {
	s := args.Schedule
	if s == nil || s.Id <= 0 || s.AgentId <= 0 || !strings.HasPrefix(args.Slot.Key, fmt.Sprint(s.Id)+":") || len(args.Slot.Key) > 128 {
		return nil, status.Error(codes.InvalidArgument, "invalid occurrence identity")
	}
	values, err := gogameValues(&gogame.Match{}, args.Row)
	if err != nil {
		return nil, err
	}
	if gogameNumber(values["agent_id"]) != s.AgentId || gogameNumber(values["state"]) != 1 || gogameNumber(values["start_time"]) != args.Slot.StartTime {
		return nil, status.Error(codes.InvalidArgument, "occurrence does not match schedule")
	}
	var saved map[string]any
	err = dd.manager.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current gogame.MatchSchedule
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, s.Id).Error; e != nil {
			return e
		}
		if current.Enabled == 0 || current.AgentId != s.AgentId || current.Payload != s.Payload || current.ScheduleKind != s.ScheduleKind || current.Timezone != s.Timezone || current.DailyTime != s.DailyTime || current.StartTime != s.StartTime {
			return nil
		}
		found, e := gogameRows(ctx, tx, "SELECT match_id FROM gg_match_occurrence WHERE occurrence_key=?", args.Slot.Key)
		if e != nil {
			return e
		}
		if len(found) > 0 {
			saved, e = gogameOne(ctx, tx, "gg_match", gogameNumber(found[0]["match_id"]))
			return e
		}
		saved, e = gogameInsert(ctx, tx, "gg_match", values)
		if e != nil {
			return e
		}
		_, e = tx.Statement.ConnPool.ExecContext(ctx, "INSERT INTO gg_match_occurrence (occurrence_key,schedule_id,match_id) VALUES (?,?,?)", args.Slot.Key, s.Id, saved["id"])
		return e
	})
	return saved, err
}

func (dd *DBDao) WithGogameMatch(ctx context.Context, id int64, work func(map[string]any, map[string]any) error) (bool, error) {
	if id <= 0 {
		return false, status.Error(codes.InvalidArgument, "match id required")
	}
	found := false
	err := dd.manager.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rows, err := gogameRows(ctx, tx, "SELECT * FROM gg_match WHERE id=? FOR UPDATE", id)
		if err != nil || len(rows) == 0 {
			return err
		}
		found = true
		row := rows[0]
		beforeJSON, err := json.Marshal(row)
		if err != nil {
			return err
		}
		before := map[string]json.RawMessage{}
		if err = json.Unmarshal(beforeJSON, &before); err != nil {
			return err
		}
		agent := gogameNumber(row["agent_id"])
		docs, err := gogameRows(ctx, tx, "SELECT payload FROM gg_match_runtime WHERE match_id=?", id)
		if err != nil {
			return err
		}
		state := map[string]any{"registrations": map[string]any{}, "outbox": []any{}}
		if len(docs) > 0 {
			if err := DecodeGogameJSON(docs[0]["payload"].(string), &state); err != nil {
				return err
			}
		}
		if err := work(row, state); err != nil {
			return err
		}
		if gogameNumber(row["id"]) != id || gogameNumber(row["agent_id"]) != agent {
			return status.Error(codes.PermissionDenied, "match identity is immutable")
		}
		if _, ok := state["registrations"].(map[string]any); !ok {
			return status.Error(codes.InvalidArgument, "registration journal required")
		}
		if _, ok := state["outbox"].([]any); !ok {
			return status.Error(codes.InvalidArgument, "outbox required")
		}
		raw, err := json.Marshal(row)
		if err != nil {
			return err
		}
		fields := map[string]json.RawMessage{}
		if err = json.Unmarshal(raw, &fields); err != nil {
			return err
		}
		delete(fields, "id")
		changed := map[string]json.RawMessage{}
		for key, value := range fields {
			if !bytes.Equal(before[key], value) {
				changed[key] = value
			}
		}
		for key := range before {
			if key != "id" {
				if _, ok := fields[key]; !ok {
					return status.Error(codes.InvalidArgument, "match fields cannot be removed")
				}
			}
		}
		values, err := gogameValues(&gogame.Match{}, changed)
		if err != nil {
			return err
		}
		if err = gogameUpdate(ctx, tx, "gg_match", id, values); err != nil {
			return err
		}
		payload, err := json.Marshal(state)
		if err != nil {
			return err
		}
		if len(payload) > GogamePayloadLimit {
			return status.Error(codes.ResourceExhausted, "match journal too large")
		}
		_, err = tx.Statement.ConnPool.ExecContext(ctx, "INSERT INTO gg_match_runtime (match_id,payload) VALUES (?,?) ON DUPLICATE KEY UPDATE payload=VALUES(payload)", id, string(payload))
		return err
	})
	return found, err
}

func (dd *DBDao) gogameParticipants(ctx context.Context, operation string) (any, error) {
	docs, err := gogameRows(ctx, dd.manager, "SELECT match_id,payload FROM gg_match_runtime")
	if err != nil {
		return nil, err
	}
	ids := map[int64]bool{}
	projection := make([]map[string]any, 0)
	for _, doc := range docs {
		id := gogameNumber(doc["match_id"])
		ids[id] = true
		var state struct {
			Registrations map[string]struct {
				Status string `json:"status"`
				UserID int64  `json:"userId"`
				Name   string `json:"name"`
				Avatar string `json:"avatar"`
				Points int64  `json:"points"`
				Rank   int64  `json:"rank"`
				RoomID int64  `json:"roomId"`
				Plan   struct {
					Gold     int64  `json:"gold"`
					Service  int64  `json:"service"`
					Currency string `json:"currency"`
				} `json:"plan"`
			} `json:"registrations"`
			Tables []map[string]any `json:"tables"`
		}
		if err = json.Unmarshal([]byte(doc["payload"].(string)), &state); err != nil {
			return nil, err
		}
		if operation == "listMatchRooms" {
			projection = append(projection, state.Tables...)
			continue
		}
		for _, e := range state.Registrations {
			playerStatus := 2
			switch e.Status {
			case "registered":
				playerStatus = 1
			case "playing":
				playerStatus = 4
			case "eliminated", "winner":
			default:
				continue
			}
			projection = append(projection, map[string]any{"match_id": id, "player_id": e.UserID, "name": e.Name, "avatar": e.Avatar, "points": e.Points, "rank": e.Rank, "status": playerStatus, "room_id": e.RoomID, "paid_gold": e.Plan.Gold, "paid_service": e.Plan.Service, "currency": e.Plan.Currency})
		}
	}
	table := "gg_match_player"
	if operation == "listMatchRooms" {
		table = "gg_match_room"
	}
	legacy, err := gogameRows(ctx, dd.db, "SELECT * FROM "+table)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0, len(legacy)+len(projection))
	for _, row := range legacy {
		if !ids[gogameNumber(row["match_id"])] {
			result = append(result, row)
		}
	}
	return append(result, projection...), nil
}
