package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestScratchHistoryReadsUnifiedSettlementIndexAndChecksOriginalRoomIdentity(t *testing.T) {
	for _, room := range []int{61, 63} {
		t.Run(fmt.Sprint(room), func(t *testing.T) {
			calls := 0
			dd, es, base := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				raw, _ := io.ReadAll(r.Body)
				if !strings.Contains(r.URL.Path, "go_gp_settlement") || strings.Contains(r.URL.Path, "gogame_records") {
					t.Error("wrong history index", r.URL.Path)
				}
				for _, scope := range []string{`"userId":42`, `"agentId":7`, `"gameId":91061`, `"currency":{"query":"CNY"}`, `"complete":true`, `"playedDate"`, `"order":"desc"`} {
					if !strings.Contains(string(raw), scope) {
						t.Error("missing financial query scope", scope, string(raw))
					}
				}
				init := fmt.Sprintf(`{"roomType":%d,"roomId":61010026,"gameRoomId":62010026,"serialNo":"original","faceValue":100,"amount":-100,"huCount":0,"multiplier":0,"huMask":[2,2,2,2,2,2,2,2,2,2,2,2,2,2,2,2,2,2,2,2]}`, room)
				rows := []any{}
				for _, override := range []map[string]any{{}, {"userId": 43}, {"agentId": 8}, {"currency": "USD"}, {"gameId": 91063}, {"complete": false}, {"log": strings.Replace(init, "61010026", "61010027", 1)}} {
					row := map[string]any{"userId": 42, "agentId": 7, "gameId": 91061, "currency": "CNY", "complete": true, "log": init, "init": `{"roomId":999,"serialNo":"wrong-init"}`}
					for key, value := range override {
						row[key] = value
					}
					rows = append(rows, map[string]any{"_source": row})
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{"hits": map[string]any{"hits": rows}})
			})
			q := ScratchHistoryQuery{UserID: 42, AgentID: 7, Currency: "CNY", GameID: 91061, RoomType: room, RoomID: 61010026, GameRoomID: 62010026, Size: 100, StartTime: base.StartTime, EndTime: base.EndTime}
			result, err := dd.ScratchHistory(context.Background(), es, q)
			if err != nil {
				t.Fatal(err)
			}
			rows := result.(map[string]any)["records"]
			raw, _ := json.Marshal(rows)
			var decoded []json.RawMessage
			_ = json.Unmarshal(raw, &decoded)
			if calls != 1 || len(decoded) != 1 || !strings.Contains(string(raw), "original") {
				t.Fatal("wrong original financial history", string(raw))
			}
			q.AgentID = 8
			if _, err = dd.ScratchHistory(context.Background(), es, q); err == nil || calls != 1 {
				t.Fatal("spoofed account reached ES", err, calls)
			}
		})
	}
}

func TestScratchHistoryReadsFurtherPagesWhenOtherRoomsFillFirstFinancialPage(t *testing.T) {
	calls := 0
	dd, es, base := centerFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		var query map[string]any
		json.NewDecoder(r.Body).Decode(&query)
		if query["from"] != float64((calls-1)*100) {
			t.Error("wrong source pagination", query)
		}
		room, count := 61010027, 100
		if calls == 2 {
			room, count = 61010026, 1
		}
		hits := []any{}
		for i := 0; i < count; i++ {
			hits = append(hits, map[string]any{"_source": map[string]any{"userId": 42, "agentId": 7, "gameId": 91061, "currency": "CNY", "complete": true, "log": fmt.Sprintf(`{"roomType":61,"roomId":%d,"gameRoomId":62010026,"serialNo":"paid"}`, room)}})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"hits": map[string]any{"hits": hits}})
	})
	result, err := dd.ScratchHistory(context.Background(), es, ScratchHistoryQuery{UserID: 42, AgentID: 7, Currency: "CNY", GameID: 91061, RoomType: 61, RoomID: 61010026, GameRoomID: 62010026, Size: 100, StartTime: base.StartTime, EndTime: base.EndTime})
	if err != nil || calls != 2 {
		t.Fatal(result, err, calls)
	}
}
