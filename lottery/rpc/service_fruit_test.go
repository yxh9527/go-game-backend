package rpc

import (
	"testing"

	"micro_service/services"
)

// TestFruitSettleRecordID 验证同一房间内每位玩家使用自己的数字注单号，且不修改房间幂等标识。
func TestFruitSettleRecordID(t *testing.T) {
	const roundID = "07d194f1-0f5e-436e-accf-07c36d7c74d2"
	req := &services.FruitSettleRoundReq{AgentId: 0, GameId: 7001, CurrencyType: "CNY", RoundId: roundID}
	for _, item := range []struct {
		userID     uint32
		recordID   string
		recordJSON string
	}{
		{34, "27536544860", `{"recordId":"27536544860","userId":34,"gameId":7001,"currencySymbol":"CNY"}`},
		{35, "27536544861", `{"recordId":"27536544861","userId":35,"agentId":0,"gameId":7001,"currencySymbol":"CNY"}`},
	} {
		player := &services.FruitSettlePlayer{UserId: item.userID, RecordJson: item.recordJSON}
		id, ok := fruitSettleRecordID(req, player)
		if !ok || id != item.recordID {
			t.Fatalf("user %d: got %q, %v; want %q", item.userID, id, ok, item.recordID)
		}
		if req.RoundId != roundID || player.RecordJson != item.recordJSON {
			t.Fatal("record parsing changed the room ID or original record")
		}
	}
}

// TestFruitSettleRecordIDRejectsInvalid 验证错误记录在入账前被拒绝，避免把别人的注单号写给当前玩家。
func TestFruitSettleRecordIDRejectsInvalid(t *testing.T) {
	req := &services.FruitSettleRoundReq{GameId: 7001, CurrencyType: "CNY", RoundId: "room"}
	for _, recordJSON := range []string{
		"", "null", "{", "[]",
		`{"userId":34,"gameId":7001,"currencySymbol":"CNY"}`,
		`{"recordId":" ","userId":34,"gameId":7001,"currencySymbol":"CNY"}`,
		`{"recordId":" 27536544860","userId":34,"gameId":7001,"currencySymbol":"CNY"}`,
		`{"recordId":27536544860,"userId":34,"gameId":7001,"currencySymbol":"CNY"}`,
		`{"recordId":"27536544860","userId":35,"gameId":7001,"currencySymbol":"CNY"}`,
		`{"recordId":"27536544860","userId":34,"gameId":7002,"currencySymbol":"CNY"}`,
		`{"recordId":"27536544860","userId":34,"gameId":7001,"currencySymbol":"USD"}`,
		`{"recordId":"27536544860","userId":34,"agentId":1,"gameId":7001,"currencySymbol":"CNY"}`,
	} {
		id, ok := fruitSettleRecordID(req, &services.FruitSettlePlayer{UserId: 34, RecordJson: recordJSON})
		if ok || id != "" {
			t.Fatalf("accepted invalid record: %s", recordJSON)
		}
	}
}
