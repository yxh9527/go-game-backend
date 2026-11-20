package dao

import (
	"app/entity/view"
	"app/esindex"
	"context"
	"encoding/json"
	"fmt"
	"github.com/olivere/elastic/v7"
)

func financialRoundKey(gameID uint32, roundID string) string {
	if gameID == 0 || roundID == "" {
		return ""
	}
	return fmt.Sprintf("%d:%s", gameID, roundID)
}

// Read original cash movements separately from final game summaries. One
// round's bills replace that round's net fallback, never another game's row.
func (es *ESDao) originalFundBills(ctx context.Context, q UserCenterQuery) ([]map[string]any, map[string]bool, int64, error) {
	filter := elastic.NewBoolQuery().Filter(elastic.NewTermQuery("userId", q.UserID), elastic.NewTermQuery("agentId", q.AgentID), elastic.NewMatchPhraseQuery("currency", q.Currency), elastic.NewRangeQuery("createTime").Gte(q.StartTime/1000).Lte(q.EndTime/1000))
	source := elastic.NewFetchSourceContext(true).Include("agentId", "userId", "gameId", "currency", "roundId", "flowingWaterOn", "createTime", "desc", "bet", "award", "currentScore")
	logs := make([]map[string]any, 0, q.Page*q.Size)
	covered := map[string]bool{}
	seen := map[string]bool{}
	count := int64(0)
	err := es.scanHistoryDocuments(ctx, esindex.FlowingWater(), filter, "createTime", "flowingWaterOn.keyword", source, func(hit *elastic.SearchHit) (bool, error) {
		var bill view.Bill
		if err := json.Unmarshal(hit.Source, &bill); err != nil {
			return false, err
		}
		if bill.UserId != int64(q.UserID) || bill.AgentId != int64(q.AgentID) || bill.CurrencyType != q.Currency {
			return false, nil
		}
		id := hit.Id
		if id == "" {
			id = bill.FlowingWaterOn
		}
		if id == "" {
			return false, fmt.Errorf("original bill identity missing")
		}
		identity := id
		if bill.FlowingWaterOn != "" {
			identity = "flow:" + bill.FlowingWaterOn
		}
		if seen[identity] {
			return false, nil
		}
		seen[identity] = true
		bill.Normalize()
		point, err := centerMinor(bill.Bets.Add(bill.Award))
		if err != nil {
			return false, err
		}
		balance, err := centerMinor(bill.UserScore)
		if err != nil || balance < 0 {
			return false, fmt.Errorf("invalid original bill balance")
		}
		if bill.CreatTime <= 0 || bill.CreatTime > cashMax/1000 || bill.GameId < 0 || uint64(bill.GameId) > uint64(^uint32(0)) {
			return false, fmt.Errorf("invalid original bill scope")
		}
		if key := financialRoundKey(uint32(bill.GameId), bill.OfficeNumber); key != "" {
			covered[key] = true
		}
		count++
		if len(logs) < q.Page*q.Size {
			order := bill.FlowingWaterOn
			if order == "" {
				order = id
			}
			logs = append(logs, map[string]any{"id": "bill:" + id, "orderKey": "bill:" + order, "sourceType": 2, "sourceId": bill.GameId, "point": point, "lastPoint": balance, "createTime": bill.CreatTime * 1000})
		}
		return false, nil
	})
	return logs, covered, count, err
}
