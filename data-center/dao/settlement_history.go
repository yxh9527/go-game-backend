package dao

import (
	"app/entity"
	"context"
	"encoding/json"
	"fmt"
	"github.com/olivere/elastic/v7"
	"github.com/shopspring/decimal"
	"sort"
	"time"
)

// Read the original completed financial log. Display indexes are projections,
// never a fallback authority for a cash game history request.
func settlementHistoryRecord(financial *entity.CacheRecordsReq) (*UserCenterRecord, error) {
	var envelope struct {
		Report json.RawMessage `json:"completedDisplayReport"`
	}
	if json.Unmarshal([]byte(financial.Log), &envelope) != nil || len(envelope.Report) == 0 {
		return nil, nil
	}
	var row UserCenterRecord
	if json.Unmarshal(envelope.Report, &row) != nil {
		return nil, fmt.Errorf("invalid original settlement history")
	}
	if row.UserID != financial.UserId || row.AgentID != financial.AgentId || row.Currency != financial.Currency || row.RoomType == 101 || row.RoomType <= 0 || row.RoomID <= 0 || row.Inning <= 0 || row.ID == "" || row.EndedAt <= 0 {
		return nil, fmt.Errorf("original settlement history identity mismatch")
	}
	var detail map[string]json.RawMessage
	if len(row.ID) > 256 || row.RoomID > cashMax || row.Inning > cashMax || row.EndedAt > cashMax || row.RoomType > 10000 || json.Unmarshal(row.Detail, &detail) != nil || detail == nil {
		return nil, fmt.Errorf("invalid original settlement detail")
	}
	bet, err := centerMinor(decimal.NewFromFloat(financial.Bet))
	if err != nil || bet < 0 {
		return nil, fmt.Errorf("invalid financial history bet")
	}
	net, err := centerMinor(decimal.NewFromFloat(financial.Win).Sub(decimal.NewFromFloat(financial.Bet)))
	if err != nil {
		return nil, err
	}
	if row.EffectiveBet != bet || row.Net != net {
		return nil, fmt.Errorf("original settlement history amounts mismatch")
	}
	balance, err := centerMinor(decimal.NewFromFloat(financial.Balance))
	if err != nil || balance < 0 {
		return nil, fmt.Errorf("invalid confirmed settlement balance")
	}
	row.Balance = &balance
	row.FinancialRoundID = financial.RoundID
	row.FinancialGameID = financial.GameId
	row.Aggregate = nil
	return &row, nil
}

func historyMatches(row UserCenterRecord, q UserCenterQuery) bool {
	return row.UserID == q.UserID && row.AgentID == q.AgentID && row.Currency == q.Currency && row.EndedAt >= q.StartTime && row.EndedAt <= q.EndTime &&
		(q.RoomType == 0 || row.RoomType == q.RoomType) && (q.RoomID == 0 || row.RoomID == q.RoomID) && (q.Inning == 0 || row.Inning == q.Inning) &&
		(q.InningFrom == 0 || row.Inning >= q.InningFrom) && (q.InningTo == 0 || row.Inning <= q.InningTo) && (q.Kind != "battles" || row.PrivateRoom == q.PrivateRoom)
}

func (es *ESDao) settlementHistory(ctx context.Context, q UserCenterQuery) ([]UserCenterRecord, int64, float64, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	filter := elastic.NewBoolQuery().Filter(elastic.NewTermQuery("userId", q.UserID), elastic.NewTermQuery("agentId", q.AgentID), elastic.NewMatchPhraseQuery("currency", q.Currency), elastic.NewTermQuery("complete", true), elastic.NewRangeQuery("playedDate").Gte(q.StartTime).Lte(q.EndTime))
	rows := make([]UserCenterRecord, 0)
	seen := map[string]bool{}
	err := es.scanSettlementDocuments(ctx, filter, func(hit *elastic.SearchHit) (bool, error) {
		var financial entity.CacheRecordsReq
		if json.Unmarshal(hit.Source, &financial) != nil {
			return false, fmt.Errorf("invalid financial history document")
		}
		if !financial.Complete || financial.UserId != q.UserID || financial.AgentId != q.AgentID || financial.Currency != q.Currency {
			return false, nil
		}
		row, err := settlementHistoryRecord(&financial)
		if err != nil {
			return false, err
		}
		if row == nil || !historyMatches(*row, q) || seen[row.ID] {
			return false, nil
		}
		if q.Kind == "replay" {
			var detail map[string]json.RawMessage
			_ = json.Unmarshal(row.Detail, &detail)
			if len(detail["replay"]) == 0 {
				return false, nil
			}
		}
		seen[row.ID] = true
		rows = append(rows, *row)
		return false, nil
	})
	if err != nil {
		return nil, 0, 0, err
	}
	return pageSettlementHistory(rows, q)
}

func pageSettlementHistory(rows []UserCenterRecord, q UserCenterQuery) ([]UserCenterRecord, int64, float64, error) {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].EndedAt == rows[j].EndedAt {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].EndedAt < rows[j].EndedAt
	})
	totalScore := int64(0)
	for _, row := range rows {
		totalScore += row.Net
		if totalScore > cashMax || totalScore < -cashMax {
			return nil, 0, 0, fmt.Errorf("history total outside score range")
		}
	}
	if q.Kind == "battles" && tableCenterRoom(q.RoomType) {
		groups := map[int64][]UserCenterRecord{}
		for _, row := range rows {
			groups[row.RoomID] = append(groups[row.RoomID], row)
		}
		rows = nil
		for _, group := range groups {
			latest := group[len(group)-1]
			net, bet := int64(0), int64(0)
			for _, row := range group {
				if bet > cashMax-row.EffectiveBet || net+row.Net > cashMax || net+row.Net < -cashMax {
					return nil, 0, 0, fmt.Errorf("room history total outside score range")
				}
				net += row.Net
				bet += row.EffectiveBet
			}
			latest.Aggregate = map[string]any{"net": net, "effectiveBet": bet, "totalHands": int64(len(group)), "startedAt": group[0].EndedAt}
			rows = append(rows, latest)
		}
	}
	total := int64(len(rows))
	ascending := q.Kind == "hands" && !q.SortDesc
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].EndedAt == rows[j].EndedAt {
			if ascending {
				return rows[i].ID < rows[j].ID
			}
			return rows[i].ID > rows[j].ID
		}
		if ascending {
			return rows[i].EndedAt < rows[j].EndedAt
		}
		return rows[i].EndedAt > rows[j].EndedAt
	})
	from, size := (q.Page-1)*q.Size, q.Size
	if q.Kind == "funds" {
		from = 0
		size = q.Page * q.Size
	}
	if q.Kind == "room" {
		from = 0
		size = len(rows)
	}
	if q.AllInnings {
		from = 0
		size = len(rows)
	}
	if from > len(rows) {
		from = len(rows)
	}
	end := from + size
	if end > len(rows) {
		end = len(rows)
	}
	return rows[from:end], total, float64(totalScore), nil
}
