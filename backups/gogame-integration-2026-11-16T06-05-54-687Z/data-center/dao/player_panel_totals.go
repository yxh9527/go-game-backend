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
	var total int64
	if err := dd.db.WithContext(ctx).Table("gg_player_panel_hand").Where("agent_id=? AND player_id=? AND currency=? AND room_type=? AND ended_at<=?", in.AgentID, in.TargetID, in.Currency, in.RoomType, now).Count(&total).Error; err != nil {
		return nil, err
	}
	return map[string]any{"records": []any{}, "complete": false, "statistics": map[string]any{"totalHands": total}}, nil
}
