package unified

import (
	"app/entity"
	"encoding/json"
	"fmt"
	"github.com/shopspring/decimal"
)

// Derive only values already fixed on the sold ticket and completed settlement.
// This can run after an instance change without drawing, claiming, or paying.
func scratchDisplayReport(round *financeRound, financial *entity.CacheRecordsReq) (map[string]any, error) {
	if round == nil || round.Scratch == nil || round.Settlement == nil || round.Settlement.Status != financeSettlementStatusCompleted || round.State != string(financeStateSettled) || round.Settlement.Mode != financeModeNormal {
		return nil, nil
	}
	ticket := round.Scratch
	if financial == nil || !financial.Complete || financial.UserId != ticket.UserID || financial.AgentId != round.Agent || financial.Currency != ticket.Currency || financial.RoundID != round.RoundID || ticket.RoundID != round.RoundID || round.Settlement.SettlementID != ticket.ClaimID {
		return nil, fmt.Errorf("scratch final display identity mismatch")
	}
	if ticket.RoomType != 61 && ticket.RoomType != 63 || ticket.RoomID <= 0 || ticket.Serial == "" || ticket.Face <= 0 || ticket.Win < 0 || len(ticket.Mask) != 20 || round.Settlement.CompletedAt <= 0 {
		return nil, fmt.Errorf("invalid final ticket display source")
	}
	if !decimal.NewFromFloat(financial.Bet).Equal(decimalFromCent(ticket.Face)) || !decimal.NewFromFloat(financial.Win).Equal(decimalFromCent(ticket.Win)) {
		return nil, fmt.Errorf("scratch display amounts differ from sold ticket")
	}
	// The ticket's immutable public record was created when it was sold. Keep
	// those rule/result fields rather than interpreting a current reel profile.
	var original map[string]any
	if json.Unmarshal([]byte(ticket.Record), &original) != nil {
		return nil, fmt.Errorf("original paid ticket record invalid")
	}
	if original["serialNo"] != ticket.Serial || original["faceValue"] != float64(ticket.Face) || financial.Log != ticket.Record {
		return nil, fmt.Errorf("paid ticket record differs from original financial report")
	}
	count := 0
	for _, symbol := range ticket.Mask {
		if symbol == 1 {
			count++
		}
	}
	if count != ticket.Count {
		return nil, fmt.Errorf("paid ticket mask and count disagree")
	}
	game := "ggl-mjhl"
	if ticket.RoomType == 63 {
		game = "ggl-queen"
	}
	info := map[string]any{"serialNo": ticket.Serial, "faceValue": ticket.Face, "amount": ticket.Win - ticket.Face, "huMask": ticket.Mask, "huCount": ticket.Count, "multiplier": ticket.Multiplier}
	detail := map[string]any{"gameRoomId": ticket.RoomID, "currInning": 1, "serialFull": ticket.Serial, "faceValue": ticket.Face, "winAmount": ticket.Win, "huMask": ticket.Mask, "huCount": ticket.Count, "multiplier": ticket.Multiplier, "winInfo": info,
		"firstGameList": []any{}, "innerPlayerList": []any{}, "gameEndData": `{"playerList":[],"resultList":[]}`, "currentInningJoin": true,
		"reportMsg": map[string]any{"currentInningJoin": true, "innerReport": false, "reportFree": 0, "playerList": []any{}}}
	return map[string]any{"id": fmt.Sprintf("%d:%s:%s:%d", round.Agent, game, ticket.Serial, ticket.UserID), "userId": ticket.UserID, "agentId": round.Agent, "currency": ticket.Currency,
		"gameId": game, "roomType": ticket.RoomType, "roomId": ticket.RoomID, "inning": 1, "endedAt": round.Settlement.CompletedAt * 1000, "playMode": 0, "privateRoom": false,
		"effectiveBet": ticket.Face, "net": ticket.Win - ticket.Face, "name": financial.Account, "room": map[string]any{}, "players": []any{}, "detail": detail}, nil
}

func attachScratchDisplayReport(round *financeRound, financial *entity.CacheRecordsReq) error {
	display, err := scratchDisplayReport(round, financial)
	if err != nil || display == nil {
		return err
	}
	var payload map[string]any
	if json.Unmarshal([]byte(financial.Log), &payload) != nil {
		return fmt.Errorf("invalid original scratch financial result")
	}
	payload["completedDisplayReport"] = display
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	financial.Log = string(raw)
	return nil
}

func scratchDisplaySnapshotStored(round *financeRound) bool {
	if round == nil || round.Scratch == nil || round.Settlement == nil {
		return false
	}
	for _, financial := range round.Settlement.Reports {
		display, err := scratchDisplayReport(round, financial)
		if err == nil && display != nil {
			return true
		}
	}
	return false
}
