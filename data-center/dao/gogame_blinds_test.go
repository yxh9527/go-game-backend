package dao

import (
	"context"
	"database/sql/driver"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"testing"
)

func TestBlindReferencesIncludeSchedulePayloads(t *testing.T) {
	for _, test := range []struct {
		table  string
		values map[string]any
		want   int64
		bad    bool
	}{
		{"gg_match", map[string]any{"blinds_id": int64(12)}, 12, false},
		{"gg_match_schedule", map[string]any{"payload": `{"blinds_id":12}`}, 12, false},
		{"gg_match_schedule", map[string]any{"payload": `{"blinds_id":"12"}`}, 12, false},
		{"gg_match", map[string]any{}, 0, false},
		{"gg_match", map[string]any{"blinds_id": 0}, 0, false},
		{"gg_match", map[string]any{"blinds_id": 1.5}, 0, true},
		{"gg_match", map[string]any{"blinds_id": -1}, 0, true},
		{"gg_match_schedule", map[string]any{"payload": "invalid"}, 0, true},
	} {
		got, err := gogameBlindReference(test.table, test.values)
		if (err != nil) != test.bad || got != test.want {
			t.Fatalf("%+v got=%d err=%v", test, got, err)
		}
	}
}

func TestDeleteBlindsRejectsReferencesAndRollsBack(t *testing.T) {
	for _, kind := range []string{"match", "schedule", "unused", "missing"} {
		t.Run(kind, func(t *testing.T) {
			first := gogameSQLStep{contains: "gg_blinds WHERE id=? FOR UPDATE", columns: []string{"id", "name"}, values: [][]driver.Value{{int64(12), "template"}}}
			if kind == "missing" {
				first.values = nil
			}
			steps := []gogameSQLStep{first}
			if kind != "missing" {
				step := gogameSQLStep{contains: "gg_match WHERE blinds_id=?", columns: []string{"id"}}
				if kind == "match" {
					step.values = [][]driver.Value{{int64(7)}}
				}
				steps = append(steps, step)
				if kind != "match" {
					step = gogameSQLStep{contains: "JSON_EXTRACT(payload,'$.blinds_id')", columns: []string{"id"}}
					if kind == "schedule" {
						step.values = [][]driver.Value{{int64(8)}}
					}
					steps = append(steps, step)
					if kind == "unused" {
						steps = append(steps, gogameSQLStep{contains: "DELETE FROM gg_blinds WHERE id=?"})
					}
				}
			}
			db, script := gogameTestDB(t, steps...)
			saved, err := (&DBDao{manager: db}).gogameDeleteBlinds(context.Background(), 12)
			want := codes.FailedPrecondition
			if kind == "missing" {
				want = codes.NotFound
			}
			if kind == "unused" {
				want = codes.OK
			}
			if status.Code(err) != want {
				t.Fatalf("err=%v", err)
			}
			end := "rollback"
			if kind == "unused" {
				end = "commit"
				if gogameNumber(saved["id"]) != 12 {
					t.Fatal(saved)
				}
			}
			if len(script.steps) != 0 || script.log[len(script.log)-1] != end {
				t.Fatal(script.log)
			}
		})
	}
}

func TestChangingMatchToMissingBlindTemplateNeverWrites(t *testing.T) {
	db, script := gogameTestDB(t, gogameSQLStep{contains: "SELECT id FROM gg_blinds WHERE id=? FOR UPDATE", columns: []string{"id"}})
	err := gogameUpdate(context.Background(), db, "gg_match", 8, map[string]any{"blinds_id": int64(12)})
	if status.Code(err) != codes.FailedPrecondition || strings.Join(script.log, "|") == "" || script.log[len(script.log)-1] != "rollback" {
		t.Fatal(err, script.log)
	}
}
