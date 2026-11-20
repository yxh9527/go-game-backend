package dao

import (
	"app/esindex"
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math"
	"time"
)

func validateCenterReportBatch(records []UserCenterRecord, index string) error {
	if index != centerIndex() && index != esindex.Name("gogame_replays") {
		return status.Error(codes.InvalidArgument, "invalid report destination")
	}
	if len(records) < 1 || len(records) > 100 {
		return status.Error(codes.InvalidArgument, "invalid report batch")
	}
	now := time.Now().UnixMilli()
	for _, r := range records {
		if !cashID.MatchString(r.ID) || r.UserID == 0 || r.AgentID == 0 || !cashCurrency.MatchString(r.Currency) || r.RoomType <= 0 || r.RoomID <= 0 || r.RoomID > cashMax || r.Inning <= 0 || r.Inning > cashMax || r.EndedAt < now-31*86400000 || r.EndedAt > now+300000 || r.EffectiveBet < 0 || r.EffectiveBet > cashMax || math.Abs(float64(r.Net)) > float64(cashMax) || r.Balance != nil && (*r.Balance < 0 || *r.Balance > cashMax) || len(r.Detail) > 256000 || !json.Valid(r.Detail) || len(r.Players) > 0 && !json.Valid(r.Players) {
			return status.Error(codes.InvalidArgument, "invalid settled report")
		}
		if index == esindex.Name("gogame_replays") && r.RoomType != 19 && r.RoomType != 101 {
			return status.Error(codes.InvalidArgument, "invalid replay game")
		}
	}
	return nil
}

func (dd *DBDao) storeCenterReports(ctx context.Context, es *ESDao, records []UserCenterRecord, index string) (any, error) {
	if err := validateCenterReportBatch(records, index); err != nil {
		return nil, err
	}
	verified := map[string]bool{}
	for _, r := range records {
		key := fmt.Sprintf("%d:%d:%s", r.AgentID, r.UserID, r.Currency)
		if !verified[key] {
			if err := dd.centerIdentity(ctx, r.UserID, r.AgentID, r.Currency); err != nil {
				return nil, err
			}
			verified[key] = true
		}
	}
	if _, err := dd.indexCenterRecords(ctx, es, records, index); err != nil {
		return nil, err
	}
	return map[string]any{"stored": len(records), "durable": true, "indexed": true}, nil
}
