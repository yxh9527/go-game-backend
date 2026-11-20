package dao

import (
	"app/tables/gogame"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

type playerPanelRecord struct {
	ID            string          `json:"id"`
	AgentID       uint32          `json:"agentId"`
	UserID        uint32          `json:"userId"`
	Currency      string          `json:"currency"`
	RoomID        int64           `json:"roomId"`
	Inning        int64           `json:"inning"`
	EndedAt       int64           `json:"endedAt"`
	Net           int64           `json:"net"`
	EffectiveBet  int64           `json:"effectiveBet"`
	PlayMode      int             `json:"playMode"`
	Players       json.RawMessage `json:"players"`
	CommonCards   json.RawMessage `json:"commonCards"`
	Events        json.RawMessage `json:"events"`
	Raised        bool            `json:"raised"`
	Voluntary     bool            `json:"voluntary"`
	CountersKnown bool            `json:"countersKnown"`
}
type playerPanelInput struct {
	Page     int    `json:"page"`
	Size     int    `json:"size"`
	AgentID  uint32 `json:"agentId"`
	UserID   uint32 `json:"userId"`
	TargetID uint32 `json:"targetId"`
	Currency string `json:"currency"`
	RoomID   int64  `json:"roomId"`
	PropID   int    `json:"propId"`
	Patch    struct {
		Name    *string `json:"remarkName"`
		Message *string `json:"remarkMessage"`
		Mark    *int    `json:"mark"`
	} `json:"patch"`
	Records []playerPanelRecord `json:"records"`
}

func panelDay(now time.Time) (time.Time, time.Time, time.Time) {
	local := now.In(time.FixedZone("Asia/Shanghai", 8*3600))
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
	week := today.AddDate(0, 0, -(int(today.Weekday())+6)%7)
	return today, week, today.AddDate(0, 0, -6)
}
func panelNote(row gogame.PlayerNote) map[string]any {
	return map[string]any{"remarkName": row.RemarkName, "remarkMessage": row.RemarkMessage, "mark": row.Mark}
}
func panelRatio(n, total int64) float64 {
	if total == 0 {
		return 0
	}
	return math.Round(float64(n)*10000/float64(total)) / 100
}
func (dd *DBDao) gogamePlayerPanel(ctx context.Context, operation, payload string) (any, error) {
	if dd.db == nil {
		return nil, status.Error(codes.Unavailable, "player panel database unavailable")
	}
	var in playerPanelInput
	if err := DecodeGogameJSON(payload, &in); err != nil {
		return nil, err
	}
	if operation == "playerPanelWrite" {
		return dd.writePlayerPanel(ctx, in.Records)
	}
	if in.UserID == 0 || in.UserID >= 90000000 || !cashCurrency.MatchString(in.Currency) {
		return nil, status.Error(codes.InvalidArgument, "invalid player panel account")
	}
	if err := dd.centerIdentity(ctx, in.UserID, in.AgentID, in.Currency); err != nil {
		return nil, err
	}
	now := time.Now()
	today, week, seven := panelDay(now)
	switch operation {
	case "playerPanelQuota", "playerPanelGift":
		row := gogame.PlayerEmojiDaily{AgentId: in.AgentID, PlayerId: in.UserID, Day: today.Format("2006-01-02")}
		query := "agent_id=? AND player_id=? AND day=?"
		if operation == "playerPanelQuota" {
			err := dd.db.WithContext(ctx).Where(query, row.AgentId, row.PlayerId, row.Day).First(&row).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
			return map[string]any{"remaining": max(0, 20-row.Used)}, nil
		}
		if in.TargetID == 0 || in.TargetID == in.UserID || in.PropID < 0 || in.PropID > 14 {
			return nil, status.Error(codes.InvalidArgument, "invalid ordinary prop")
		}
		err := dd.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
				return err
			}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(query, row.AgentId, row.PlayerId, row.Day).First(&row).Error; err != nil {
				return err
			}
			if row.Used >= 20 {
				return status.Error(codes.FailedPrecondition, "daily free props exhausted")
			}
			row.Used++
			return tx.Model(&gogame.PlayerEmojiDaily{}).Where(query, row.AgentId, row.PlayerId, row.Day).Update("used", row.Used).Error
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"remaining": 20 - row.Used}, nil
	case "playerPanelNoteGet", "playerPanelNoteSet":
		if in.TargetID == 0 {
			return nil, status.Error(codes.InvalidArgument, "target required")
		}
		row := gogame.PlayerNote{AgentId: in.AgentID, PlayerId: in.UserID, TargetId: in.TargetID}
		query := "agent_id=? AND player_id=? AND target_id=?"
		if operation == "playerPanelNoteGet" {
			err := dd.db.WithContext(ctx).Where(query, row.AgentId, row.PlayerId, row.TargetId).First(&row).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
			return panelNote(row), nil
		}
		valid := func(v *string, limit int) bool {
			return v == nil || utf8.ValidString(*v) && utf8.RuneCountInString(*v) <= limit && !strings.ContainsAny(*v, ";\x00\n\r\t")
		}
		if !valid(in.Patch.Name, 32) || !valid(in.Patch.Message, 256) || in.Patch.Mark != nil && (*in.Patch.Mark < 0 || *in.Patch.Mark > 6) {
			return nil, status.Error(codes.InvalidArgument, "invalid player note")
		}
		if in.Patch.Name == nil && in.Patch.Message == nil && in.Patch.Mark == nil {
			return nil, status.Error(codes.InvalidArgument, "empty player note")
		}
		err := dd.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
				return err
			}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(query, row.AgentId, row.PlayerId, row.TargetId).First(&row).Error; err != nil {
				return err
			}
			if in.Patch.Name != nil {
				row.RemarkName = strings.TrimSpace(*in.Patch.Name)
			}
			if in.Patch.Message != nil {
				row.RemarkMessage = strings.TrimSpace(*in.Patch.Message)
			}
			if in.Patch.Mark != nil {
				row.Mark = *in.Patch.Mark
			}
			row.UpdatedAt = now.UnixMilli()
			return tx.Save(&row).Error
		})
		if err != nil {
			return nil, err
		}
		return panelNote(row), nil
	case "playerPanelRecords":
		if in.TargetID == 0 || in.TargetID >= 90000000 || in.RoomID <= 0 || in.Page < 1 || in.Size < 1 || in.Size > 100 || (in.Page-1)*in.Size >= 10000 {
			return nil, status.Error(codes.InvalidArgument, "invalid player records query")
		}
		if err := dd.centerIdentity(ctx, in.TargetID, in.AgentID, in.Currency); err != nil {
			return nil, err
		}
		var totals struct{ Total, Score int64 }
		where := "agent_id=? AND player_id=? AND currency=? AND ended_at>=? AND ended_at<=?"
		if err := dd.db.WithContext(ctx).Raw("SELECT COUNT(*) AS total,COALESCE(SUM(net),0) AS score FROM gg_player_panel_hand WHERE "+where, in.AgentID, in.TargetID, in.Currency, seven.UnixMilli(), now.UnixMilli()).Scan(&totals).Error; err != nil {
			return nil, err
		}
		var rows []gogame.PlayerPanelHand
		if err := dd.db.WithContext(ctx).Where(where, in.AgentID, in.TargetID, in.Currency, seven.UnixMilli(), now.UnixMilli()).Order("ended_at DESC,record_key DESC").Limit(in.Size).Offset((in.Page - 1) * in.Size).Find(&rows).Error; err != nil {
			return nil, err
		}
		records := make([]json.RawMessage, 0, len(rows))
		for _, r := range rows {
			records = append(records, json.RawMessage(r.Payload))
		}
		return map[string]any{"records": records, "total": totals.Total, "score": totals.Score, "paged": true}, nil
	case "playerPanelRead":
		if in.TargetID == 0 || in.TargetID >= 90000000 || in.RoomID <= 0 {
			return nil, status.Error(codes.InvalidArgument, "invalid player statistics target")
		}
		if err := dd.centerIdentity(ctx, in.TargetID, in.AgentID, in.Currency); err != nil {
			return nil, err
		}
		var stats struct{ TotalHands, Known, Raised, Voluntary, TableHands, TableVoluntary, TodayScore, WeekScore, SevenHands int64 }
		sql := "SELECT COUNT(*) AS total_hands,COALESCE(SUM(counters_known),0) AS known,COALESCE(SUM(raised),0) AS raised,COALESCE(SUM(voluntary),0) AS voluntary,COALESCE(SUM(room_id=?),0) AS table_hands,COALESCE(SUM(CASE WHEN room_id=? THEN voluntary ELSE 0 END),0) AS table_voluntary,COALESCE(SUM(CASE WHEN ended_at>=? THEN net ELSE 0 END),0) AS today_score,COALESCE(SUM(CASE WHEN ended_at>=? THEN net ELSE 0 END),0) AS week_score,COALESCE(SUM(ended_at>=?),0) AS seven_hands FROM gg_player_panel_hand WHERE agent_id=? AND player_id=? AND currency=?"
		if err := dd.db.WithContext(ctx).Raw(sql, in.RoomID, in.RoomID, today.UnixMilli(), week.UnixMilli(), seven.UnixMilli(), in.AgentID, in.TargetID, in.Currency).Scan(&stats).Error; err != nil {
			return nil, err
		}
		var rows []gogame.PlayerPanelHand
		if err := dd.db.WithContext(ctx).Where("agent_id=? AND player_id=? AND currency=? AND ended_at>=?", in.AgentID, in.TargetID, in.Currency, seven.UnixMilli()).Order("ended_at DESC, record_key DESC").Limit(10000).Find(&rows).Error; err != nil {
			return nil, err
		}
		records := make([]json.RawMessage, 0, len(rows))
		for _, r := range rows {
			records = append(records, json.RawMessage(r.Payload))
		}
		summary := map[string]any{"totalHands": stats.TotalHands, "todayScore": stats.TodayScore, "oneWeekScore": stats.WeekScore, "oneWeekHands": stats.SevenHands,
			"flipManualBetPercent": panelRatio(stats.Raised, stats.TotalHands), "joinPercent": panelRatio(stats.Voluntary, stats.TotalHands), "tableJoinPercent": panelRatio(stats.TableVoluntary, stats.TableHands)}
		if stats.Known != stats.TotalHands {
			summary["flipManualBetPercent"] = "--"
			summary["joinPercent"] = "--"
			summary["tableJoinPercent"] = "--"
		}
		return map[string]any{"records": records, "statistics": summary, "complete": int64(len(rows)) == stats.SevenHands}, nil
	}
	return nil, status.Error(codes.Unimplemented, "unknown player panel operation")
}
func (dd *DBDao) writePlayerPanel(ctx context.Context, records []playerPanelRecord) (any, error) {
	if len(records) == 0 || len(records) > 100 {
		return nil, status.Error(codes.InvalidArgument, "invalid player hand batch")
	}
	rows := make([]gogame.PlayerPanelHand, 0, len(records))
	verified := make(map[string]bool)
	for _, r := range records {
		if r.UserID == 0 || r.UserID >= 90000000 || r.ID == "" || len(r.ID) > 256 || r.RoomID <= 0 || r.Inning <= 0 || r.EndedAt <= 0 || !cashCurrency.MatchString(r.Currency) || r.Net > cashMax || r.Net < -cashMax || r.EffectiveBet < 0 || r.EffectiveBet > cashMax {
			return nil, status.Error(codes.InvalidArgument, "invalid settled player hand")
		}
		scope := fmt.Sprintf("%d:%d:%s", r.AgentID, r.UserID, r.Currency)
		if !verified[scope] {
			if err := dd.centerIdentity(ctx, r.UserID, r.AgentID, r.Currency); err != nil {
				return nil, err
			}
			verified[scope] = true
		}
		var players []map[string]any
		if err := json.Unmarshal(r.Players, &players); err != nil || len(players) == 0 {
			return nil, status.Error(codes.InvalidArgument, "player hand participants required")
		}
		raw, err := json.Marshal(r)
		if err != nil {
			return nil, err
		}
		rows = append(rows, gogame.PlayerPanelHand{RecordKey: fmt.Sprintf("%x", sha256.Sum256([]byte(scope+":"+r.ID))), AgentId: r.AgentID, PlayerId: r.UserID, Currency: r.Currency,
			RoomId: r.RoomID, EndedAt: r.EndedAt, Net: r.Net, Raised: r.Raised, Voluntary: r.Voluntary, CountersKnown: r.CountersKnown, Payload: string(raw)})
	}
	err := dd.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error })
	if err != nil {
		return nil, err
	}
	return map[string]any{"stored": len(rows)}, nil
}
