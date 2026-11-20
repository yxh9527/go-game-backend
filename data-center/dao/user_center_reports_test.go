package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"strings"
	"testing"
	"time"
)

func reportFixture() UserCenterRecord {
	return UserCenterRecord{ID: "report-one", UserID: 42, AgentID: 7, Currency: "CNY", GameID: "holdem", RoomType: 19, RoomID: 700000, Inning: 1, EndedAt: time.Now().UnixMilli(), EffectiveBet: 100, Net: 50, Players: json.RawMessage(`[]`), Detail: json.RawMessage(`{}`)}
}

func TestReportsWriteDirectlyToESWithoutMySQLRawStorage(t *testing.T) {
	for _, kind := range []string{"cash", "replay"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			dd, es, _ := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"errors":false,"items":[{"create":{"status":201}}]}`)
			})
			db, script := gogameTestDB(t)
			dd.db = db
			record := reportFixture()
			var result any
			var err error
			if kind == "replay" {
				record.RoomType = 101
				result, err = dd.WriteHoldemReplays(context.Background(), es, []UserCenterRecord{record})
			} else {
				result, err = dd.WriteUserCenter(context.Background(), es, []UserCenterRecord{record})
			}
			if err != nil {
				t.Fatal(err)
			}
			receipt := result.(map[string]any)
			if receipt["stored"] != 1 || receipt["indexed"] != true || receipt["durable"] != true || calls != 1 || len(script.log) != 0 {
				t.Fatal(receipt, calls, script.log)
			}
		})
	}
}

func TestESFailureOrPartialBulkCannotAcknowledgeReportAndRetryUsesOriginalIDs(t *testing.T) {
	calls := 0
	ids := []string{}
	record := reportFixture()
	dd, es, _ := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"_id":%q,"found":true,"_source":%s}`, strings.TrimPrefix(r.URL.Path, "/go_gogame_records/_doc/"), mustReportJSON(t, record))
			return
		}
		calls++
		w.Header().Set("Content-Type", "application/json")
		var head map[string]map[string]any
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&head); err != nil {
			t.Error(err)
			return
		}
		ids = append(ids, head["create"]["_id"].(string))
		if calls == 1 {
			w.WriteHeader(503)
			fmt.Fprint(w, `{"error":"offline"}`)
			return
		}
		if calls == 2 {
			fmt.Fprint(w, `{"errors":true,"items":[{"create":{"status":429}}]}`)
			return
		}
		fmt.Fprint(w, `{"errors":true,"items":[{"create":{"status":409}}]}`)
	})
	for i := 0; i < 2; i++ {
		if _, err := dd.WriteUserCenter(context.Background(), es, []UserCenterRecord{record}); err == nil {
			t.Fatal("failed ES write acknowledged")
		}
	}
	value, err := dd.WriteUserCenter(context.Background(), es, []UserCenterRecord{record})
	if err != nil {
		t.Fatal(err)
	}
	if value.(map[string]any)["indexed"] != true || calls != 3 || ids[0] != ids[1] || ids[1] != ids[2] {
		t.Fatal(value, ids)
	}
}

func TestDirectESReportsRejectMissingESAndForeignOwnersWithoutCreatingSQLCopies(t *testing.T) {
	dd, es, _ := centerFixture(t, func(http.ResponseWriter, *http.Request) { t.Fatal("unauthorized record reached ES") })
	record := reportFixture()
	if _, err := dd.WriteUserCenter(context.Background(), nil, []UserCenterRecord{record}); err == nil {
		t.Fatal("missing ES acknowledged")
	}
	record.AgentID = 8
	if _, err := dd.WriteUserCenter(context.Background(), es, []UserCenterRecord{record}); err == nil {
		t.Fatal("foreign owner acknowledged")
	}
}

func mustReportJSON(t *testing.T, record UserCenterRecord) string {
	t.Helper()
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestESConflictVerifiesOriginalPayloadAndRejectsChangedOrUnavailableDocuments(t *testing.T) {
	for _, variant := range []string{"same", "changed", "missing", "failed"} {
		t.Run(variant, func(t *testing.T) {
			record := reportFixture()
			gets := 0
			dd, es, _ := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method != "GET" {
					fmt.Fprint(w, `{"errors":true,"items":[{"create":{"status":409}}]}`)
					return
				}
				gets++
				if variant == "failed" {
					w.WriteHeader(503)
					fmt.Fprint(w, `{"error":"offline"}`)
					return
				}
				if variant == "missing" {
					w.WriteHeader(404)
					fmt.Fprint(w, `{"found":false}`)
					return
				}
				original := record
				if variant == "changed" {
					original.Net++
				}
				fmt.Fprintf(w, `{"_id":%q,"found":true,"_source":%s}`, strings.TrimPrefix(r.URL.Path, "/go_gogame_records/_doc/"), mustReportJSON(t, original))
			})
			_, err := dd.WriteUserCenter(context.Background(), es, []UserCenterRecord{record})
			if variant == "same" {
				if err != nil || gets != 1 {
					t.Fatal(err, gets)
				}
				return
			}
			if err == nil || gets != 1 {
				t.Fatal("unconfirmed same-ID report was acknowledged", err, gets)
			}
			if variant == "changed" && status.Code(err) != codes.FailedPrecondition {
				t.Fatal(err)
			}
		})
	}
}
