package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"sort"
)

type centerMatchEntry struct {
	UserID  int64  `json:"userId"`
	Name    string `json:"name"`
	Avatar  string `json:"avatar"`
	Status  string `json:"status"`
	Rank    int64  `json:"rank"`
	Receipt *struct {
		PaidGold    int64 `json:"paidGold"`
		PaidService int64 `json:"paidService"`
	} `json:"receipt"`
	Refund *struct {
		Refunded int64 `json:"refunded"`
	} `json:"refund"`
}

func centerMilliseconds(n int64) int64 {
	if n > 0 && n < 100000000000 {
		return n * 1000
	}
	return n
}
func (dd *DBDao) centerMatches(ctx context.Context, q UserCenterQuery) (any, error) {
	if dd.manager == nil {
		return nil, status.Error(codes.Unavailable, "tournament records unavailable")
	}
	sql := "SELECT m.*,r.payload FROM gg_match m JOIN gg_match_runtime r ON r.match_id=m.id WHERE m.agent_id=? AND m.currency=? AND ((m.start_time BETWEEN ? AND ?) OR (m.start_time BETWEEN ? AND ?))"
	args := []any{q.AgentID, q.Currency, q.StartTime, q.EndTime, q.StartTime / 1000, q.EndTime / 1000}
	if q.Kind == "match" {
		sql += " AND m.id=?"
		args = append(args, q.RoomID)
	}
	docs, err := gogameRows(ctx, dd.manager, sql+" ORDER BY m.start_time DESC LIMIT 1001", args...)
	if err != nil {
		return nil, err
	}
	if len(docs) > 1000 {
		return nil, status.Error(codes.ResourceExhausted, "too many tournament records")
	}
	matches := make([]map[string]any, 0)
	var total int64
	for _, doc := range docs {
		var state struct {
			Registrations map[string]centerMatchEntry `json:"registrations"`
			Prizes        []struct {
				Status string `json:"status"`
				Plan   struct {
					UserID   int64  `json:"userId"`
					Kind     string `json:"kind"`
					Amount   int64  `json:"amount"`
					Currency string `json:"currency"`
				} `json:"plan"`
				Receipt *struct {
					Amount   int64  `json:"amount"`
					Currency string `json:"currency"`
				} `json:"receipt"`
			} `json:"prizes"`
		}
		if err = json.Unmarshal([]byte(fmt.Sprint(doc["payload"])), &state); err != nil {
			return nil, err
		}
		own, ok := state.Registrations[fmt.Sprint(q.UserID)]
		if !ok || own.UserID != int64(q.UserID) || own.Status == "pending" || own.Status == "failed" {
			continue
		}
		rewards, tickets := map[int64]int64{}, map[int64]int64{}
		for _, p := range state.Prizes {
			if p.Status != "paid" || p.Receipt == nil || p.Receipt.Amount != p.Plan.Amount || p.Receipt.Currency != p.Plan.Currency {
				continue
			}
			if p.Plan.Kind == "money" && p.Plan.Currency == q.Currency {
				rewards[p.Plan.UserID] += p.Plan.Amount
			}
			if p.Plan.Kind == "ticket" {
				tickets[p.Plan.UserID] += p.Plan.Amount
			}
		}
		ranks := make([]map[string]any, 0)
		var myRank map[string]any
		for _, e := range state.Registrations {
			if e.Status == "pending" || e.Status == "failed" || e.Status == "refunded" {
				continue
			}
			rank := map[string]any{"uid": e.UserID, "rank": e.Rank, "userName": e.Name, "icon": e.Avatar, "point": rewards[e.UserID], "ticketNum": tickets[e.UserID]}
			if e.Rank > 0 {
				ranks = append(ranks, rank)
			}
			if e.UserID == int64(q.UserID) {
				myRank = rank
			}
		}
		sort.Slice(ranks, func(i, j int) bool { return ranks[i]["rank"].(int64) < ranks[j]["rank"].(int64) })
		start, end := centerMilliseconds(gogameNumber(doc["start_time"])), centerMilliseconds(gogameNumber(doc["end_time"]))
		duration := int64(0)
		if end > start {
			duration = end - start
		}
		paid := int64(0)
		if own.Receipt != nil {
			paid = own.Receipt.PaidGold + own.Receipt.PaidService
		}
		refund := int64(0)
		if own.Refund != nil {
			refund = own.Refund.Refunded
		}
		total += rewards[int64(q.UserID)] - paid + refund
		rank := own.Rank
		if rank <= 0 && gogameNumber(doc["state"]) != 3 {
			rank = -1
		}
		item := map[string]any{"id": gogameNumber(doc["id"]), "championshipName": doc["championship_name"], "roomType": 101, "gameType": gogameNumber(doc["game_type"]),
			"startDate": start, "status": gogameNumber(doc["state"]), "myRank": rank, "rewardAmount": rewards[int64(q.UserID)], "ticketNum": tickets[int64(q.UserID)],
			"currentPlayer": gogameNumber(doc["current_player"]), "initPoint": gogameNumber(doc["init_point"]), "registerGold": gogameNumber(doc["register_gold"]),
			"blindType": gogameNumber(doc["blind_type"]), "forceShowCard": gogameNumber(doc["force_show_card"]) != 0, "gameTime": duration,
			"increaseBindTime": gogameNumber(doc["increase_bind_time"]), "MTTUserRanks": ranks, "myUserRank": myRank}
		matches = append(matches, item)
	}
	if q.Kind == "match" {
		if len(matches) == 0 {
			return map[string]any{"match": nil}, nil
		}
		return map[string]any{"match": matches[0]}, nil
	}
	return map[string]any{"matches": matches, "totalScore": total}, nil
}
