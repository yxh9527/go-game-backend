package dao

import (
	"app/entity"
	"app/esindex"
	"context"
	"encoding/json"
	"fmt"
	"github.com/olivere/elastic/v7"
	"time"
)

type ScratchHistoryQuery struct {
	UserID     uint32 `json:"userId"`
	AgentID    uint32 `json:"agentId"`
	Currency   string `json:"currency"`
	GameID     uint32 `json:"gameId"`
	RoomType   int    `json:"roomType"`
	RoomID     int64  `json:"roomId"`
	GameRoomID int64  `json:"gameRoomId"`
	Size       int    `json:"size"`
	StartTime  int64  `json:"startTime"`
	EndTime    int64  `json:"endTime"`
}

// The source is the original financial document for every paid scratch ticket.
// No display index, Redis history list or current finance snapshot is consulted.
func (dd *DBDao) ScratchHistory(ctx context.Context, es *ESDao, q ScratchHistoryQuery) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if es == nil || es.es == nil {
		return nil, fmt.Errorf("settlement ES unavailable")
	}
	if q.GameID == 0 || (q.RoomType != 61 && q.RoomType != 63) || q.GameRoomID <= 0 || q.GameRoomID > cashMax {
		return nil, fmt.Errorf("invalid scratch history scope")
	}
	if err := validateCenterQuery(UserCenterQuery{UserID: q.UserID, AgentID: q.AgentID, Currency: q.Currency, Kind: "hands", Page: 1, Size: q.Size, RoomType: q.RoomType, RoomID: q.RoomID, StartTime: q.StartTime, EndTime: q.EndTime}); err != nil {
		return nil, err
	}
	if q.RoomID <= 0 {
		return nil, fmt.Errorf("scratch history room required")
	}
	if err := dd.centerIdentity(ctx, q.UserID, q.AgentID, q.Currency); err != nil {
		return nil, err
	}
	filter := elastic.NewBoolQuery().Filter(elastic.NewTermQuery("userId", q.UserID), elastic.NewTermQuery("agentId", q.AgentID), elastic.NewTermQuery("gameId", q.GameID), elastic.NewMatchPhraseQuery("currency", q.Currency), elastic.NewTermQuery("complete", true), elastic.NewRangeQuery("playedDate").Gte(q.StartTime).Lte(q.EndTime))
	rows := make([]*entity.CacheRecordsReq, 0, q.Size)
	// Original financial documents encode the round detail in log JSON. Inspect bounded pages
	// until this room has its requested count; never return a partial success
	// when the existing ES result window is exhausted.
	for from := 0; from < 10000; from += 100 {
		response, err := es.es.Search().Index(esindex.Settlement()).Query(filter).From(from).Size(100).Sort("playedDate", false).Sort("hash.keyword", false).Do(ctx)
		if err != nil {
			return nil, err
		}
		if response == nil || response.Hits == nil {
			return nil, fmt.Errorf("invalid settlement history response")
		}
		for _, hit := range response.Hits.Hits {
			var row entity.CacheRecordsReq
			if json.Unmarshal(hit.Source, &row) != nil {
				return nil, fmt.Errorf("invalid original settlement record")
			}
			if row.UserId != q.UserID || row.AgentId != q.AgentID || row.GameId != q.GameID || row.Currency != q.Currency || !row.Complete {
				continue
			}
			var source struct {
				RoomID     int64  `json:"roomId"`
				GameRoomID int64  `json:"gameRoomId"`
				RoomType   int    `json:"roomType"`
				Serial     string `json:"serialNo"`
			}
			if json.Unmarshal([]byte(row.Log), &source) != nil || source.Serial == "" {
				continue
			}
			if source.RoomID != q.RoomID || source.GameRoomID != q.GameRoomID || source.RoomType != q.RoomType {
				continue
			}
			// Return the original log and account identity. Do not substitute a
			// display document or init payload for the round detail.
			rows = append(rows, &entity.CacheRecordsReq{UserId: row.UserId, AgentId: row.AgentId, GameId: row.GameId, Currency: row.Currency, Log: row.Log})
			if len(rows) == q.Size {
				return map[string]any{"records": rows}, nil
			}
		}
		if len(response.Hits.Hits) < 100 {
			return map[string]any{"records": rows}, nil
		}
	}
	return nil, fmt.Errorf("scratch history exceeds bounded settlement search window")
}
