package dao

import (
	"app/entity"
	"bufio"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/olivere/elastic/v7"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

type finalSQLStep struct {
	contains string
	columns  []string
	values   [][]driver.Value
	err      error
	check    func([]driver.NamedValue)
}
type finalSQLScript struct {
	t     *testing.T
	steps []finalSQLStep
	log   []string
}
type finalSQLDriver struct{ script *finalSQLScript }
type finalSQLConn struct{ script *finalSQLScript }
type finalSQLTx struct{ script *finalSQLScript }
type finalSQLRows struct {
	columns []string
	values  [][]driver.Value
}

var finalSQLSequence atomic.Int64

func (d finalSQLDriver) Open(string) (driver.Conn, error) { return &finalSQLConn{d.script}, nil }
func (c *finalSQLConn) Close() error                      { return nil }
func (c *finalSQLConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (c *finalSQLConn) Begin() (driver.Tx, error) {
	c.script.log = append(c.script.log, "begin")
	return &finalSQLTx{c.script}, nil
}
func (tx *finalSQLTx) Commit() error   { tx.script.log = append(tx.script.log, "commit"); return nil }
func (tx *finalSQLTx) Rollback() error { tx.script.log = append(tx.script.log, "rollback"); return nil }
func (c *finalSQLConn) step(query string, args []driver.NamedValue) finalSQLStep {
	c.script.t.Helper()
	c.script.log = append(c.script.log, query)
	if len(c.script.steps) == 0 {
		c.script.t.Fatal("unexpected SQL", query)
	}
	step := c.script.steps[0]
	c.script.steps = c.script.steps[1:]
	if !strings.Contains(query, step.contains) {
		c.script.t.Fatalf("SQL %q missing %q", query, step.contains)
	}
	if step.check != nil {
		step.check(args)
	}
	return step
}
func (c *finalSQLConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	step := c.step(query, args)
	return driver.RowsAffected(1), step.err
}
func (c *finalSQLConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	step := c.step(query, args)
	return &finalSQLRows{step.columns, step.values}, step.err
}
func (r *finalSQLRows) Columns() []string { return r.columns }
func (r *finalSQLRows) Close() error      { return nil }
func (r *finalSQLRows) Next(values []driver.Value) error {
	if len(r.values) == 0 {
		return io.EOF
	}
	copy(values, r.values[0])
	r.values = r.values[1:]
	return nil
}
func finalReportDB(t *testing.T, steps ...finalSQLStep) (*DBDao, *finalSQLScript) {
	script := &finalSQLScript{t: t, steps: steps}
	name := fmt.Sprintf("final-record-%d", finalSQLSequence.Add(1))
	sql.Register(name, finalSQLDriver{script})
	connection, err := sql.Open(name, "")
	if err != nil {
		t.Fatal(err)
	}
	connection.SetMaxOpenConns(1)
	t.Cleanup(func() { connection.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: connection, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	return &DBDao{player: db}, script
}

func TestFinalRecordAndBillStoredInExistingTableBeforeReturn(t *testing.T) {
	record := &entity.CacheRecordsReq{AgentId: 7, UserId: 42, Currency: "CNY", RoundID: "original", RowVersion: 10, PlayedDate: 123, Complete: true, Log: `{"cards":[1,2]}`}
	bill := &entity.CacheBillsReq{AgentId: 7, UserId: 42, Currency: "CNY", RoundID: "original", FlowingWaterOn: "actual-bill", Bet: -1, Award: 2, CurrentScore: 10, CreateTime: 123}
	payloads := []string{}
	check := func(args []driver.NamedValue) {
		if args[0].Value != int64(42) || args[1].Value != "CNY" || !strings.HasPrefix(args[2].Value.(string), financialReportPrefix) {
			t.Fatal(args)
		}
		payloads = append(payloads, args[3].Value.(string))
	}
	storedRecord, _ := financialRecord(record)
	recordJSON, _ := json.Marshal(storedRecord)
	storedBill, _ := financialBill(bill)
	billJSON, _ := json.Marshal(storedBill)
	db, script := finalReportDB(t, finalSQLStep{contains: "ON DUPLICATE KEY UPDATE storage_key=VALUES(storage_key)", check: check},
		finalSQLStep{contains: "SELECT payload FROM fg_game_storage", columns: []string{"payload"}, values: [][]driver.Value{{string(recordJSON)}}},
		finalSQLStep{contains: "ON DUPLICATE KEY UPDATE storage_key=VALUES(storage_key)", check: check},
		finalSQLStep{contains: "SELECT payload FROM fg_game_storage", columns: []string{"payload"}, values: [][]driver.Value{{string(billJSON)}}})
	if err := db.StoreFinancialRecords(context.Background(), []*entity.CacheRecordsReq{record}, []*entity.CacheBillsReq{bill}); err != nil {
		t.Fatal(err)
	}
	if len(script.steps) != 0 || script.log[0] != "begin" || script.log[len(script.log)-1] != "commit" {
		t.Fatal(script.log)
	}
	var stored finalFinancialRecord
	if json.Unmarshal([]byte(payloads[0]), &stored) != nil || stored.Record.Log != record.Log || stored.Record.PlayedDate != 123 || stored.Record.RowVersion != 10 {
		t.Fatal(payloads)
	}
	if json.Unmarshal([]byte(payloads[1]), &stored) != nil || stored.Bill.FlowingWaterOn != "actual-bill" || stored.Bill.CreateTime != 123 {
		t.Fatal(payloads)
	}
	record.Log = "changed after SaveRecord"
	if strings.Contains(payloads[0], "changed after") {
		t.Fatal("stored snapshot kept mutable pointer")
	}
}

func TestFinalRecordStorageFailureRollsBackAndCannotPretendDurable(t *testing.T) {
	record := &entity.CacheRecordsReq{AgentId: 7, UserId: 42, Currency: "CNY", RoundID: "original", RowVersion: 10}
	db, script := finalReportDB(t, finalSQLStep{contains: "INSERT INTO fg_game_storage", err: errors.New("SQL unavailable")})
	if err := db.StoreFinancialRecords(context.Background(), []*entity.CacheRecordsReq{record}, nil); err == nil {
		t.Fatal("SQL failure hidden")
	}
	if script.log[len(script.log)-1] != "rollback" {
		t.Fatal(script.log)
	}
}

func TestSameFinancialVersionCannotReplaceAnotherOriginalPayload(t *testing.T) {
	record := &entity.CacheRecordsReq{AgentId: 7, UserId: 42, Currency: "CNY", RoundID: "original", RowVersion: 10, Log: "new"}
	previous := *record
	previous.Log = "old"
	row, _ := financialRecord(&previous)
	raw, _ := json.Marshal(row)
	db, script := finalReportDB(t, finalSQLStep{contains: "ON DUPLICATE KEY UPDATE"}, finalSQLStep{contains: "FOR UPDATE", columns: []string{"payload"}, values: [][]driver.Value{{string(raw)}}})
	if err := db.StoreFinancialRecords(context.Background(), []*entity.CacheRecordsReq{record}, nil); err == nil {
		t.Fatal("same version replaced original content")
	}
	if script.log[len(script.log)-1] != "rollback" {
		t.Fatal(script.log)
	}
}

func TestFinalRecordExternalVersionPreventsOlderInstanceFromOverwritingNewResult(t *testing.T) {
	highest := int64(0)
	final := ""
	billDocs := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scanner := bufio.NewScanner(r.Body)
		statuses := []int{}
		for scanner.Scan() {
			var header map[string]map[string]any
			_ = json.Unmarshal(scanner.Bytes(), &header)
			scanner.Scan()
			var row map[string]any
			_ = json.Unmarshal(scanner.Bytes(), &row)
			if request, ok := header["index"]; ok {
				if request["version_type"] != "external_gte" {
					t.Error("record has no external version")
				}
				version := int64(request["version"].(float64))
				if version < highest {
					statuses = append(statuses, 409)
				} else {
					highest = version
					final = row["log"].(string)
					statuses = append(statuses, 201)
				}
			} else {
				request := header["create"]
				id := request["_id"].(string)
				if billDocs[id] {
					statuses = append(statuses, 409)
				} else {
					billDocs[id] = true
					statuses = append(statuses, 201)
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"items":[`)
		for i, code := range statuses {
			if i > 0 {
				fmt.Fprint(w, ",")
			}
			fmt.Fprintf(w, `{"index":{"status":%d}}`, code)
		}
		fmt.Fprint(w, `]}`)
	}))
	defer server.Close()
	client, err := elastic.NewClient(elastic.SetURL(server.URL), elastic.SetSniff(false), elastic.SetHealthcheck(false))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Stop()
	es := NewESDao(client)
	record := &entity.CacheRecordsReq{AgentId: 7, UserId: 42, Currency: "CNY", RoundID: "original", RowVersion: 20, Log: "new"}
	latest, _ := financialRecord(record)
	record.RowVersion = 10
	record.Log = "old"
	old, _ := financialRecord(record)
	bill, _ := financialBill(&entity.CacheBillsReq{AgentId: 7, UserId: 42, Currency: "CNY", FlowingWaterOn: "original"})
	for _, rows := range [][]finalFinancialRecord{{latest, bill}, {old, bill}, {latest, bill}} {
		if err = es.indexFinalFinancialRows(context.Background(), rows); err != nil {
			t.Fatal(err)
		}
	}
	if highest != 20 || final != "new" || len(billDocs) != 1 {
		t.Fatal(highest, final, billDocs)
	}
}

func TestNewDatabaseHandleProjectsOriginalRecordAndRetainsItOnESFailure(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(fmt.Sprint(failed), func(t *testing.T) {
			record := &entity.CacheRecordsReq{AgentId: 7, UserId: 42, Currency: "CNY", RoundID: "original", RowVersion: 10, Complete: true, Log: "original"}
			row, _ := financialRecord(record)
			raw, _ := json.Marshal(row)
			step := finalSQLStep{contains: "FROM `fg_game_storage`", columns: []string{"id", "user_id", "game_id", "currency_symbol", "storage_key", "payload", "expire_at", "updated_at"}, values: [][]driver.Value{{int64(8), int64(42), int64(0), "CNY", "final-record:original", string(raw), int64(0), int64(123)}}}
			steps := []finalSQLStep{step}
			if !failed {
				steps = append(steps, finalSQLStep{contains: "UPDATE `fg_game_storage` SET `expire_at`=?", check: func(args []driver.NamedValue) {
					if args[0].Value.(int64) <= 0 {
						t.Fatal(args)
					}
				}})
			}
			db, script := finalReportDB(t, steps...)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if failed {
					w.WriteHeader(503)
					fmt.Fprint(w, `{"error":"offline"}`)
				} else {
					fmt.Fprint(w, `{"items":[{"index":{"status":201}}]}`)
				}
			}))
			defer server.Close()
			client, err := elastic.NewClient(elastic.SetURL(server.URL), elastic.SetSniff(false), elastic.SetHealthcheck(false), elastic.SetMaxRetries(0))
			if err != nil {
				t.Fatal(err)
			}
			defer client.Stop()
			count, err := db.ProjectFinalFinancialRecords(context.Background(), NewESDao(client))
			if (err != nil) != failed {
				t.Fatal(count, err)
			}
			if !failed && count != 1 {
				t.Fatal(count)
			}
			if len(script.steps) != 0 {
				t.Fatal(script.log)
			}
		})
	}
}
