package dao

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"testing"
)

func TestImmutableFairCommitmentAndMonotonicDisclosure(t *testing.T) {
	original := map[string]any{"proofId": "proof", "agentId": 7, "gameRoomId": 11, "currentInning": 1, "pokerList": []any{map[string]any{"encryptedStr": "cipher", "secretKey": "key"}}, "viewers": []any{int64(1), int64(2)}, "revealed": []any{}, "revealedCards": []any{}, "finishedAt": int64(0), "boardCount": int64(0)}
	first, err := fairCommitment(original)
	if err != nil {
		t.Fatal(err)
	}
	changed := map[string]any{}
	for k, v := range original {
		changed[k] = v
	}
	changed["viewers"] = []any{int64(1), int64(2), int64(3)}
	changed["finishedAt"] = int64(100)
	changed["boardCount"] = int64(3)
	second, err := fairCommitment(changed)
	if err != nil || first != second {
		t.Fatal("public reveal changed commitment")
	}
	if err = validateProofUpdate(original, changed); err != nil {
		t.Fatal(err)
	}
	if err = validateProofUpdate(changed, original); err == nil {
		t.Fatal("disclosure rollback accepted")
	}
	changed["pokerList"] = []any{map[string]any{"encryptedStr": "different"}}
	third, _ := fairCommitment(changed)
	if third == first {
		t.Fatal("deck replacement retained commitment")
	}
	if !fairViewer(original, 1) || fairViewer(original, 3) {
		t.Fatal("viewer isolation failed")
	}
}

func TestSmartStoppedTaskCannotBeRestartedByNewerVersion(t *testing.T) {
	task := map[string]any{"taskId": "test", "agentId": 7, "userId": 42, "roomType": 58, "roomId": 11, "version": 2, "status": "stopped", "instanceId": "one", "areaId": 1, "betAmount": 100, "stopCountLimit": 10, "takeProfit": 0, "stopLoss": 0, "strategy": "FIXED", "currency": "CNY"}
	raw, _ := json.Marshal(task)
	db, script := gogameTestDB(t, gogameSQLStep{contains: "SELECT version,payload FROM gg_smart_task", columns: []string{"version", "payload"}, values: [][]driver.Value{{int64(2), string(raw)}}})
	task["version"] = 3
	task["status"] = "running"
	request, _ := json.Marshal(map[string]any{"task": task})
	_, err := (&DBDao{manager: db}).gogameSmartTask(context.Background(), "smartTaskSave", string(request))
	if err == nil || len(script.steps) != 0 || script.log[len(script.log)-1] != "rollback" {
		t.Fatal(err, script.log)
	}
}
