package dao

import (
	"app/tables/gogame"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type gogameSQLStep struct {
	contains string
	columns  []string
	values   [][]driver.Value
	err      error
	check    func([]driver.NamedValue)
}
type gogameSQLScript struct {
	t     *testing.T
	steps []gogameSQLStep
	log   []string
}
type gogameSQLDriver struct{ script *gogameSQLScript }
type gogameSQLConn struct{ script *gogameSQLScript }
type gogameSQLTx struct{ script *gogameSQLScript }
type gogameSQLRows struct {
	columns []string
	values  [][]driver.Value
}

var gogameSQLSequence atomic.Int64

func (d gogameSQLDriver) Open(string) (driver.Conn, error) { return &gogameSQLConn{d.script}, nil }
func (c *gogameSQLConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (c *gogameSQLConn) Close() error { return nil }
func (c *gogameSQLConn) Begin() (driver.Tx, error) {
	c.script.log = append(c.script.log, "begin")
	return &gogameSQLTx{c.script}, nil
}
func (t *gogameSQLTx) Commit() error   { t.script.log = append(t.script.log, "commit"); return nil }
func (t *gogameSQLTx) Rollback() error { t.script.log = append(t.script.log, "rollback"); return nil }
func (c *gogameSQLConn) step(query string, args []driver.NamedValue) gogameSQLStep {
	c.script.t.Helper()
	c.script.log = append(c.script.log, query)
	if len(c.script.steps) == 0 {
		c.script.t.Fatalf("unexpected SQL: %s", query)
	}
	step := c.script.steps[0]
	c.script.steps = c.script.steps[1:]
	if !strings.Contains(query, step.contains) {
		c.script.t.Fatalf("SQL %q does not contain %q", query, step.contains)
	}
	if step.check != nil {
		step.check(args)
	}
	return step
}
func (c *gogameSQLConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	step := c.step(query, args)
	return &gogameSQLRows{step.columns, step.values}, step.err
}
func (c *gogameSQLConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	step := c.step(query, args)
	return driver.RowsAffected(1), step.err
}
func (r *gogameSQLRows) Columns() []string { return r.columns }
func (r *gogameSQLRows) Close() error      { return nil }
func (r *gogameSQLRows) Next(values []driver.Value) error {
	if len(r.values) == 0 {
		return io.EOF
	}
	copy(values, r.values[0])
	r.values = r.values[1:]
	return nil
}
func gogameTestDB(t *testing.T, steps ...gogameSQLStep) (*gorm.DB, *gogameSQLScript) {
	t.Helper()
	script := &gogameSQLScript{t: t, steps: steps}
	name := "gogame-script-" + string(rune('A'+gogameSQLSequence.Add(1)))
	sql.Register(name, gogameSQLDriver{script})
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	gdb, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	return gdb, script
}

func TestGogameTransactionCommitsJournalAndRowUnderOneLock(t *testing.T) {
	db, script := gogameTestDB(t,
		gogameSQLStep{contains: "SELECT * FROM gg_match WHERE id=? FOR UPDATE", columns: []string{"id", "agent_id", "state"}, values: [][]driver.Value{{int64(9), int64(11), int64(1)}}},
		gogameSQLStep{contains: "SELECT payload FROM gg_match_runtime", columns: []string{"payload"}},
		gogameSQLStep{contains: "UPDATE gg_match SET", check: func(args []driver.NamedValue) {
			if args[0].Value != int64(4) {
				t.Fatalf("state not committed: %v", args)
			}
		}},
		gogameSQLStep{contains: "INSERT INTO gg_match_runtime", check: func(args []driver.NamedValue) {
			if !strings.Contains(args[1].Value.(string), "refund_pending") {
				t.Fatal("missing refund journal")
			}
		}},
	)
	found, err := (&DBDao{manager: db}).WithGogameMatch(context.Background(), 9, func(row, state map[string]any) error {
		row["state"] = int64(4)
		state["registrations"].(map[string]any)["10001"] = map[string]any{"status": "refund_pending"}
		return nil
	})
	if err != nil || !found {
		t.Fatalf("transaction failed: %v", err)
	}
	if len(script.steps) != 0 || script.log[0] != "begin" || script.log[len(script.log)-1] != "commit" {
		t.Fatalf("not atomic: %v", script.log)
	}
}

func TestGogameTransactionRollsBackCallbackOrIdentityChange(t *testing.T) {
	for _, reason := range []string{"callback", "agent"} {
		t.Run(reason, func(t *testing.T) {
			db, script := gogameTestDB(t, gogameSQLStep{contains: "FOR UPDATE", columns: []string{"id", "agent_id"}, values: [][]driver.Value{{int64(9), int64(11)}}}, gogameSQLStep{contains: "SELECT payload", columns: []string{"payload"}})
			_, err := (&DBDao{manager: db}).WithGogameMatch(context.Background(), 9, func(row, state map[string]any) error {
				if reason == "callback" {
					return errors.New("payment failed")
				}
				row["agent_id"] = 22
				return nil
			})
			if err == nil || script.log[len(script.log)-1] != "rollback" {
				t.Fatalf("expected rollback, got %v %v", err, script.log)
			}
		})
	}
}

func TestGogameStorageRejectsSQLAndUnknownFieldsBeforeQuery(t *testing.T) {
	dd := &DBDao{}
	if _, err := dd.GogameCall(context.Background(), "SELECT * FROM user_info", "{}"); status.Code(err) != codes.InvalidArgument {
		t.Fatal(err)
	}
	if _, err := dd.GogameCall(context.Background(), "listRooms", `{"sql":"DELETE FROM gg_match"}`); status.Code(err) != codes.InvalidArgument {
		t.Fatal(err)
	}
	if _, err := gogameValues(&gogame.Match{}, map[string]json.RawMessage{"id": json.RawMessage("1")}); status.Code(err) != codes.InvalidArgument {
		t.Fatal(err)
	}
	if _, err := gogameValues(&gogame.Match{}, map[string]json.RawMessage{"state;DELETE": json.RawMessage("1")}); status.Code(err) != codes.InvalidArgument {
		t.Fatal(err)
	}
	if _, err := dd.GogameCall(context.Background(), "persistScore", `{"userId":10001,"score":100}`); status.Code(err) != codes.FailedPrecondition {
		t.Fatal("snapshot cannot impersonate financial authority", err)
	}
}

func TestGogameParticipantProjectionKeepsNativeStatusAndOverridesLegacyRows(t *testing.T) {
	runtime := `{"registrations":{"10001":{"status":"playing","userId":10001,"name":"player","avatar":"16","points":100,"rank":1,"roomId":81000000,"plan":{"gold":10,"service":1,"currency":"CNY"}}},"tables":[{"match_id":9,"room_id":81000000,"player_nums":1}]}`
	for _, operation := range []string{"listMatchPlayers", "listMatchRooms"} {
		t.Run(operation, func(t *testing.T) {
			db, _ := gogameTestDB(t, gogameSQLStep{contains: "SELECT match_id,payload", columns: []string{"match_id", "payload"}, values: [][]driver.Value{{int64(9), runtime}}},
				gogameSQLStep{contains: "SELECT * FROM gg_match_", columns: []string{"match_id"}, values: [][]driver.Value{{int64(9)}, {int64(10)}}})
			value, err := (&DBDao{manager: db, db: db}).gogameParticipants(context.Background(), operation)
			if err != nil {
				t.Fatal(err)
			}
			rows := value.([]map[string]any)
			if len(rows) != 2 || rows[0]["match_id"] != int64(10) {
				t.Fatalf("bad projection %v", rows)
			}
			if operation == "listMatchPlayers" && rows[1]["status"] != 4 {
				t.Fatal("native playing status must be 4")
			}
		})
	}
}

func TestGogameScheduleCannotMoveToAnotherAgent(t *testing.T) {
	db, script := gogameTestDB(t, gogameSQLStep{contains: "FOR UPDATE", columns: []string{"id", "agent_id"}, values: [][]driver.Value{{int64(3), int64(11)}}})
	_, err := (&DBDao{manager: db}).GogameCall(context.Background(), "saveSchedule", `{"id":3,"row":{"agent_id":22,"schedule_kind":"once"}}`)
	if status.Code(err) != codes.PermissionDenied || script.log[len(script.log)-1] != "rollback" {
		t.Fatalf("agent migration accepted: %v", err)
	}
}

func TestGogameOccurrenceLookupUsesTheLockedConnection(t *testing.T) {
	db, script := gogameTestDB(t,
		gogameSQLStep{contains: "gg_match_schedule", columns: []string{"id", "agent_id", "payload", "schedule_kind", "timezone", "daily_time", "start_time", "enabled"}, values: [][]driver.Value{{int64(3), int64(11), "{}", "daily", "UTC", "12:00", int64(0), int64(1)}}},
		gogameSQLStep{contains: "SELECT match_id FROM gg_match_occurrence", columns: []string{"match_id"}, values: [][]driver.Value{{int64(9)}}},
		gogameSQLStep{contains: "SELECT * FROM gg_match WHERE id=?", columns: []string{"id", "agent_id"}, values: [][]driver.Value{{int64(9), int64(11)}}},
	)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result, err := (&DBDao{manager: db}).GogameCall(ctx, "ensureOccurrence", `{"schedule":{"id":3,"agent_id":11,"payload":"{}","schedule_kind":"daily","timezone":"UTC","daily_time":"12:00","start_time":0,"enabled":1},"slot":{"key":"3:2026-10-30","startTime":1000000},"row":{"agent_id":11,"state":1,"start_time":1000000}}`)
	if err != nil || result.(map[string]any)["id"] != int64(9) {
		t.Fatalf("occurrence lookup failed %v %v", result, err)
	}
	if len(script.steps) != 0 || script.log[len(script.log)-1] != "commit" {
		t.Fatal(script.log)
	}
}

func TestGogameUnchangedNullableFieldsArePreserved(t *testing.T) {
	db, script := gogameTestDB(t,
		gogameSQLStep{contains: "FOR UPDATE", columns: []string{"id", "agent_id", "ticket_name"}, values: [][]driver.Value{{int64(9), int64(11), nil}}},
		gogameSQLStep{contains: "SELECT payload", columns: []string{"payload"}},
		gogameSQLStep{contains: "INSERT INTO gg_match_runtime"},
	)
	_, err := (&DBDao{manager: db}).WithGogameMatch(context.Background(), 9, func(row, state map[string]any) error { return nil })
	if err != nil || len(script.steps) != 0 {
		t.Fatalf("unchanged null values should not be rewritten %v", err)
	}
}
