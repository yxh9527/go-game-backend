package dao

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func reportFixture() UserCenterRecord {
	return UserCenterRecord{ID: "report-one", UserID: 42, AgentID: 7, Currency: "CNY", GameID: "holdem", RoomType: 19, RoomID: 700000, Inning: 1, EndedAt: time.Now().UnixMilli(), EffectiveBet: 100, Net: 50, Players: json.RawMessage(`[]`), Detail: json.RawMessage(`{}`)}
}

func TestFinalReportStoredBeforeAcknowledgementEvenWhenESFails(t *testing.T) {
	for _, outage := range []bool{false, true} {
		t.Run(fmt.Sprint(outage), func(t *testing.T) {
			record := reportFixture()
			payload, _ := json.Marshal(centerReport{Index: centerIndex(), Record: record})
			dd, es, _ := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if outage {
					w.WriteHeader(503)
					fmt.Fprint(w, `{"error":"offline"}`)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"errors":false,"items":[{"create":{"status":201}}]}`)
			})
			steps := []gogameSQLStep{{contains: "INSERT INTO fg_game_storage", check: func(args []driver.NamedValue) {
				if args[3].Value != string(payload) {
					t.Fatal(args)
				}
			}}, {contains: "SELECT payload FROM fg_game_storage", columns: []string{"payload"}, values: [][]driver.Value{{string(payload)}}}}
			if !outage {
				steps = append(steps, gogameSQLStep{contains: "SET expire_at=?"})
			}
			db, script := gogameTestDB(t, steps...)
			dd.db = db
			result, err := dd.WriteUserCenter(context.Background(), es, []UserCenterRecord{record})
			if err != nil {
				t.Fatal(err)
			}
			receipt := result.(map[string]any)
			if receipt["durable"] != true || receipt["stored"] != 1 || receipt["indexed"] == outage {
				t.Fatal(receipt)
			}
			if len(script.steps) != 0 || script.log[0] != "begin" {
				t.Fatal(script.log)
			}
			// The SQL commit precedes any later acknowledgement/index marker.
			commit := -1
			for i, entry := range script.log {
				if entry == "commit" {
					commit = i
				}
			}
			if commit < 0 {
				t.Fatal("final record was not committed", script.log)
			}
		})
	}
}

func TestRecordRetriesPreserveOriginalAndRejectConflictingPayloads(t *testing.T) {
	for _, changed := range []bool{false, true} {
		t.Run(fmt.Sprint(changed), func(t *testing.T) {
			record := reportFixture()
			original, _ := json.Marshal(centerReport{Index: centerIndex(), Record: record})
			if changed {
				record.Net++
			}
			db, script := gogameTestDB(t, gogameSQLStep{contains: "INSERT INTO fg_game_storage"}, gogameSQLStep{contains: "FOR UPDATE", columns: []string{"payload"}, values: [][]driver.Value{{string(original)}}})
			_, err := (&DBDao{db: db}).storeCenterReportsSQL(context.Background(), []UserCenterRecord{record}, centerIndex())
			if (err != nil) != changed {
				t.Fatal(err)
			}
			end := "commit"
			if changed {
				end = "rollback"
			}
			if script.log[len(script.log)-1] != end {
				t.Fatal(script.log)
			}
		})
	}
}

func TestNewProcessProjectsStoredReportWithoutCallingWalletOrRecreatingRecord(t *testing.T) {
	record := reportFixture()
	payload, _ := json.Marshal(centerReport{Index: centerIndex(), Record: record})
	key := "uc-report:old-process"
	dd, es, _ := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"errors":true,"items":[{"create":{"status":409}}]}`)
	})
	db, script := gogameTestDB(t, gogameSQLStep{contains: "WHERE game_id=0 AND expire_at=0", columns: []string{"storage_key", "payload"}, values: [][]driver.Value{{key, string(payload)}}}, gogameSQLStep{contains: "UPDATE fg_game_storage SET expire_at=?", check: func(args []driver.NamedValue) {
		if args[1].Value != key {
			t.Fatal(args)
		}
	}})
	dd.db = db
	count, err := dd.FlushCenterReports(context.Background(), es)
	if err != nil || count != 1 || len(script.steps) != 0 {
		t.Fatal(count, err, script.log)
	}
}

func TestSQLFailureCannotBeAcknowledgedAsDurable(t *testing.T) {
	record := reportFixture()
	dd, es, _ := centerFixture(t, func(http.ResponseWriter, *http.Request) { t.Fatal("ES called before final record stored") })
	db, script := gogameTestDB(t, gogameSQLStep{contains: "INSERT INTO fg_game_storage", err: errors.New("SQL unavailable")})
	dd.db = db
	if _, err := dd.WriteUserCenter(context.Background(), es, []UserCenterRecord{record}); err == nil {
		t.Fatal("storage failure acknowledged")
	}
	if script.log[len(script.log)-1] != "rollback" {
		t.Fatal(script.log)
	}
}

func TestReportExpiryNeverDeletesUnindexedOrGameRecoveryRecords(t *testing.T) {
	db, script := gogameTestDB(t, gogameSQLStep{contains: "WHERE game_id=0 AND storage_key LIKE ? AND expire_at>0 AND expire_at<? LIMIT 100", check: func(args []driver.NamedValue) {
		if args[0].Value != "uc-report:%" {
			t.Fatal(args)
		}
	}})
	if err := (&DBDao{db: db}).ExpireIndexedCenterReports(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(script.steps) != 0 {
		t.Fatal(script.log)
	}
}
