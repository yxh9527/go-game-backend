package dao

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
)

func TestGogameFinanceCatalogueProjectsOnlyClientMappings(t *testing.T) {
	db, script := gogameTestDB(t, gogameSQLStep{contains: "FROM `gp_game`", columns: []string{"id", "number", "gameType", "clientGameType", "roomType", "confName", "state", "isFrozen"}, values: [][]driver.Value{
		{int64(1), int64(2006), int64(1), nil, nil, "test_legacy", int64(1), int64(0)},
		{int64(2), int64(91001), int64(3), int64(19), int64(19), "test_holdem", int64(1), int64(0)},
	}})
	result, err := (&DBDao{manager: db}).GogameCall(context.Background(), "listFinanceGames", "{}")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(result)
	if string(raw) != `[{"gameType":19,"roomType":19,"number":91001,"symbol":"test_holdem","state":1,"isFrozen":0}]` {
		t.Fatalf("projection=%s", raw)
	}
	if len(script.steps) != 0 {
		t.Fatal("catalogue query missing")
	}
}
func TestGogameFinanceCatalogueRejectsInvalidMappingAndBrowserNumber(t *testing.T) {
	db, _ := gogameTestDB(t, gogameSQLStep{contains: "FROM `gp_game`", columns: []string{"number", "clientGameType", "roomType", "confName", "state"}, values: [][]driver.Value{{int64(91001), int64(19), nil, "test_holdem", int64(1)}}})
	if _, err := (&DBDao{manager: db}).GogameCall(context.Background(), "listFinanceGames", "{}"); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("invalid route: %v", err)
	}
	if _, err := (&DBDao{}).GogameCall(context.Background(), "listFinanceGames", `{"number":19}`); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("untrusted number: %v", err)
	}
}
