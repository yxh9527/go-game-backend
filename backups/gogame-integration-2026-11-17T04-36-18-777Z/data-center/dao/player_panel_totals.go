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

func (dd *DBDao) nonHoldemPanelTotals(ctx context.Context, in playerPanelInput, now int64) (any, error) {
	if !panelSeatRoomType(in.RoomType) {
		return nil, status.Error(codes.InvalidArgument, "invalid player statistics game")
	}
	var stats struct{ Total, EffectiveBetHands, NetWinHands int64 }
	if err := dd.db.WithContext(ctx).Table("gg_player_panel_hand").Select("COUNT(*) AS total,COALESCE(SUM(CASE WHEN CAST(JSON_UNQUOTE(JSON_EXTRACT(payload,'$.effectiveBet')) AS DECIMAL(20,4))>0 THEN 1 ELSE 0 END),0) AS effective_bet_hands,COALESCE(SUM(CASE WHEN net>0 THEN 1 ELSE 0 END),0) AS net_win_hands").Where("agent_id=? AND player_id=? AND currency=? AND room_type=? AND ended_at<=?", in.AgentID, in.TargetID, in.Currency, in.RoomType, now).Scan(&stats).Error; err != nil {
		return nil, err
	}
	if stats.Total < 0 || stats.Total > cashMax || stats.EffectiveBetHands < 0 || stats.EffectiveBetHands > stats.Total || stats.NetWinHands < 0 || stats.NetWinHands > stats.Total {
		return nil, status.Error(codes.Unavailable, "invalid saved player hand totals")
	}
	return map[string]any{"records": []any{}, "complete": false, "statistics": map[string]any{"totalHands": stats.Total, "effectiveBetHands": stats.EffectiveBetHands, "effectiveBetHandPercent": panelRatio(stats.EffectiveBetHands, stats.Total), "netWinHands": stats.NetWinHands, "winPercent": panelRatio(stats.NetWinHands, stats.Total)}}, nil
}
