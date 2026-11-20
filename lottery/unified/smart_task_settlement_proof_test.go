package unified

import (
	"encoding/json"
	"strings"
	"testing"
)

func smartProofRound() *financeRound {
	round := reportRound(1)
	round.GameID, round.Level = 92058, 1
	round.Settlement.SmartTaskProofVersion = 1
	row := round.Settlement.Reports[0]
	row.GameId, row.Level, row.Win = round.GameID, round.Level, row.Bet
	row.RowVersion = 123456789
	row.Log = `{"actualResult":{"banker":[1,2],"player":[3,4]},"completedDisplayReport":{"smartTask":{"taskId":"7:42:fixed-one","attemptId":"attempt-one","inning":1}},"smartTaskSettlement":{"mode":"VOID_REFUND","completedAt":999}}`
	return round
}

func TestSmartTaskProofUsesOriginalCompletedFinancialStateAndReplacesCallerMarker(t *testing.T) {
	round := smartProofRound()
	original := round.Settlement.Reports[0].Log
	rows, err := settlementReportBatch(round)
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(rows[0].Log), &envelope); err != nil {
		t.Fatal(err)
	}
	var proof struct {
		Version      int    `json:"version"`
		Mode         string `json:"mode"`
		State        string `json:"state"`
		RoundID      string `json:"roundId"`
		SettlementID string `json:"settlementId"`
		CompletedAt  int64  `json:"completedAt"`
		TaskID       string `json:"taskId"`
		AttemptID    string `json:"attemptId"`
		Inning       int64  `json:"inning"`
	}
	if err := json.Unmarshal(envelope["smartTaskSettlement"], &proof); err != nil {
		t.Fatal(err)
	}
	if proof.Version != 1 || proof.Mode != financeModeNormal || proof.State != string(financeStateSettled) || proof.RoundID != round.RoundID || proof.SettlementID != round.Settlement.SettlementID || proof.CompletedAt != round.Settlement.CompletedAt || proof.TaskID != "7:42:fixed-one" || proof.AttemptID != "attempt-one" || proof.Inning != 1 {
		t.Fatal(proof)
	}
	if string(envelope["actualResult"]) != `{"banker":[1,2],"player":[3,4]}` || !strings.Contains(string(envelope["completedDisplayReport"]), `"attempt-one"`) {
		t.Fatal("original log detail changed", rows[0].Log)
	}
	if rows[0].Balance != 12.34 || rows[0].Bet != 1 || rows[0].Win != 1 || rows[0].RowVersion != 123456789 || round.Settlement.Reports[0].Log != original {
		t.Fatal("projection changed original financial data")
	}
	// Reconstruct the handler state only from the original persisted snapshot.
	raw, _ := json.Marshal(round)
	var loaded financeRound
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatal(err)
	}
	repeated, err := settlementReportBatch(&loaded)
	if err != nil || repeated[0].Log != rows[0].Log || repeated[0].RowVersion != rows[0].RowVersion {
		t.Fatal("projection retry changed the immutable document", err)
	}
}

func TestSmartTaskRefundOtherModesAndNonTaskLogsCannotRetainCallerNormalProof(t *testing.T) {
	for _, variant := range []string{"refund", "banker", "seat", "no-task"} {
		t.Run(variant, func(t *testing.T) {
			round := smartProofRound()
			switch variant {
			case "refund":
				round.State = string(financeStateVoided)
				round.Settlement.Mode = financeModeVoidRefund
			case "banker":
				round.Settlement.Mode = financeModeBankerNet
			case "seat":
				round.Settlement.Mode = financeModeSeatNet
			case "no-task":
				round.Settlement.Reports[0].Log = `{"actualResult":1,"smartTaskSettlement":{"version":1,"mode":"NORMAL"}}`
			}
			rows, err := settlementReportBatch(round)
			if err != nil || len(rows) != 1 {
				t.Fatal(rows, err)
			}
			var envelope map[string]json.RawMessage
			if err := json.Unmarshal([]byte(rows[0].Log), &envelope); err != nil {
				t.Fatal(err)
			}
			if envelope["smartTaskSettlement"] != nil || envelope["actualResult"] == nil {
				t.Fatal("caller financial proof survived incompatible original state", rows[0].Log)
			}
		})
	}
}

func TestNewSmartTaskProofCannotBeIssuedForUnconfirmedOrMismatchedMoney(t *testing.T) {
	for _, variant := range []string{"claimed", "failed-account", "missing-id", "missing-time", "wrong-game", "wrong-level"} {
		t.Run(variant, func(t *testing.T) {
			round := smartProofRound()
			switch variant {
			case "claimed":
				round.Settlement.Status = financeSettlementStatusClaimed
			case "failed-account":
				round.Settlement.Results[0].Code = 1
			case "missing-id":
				round.Settlement.SettlementID = ""
			case "missing-time":
				round.Settlement.CompletedAt = 0
			case "wrong-game":
				round.Settlement.Reports[0].GameId++
			case "wrong-level":
				round.Settlement.Reports[0].Level++
			}
			rows, err := settlementReportBatch(round)
			if variant == "claimed" {
				if err != nil || len(rows) != 0 {
					t.Fatal("claimed result became final", rows, err)
				}
				return
			}
			if err == nil || rows != nil {
				t.Fatal("invalid financial proof accepted", rows, err)
			}
		})
	}
}

func TestLegacyAcceptedSmartReportsKeepTheirOriginalFormatOnHotUpdate(t *testing.T) {
	round := smartProofRound()
	round.Settlement.SmartTaskProofVersion = 0
	original := round.Settlement.Reports[0].Log
	rows, err := settlementReportBatch(round)
	if err != nil || rows[0].Log != original || rows[0].RowVersion != round.Settlement.Reports[0].RowVersion {
		t.Fatal("hot update changed previously accepted report payload", err)
	}
	// Ordinary and non-object historical logs retain their established format.
	for _, log := range []string{`{"actualResult":1}`, `"legacy"`, `not-json`} {
		round := smartProofRound()
		round.Settlement.Reports[0].Log = log
		rows, err := settlementReportBatch(round)
		if err != nil || rows[0].Log != log {
			t.Fatal(log, rows, err)
		}
	}
}
