package gogame

import (
	"encoding/json"
	"testing"
)

func TestAwardJSONHasCurrency(t *testing.T) {
	var ranks []AwardRank
	if err := json.Unmarshal([]byte(mustJSON([]AwardRank{
		{Rank: 1, Items: []AwardItem{{Kind: RewardKindMoney, Currency: "CNY", Amount: "10000"}}},
	})), &ranks); err != nil {
		t.Fatal(err)
	}
	if ranks[0].Items[0].Currency == "" {
		t.Fatal("money reward missing currency")
	}
}

func TestEnvelopeDataIsString(t *testing.T) {
	raw, err := NewEnvelope(EventMatchState, map[string]any{"id": 1, "state": MatchStateRegistering})
	if err != nil {
		t.Fatal(err)
	}
	var env Envelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		t.Fatal(err)
	}
	if env.Event != EventMatchState {
		t.Fatalf("event=%s", env.Event)
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(env.Data), &data); err != nil {
		t.Fatal(err)
	}
	if data["id"].(float64) != 1 {
		t.Fatalf("id=%v", data["id"])
	}
}

func TestValidateAwardJSONRequiresCurrency(t *testing.T) {
	if err := ValidateAwardJSON(`[{"rank":1,"items":[{"kind":"money","amount":"10000"}]}]`); err == nil {
		t.Fatal("expected currency error")
	}
	if err := ValidateAwardJSON(`[{"rank":1,"items":[{"kind":"money","currency":"CNY","amount":"10000"}]}]`); err != nil {
		t.Fatal(err)
	}
}

func TestChampionshipNameIsJSONString(t *testing.T) {
	s := langName("德州每日免费赛", "Daily Free Tournament", "NLH Daily Freeroll")
	var items []map[string]string
	if err := json.Unmarshal([]byte(s), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 || items[0]["lang"] != "zh_cn" {
		t.Fatalf("unexpected lang payload: %s", s)
	}
}
