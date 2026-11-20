package dao

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func panelSeatRoomType(roomType int) bool {
	switch roomType {
	case 19, 24, 25, 26, 27, 28, 30, 31, 32:
		return true
	}
	return false
}

type nativePanelTotals struct{ Total, EffectiveBetHands, NetWinHands, Known, Voluntary, Raised, TableHands, TableKnown, TableVoluntary int64 }

func zjhPanelActions(stats nativePanelTotals, total int64, summary map[string]any) error {
	if stats.Known < 0 || stats.Known > total || stats.Voluntary < 0 || stats.Voluntary > stats.Known || stats.Raised < 0 || stats.Raised > stats.Voluntary || stats.TableHands < 0 || stats.TableHands > total || stats.TableKnown < 0 || stats.TableKnown > min(stats.Known, stats.TableHands) || stats.TableVoluntary < 0 || stats.TableVoluntary > min(stats.Voluntary, stats.TableKnown) {
		return status.Error(codes.Unavailable, "invalid saved ZJH action totals")
	}
	summary["behaviorHands"], summary["voluntaryHands"], summary["raisedHands"] = stats.Known, stats.Voluntary, stats.Raised
	summary["tableHands"], summary["tableBehaviorHands"], summary["tableVoluntaryHands"] = stats.TableHands, stats.TableKnown, stats.TableVoluntary
	return nil
}

func (dd *DBDao) nonHoldemPanelTotals(ctx context.Context, in playerPanelInput, now int64) (any, error) {
	if !panelSeatRoomType(in.RoomType) {
		return nil, status.Error(codes.InvalidArgument, "invalid player statistics game")
	}
	var stats nativePanelTotals
	query := "COUNT(*) AS total,COALESCE(SUM(CASE WHEN CAST(JSON_UNQUOTE(JSON_EXTRACT(payload,'$.effectiveBet')) AS DECIMAL(20,4))>0 THEN 1 ELSE 0 END),0) AS effective_bet_hands,COALESCE(SUM(CASE WHEN net>0 THEN 1 ELSE 0 END),0) AS net_win_hands"
	var args []any
	if in.RoomType == 25 {
		query += ",COALESCE(SUM(counters_known),0) AS known,COALESCE(SUM(CASE WHEN counters_known=1 AND voluntary=1 THEN 1 ELSE 0 END),0) AS voluntary,COALESCE(SUM(CASE WHEN counters_known=1 AND raised=1 THEN 1 ELSE 0 END),0) AS raised,COALESCE(SUM(room_id=?),0) AS table_hands,COALESCE(SUM(CASE WHEN room_id=? AND counters_known=1 THEN 1 ELSE 0 END),0) AS table_known,COALESCE(SUM(CASE WHEN room_id=? AND counters_known=1 AND voluntary=1 THEN 1 ELSE 0 END),0) AS table_voluntary"
		args = []any{in.RoomID, in.RoomID, in.RoomID}
	}
	if err := dd.db.WithContext(ctx).Table("gg_player_panel_hand").Select(query, args...).Where("agent_id=? AND player_id=? AND currency=? AND room_type=? AND ended_at<=?", in.AgentID, in.TargetID, in.Currency, in.RoomType, now).Scan(&stats).Error; err != nil {
		return nil, err
	}
	if stats.Total < 0 || stats.Total > cashMax || stats.EffectiveBetHands < 0 || stats.EffectiveBetHands > stats.Total || stats.NetWinHands < 0 || stats.NetWinHands > stats.Total {
		return nil, status.Error(codes.Unavailable, "invalid saved player hand totals")
	}
	summary := map[string]any{"totalHands": stats.Total, "effectiveBetHands": stats.EffectiveBetHands, "effectiveBetHandPercent": panelRatio(stats.EffectiveBetHands, stats.Total), "netWinHands": stats.NetWinHands, "winPercent": panelRatio(stats.NetWinHands, stats.Total)}
	if in.RoomType == 25 {
		if err := zjhPanelActions(stats, stats.Total, summary); err != nil {
			return nil, err
		}
	}
	return map[string]any{"records": []any{}, "complete": false, "statistics": summary}, nil
}
