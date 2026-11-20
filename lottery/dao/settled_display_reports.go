package dao

import (
	"app/entity"
	"app/esindex"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/olivere/elastic/v7"
	"github.com/shopspring/decimal"
	"time"
)

// The display document is captured by the game before settlement commit and
// travels in the existing immutable result. No missing cards/results are made up.
func settledDisplayDocuments(record *entity.CacheRecordsReq) (map[string]any, error) {
	if record == nil {
		return nil, fmt.Errorf("nil final financial report")
	}
	var envelope struct {
		Report json.RawMessage `json:"completedDisplayReport"`
	}
	if json.Unmarshal([]byte(record.Log), &envelope) != nil || len(envelope.Report) == 0 {
		return nil, nil
	}
	if len(envelope.Report) > 256000 {
		return nil, fmt.Errorf("final display report exceeds size limit")
	}
	var report map[string]any
	decoder := json.NewDecoder(bytes.NewReader(envelope.Report))
	decoder.UseNumber()
	if decoder.Decode(&report) != nil {
		return nil, fmt.Errorf("invalid final display report")
	}
	number := func(key string) (int64, bool) {
		raw, ok := report[key].(json.Number)
		if !ok {
			return 0, false
		}
		n, err := raw.Int64()
		return n, err == nil
	}
	agent, ok := number("agentId")
	if !ok || agent != int64(record.AgentId) {
		return nil, fmt.Errorf("final display agent mismatch")
	}
	uid, ok := number("userId")
	if !ok || uid != int64(record.UserId) {
		return nil, fmt.Errorf("final display player mismatch")
	}
	if report["currency"] != record.Currency {
		return nil, fmt.Errorf("final display currency mismatch")
	}
	id, ok := report["id"].(string)
	if !ok || id == "" || len(id) > 128 {
		return nil, fmt.Errorf("invalid final display identity")
	}
	for _, field := range []string{"roomId", "inning", "endedAt"} {
		n, ok := number(field)
		if !ok || n <= 0 || n > 9007199254740991 {
			return nil, fmt.Errorf("invalid final display %s", field)
		}
	}
	net, ok := number("net")
	actual := decimal.NewFromFloat(record.Win).Sub(decimal.NewFromFloat(record.Bet)).Shift(2)
	if !ok || !actual.IsInteger() || !actual.Equal(decimal.NewFromInt(net)) {
		return nil, fmt.Errorf("final display net does not match settled money")
	}
	bet, ok := number("effectiveBet")
	if !ok || bet < 0 || !decimal.NewFromFloat(record.Bet).Shift(2).Equal(decimal.NewFromInt(bet)) {
		return nil, fmt.Errorf("final display bet does not match settled money")
	}
	roomType, ok := number("roomType")
	if !ok || roomType <= 0 || roomType == 101 {
		return nil, fmt.Errorf("invalid financial display room type")
	}
	if _, ok = report["detail"].(map[string]any); !ok {
		return nil, fmt.Errorf("final display details missing")
	}
	balance := decimal.NewFromFloat(record.Balance).Shift(2)
	if !balance.IsInteger() || balance.IsNegative() {
		return nil, fmt.Errorf("invalid settled display balance")
	}
	report["balance"] = balance.IntPart()
	delete(report, "replayOnly")
	delete(report, "aggregate")
	return report, nil
}

func (es *ESDao) BulkSettledDisplayReports(records []*entity.CacheRecordsReq) error {
	if es == nil || es.Client == nil {
		return fmt.Errorf("ES unavailable")
	}
	bulk := es.Client.Bulk().Retrier(elastic.NewStopRetrier())
	count := 0
	for _, financial := range records {
		report, err := settledDisplayDocuments(financial)
		if err != nil {
			return err
		}
		if report == nil {
			continue
		}
		id := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d:%d:%s:%s", financial.AgentId, financial.UserId, financial.Currency, report["id"]))))
		bulk.Add(elastic.NewBulkIndexRequest().Index(esindex.Name("gogame_records")).Id(id).OpType("create").Doc(report))
		count++
		detail := report["detail"].(map[string]any)
		if detail["replay"] != nil {
			bulk.Add(elastic.NewBulkIndexRequest().Index(esindex.Name("gogame_replays")).Id(id).OpType("create").Doc(report))
			count++
		}
	}
	if count == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	response, err := bulk.Do(ctx)
	if err != nil {
		return err
	}
	if response == nil || len(response.Items) != count {
		return fmt.Errorf("final display bulk response incomplete")
	}
	for _, item := range response.Items {
		if len(item) != 1 {
			return fmt.Errorf("invalid final display bulk item")
		}
		for _, result := range item {
			if result == nil || result.Status < 200 || result.Status >= 300 && result.Status != 409 {
				return fmt.Errorf("final display indexing failed")
			}
		}
	}
	return nil
}
