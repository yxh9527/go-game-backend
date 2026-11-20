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

type nativePanelTotals struct{ Total, EffectiveBetHands, NetWinHands, Known, Voluntary, Raised, TableHands, TableKnown, TableVoluntary, Doubled, Split, InsuranceBuy, BetOpportunities, AboveMinimumBet, ManualGrab, PositiveGrab, ClientArrangement, ExtendedArrangement, ClientDiscard, ServerDiscard int64 }

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
	if in.RoomType != 19 {
		query += ",COALESCE(SUM(counters_known),0) AS known,COALESCE(SUM(CASE WHEN counters_known=1 AND voluntary=1 THEN 1 ELSE 0 END),0) AS voluntary,COALESCE(SUM(CASE WHEN counters_known=1 AND raised=1 THEN 1 ELSE 0 END),0) AS raised,COALESCE(SUM(room_id=?),0) AS table_hands,COALESCE(SUM(CASE WHEN room_id=? AND counters_known=1 THEN 1 ELSE 0 END),0) AS table_known,COALESCE(SUM(CASE WHEN room_id=? AND counters_known=1 AND voluntary=1 THEN 1 ELSE 0 END),0) AS table_voluntary"
		args = []any{in.RoomID, in.RoomID, in.RoomID}
	}
	if in.RoomType == 31 {
		query += ",COALESCE(SUM(CASE WHEN counters_known=1 AND JSON_UNQUOTE(JSON_EXTRACT(payload,'$.blackjackActions.doubled'))='true' THEN 1 ELSE 0 END),0) AS doubled,COALESCE(SUM(CASE WHEN counters_known=1 AND JSON_UNQUOTE(JSON_EXTRACT(payload,'$.blackjackActions.split'))='true' THEN 1 ELSE 0 END),0) AS split,COALESCE(SUM(CASE WHEN counters_known=1 AND JSON_UNQUOTE(JSON_EXTRACT(payload,'$.blackjackActions.insuranceBuy'))='true' THEN 1 ELSE 0 END),0) AS insurance_buy"
	}
	if in.RoomType == 27 {
		query += ",COALESCE(SUM(CASE WHEN counters_known=1 AND JSON_UNQUOTE(JSON_EXTRACT(payload,'$.arrangementChoices.clientSubmitted'))='true' THEN 1 ELSE 0 END),0) AS client_arrangement,COALESCE(SUM(CASE WHEN counters_known=1 AND JSON_UNQUOTE(JSON_EXTRACT(payload,'$.arrangementChoices.extended'))='true' THEN 1 ELSE 0 END),0) AS extended_arrangement"
	}
	if in.RoomType == 26 || in.RoomType == 28 || in.RoomType == 30 {
		query += ",COALESCE(SUM(CASE WHEN counters_known=1 AND JSON_UNQUOTE(JSON_EXTRACT(payload,'$.mahjongDiscards.clientDiscard'))='true' THEN 1 ELSE 0 END),0) AS client_discard,COALESCE(SUM(CASE WHEN counters_known=1 AND JSON_UNQUOTE(JSON_EXTRACT(payload,'$.mahjongDiscards.serverDiscard'))='true' THEN 1 ELSE 0 END),0) AS server_discard"
	}
	if in.RoomType == 24 || in.RoomType == 32 {
		query += ",COALESCE(SUM(CASE WHEN counters_known=1 AND JSON_UNQUOTE(JSON_EXTRACT(payload,'$.seatChoices.betOpportunity'))='true' THEN 1 ELSE 0 END),0) AS bet_opportunities,COALESCE(SUM(CASE WHEN counters_known=1 AND JSON_UNQUOTE(JSON_EXTRACT(payload,'$.seatChoices.aboveMinimumBet'))='true' THEN 1 ELSE 0 END),0) AS above_minimum_bet,COALESCE(SUM(CASE WHEN counters_known=1 AND JSON_UNQUOTE(JSON_EXTRACT(payload,'$.seatChoices.manualGrab'))='true' THEN 1 ELSE 0 END),0) AS manual_grab,COALESCE(SUM(CASE WHEN counters_known=1 AND JSON_UNQUOTE(JSON_EXTRACT(payload,'$.seatChoices.positiveGrab'))='true' THEN 1 ELSE 0 END),0) AS positive_grab"
	}
	if err := dd.db.WithContext(ctx).Table("gg_player_panel_hand").Select(query, args...).Where("agent_id=? AND player_id=? AND currency=? AND room_type=? AND ended_at<=?", in.AgentID, in.TargetID, in.Currency, in.RoomType, now).Scan(&stats).Error; err != nil {
		return nil, err
	}
	if stats.Total < 0 || stats.Total > cashMax || stats.EffectiveBetHands < 0 || stats.EffectiveBetHands > stats.Total || stats.NetWinHands < 0 || stats.NetWinHands > stats.Total {
		return nil, status.Error(codes.Unavailable, "invalid saved player hand totals")
	}
	summary := map[string]any{"totalHands": stats.Total, "effectiveBetHands": stats.EffectiveBetHands, "effectiveBetHandPercent": panelRatio(stats.EffectiveBetHands, stats.Total), "netWinHands": stats.NetWinHands, "winPercent": panelRatio(stats.NetWinHands, stats.Total)}
	if in.RoomType != 19 {
		if err := zjhPanelActions(stats, stats.Total, summary); err != nil {
			return nil, err
		}
	}
	if in.RoomType == 31 {
		if stats.Raised != 0 || stats.Doubled < 0 || stats.Doubled > stats.Known || stats.Split < 0 || stats.Split > stats.Known || stats.InsuranceBuy < 0 || stats.InsuranceBuy > stats.Known {
			return nil, status.Error(codes.Unavailable, "invalid saved blackjack action totals")
		}
		summary["doubleHands"], summary["splitHands"], summary["insuranceBuyHands"] = stats.Doubled, stats.Split, stats.InsuranceBuy
	}
	if in.RoomType == 27 {
		if stats.Voluntary != 0 || stats.Raised != 0 || stats.ClientArrangement < 0 || stats.ClientArrangement > stats.Known || stats.ExtendedArrangement < 0 || stats.ExtendedArrangement > stats.Known {
			return nil, status.Error(codes.Unavailable, "invalid saved arrangement choice totals")
		}
		summary["clientArrangementHands"], summary["extendedArrangementHands"] = stats.ClientArrangement, stats.ExtendedArrangement
	}
	if in.RoomType == 26 || in.RoomType == 28 || in.RoomType == 30 {
		if stats.Voluntary != 0 || stats.Raised != 0 || stats.ClientDiscard < 0 || stats.ClientDiscard > stats.Known || stats.ServerDiscard < 0 || stats.ServerDiscard > stats.Known {
			return nil, status.Error(codes.Unavailable, "invalid saved mahjong discard totals")
		}
		summary["clientDiscardHands"], summary["serverDiscardHands"] = stats.ClientDiscard, stats.ServerDiscard
	}
	if in.RoomType == 24 || in.RoomType == 32 {
		if stats.Raised != 0 || stats.BetOpportunities < stats.Voluntary || stats.BetOpportunities > stats.Known || stats.AboveMinimumBet < 0 || stats.AboveMinimumBet > stats.Voluntary || stats.ManualGrab < 0 || stats.ManualGrab > stats.Known || stats.PositiveGrab < 0 || stats.PositiveGrab > stats.ManualGrab || in.RoomType == 32 && (stats.BetOpportunities != stats.Known || stats.ManualGrab != 0 || stats.PositiveGrab != 0) {
			return nil, status.Error(codes.Unavailable, "invalid saved seat choice totals")
		}
		summary["betOpportunityHands"], summary["aboveMinimumBetHands"], summary["manualGrabHands"], summary["positiveGrabHands"] = stats.BetOpportunities, stats.AboveMinimumBet, stats.ManualGrab, stats.PositiveGrab
	}
	return map[string]any{"records": []any{}, "complete": false, "statistics": summary}, nil
}
