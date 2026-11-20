package unified

import (
	"app/entity/view"
	"app/esindex"
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/olivere/elastic/v7"
	"lottery/dao"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPoolBulkLogUsesSharedGoIndexPrefix(t *testing.T) {
	var index string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scanner := bufio.NewScanner(r.Body)
		if !scanner.Scan() {
			t.Error("missing bulk header")
			return
		}
		var header map[string]map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &header); err != nil {
			t.Error(err)
			return
		}
		index, _ = header["index"]["_index"].(string)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"errors":false,"items":[{"index":{"status":201}}]}`)
	}))
	defer server.Close()
	client, err := elastic.NewClient(elastic.SetURL(server.URL), elastic.SetSniff(false), elastic.SetHealthcheck(false))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Stop()
	service := &LotteryService{es: dao.NewESDao(client)}
	if err = service.BulkPoolLog([]*view.PoolLogItem{{}}); err != nil {
		t.Fatal(err)
	}
	if index != esindex.PoolRecordLog() || index != "go_pool_record_log" {
		t.Fatalf("unexpected pool log index %q", index)
	}
}
