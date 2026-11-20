package unified

import (
	"app/entity"
	"encoding/json"
	"fmt"
)

// The financial service owns this marker. Never forward a caller-provided
// marker as evidence that a zero-net game was normally settled, not refunded.
func attachSmartTaskSettlementProof(round *financeRound, row *entity.CacheRecordsReq) error {
	if round.Settlement == nil || round.Settlement.SmartTaskProofVersion != 1 {
		return nil // Keep already accepted legacy document versions unchanged.
	}
	var envelope map[string]json.RawMessage
	if json.Unmarshal([]byte(row.Log), &envelope) != nil || envelope == nil {
		return nil // Preserve legacy non-object logs verbatim.
	}
	_, hadProof := envelope["smartTaskSettlement"]
	delete(envelope, "smartTaskSettlement")
	var display struct {
		SmartTask struct {
			TaskID    string `json:"taskId"`
			AttemptID string `json:"attemptId"`
			Inning    int64  `json:"inning"`
		} `json:"smartTask"`
	}
	_ = json.Unmarshal(envelope["completedDisplayReport"], &display)
	ref := display.SmartTask
	settlement := round.Settlement
	eligible := round.State == string(financeStateSettled) && settlement != nil && settlement.Status == financeSettlementStatusCompleted && settlement.Mode == financeModeNormal &&
		ref.TaskID != "" && ref.AttemptID != "" && ref.Inning > 0
	if eligible {
		if settlement.SettlementID == "" || settlement.CompletedAt <= 0 || row.RoundID != round.RoundID || row.AgentId != round.Agent || row.GameId != round.GameID || row.Level != round.Level || !row.Complete {
			return fmt.Errorf("invalid completed smart task financial identity")
		}
		proof, err := json.Marshal(struct {
			Version      int    `json:"version"`
			State        string `json:"state"`
			Mode         string `json:"mode"`
			RoundID      string `json:"roundId"`
			SettlementID string `json:"settlementId"`
			CompletedAt  int64  `json:"completedAt"`
			TaskID       string `json:"taskId"`
			AttemptID    string `json:"attemptId"`
			Inning       int64  `json:"inning"`
		}{1, round.State, settlement.Mode, round.RoundID, settlement.SettlementID, settlement.CompletedAt, ref.TaskID, ref.AttemptID, ref.Inning})
		if err != nil {
			return err
		}
		envelope["smartTaskSettlement"] = proof
	}
	if !eligible && !hadProof {
		return nil
	}
	raw, err := json.Marshal(envelope)
	if err == nil {
		row.Log = string(raw)
	}
	return err
}
