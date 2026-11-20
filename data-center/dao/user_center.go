package dao

import (
	"app/esindex"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/olivere/elastic/v7"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math"
	"sort"
	"time"
)

type UserCenterQuery struct {
	UserID      uint32 `json:"userId"`
	AgentID     uint32 `json:"agentId"`
	Currency    string `json:"currency"`
	Kind        string `json:"kind"`
	Page        int    `json:"page"`
	Size        int    `json:"size"`
	StartTime   int64  `json:"startTime"`
	EndTime     int64  `json:"endTime"`
	RoomType    int    `json:"roomType"`
	RoomID      int64  `json:"roomId"`
	Inning      int64  `json:"inning"`
	InningFrom  int64  `json:"inningFrom"`
	InningTo    int64  `json:"inningTo"`
	SortDesc    bool   `json:"sortDesc"`
	PrivateRoom bool   `json:"privateRoom"`
	AllInnings  bool   `json:"allInnings,omitempty"`
}
type UserCenterRecord struct {
	FinancialRoundID string          `json:"-"`
	FinancialGameID  uint32          `json:"-"`
	SmartTask        map[string]any  `json:"smartTask,omitempty"`
	ID               string          `json:"id"`
	UserID           uint32          `json:"userId"`
	AgentID          uint32          `json:"agentId"`
	Currency         string          `json:"currency"`
	GameID           string          `json:"gameId"`
	RoomType         int             `json:"roomType"`
	RoomID           int64           `json:"roomId"`
	Inning           int64           `json:"inning"`
	EndedAt          int64           `json:"endedAt"`
	PlayMode         int             `json:"playMode"`
	PrivateRoom      bool            `json:"privateRoom"`
	EffectiveBet     int64           `json:"effectiveBet"`
	Net              int64           `json:"net"`
	Balance          *int64          `json:"balance,omitempty"`
	TakeIn           *int64          `json:"takeIn,omitempty"`
	Name             string          `json:"name"`
	Room             map[string]any  `json:"room"`
	Players          json.RawMessage `json:"players"`
	Detail           json.RawMessage `json:"detail"`
	Aggregate        map[string]any  `json:"aggregate,omitempty"`
}

