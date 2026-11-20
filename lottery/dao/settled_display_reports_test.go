package dao

import (
	"app/entity"
	"app/esindex"
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/olivere/elastic/v7"
	"net/http"
	"net/http/httptest"
	"testing"
)

func displayReportFixture() *entity.CacheRecordsReq {
	report := map[string]any{"id": "7:holdem:700000:42", "userId": 42, "agentId": 7, "currency": "CNY", "roomType": 19, "roomId": 700000, "inning": 1, "endedAt": 123456789, "effectiveBet": 100, "net": 100, "balance": -1, "detail": map[string]any{"replay": map[string]any{"original": true}}, "players": []any{}}
	payload, _ := json.Marshal(map[string]any{"score": 100, "completedDisplayReport": report})
	return &entity.CacheRecordsReq{AgentId: 7, UserId: 42, Currency: "CNY", Bet: 1, Win: 2, Balance: 12.34, Log: string(payload)}
}
func TestSettledDisplayProjectionUsesOriginalIdentityAndAuthoritativeMoney(t *testing.T) {
	record := displayReportFixture()
	report, err := settledDisplayDocuments(record)
	if err != nil {
		t.Fatal(err)
	}
	if report["balance"] != int64(1234) || report["id"] != "7:holdem:700000:42" {
		t.Fatal(report)
	}
	for _, field := range []string{"agentId", "userId", "currency", "net", "effectiveBet"} {
		var envelope map[string]any
		_ = json.Unmarshal([]byte(record.Log), &envelope)
		row := envelope["completedDisplayReport"].(map[string]any)
		if field == "currency" {
			row[field] = "USD"
		} else {
			row[field] = 999
		}
		raw, _ := json.Marshal(envelope)
		changed := *record
		changed.Log = string(raw)
		if _, err = settledDisplayDocuments(&changed); err == nil {
			t.Fatal("inconsistent display report accepted", field)
		}
	}
	legacy := *record
	legacy.Log = `{"score":100}`
	if report, err = settledDisplayDocuments(&legacy); err != nil || report != nil {
		t.Fatal("invented report for legacy result", report, err)
	}
}
func TestSettledDisplayBulkCreatesSameDocumentsAsDataCenterAndAcceptsOriginalRetry(t *testing.T) {
	headers := []map[string]map[string]any{}
	documents := []map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scanner := bufio.NewScanner(r.Body)
		for scanner.Scan() {
			var header map[string]map[string]any
			_ = json.Unmarshal(scanner.Bytes(), &header)
			headers = append(headers, header)
			scanner.Scan()
			var document map[string]any
			_ = json.Unmarshal(scanner.Bytes(), &document)
			documents = append(documents, document)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"errors":true,"items":[{"create":{"status":409}},{"create":{"status":201}}]}`)
	}))
	defer server.Close()
	client, err := elastic.NewClient(elastic.SetURL(server.URL), elastic.SetSniff(false), elastic.SetHealthcheck(false))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Stop()
	record := displayReportFixture()
	if err = NewESDao(client).BulkSettledDisplayReports([]*entity.CacheRecordsReq{record}); err != nil {
		t.Fatal(err)
	}
	id := fmt.Sprintf("%x", sha256.Sum256([]byte("7:42:CNY:7:holdem:700000:42")))
	if len(headers) != 2 || headers[0]["create"]["_id"] != id || headers[1]["create"]["_id"] != id || headers[0]["create"]["_index"] != esindex.Name("gogame_records") || headers[1]["create"]["_index"] != esindex.Name("gogame_replays") {
		t.Fatal(headers)
	}
	if documents[0]["balance"] != float64(1234) {
		t.Fatal(documents)
	}
}
