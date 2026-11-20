package dao

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/olivere/elastic/v7"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSettlementCursorReadsExactAndBeyondTenThousandWithoutOffsetOrExtraSourceCopies(t *testing.T) {
	for _, total := range []int{10000, 10001} {
		t.Run(fmt.Sprint(total), func(t *testing.T) {
			calls, visited := 0, 0
			_, es, _ := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
				var query map[string]any
				if json.NewDecoder(r.Body).Decode(&query) != nil {
					t.Fatal("invalid query")
				}
				if query["from"] != nil {
					t.Error("offset used", query)
				}
				if query["size"] != float64(100) {
					t.Error("unbounded page", query)
				}
				source, _ := json.Marshal(query["_source"])
				if !strings.Contains(string(source), `"log"`) || strings.Contains(string(source), `"init"`) {
					t.Error("source copied wrong detail", string(source))
				}
				start := calls * 100
				if calls > 0 {
					cursor, ok := query["search_after"].([]any)
					if !ok || len(cursor) != 2 || cursor[1] != fmt.Sprintf("h-%05d", start-1) {
						t.Error("cursor did not use last original document", query)
					}
				}
				calls++
				end := start + 100
				if end > total {
					end = total
				}
				hits := make([]any, 0)
				for i := start; i < end; i++ {
					hits = append(hits, map[string]any{"sort": []any{1700000000000 - i, fmt.Sprintf("h-%05d", i)}, "_source": map[string]any{"userId": 42, "agentId": 7, "complete": true, "hash": fmt.Sprintf("h-%05d", i), "log": "{}"}})
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{"hits": map[string]any{"hits": hits}})
			})
			err := es.scanSettlementDocuments(context.Background(), elastic.NewTermQuery("userId", 42), func(hit *elastic.SearchHit) (bool, error) { visited++; return false, nil })
			if err != nil || visited != total || calls != 101 {
				t.Fatal("valid large history failed or truncated", visited, calls, err)
			}
		})
	}
}

func TestSettlementCursorStopsWhenRequestedRowsAreComplete(t *testing.T) {
	calls, visited := 0, 0
	_, es, _ := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		hits := make([]any, 100)
		for i := range hits {
			hits[i] = map[string]any{"_source": map[string]any{"log": "original"}}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"hits": map[string]any{"hits": hits}})
	})
	err := es.scanSettlementDocuments(context.Background(), elastic.NewMatchAllQuery(), func(hit *elastic.SearchHit) (bool, error) { visited++; return visited == 3, nil })
	if err != nil || calls != 1 || visited != 3 {
		t.Fatal("early stop ignored", calls, visited, err)
	}
}

func TestSettlementCursorRejectsMissingOrRepeatedContinuationInsteadOfLooping(t *testing.T) {
	for _, missing := range []bool{true, false} {
		t.Run(fmt.Sprint(missing), func(t *testing.T) {
			calls := 0
			_, es, _ := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				hits := make([]any, 100)
				for i := range hits {
					row := map[string]any{"_source": map[string]any{"log": "{}"}}
					if !missing {
						row["sort"] = []any{123, "same-hash"}
					}
					hits[i] = row
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{"hits": map[string]any{"hits": hits}})
			})
			err := es.scanSettlementDocuments(context.Background(), elastic.NewMatchAllQuery(), func(*elastic.SearchHit) (bool, error) { return false, nil })
			if err == nil || calls > 2 {
				t.Fatal("invalid cursor did not stop", calls, err)
			}
		})
	}
}

func TestSettlementCursorRetainsByteLimitAndVisitorFailures(t *testing.T) {
	_, es, _ := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"hits": map[string]any{"hits": []any{map[string]any{"_source": map[string]any{"log": strings.Repeat("x", settlementScanBytes)}}}}})
	})
	visited := false
	err := es.scanSettlementDocuments(context.Background(), elastic.NewMatchAllQuery(), func(*elastic.SearchHit) (bool, error) { visited = true; return false, nil })
	if err == nil || visited {
		t.Fatal("oversized original detail bypassed read budget", err)
	}
	_, small, _ := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"hits":{"hits":[{"_source":{"log":"{}"}}]}}`)
	})
	wanted := errors.New("original document validation failed")
	err = small.scanSettlementDocuments(context.Background(), elastic.NewMatchAllQuery(), func(*elastic.SearchHit) (bool, error) { return false, wanted })
	if !errors.Is(err, wanted) {
		t.Fatal("validation failure hidden as empty history", err)
	}
}

func TestSettlementCursorHonorsCallerCancellation(t *testing.T) {
	_, es, _ := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(time.Second):
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	err := es.scanSettlementDocuments(ctx, elastic.NewMatchAllQuery(), func(*elastic.SearchHit) (bool, error) { return false, nil })
	if err == nil || ctx.Err() == nil {
		t.Fatal("cancelled search continued", err)
	}
}