func centerIndex() string { return esindex.Name("gogame_records") }
func validateCenterQuery(q UserCenterQuery) error {
	if q.AllInnings && (q.Kind != "replay" || q.RoomID <= 0 || q.Page != 1 || (q.RoomType != 0 && q.RoomType != 19 && q.RoomType != 101)) {
		return status.Error(codes.InvalidArgument, "invalid full replay scope")
	}
	if q.InningFrom < 0 || q.InningTo < 0 || q.InningFrom > cashMax || q.InningTo > cashMax || q.InningFrom > 0 && q.InningTo > 0 && q.InningFrom > q.InningTo {
		return status.Error(codes.InvalidArgument, "invalid replay range")
	}
	if q.UserID == 0 || !cashCurrency.MatchString(q.Currency) || q.Page < 1 || q.Page > 1000 || q.Size < 1 || q.Size > 100 || (q.Page-1)*q.Size >= 10000 || q.StartTime <= 0 || q.EndTime < q.StartTime || q.EndTime-q.StartTime > 31*86400000 || q.RoomType < 0 || q.RoomType > 10000 || q.RoomID < 0 || q.RoomID > cashMax || q.Inning < 0 || q.Inning > cashMax {
		return status.Error(codes.InvalidArgument, "invalid user-center query")
	}
	switch q.Kind {
	case "battles", "hands", "round", "room", "funds", "tickets", "matches", "match", "replay":
		return nil
	}
	return status.Error(codes.InvalidArgument, "invalid user-center kind")
}
func (dd *DBDao) centerIdentity(ctx context.Context, uid, agent uint32, currency string) error {
	if uid == 0 || !cashCurrency.MatchString(currency) {
		return status.Error(codes.InvalidArgument, "invalid user-center account")
	}
	if rd := RedisIns(); rd != nil && rd.Client != nil {
		p, err := rd.GetPlayer(uid, 0)
		if err != nil {
			return err
		}
		if p != nil {
			if p.AgentId != agent || p.CurrencyType != currency {
				return status.Error(codes.PermissionDenied, "user-center account mismatch")
			}
			return nil
		}
	}
	if dd.db == nil {
		return status.Error(codes.Unavailable, "user-center identity storage unavailable")
	}
	p, err := dd.GetPlayer(ctx, uid)
	if err != nil {
		return err
	}
	if uint32(p.ProxyId) != agent || p.CurrencyType != currency {
		return status.Error(codes.PermissionDenied, "user-center account mismatch")
	}
	return nil
}
func centerFilter(q UserCenterQuery) *elastic.BoolQuery {
	filters := []elastic.Query{elastic.NewTermQuery("userId", q.UserID), elastic.NewTermQuery("agentId", q.AgentID), elastic.NewMatchPhraseQuery("currency", q.Currency), elastic.NewRangeQuery("endedAt").Gte(q.StartTime).Lte(q.EndTime)}
	if q.RoomType > 0 {
		filters = append(filters, elastic.NewTermQuery("roomType", q.RoomType))
	}
	if q.RoomID > 0 {
		filters = append(filters, elastic.NewTermQuery("roomId", q.RoomID))
	}
	if q.Inning > 0 {
		filters = append(filters, elastic.NewTermQuery("inning", q.Inning))
	}
	if q.InningFrom > 0 || q.InningTo > 0 {
		rangeQuery := elastic.NewRangeQuery("inning")
		if q.InningFrom > 0 {
			rangeQuery = rangeQuery.Gte(q.InningFrom)
		}
		if q.InningTo > 0 {
			rangeQuery = rangeQuery.Lte(q.InningTo)
		}
		filters = append(filters, rangeQuery)
	}
	if q.Kind == "battles" {
		filters = append(filters, elastic.NewTermQuery("privateRoom", q.PrivateRoom))
	}
	return elastic.NewBoolQuery().Filter(filters...)
}
func scoreMetric(aggs elastic.Aggregations, name string) float64 {
	if s, ok := aggs.Sum(name); ok && s.Value != nil {
		return *s.Value
	}
	return 0
}
func centerTotal(resp *elastic.SearchResult) int64 {
	if resp.Hits != nil && resp.Hits.TotalHits != nil {
		return resp.Hits.TotalHits.Value
	}
	return 0
}
func tableCenterRoom(room int) bool {
	switch room {
	case 19, 20, 21, 24, 25, 26, 27, 28, 30, 31, 32:
		return true
	}
	return false
}
func (es *ESDao) centerRecords(ctx context.Context, q UserCenterQuery) ([]UserCenterRecord, int64, float64, error) {
	if es == nil || es.es == nil {
		return nil, 0, 0, status.Error(codes.Unavailable, "history ES unavailable")
	}
	if q.Kind == "replay" && q.RoomType == 101 {
		return es.competitionReplayRecords(ctx, q)
	}
	rows, total, net, err := es.settlementHistory(ctx, q)
	if err != nil {
		return nil, 0, 0, err
	}
	// The HTTP detail route may omit room type. Match points never produce a
	// cash settlement: query that source separately, only for competition 101.
	if q.Kind == "replay" && q.RoomType == 0 && len(rows) == 0 {
		q.RoomType = 101
		return es.competitionReplayRecords(ctx, q)
	}
	return rows, total, net, nil
}

