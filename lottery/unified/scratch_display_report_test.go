package unified

import (
	"app/entity"
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/olivere/elastic/v7"
	"lottery/dao"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func paidScratchReport(roomType int) *financeRound {
	kind := "ggl1"
	if roomType == 63 {
		kind = "ggl2"
	}
	mask := make([]int, 20)
	for i := range mask {
		mask[i] = 2
	}
	mask[0] = 1
	ticket := &scratchTicket{Serial: "sold-original-ticket", RoundID: "paid-original-round", UserID: 42, Currency: "CNY", Kind: kind, Face: 100, Win: 50, Mask: mask, Count: 1, Multiplier: .5, RoomID: 61010026, GameRoomID: 62010026, RoomType: roomType, ClaimID: "original-claim"}
	record, _ := json.Marshal(map[string]any{"serialNo": ticket.Serial, "faceValue": ticket.Face, "amount": ticket.Win - ticket.Face, "huMask": mask, "huCount": 1, "multiplier": .5})
	ticket.Record = string(record)
	return &financeRound{RoundID: ticket.RoundID, Agent: 7, GameID: 91061, State: string(financeStateSettled), Scratch: ticket, Settlement: &financeSettlement{SettlementID: ticket.ClaimID, Mode: financeModeNormal, Status: financeSettlementStatusCompleted, CompletedAt: time.Now().Unix(), Results: []*financeSettlementResult{{UserID: 42, Code: 0, Currency: "999.50"}}, Reports: []*entity.CacheRecordsReq{{AgentId: 7, UserId: 42, RoundID: ticket.RoundID, Currency: "CNY", Bet: 1, Win: .5, Complete: true, Log: ticket.Record}}}}
}

func TestPaidTicketDisplayIsDerivedFromOriginalTicketAndOriginalClaimTime(t *testing.T) {
	for _, roomType := range []int{61, 63} {
		round := paidScratchReport(roomType)
		original := round.Settlement.Reports[0].Log
		if !scratchDisplaySnapshotStored(round) {
			t.Fatal("completed original ticket not recognized")
		}
		reports, err := settlementReportBatch(round)
		if err != nil || len(reports) != 1 {
			t.Fatal(reports, err)
		}
		var envelope map[string]any
		if json.Unmarshal([]byte(reports[0].Log), &envelope) != nil {
			t.Fatal(reports)
		}
		display := envelope["completedDisplayReport"].(map[string]any)
		detail := display["detail"].(map[string]any)
		game := "ggl-mjhl"
		if roomType == 63 {
			game = "ggl-queen"
		}
		if display["id"] != "7:"+game+":sold-original-ticket:42" || display["gameId"] != game || display["endedAt"] != float64(round.Settlement.CompletedAt*1000) || display["roomId"] != float64(round.Scratch.RoomID) || display["net"] != float64(-50) || display["effectiveBet"] != float64(100) {
			t.Fatal(display)
		}
		if detail["serialFull"] != round.Scratch.Serial || detail["huCount"] != float64(1) || detail["winInfo"].(map[string]any)["amount"] != float64(-50) {
			t.Fatal(detail)
		}
		if round.Settlement.Reports[0].Log != original {
			t.Fatal("projection rewrote original ticket")
		}
		repeated, err := settlementReportBatch(round)
		if err != nil || repeated[0].Log != reports[0].Log {
			t.Fatal("retry changed original display", err)
		}
	}
}

func TestUnclaimedRefundedOrInconsistentTicketsCannotMasqueradeAsCompletedDisplays(t *testing.T) {
	for _, bad := range []string{"pending", "refund", "amount", "identity", "mask", "wrong-claim"} {
		round := paidScratchReport(61)
		switch bad {
		case "pending":
			round.Settlement.Status = financeSettlementStatusClaimed
		case "refund":
			round.State = string(financeStateVoided)
			round.Settlement.Mode = financeModeVoidRefund
		case "amount":
			round.Scratch.Win++
		case "identity":
			round.Settlement.Reports[0].UserId = 43
		case "mask":
			round.Scratch.Count++
		case "wrong-claim":
			round.Settlement.SettlementID = "another"
		}
		if scratchDisplaySnapshotStored(round) {
			t.Fatal("invalid ticket confirmed display", bad)
		}
	}
}

func TestNewServiceIndexesPaidTicketWithoutGameResponseOrSecondClaim(t *testing.T) {
	indices := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scanner := bufio.NewScanner(r.Body)
		count := 0
		for scanner.Scan() {
			var head map[string]map[string]any
			_ = json.Unmarshal(scanner.Bytes(), &head)
			for _, operation := range head {
				indices = append(indices, operation["_index"].(string))
			}
			scanner.Scan()
			count++
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"errors":false,"items":[`)
		for i := 0; i < count; i++ {
			if i > 0 {
				fmt.Fprint(w, ",")
			}
			fmt.Fprint(w, `{"index":{"status":201}}`)
		}
		fmt.Fprint(w, `]}`)
	}))
	defer server.Close()
	client, err := elastic.NewClient(elastic.SetURL(server.URL), elastic.SetSniff(false), elastic.SetHealthcheck(false))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Stop()
	round := paidScratchReport(61)
	raw, _ := json.Marshal(round)
	var restored financeRound
	_ = json.Unmarshal(raw, &restored)
	service := &LotteryService{es: dao.NewESDao(client), roundLock: &sync.RWMutex{}, rounds: map[string]*financeRound{restored.RoundID: &restored}}
	if _, err = service.projectSettlementReports(1); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(indices) != "[go_gp_settlement go_gogame_records]" {
		t.Fatal(indices)
	}
	if !restored.Settlement.ReportsIndexed || restored.Scratch.Claimed {
		t.Fatal("indexing re-claimed a ticket or failed confirmation")
	}
}
