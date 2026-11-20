package dao

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
)

func TestAdminRuntimeReadHasNoWriteOrRowLock(t *testing.T) {
	db, script := gogameTestDB(t, gogameSQLStep{contains: "LEFT JOIN gg_match_runtime", columns: []string{"id", "agent_id", "state", "runtime_payload"}, values: [][]driver.Value{{int64(8), int64(7), int64(2), `{"registrations":{},"clock":{"currentBlindLv":3}}`}}})
	value, err := (&DBDao{manager: db}).GogameCall(context.Background(), "readMatch", `{"id":8}`)
	if err != nil {
		t.Fatal(err)
	}
	record := value.(map[string]any)
	if gogameNumber(record["row"].(map[string]any)["agent_id"]) != 7 {
		t.Fatal(record)
	}
	if len(script.steps) != 0 || len(script.log) != 1 {
		t.Fatal("read unexpectedly opened a transaction or wrote state", script.log)
	}
}

func TestOperatorTicketIssuanceAtomicReplayAndFailure(t *testing.T) {
	for _, kind := range []string{"new", "replay", "conflict", "inventory", "wrong-agent", "insert-failure"} {
		t.Run(kind, func(t *testing.T) {
			input := ticketAdminInput{ID: 3, UserID: 42, AgentID: 7, Count: 2, RequestID: "issue:test"}
			remain := int64(10)
			if kind == "inventory" {
				remain = 1
			}
			steps := []gogameSQLStep{{contains: "SELECT remain,disabled", columns: []string{"remain", "disabled"}, values: [][]driver.Value{{remain, int64(0)}}}}
			prior := gogameSQLStep{contains: "WHERE issue_key=?", columns: []string{"id", "ticket_id", "owner", "issue_agent"}}
			if kind == "replay" || kind == "conflict" {
				prior.values = [][]driver.Value{{int64(1), int64(3), int64(42), int64(7)}, {int64(2), int64(3), int64(42), int64(7)}}
				if kind == "conflict" {
					input.Count = 3
				}
			}
			steps = append(steps, prior)
			if kind == "new" || kind == "wrong-agent" || kind == "insert-failure" {
				account := gogameSQLStep{contains: "proxy_id=? FOR UPDATE", columns: []string{"user_id"}, values: [][]driver.Value{{int64(42)}}}
				if kind == "wrong-agent" {
					account.values = nil
				}
				steps = append(steps, account)
				if kind != "wrong-agent" {
					steps = append(steps, gogameSQLStep{contains: "remain=remain-?"})
					insert := gogameSQLStep{contains: "issue_key,issue_ordinal,issue_agent"}
					if kind == "insert-failure" {
						insert.err = errors.New("insert failed")
					}
					steps = append(steps, insert)
					if kind == "new" {
						steps = append(steps, insert)
					}
				}
			}
			db, script := gogameTestDB(t, steps...)
			result, err := (&DBDao{manager: db}).issueTicketsSQL(context.Background(), "`player`.`gg_ticket_hold`", input)
			ok := kind == "new" || kind == "replay"
			if (err == nil) != ok {
				t.Fatal(kind, err)
			}
			end := "rollback"
			if ok {
				end = "commit"
				if result["issued"] != int64(2) {
					t.Fatal(result)
				}
			}
			if len(script.steps) != 0 || script.log[len(script.log)-1] != end {
				t.Fatal(script.log)
			}
		})
	}
}

func TestDraftDeleteNeverErasesRegisteredOrFinancialHistory(t *testing.T) {
	for _, kind := range []string{"unused", "registered", "running"} {
		t.Run(kind, func(t *testing.T) {
			state := int64(0)
			if kind == "running" {
				state = 2
			}
			steps := []gogameSQLStep{{contains: "SELECT * FROM gg_match WHERE id=? FOR UPDATE", columns: []string{"id", "state"}, values: [][]driver.Value{{int64(8), state}}}}
			if kind != "running" {
				payload := `{"registrations":{},"prizes":[],"tables":[]}`
				if kind == "registered" {
					payload = `{"registrations":{"42":{"status":"refunded"}}}`
				}
				steps = append(steps, gogameSQLStep{contains: "SELECT payload FROM gg_match_runtime", columns: []string{"payload"}, values: [][]driver.Value{{payload}}})
				if kind == "unused" {
					steps = append(steps, gogameSQLStep{contains: "SELECT match_id FROM gg_match_occurrence", columns: []string{"match_id"}}, gogameSQLStep{contains: "SELECT id FROM gg_ticket", columns: []string{"id"}}, gogameSQLStep{contains: "DELETE FROM gg_match_runtime"}, gogameSQLStep{contains: "DELETE FROM gg_match WHERE id=? AND state=0"})
				}
			}
			db, script := gogameTestDB(t, steps...)
			_, err := (&DBDao{manager: db}).gogameDeleteDraft(context.Background(), 8)
			end := "rollback"
			if kind == "unused" {
				end = "commit"
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil {
				t.Fatal("unsafe delete accepted")
			}
			if len(script.steps) != 0 || script.log[len(script.log)-1] != end {
				t.Fatal(script.log)
			}
		})
	}
}