func (es *ESDao) competitionReplayRecords(ctx context.Context, q UserCenterQuery) ([]UserCenterRecord, int64, float64, error) {
	if q.Kind != "replay" || q.RoomType != 101 {
		return nil, 0, 0, status.Error(codes.InvalidArgument, "competition replay scope required")
	}
	if q.AllInnings {
		rows := make([]UserCenterRecord, 0)
		err := es.scanCompetitionReplayDocuments(ctx, centerFilter(q), func(hit *elastic.SearchHit) (bool, error) {
			var row UserCenterRecord
			if err := json.Unmarshal(hit.Source, &row); err != nil {
				return false, err
			}
			if row.RoomType == 101 && historyMatches(row, q) {
				rows = append(rows, row)
			}
			return false, nil
		})
		if err != nil {
			return nil, 0, 0, err
		}
		return rows, int64(len(rows)), 0, nil
	}
	response, err := es.es.Search().Index(esindex.Name("gogame_replays")).IgnoreUnavailable(true).Query(centerFilter(q)).TrackTotalHits(true).
		From((q.Page-1)*q.Size).Size(q.Size).Sort("endedAt", false).Sort("id.keyword", false).Do(ctx)
	if err != nil {
		return nil, 0, 0, err
	}
	rows := make([]UserCenterRecord, 0)
	if response != nil && response.Hits != nil {
		for _, hit := range response.Hits.Hits {
			var row UserCenterRecord
			if err = json.Unmarshal(hit.Source, &row); err != nil {
				return nil, 0, 0, err
			}
			if row.RoomType == 101 && historyMatches(row, q) {
				rows = append(rows, row)
			}
		}
	}
	if response == nil {
		return nil, 0, 0, fmt.Errorf("invalid competition replay response")
	}
	return rows, centerTotal(response), 0, nil
}
func centerMinor(value decimal.Decimal) (int64, error) {
	n := value.Shift(2)
	if !n.IsInteger() || n.Abs().GreaterThan(decimal.NewFromInt(cashMax)) {
		return 0, status.Error(codes.FailedPrecondition, "invalid stored user-center money")
	}
	return n.IntPart(), nil
}
func (dd *DBDao) centerMoneyLogs(ctx context.Context, es *ESDao, q UserCenterQuery) ([]map[string]any, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	// Deduplicate before paging, otherwise removal of already represented
	// settlements can hide older fallback rows outside the original first page.
	historyQuery := q
	historyQuery.Kind = "room"
	records, _, _, err := es.centerRecords(ctx, historyQuery)
	if err != nil {
		return nil, 0, err
	}
	logs, covered, count, err := es.originalFundBills(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	for _, r := range records {
		if r.Balance != nil && !covered[financialRoundKey(r.FinancialGameID, r.FinancialRoundID)] {
			logs = append(logs, map[string]any{"id": r.ID, "sourceType": 2, "sourceId": r.RoomID, "point": r.Net, "lastPoint": *r.Balance, "createTime": r.EndedAt})
			count++
		}
	}
	if dd.manager == nil {
		return nil, 0, status.Error(codes.Unavailable, "user-center transfer storage unavailable")
	}
	type scoreLog struct {
		ID        int64
		LogOn     string
		LogType   int
		Score     decimal.Decimal
		UserScore decimal.Decimal
		Created   int64
	}
	var transfers []scoreLog
	var total int64
	base := dd.manager.WithContext(ctx).Table("gp_user_score_log").Where("userId=? AND agentId=? AND logType IN (1,2) AND createtime>=? AND createtime<=?", q.UserID, q.AgentID, q.StartTime/1000, q.EndTime/1000)
	if err = base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err = base.Select("id, logOn AS log_on, logType AS log_type, score, userScore AS user_score, createtime AS created").Order("createtime DESC, id DESC").Limit(q.Page * q.Size).Scan(&transfers).Error; err != nil {
		return nil, 0, err
	}
	count += total
	for _, r := range transfers {
		point, e := centerMinor(r.Score.Abs())
		if e != nil {
			return nil, 0, e
		}
		balance, e := centerMinor(r.UserScore)
		if e != nil {
			return nil, 0, e
		}
		kind := 26
		if r.LogType == 2 {
			point = -point
			kind = 27
		}
		logs = append(logs, map[string]any{"id": "transfer:" + r.LogOn, "orderKey": fmt.Sprintf("transfer:%020d", r.ID), "sourceType": kind, "sourceId": q.AgentID, "point": point, "lastPoint": balance, "createTime": r.Created * 1000})
	}
	sort.SliceStable(logs, func(i, j int) bool {
		a, b := logs[i]["createTime"].(int64), logs[j]["createTime"].(int64)
		if a == b {
			left, right := logs[i]["orderKey"], logs[j]["orderKey"]
			if left == nil {
				left = logs[i]["id"]
			}
			if right == nil {
				right = logs[j]["id"]
			}
			return fmt.Sprint(left) > fmt.Sprint(right)
		}
		return a > b
	})
	from := (q.Page - 1) * q.Size
	if from > len(logs) {
		from = len(logs)
	}
	end := from + q.Size
	if end > len(logs) {
		end = len(logs)
	}
	for _, log := range logs[from:end] {
		delete(log, "orderKey")
	}
	return logs[from:end], count, nil
}
func (dd *DBDao) UserCenter(ctx context.Context, es *ESDao, q UserCenterQuery) (any, error) {
	if err := validateCenterQuery(q); err != nil {
		return nil, err
	}
	if err := dd.centerIdentity(ctx, q.UserID, q.AgentID, q.Currency); err != nil {
		return nil, err
	}
	if q.Kind == "tickets" {
		return dd.centerTickets(ctx, q)
	}
	if q.Kind == "matches" || q.Kind == "match" {
		return dd.centerMatches(ctx, q)
	}
	if q.Kind == "funds" {
		logs, total, err := dd.centerMoneyLogs(ctx, es, q)
		return map[string]any{"logs": logs, "total": total, "paged": true}, err
	}
	records, total, net, err := es.centerRecords(ctx, q)
	return map[string]any{"records": records, "total": total, "totalScore": net, "paged": true}, err
}
func (dd *DBDao) WriteUserCenter(ctx context.Context, es *ESDao, records []UserCenterRecord) (any, error) {
	return dd.storeCenterReports(ctx, es, records, centerIndex())
}
func (dd *DBDao) WriteHoldemReplays(ctx context.Context, es *ESDao, records []UserCenterRecord) (any, error) {
	for _, r := range records {
		if r.RoomType != 19 && r.RoomType != 101 {
			return nil, status.Error(codes.InvalidArgument, "invalid replay game")
		}
	}
	return dd.storeCenterReports(ctx, es, records, esindex.Name("gogame_replays"))
}
func (dd *DBDao) writeCenterRecords(ctx context.Context, es *ESDao, records []UserCenterRecord, index string) (any, error) {
	if len(records) == 0 || len(records) > 100 {
		return nil, status.Error(codes.InvalidArgument, "invalid user-center record batch")
	}
	verified := map[string]bool{}
	now := time.Now().UnixMilli()
	for _, r := range records {
		if !cashID.MatchString(r.ID) || r.RoomType <= 0 || r.RoomID <= 0 || r.RoomID > cashMax || r.Inning <= 0 || r.Inning > cashMax || r.EndedAt < now-31*86400000 || r.EndedAt > now+300000 || r.EffectiveBet < 0 || r.EffectiveBet > cashMax || math.Abs(float64(r.Net)) > float64(cashMax) || r.Balance != nil && (*r.Balance < 0 || *r.Balance > cashMax) || len(r.Detail) > 256000 {
			return nil, status.Error(codes.InvalidArgument, "invalid user-center record")
		}
		key := fmt.Sprintf("%d:%d:%s", r.AgentID, r.UserID, r.Currency)
		if !verified[key] {
			if err := dd.centerIdentity(ctx, r.UserID, r.AgentID, r.Currency); err != nil {
				return nil, err
			}
			verified[key] = true
		}
	}
	return dd.indexCenterRecords(ctx, es, records, index)
}

func (dd *DBDao) indexCenterRecords(ctx context.Context, es *ESDao, records []UserCenterRecord, index string) (any, error) {
	if es == nil || es.es == nil {
		return nil, status.Error(codes.Unavailable, "user-center ES unavailable")
	}
	bulk := es.es.Bulk().Retrier(elastic.NewStopRetrier()).Refresh("wait_for")
	expected := make([]UserCenterRecord, 0, len(records))
	identities := make([]string, 0, len(records))
	for _, r := range records {
		key := fmt.Sprintf("%d:%d:%s", r.AgentID, r.UserID, r.Currency)
		id := fmt.Sprintf("%x", sha256.Sum256([]byte(key+":"+r.ID)))
		r.Aggregate = nil
		expected = append(expected, r)
		identities = append(identities, id)
		bulk.Add(elastic.NewBulkIndexRequest().Index(index).Id(id).OpType("create").Doc(r))
	}
	resp, err := bulk.Do(ctx)
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Items) != len(records) {
		return nil, status.Error(codes.Unavailable, "user-center storage response incomplete")
	}
	for i, item := range resp.Items {
		if len(item) != 1 {
			return nil, status.Error(codes.Unavailable, "invalid user-center storage item")
		}
		for _, outcome := range item {
			if outcome == nil || outcome.Status < 200 || outcome.Status >= 300 && outcome.Status != 409 {
				return nil, status.Error(codes.Unavailable, "user-center record storage rejected batch")
			}
			if outcome.Status == 409 {
				if err := es.confirmOriginalCenterReport(ctx, index, identities[i], expected[i]); err != nil {
					return nil, err
				}
			}
		}
	}
	return map[string]any{"stored": len(records)}, nil
}
func (dd *DBDao) centerTickets(ctx context.Context, q UserCenterQuery) (any, error) {
	if dd.db == nil {
		return nil, status.Error(codes.Unavailable, "user-center ticket storage unavailable")
	}
	type ticketEvent struct {
		ID       string
		Created  int64
		Num      int
		SourceID int
	}
	// Issuance and use timestamps already exist on ticket holdings; no new ledger.
	sql := "SELECT * FROM (SELECT CONCAT('issued:',h.id) AS id,h.create_time AS created,1 AS num,0 AS source_id FROM gg_ticket_hold h JOIN gg_ticket t ON t.id=h.ticket_id WHERE h.owner=? AND t.currency=? UNION ALL SELECT CONCAT('used:',h.id),h.use_time,-1,1 FROM gg_ticket_hold h JOIN gg_ticket t ON t.id=h.ticket_id WHERE h.owner=? AND t.currency=? AND h.is_use=1) events WHERE created>=? AND created<=?"
	args := []any{q.UserID, q.Currency, q.UserID, q.Currency, q.StartTime / 1000, q.EndTime / 1000}
	var total int64
	if err := dd.db.WithContext(ctx).Raw("SELECT COUNT(*) FROM ("+sql+") count_events", args...).Scan(&total).Error; err != nil {
		return nil, err
	}
	var events []ticketEvent
	pagedArgs := append(append([]any{}, args...), q.Size, (q.Page-1)*q.Size)
	if err := dd.db.WithContext(ctx).Raw(sql+" ORDER BY created DESC,id DESC LIMIT ? OFFSET ?", pagedArgs...).Scan(&events).Error; err != nil {
		return nil, err
	}
	logs := make([]map[string]any, 0, len(events))
	for _, e := range events {
		logs = append(logs, map[string]any{"id": e.ID, "createTime": e.Created * 1000, "num": e.Num, "sourceId": e.SourceID})
	}
	return map[string]any{"logs": logs, "total": total, "paged": true}, nil
}
