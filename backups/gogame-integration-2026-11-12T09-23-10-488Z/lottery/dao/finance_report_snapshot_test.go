package dao

import (
	"encoding/json"
	"testing"
)

func TestSeatNetCommitsFinalReportPayloadWithMoneyAndTerminalSnapshot(t *testing.T) {
	rd, batch, before, terminal := seatNetFixture(t)
	var snapshot map[string]any
	_ = json.Unmarshal([]byte(terminal), &snapshot)
	settlement := snapshot["settlement"].(map[string]any)
	settlement["reports"] = []any{map[string]any{"userId": 42, "roundID": "net-round", "log": "original-final-result", "playedDate": 123456789}}
	payload, _ := json.Marshal(snapshot)
	if _, err := rd.SettleSeatNet("net-round", "report-fingerprint", before, string(payload), batch); err != nil {
		t.Fatal(err)
	}
	raw, err := rd.HGet("lottery_finance_rounds", "net-round")
	if err != nil {
		t.Fatal(err)
	}
	var committed map[string]any
	if json.Unmarshal([]byte(raw), &committed) != nil {
		t.Fatal(raw)
	}
	final := committed["settlement"].(map[string]any)
	report := final["reports"].([]any)[0].(map[string]any)
	if report["log"] != "original-final-result" || report["playedDate"] != float64(123456789) {
		t.Fatal("committed funds lost their original report", committed)
	}
	results := final["results"].([]any)
	if results[0].(map[string]any)["currency"] != "998.77" {
		t.Fatal(committed)
	}
	// A lost acknowledgement returns the same original terminal snapshot.
	if _, err = rd.SettleSeatNet("net-round", "report-fingerprint", before, string(payload), batch); err != nil {
		t.Fatal(err)
	}
	after, _ := rd.HGet("lottery_finance_rounds", "net-round")
	if after != raw {
		t.Fatal("replay rewrote the report snapshot")
	}
}
