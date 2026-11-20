package dao

import (
	"app/batchwriter"
	"app/entity"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/olivere/elastic/v7"
)

func TestBulkRetryWithStableRecordAndBillIDs(t *testing.T) {
	for _, kind := range []string{"record", "bill"} {
		t.Run(kind, func(t *testing.T) {
			var ids [][]string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				scan := bufio.NewScanner(r.Body)
				var current []string
				for scan.Scan() {
					var head map[string]map[string]any
					if err := json.Unmarshal(scan.Bytes(), &head); err != nil {
						t.Error(err)
						return
					}
					current = append(current, head["index"]["_id"].(string))
					scan.Scan()
				}
				ids = append(ids, current)
				w.Header().Set("Content-Type", "application/json")
				if len(ids) == 1 {
					w.WriteHeader(503)
					fmt.Fprint(w, `{"error":"temporarily unavailable","status":503}`)
					return
				}
				fmt.Fprint(w, `{"errors":false,"items":[{"index":{"status":201}},{"index":{"status":200}}]}`)
			}))
			defer server.Close()
			client, err := elastic.NewClient(elastic.SetURL(server.URL), elastic.SetSniff(false), elastic.SetHealthcheck(false), elastic.SetMaxRetries(0))
			if err != nil {
				t.Fatal(err)
			}
			defer client.Stop()
			es := NewESDao(client)
			in := make(chan int, 2)
			in <- 1
			in <- 2
			close(in)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			left := batchwriter.Run(ctx, in, 2, time.Millisecond, func(batch []int) error {
				if kind == "record" {
					return es.BulkRecordsSave([]*entity.CacheRecordsReq{{AgentId: 1, UserId: 10, RoundID: "round-1"}, {AgentId: 1, UserId: 20, RoundID: "round-1"}})
				}
				return es.BulkBillsSave([]*entity.CacheBillsReq{{AgentId: 1, UserId: 10, FlowingWaterOn: "bill-1"}, {AgentId: 1, UserId: 10, FlowingWaterOn: "bill-2"}})
			}, nil)
			if len(left) != 0 || len(ids) != 2 || !reflect.DeepEqual(ids[0], ids[1]) || ids[0][0] == ids[0][1] || ids[0][0] == "" {
				t.Fatalf("left=%v ids=%v", left, ids)
			}
		})
	}
}

func TestBulkRejectsHTTP200WithFailedOrMissingItems(t *testing.T) {
	for _, response := range []string{
		`{"errors":true,"items":[{"index":{"status":201}},{"index":{"status":429,"error":{"type":"rejected","reason":"busy"}}}]}`,
		`{"errors":false,"items":[{"index":{"status":400}},{"index":{"status":201}}]}`,
		`{"errors":false,"items":[{"index":{"status":201}}]}`,
	} {
		t.Run(fmt.Sprint(len(response)), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, response)
			}))
			defer server.Close()
			client, err := elastic.NewClient(elastic.SetURL(server.URL), elastic.SetSniff(false), elastic.SetHealthcheck(false), elastic.SetMaxRetries(0))
			if err != nil {
				t.Fatal(err)
			}
			defer client.Stop()
			es := NewESDao(client)
			if err := es.BulkRecordsSave([]*entity.CacheRecordsReq{{RoundID: "a"}, {RoundID: "b"}}); err == nil {
				t.Fatal("record failure swallowed")
			}
			if err := es.BulkBillsSave([]*entity.CacheBillsReq{{FlowingWaterOn: "a"}, {FlowingWaterOn: "b"}}); err == nil {
				t.Fatal("bill failure swallowed")
			}
			if err := es.BulkBillsSave([]*entity.CacheBillsReq{{}}); err == nil || !strings.Contains(err.Error(), "identity") {
				t.Fatal("missing bill identity accepted")
			}
		})
	}
}
