package tables

import (
	"testing"

	"app/tables/gogame"
	"app/tables/manager"
)

func TestGameServerSeedsProjectToUniqueRoutes(t *testing.T) {
	seeds := GameServerSeeds()
	if len(seeds) != 24 {
		t.Fatalf("got %d game-server room seeds, want 24", len(seeds))
	}

	games := make([]*manager.Game, 0, len(seeds))
	for _, seed := range seeds {
		clientGameType, roomType := seed.ClientGameType, seed.RoomType
		games = append(games, &manager.Game{
			Number: seed.Number, ConfName: seed.Symbol, State: 1, IsFrozen: 0,
			ClientGameType: &clientGameType, RoomType: &roomType,
		})
	}

	routes, err := gogame.ProjectGameRoutes(games)
	if err != nil {
		t.Fatalf("server seeds do not form a valid route catalogue: %v", err)
	}
	if len(routes) != len(seeds) {
		t.Fatalf("projected %d routes from %d seeds", len(routes), len(seeds))
	}
}
