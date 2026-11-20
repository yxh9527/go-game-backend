package gogame

import (
	"app/tables/manager"
	"fmt"
	"testing"
)

func routeUint(value uint32) *uint32 { return &value }
func routeGame(number int, gameType, roomType uint32) *manager.Game {
	return &manager.Game{Number: number, GameType: 3, ClientGameType: routeUint(gameType), RoomType: routeUint(roomType), ConfName: fmt.Sprintf("test_route_%d", number), State: 1}
}
func TestProjectGameRoutesPreservesLegacyClassification(t *testing.T) {
	legacy := &manager.Game{Number: 2006, GameType: 1, ConfName: "test_legacy", State: 1}
	mapped := routeGame(91001, 19, 19)
	routes, err := ProjectGameRoutes([]*manager.Game{legacy, mapped, routeGame(91002, 1004, 61), routeGame(91003, 1004, 63)})
	if err != nil || len(routes) != 3 {
		t.Fatalf("routes=%v error=%v", routes, err)
	}
	if routes[0].GameType != 19 || routes[0].Number != 91001 || mapped.GameType != 3 || legacy.ClientGameType != nil {
		t.Fatal("legacy catalogue identity changed")
	}
	mapped.ClientGameType = routeUint(20)
	if routes[0].GameType != 19 {
		t.Fatal("projection aliases mutable catalogue")
	}
}
func TestProjectGameRoutesRejectsAmbiguousAndInvalidIdentities(t *testing.T) {
	for _, games := range [][]*manager.Game{
		{routeGame(91001, 19, 19), routeGame(91002, 19, 19)},
		{routeGame(91001, 19, 19), {Number: 91001, GameType: 1}},
		{routeGame(0, 19, 19)}, {routeGame(91001, 0, 19)},
		{{Number: 91001, ClientGameType: routeUint(19)}},
	} {
		if _, err := ProjectGameRoutes(games); err == nil {
			t.Fatalf("invalid routes accepted: %+v", games)
		}
	}
}
