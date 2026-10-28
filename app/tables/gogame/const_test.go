package gogame

import "testing"

func TestCanTransitMatch(t *testing.T) {
	ok := [][2]int32{
		{MatchStateDraft, MatchStateRegistering},
		{MatchStateDraft, MatchStateCancelled},
		{MatchStateRegistering, MatchStateRunning},
		{MatchStateRegistering, MatchStateCancelled},
		{MatchStateRunning, MatchStateFinished},
		{MatchStateRunning, MatchStateCancelled},
	}
	for _, c := range ok {
		if !CanTransitMatch(c[0], c[1]) {
			t.Fatalf("expected %d -> %d", c[0], c[1])
		}
	}
	bad := [][2]int32{
		{MatchStateDraft, MatchStateRunning},
		{MatchStateRegistering, MatchStateDraft},
		{MatchStateFinished, MatchStateRunning},
		{MatchStateCancelled, MatchStateRegistering},
	}
	for _, c := range bad {
		if CanTransitMatch(c[0], c[1]) {
			t.Fatalf("unexpected %d -> %d", c[0], c[1])
		}
	}
}
