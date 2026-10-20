package rpc

import (
	"context"
	"encoding/json"
	"lottery/dao"
	"math"
	"micro_service/services"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/olivere/elastic/v7"
	"github.com/shopspring/decimal"
)

func TestLuck5AmountsAreExactCents(t *testing.T) {
	for raw, expected := range map[string]int64{"0.00": 0, "10": 1000, "1.01": 101, "90071992547409.93": 9007199254740993, "92233720368547758.07": math.MaxInt64} {
		actual, ok := parseLuck5Amount(raw)
		if !ok || actual != expected {
			t.Fatalf("%s => %d %v", raw, actual, ok)
		}
	}
	for _, raw := range []string{"", "-1", "NaN", "1.001", "1e2", " 1.00", "92233720368547758.08"} {
		if _, ok := parseLuck5Amount(raw); ok {
			t.Fatalf("accepted %s", raw)
		}
	}
}

func TestLuck5RecordIdentityAndTimestamp(t *testing.T) {
	req := &services.Luck5SettleReq{UserId: 279, AgentId: 0, GameId: 7012, CurrencyType: "CNY", RoundId: "round", RecordJson: `{"recordId":"round","userId":279,"gameId":7012,"currencySymbol":"CNY","createdAtMs":1791874388273}`}
	if !validateLuck5Record(req) {
		t.Fatal("valid record rejected")
	}
	for _, record := range []string{`{"recordId":"other","userId":279,"gameId":7012,"currencySymbol":"CNY"}`, `{"recordId":"round","userId":279,"gameId":7012,"agentId":4,"currencySymbol":"CNY"}`, `{"recordId":"round","userId":279,"gameId":7012,"currencySymbol":"CNY","createdAtMs":-1}`, `null`} {
		req.RecordJson = record
		if validateLuck5Record(req) {
			t.Fatalf("accepted %s", record)
		}
	}
}

func TestLuck5RecordSaveReportsFailureAndUsesSameDocument(t *testing.T) {
	fail := true
	ids := []string{}
	logs := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ids = append(ids, request.URL.Path)
		var body map[string]any
		json.NewDecoder(request.Body).Decode(&body)
		logs = append(logs, body["log"].(string))
		writer.Header().Set("Content-Type", "application/json")
		if fail {
			writer.WriteHeader(500)
			writer.Write([]byte(`{"error":{"type":"unavailable","reason":"test"},"status":500}`))
			return
		}
		writer.Write([]byte(`{"_index":"test","_id":"record","_version":1,"result":"created"}`))
	}))
	defer server.Close()
	client, err := elastic.NewClient(elastic.SetURL(server.URL), elastic.SetSniff(false), elastic.SetHealthcheck(false))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Stop()
	service := &LotteryService{es: dao.NewESDao(client)}
	round := &dao.Luck5Round{UserId: 279, GameId: 7012, CurrencyType: "CNY", RoundId: "round", BetCent: 1000, WinCent: 40000, Balance: 139000, Symbol: "lucky", Exchange: decimal.NewFromInt(1), RecordJson: `{"recordId":"round","createdAtMs":1791874388273,"cards":[1,2,3,4,5]}`}
	if err := service.saveLuck5Record(context.Background(), round); err == nil {
		t.Fatal("failed record marked successful")
	}
	fail = false
	if err := service.saveLuck5Record(context.Background(), round); err != nil {
		t.Fatal(err)
	}
	if len(ids) < 2 || ids[0] != ids[len(ids)-1] || logs[0] != round.RecordJson || logs[len(logs)-1] != round.RecordJson {
		t.Fatal("record retry changed document or payload")
	}
}
