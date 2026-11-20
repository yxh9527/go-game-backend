package dao

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type soldSQLStep struct {
	contains string
	columns  []string
	values   [][]driver.Value
	err      error
	check    func([]driver.NamedValue)
}
type soldSQLScript struct {
	t     *testing.T
	steps []soldSQLStep
	log   []string
}
type soldSQLDriver struct{ script *soldSQLScript }
type soldSQLConn struct{ script *soldSQLScript }
type soldSQLTx struct{ script *soldSQLScript }
type soldSQLRows struct {
	columns []string
	values  [][]driver.Value
}
type soldSQLResult struct{}

var soldSQLSequence atomic.Int64

func (d soldSQLDriver) Open(string) (driver.Conn, error) { return &soldSQLConn{d.script}, nil }
func (c *soldSQLConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (c *soldSQLConn) Close() error { return nil }
func (c *soldSQLConn) Begin() (driver.Tx, error) {
	c.script.log = append(c.script.log, "begin")
	return &soldSQLTx{c.script}, nil
}
func (t *soldSQLTx) Commit() error   { t.script.log = append(t.script.log, "commit"); return nil }
func (t *soldSQLTx) Rollback() error { t.script.log = append(t.script.log, "rollback"); return nil }
func (c *soldSQLConn) step(query string, args []driver.NamedValue) soldSQLStep {
	c.script.t.Helper()
	c.script.log = append(c.script.log, query)
	if len(c.script.steps) == 0 {
		c.script.t.Fatalf("unexpected SQL %s", query)
	}
	step := c.script.steps[0]
	c.script.steps = c.script.steps[1:]
	if !strings.Contains(query, step.contains) {
		c.script.t.Fatalf("SQL %q lacks %q", query, step.contains)
	}
	if step.check != nil {
		step.check(args)
	}
	return step
}
func (c *soldSQLConn) QueryContext(_ context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	step := c.step(q, args)
	return &soldSQLRows{step.columns, step.values}, step.err
}
func (c *soldSQLConn) ExecContext(_ context.Context, q string, args []driver.NamedValue) (driver.Result, error) {
	step := c.step(q, args)
	return soldSQLResult{}, step.err
}
func (soldSQLResult) LastInsertId() (int64, error) { return 1, nil }
func (soldSQLResult) RowsAffected() (int64, error) { return 1, nil }
func (r *soldSQLRows) Columns() []string           { return r.columns }
func (r *soldSQLRows) Close() error                { return nil }
func (r *soldSQLRows) Next(out []driver.Value) error {
	if len(r.values) == 0 {
		return io.EOF
	}
	copy(out, r.values[0])
	r.values = r.values[1:]
	return nil
}
func soldTestDB(t *testing.T, steps ...soldSQLStep) (*DBDao, *soldSQLScript) {
	t.Helper()
	script := &soldSQLScript{t: t, steps: steps}
	name := fmt.Sprintf("scratch-sold-%d", soldSQLSequence.Add(1))
	sql.Register(name, soldSQLDriver{script})
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
	return &DBDao{player: gdb}, script
}
func soldOwnerRaw(user uint32) string {
	raw, _ := json.Marshal(scratchSoldOwner{Version: 1, Agent: 7, GameID: 91061, UserID: user, Currency: "CNY", Serial: "ticket", RoundID: "original"})
	return string(raw)
}

func TestScratchOwnershipAndPaidTicketCommitTogetherWithoutCopyingDetailsIntoMarketRow(t *testing.T) {
	owner := soldOwnerRaw(42)
	dd, script := soldTestDB(t,
		soldSQLStep{contains: "ON DUPLICATE KEY UPDATE", check: func(args []driver.NamedValue) {
			if len(args) < 6 || args[0].Value != int64(0) || args[1].Value != int64(91061) || args[2].Value != "*" || args[3].Value != "scratch-sold:7:ticket" || args[4].Value != owner || args[5].Value != int64(0) {
				t.Fatal("nonminimal market ownership", args)
			}
		}},
		soldSQLStep{contains: "FOR UPDATE", columns: []string{"payload", "expire_at"}, values: [][]driver.Value{{owner, int64(0)}}, check: func(args []driver.NamedValue) {
			if args[0].Value != int64(0) || args[1].Value != int64(91061) || args[2].Value != "*" || args[3].Value != "scratch-sold:7:ticket" {
				t.Fatal(args)
			}
		}},
		soldSQLStep{contains: "ON DUPLICATE KEY UPDATE", check: func(args []driver.NamedValue) {
			if args[0].Value != int64(42) || args[2].Value != "CNY" || args[3].Value != "scratch:ticket" {
				t.Fatal("paid ticket identity changed", args)
			}
		}},
	)
	if err := dd.SaveScratchGameStorage(context.Background(), 7, 42, 91061, "CNY", "scratch:ticket", `{"serialFull":"ticket","roundId":"original","userId":42,"currency":"CNY","huMask":[1,2]}`); err != nil {
		t.Fatal(err)
	}
	if len(script.steps) != 0 || script.log[0] != "begin" || script.log[len(script.log)-1] != "commit" {
		t.Fatal(script.log)
	}
}

func TestScratchOwnershipConflictAndSQLFailureRollbackBeforeDelivery(t *testing.T) {
	for _, fail := range []bool{false, true} {
		owner := soldOwnerRaw(43)
		if fail {
			owner = soldOwnerRaw(42)
		}
		steps := []soldSQLStep{{contains: "ON DUPLICATE KEY UPDATE"}, {contains: "FOR UPDATE", columns: []string{"payload", "expire_at"}, values: [][]driver.Value{{owner, int64(0)}}}}
		if fail {
			steps = append(steps, soldSQLStep{contains: "ON DUPLICATE KEY UPDATE", err: errors.New("ticket SQL failed")})
		}
		dd, script := soldTestDB(t, steps...)
		err := dd.SaveScratchGameStorage(context.Background(), 7, 42, 91061, "CNY", "scratch:ticket", `{"serialFull":"ticket","roundId":"original","userId":42,"currency":"CNY"}`)
		if err == nil || len(script.steps) != 0 || script.log[len(script.log)-1] != "rollback" {
			t.Fatal("unconfirmed ownership committed", err, script.log)
		}
	}
}

func TestScratchMarketLookupCoversOtherUsersAndCurrenciesWithIndexedAgentScope(t *testing.T) {
	dd, script := soldTestDB(t, soldSQLStep{contains: "storage_key IN", columns: []string{"storage_key", "payload", "expire_at"}, values: [][]driver.Value{{"scratch-sold:7:ticket", soldOwnerRaw(42), int64(0)}}, check: func(args []driver.NamedValue) {
		if len(args) != 5 || args[0].Value != int64(0) || args[1].Value != int64(91061) || args[2].Value != "*" || args[3].Value != "scratch-sold:7:ticket" || args[4].Value != "scratch-sold:7:free" {
			t.Fatal("query is not market scoped", args)
		}
	}})
	result, err := dd.ScratchSoldSerials(context.Background(), ScratchScope{7, 91061, 43, "USD"}, []string{"ticket", "free", "ticket"})
	if err != nil || !result["ticket"] || result["free"] || len(script.steps) != 0 {
		t.Fatal(result, err)
	}
}

func TestScratchMarketLookupRejectsCorruptOwnerInsteadOfTreatingSerialAsFree(t *testing.T) {
	for _, raw := range []string{`{}`, strings.Replace(soldOwnerRaw(42), `"agent":7`, `"agent":8`, 1), strings.Replace(soldOwnerRaw(42), `"serialFull":"ticket"`, `"serialFull":"foreign"`, 1)} {
		dd, _ := soldTestDB(t, soldSQLStep{contains: "storage_key IN", columns: []string{"storage_key", "payload", "expire_at"}, values: [][]driver.Value{{"scratch-sold:7:ticket", raw, int64(0)}}})
		if _, err := dd.ScratchSoldSerials(context.Background(), ScratchScope{7, 91061, 42, "CNY"}, []string{"ticket"}); err == nil {
			t.Fatal("corrupt owner treated as free", raw)
		}
	}
	dd, _ := soldTestDB(t, soldSQLStep{contains: "storage_key IN", err: errors.New("SQL offline")})
	if _, err := dd.ScratchSoldSerials(context.Background(), ScratchScope{7, 91061, 42, "CNY"}, []string{"ticket"}); err == nil {
		t.Fatal("lookup failure treated as free")
	}
}
