package dao

import (
	"app/entity/view"
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
	SortDesc    bool   `json:"sortDesc"`
	PrivateRoom bool   `json:"privateRoom"`
}
type UserCenterRecord struct {
	ID           string          `json:"id"`
	UserID       uint32          `json:"userId"`
	AgentID      uint32          `json:"agentId"`
	Currency     string          `json:"currency"`
	GameID       string          `json:"gameId"`
	RoomType     int             `json:"roomType"`
	RoomID       int64           `json:"roomId"`
	Inning       int64           `json:"inning"`
	EndedAt      int64           `json:"endedAt"`
	PlayMode     int             `json:"playMode"`
	PrivateRoom  bool            `json:"privateRoom"`
	EffectiveBet int64           `json:"effectiveBet"`
	Net          int64           `json:"net"`
	Balance      *int64          `json:"balance,omitempty"`
	TakeIn       *int64          `json:"takeIn,omitempty"`
	Name         string          `json:"name"`
	Room         map[string]any  `json:"room"`
	Players      json.RawMessage `json:"players"`
	Detail       json.RawMessage `json:"detail"`
	Aggregate    map[string]any  `json:"aggregate,omitempty"`
}

func centerIndex() string { return esindex.Name("gogame_records") }
func validateCenterQuery(q UserCenterQuery) error {
	if q.UserID == 0 || !cashCurrency.MatchString(q.Currency) || q.Page < 1 || q.Page > 1000 || q.Size < 1 || q.Size > 100 || (q.Page-1)*q.Size >= 10000 || q.StartTime <= 0 || q.EndTime < q.StartTime || q.EndTime-q.StartTime > 31*86400000 || q.RoomType < 0 || q.RoomType > 10000 || q.RoomID < 0 || q.RoomID > cashMax || q.Inning < 0 || q.Inning > cashMax {
		return status.Error(codes.InvalidArgument, "invalid user-center query")
	}
	switch q.Kind {
	case "battles", "hands", "round", "room", "funds", "tickets", "matches", "match":
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
		return nil, 0, 0, status.Error(codes.Unavailable, "user-center ES unavailable")
	}
	search := es.es.Search().Index(centerIndex()).IgnoreUnavailable(true).Query(centerFilter(q)).TrackTotalHits(true).Aggregation("totalScore", elastic.NewSumAggregation().Field("net"))
	if q.Kind == "battles" && tableCenterRoom(q.RoomType) {
		limit := q.Page * q.Size
		rooms := elastic.NewTermsAggregation().Field("roomId").Size(limit).OrderByAggregation("latest", false).
			SubAggregation("latest", elastic.NewMaxAggregation().Field("endedAt")).SubAggregation("started", elastic.NewMinAggregation().Field("endedAt")).
			SubAggregation("score", elastic.NewSumAggregation().Field("net")).SubAggregation("bet", elastic.NewSumAggregation().Field("effectiveBet")).
			SubAggregation("sample", elastic.NewTopHitsAggregation().Size(1).Sort("endedAt", false))
		resp, err := search.Size(0).Aggregation("rooms", rooms).Aggregation("roomCount", elastic.NewCardinalityAggregation().Field("roomId").PrecisionThreshold(40000)).Do(ctx)
		if err != nil {
			return nil, 0, 0, err
		}
		records := make([]UserCenterRecord, 0, q.Size)
		total := int64(0)
		if c, ok := resp.Aggregations.Cardinality("roomCount"); ok && c.Value != nil {
			total = int64(*c.Value)
		}
		if groups, ok := resp.Aggregations.Terms("rooms"); ok {
			for i, b := range groups.Buckets {
				if i < (q.Page-1)*q.Size {
					continue
				}
				sample, ok := b.Aggregations.TopHits("sample")
				if !ok || sample.Hits == nil || len(sample.Hits.Hits) == 0 {
					continue
				}
				var r UserCenterRecord
				if err = json.Unmarshal(sample.Hits.Hits[0].Source, &r); err != nil {
					return nil, 0, 0, err
				}
				r.Aggregate = map[string]any{"net": scoreMetric(b.Aggregations, "score"), "effectiveBet": scoreMetric(b.Aggregations, "bet"), "totalHands": b.DocCount}
				if started, ok := b.Aggregations.Min("started"); ok && started.Value != nil {
					r.Aggregate["startedAt"] = *started.Value
				}
				records = append(records, r)
			}
		}
		return records, total, scoreMetric(resp.Aggregations, "totalScore"), nil
	}
	size := q.Size
	from := (q.Page - 1) * q.Size
	if q.Kind == "funds" {
		size = q.Page * q.Size
		from = 0
		search = search.PostFilter(elastic.NewExistsQuery("balance"))
	}
	if q.Kind == "room" {
		size = 1000
		from = 0
	}
	ascending := q.Kind == "hands" && !q.SortDesc
	resp, err := search.From(from).Size(size).Sort("endedAt", ascending).Sort("id.keyword", ascending).Do(ctx)
	if err != nil {
		return nil, 0, 0, err
	}
	records := make([]UserCenterRecord, 0)
	if resp.Hits != nil {
		for _, hit := range resp.Hits.Hits {
			var r UserCenterRecord
			if err = json.Unmarshal(hit.Source, &r); err != nil {
				return nil, 0, 0, err
			}
			records = append(records, r)
		}
	}
	return records, centerTotal(resp), scoreMetric(resp.Aggregations, "totalScore"), nil
}
func centerMinor(value decimal.Decimal) (int64, error) {
	n := value.Shift(2)
	if !n.IsInteger() || n.Abs().GreaterThan(decimal.NewFromInt(cashMax)) {
		return 0, status.Error(codes.FailedPrecondition, "invalid stored user-center money")
	}
	return n.IntPart(), nil
}
func (dd *DBDao) centerMoneyLogs(ctx context.Context, es *ESDao, q UserCenterQuery) ([]map[string]any, int64, error) {
	records, count, _, err := es.centerRecords(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	logs := make([]map[string]any, 0)
	for _, r := range records {
		if r.Balance != nil {
			logs = append(logs, map[string]any{"id": r.ID, "sourceType": 2, "sourceId": r.RoomID, "point": r.Net, "lastPoint": *r.Balance, "createTime": r.EndedAt})
		}
	}
	filter := elastic.NewBoolQuery().Filter(elastic.NewTermQuery("userId", q.UserID), elastic.NewTermQuery("agentId", q.AgentID), elastic.NewMatchPhraseQuery("currency", q.Currency), elastic.NewRangeQuery("createTime").Gte(q.StartTime/1000).Lte(q.EndTime/1000))
	resp, err := es.es.Search().Index(esindex.FlowingWater()).IgnoreUnavailable(true).Query(filter).Size(q.Page*q.Size).Sort("createTime", false).TrackTotalHits(true).Do(ctx)
	if err != nil {
		return nil, 0, err
	}
	count += centerTotal(resp)
	if resp.Hits != nil {
		for _, hit := range resp.Hits.Hits {
			var b view.Bill
			if err = json.Unmarshal(hit.Source, &b); err != nil {
				return nil, 0, err
			}
			b.Normalize()
			point, e := centerMinor(b.Bets.Add(b.Award))
			if e != nil {
				return nil, 0, e
			}
			balance, e := centerMinor(b.UserScore)
			if e != nil {
				return nil, 0, e
			}
			logs = append(logs, map[string]any{"id": "bill:" + hit.Id, "sourceType": 2, "sourceId": b.GameId, "point": point, "lastPoint": balance, "createTime": b.CreatTime * 1000})
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
		logs = append(logs, map[string]any{"id": "transfer:" + r.LogOn, "sourceType": kind, "sourceId": q.AgentID, "point": point, "lastPoint": balance, "createTime": r.Created * 1000})
	}
	sort.SliceStable(logs, func(i, j int) bool { return logs[i]["createTime"].(int64) > logs[j]["createTime"].(int64) })
	from := (q.Page - 1) * q.Size
	if from > len(logs) {
		from = len(logs)
	}
	end := from + q.Size
	if end > len(logs) {
		end = len(logs)
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
	if es == nil || es.es == nil {
		return nil, status.Error(codes.Unavailable, "user-center ES unavailable")
	}
	if len(records) == 0 || len(records) > 100 {
		return nil, status.Error(codes.InvalidArgument, "invalid user-center record batch")
	}
	verified := map[string]bool{}
	now := time.Now().UnixMilli()
	bulk := es.es.Bulk().Refresh("wait_for")
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
		id := fmt.Sprintf("%x", sha256.Sum256([]byte(key+":"+r.ID)))
		r.Aggregate = nil
		bulk.Add(elastic.NewBulkIndexRequest().Index(centerIndex()).Id(id).OpType("create").Doc(r))
	}
	resp, err := bulk.Do(ctx)
	if err != nil {
		return nil, err
	}
	if len(resp.Items) != len(records) {
		return nil, status.Error(codes.Unavailable, "user-center storage response incomplete")
	}
	for _, item := range resp.Items {
		for _, outcome := range item {
			if outcome.Status >= 400 && outcome.Status != 409 {
				return nil, status.Error(codes.Unavailable, "user-center record storage rejected batch")
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
