package arena_test

import (
	"testing"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/game"
	"github.com/ognev-dev/goplease/game/ability"
)

const (
	BasID    = 1
	GritID   = 2
	FletchID = 3
	SilverID = 4
	MistID   = 5
	JulyID   = 6
)

func setupGame(t *testing.T) (*game.Arena, *game.Player, *game.Player) {
	t.Helper()
	p1 := game.NewPlayer(ds.NewID(), "Player 1", 0, game.StartingUnits(ds.NewID()))
	p2 := game.NewPlayer(ds.NewID(), "Player 2", 1, game.StartingUnits(ds.NewID()))
	ar := game.NewArena(p1, p2)
	ar.ActivePlayer = 0

	return ar, p1, p2
}

func placeUnit(t *testing.T, ar *game.Arena, playerID ds.ID, templateID int, atQ, atR int) *game.Unit {
	t.Helper()

	_, playerIdx := ar.PlayerByID(playerID)
	at := game.HexCoord{Q: atQ, R: atR}
	cell := ar.Board.Cells[at]
	cell.IsSafeZone = true
	cell.SafeZonePlayer = playerIdx
	ar.Board.Cells[at] = cell
	u, err := ar.PlaceUnitFromHandToBoard(templateID, at, playerID)
	if err != nil {
		t.Fatalf("placeUnit: %v", err)
	}

	u.OwnerID = playerID
	return u
}

func useAbility(t *testing.T, ar *game.Arena, playerID ds.ID, abID ability.ID) game.ApplyStates {
	t.Helper()

	states, err := ar.UseAbility(game.UseAbilityPayload{
		AbilityID: abID,
	}, playerID)
	if err != nil {
		t.Fatalf("useAbility %s: %v", abID, err)
	}

	return states
}

func useAbilityAt(t *testing.T, ar *game.Arena, playerID ds.ID, abID ability.ID, at game.HexCoord) game.ApplyStates {
	t.Helper()

	states, err := ar.UseAbility(game.UseAbilityPayload{
		AbilityID: abID,
		Target:    &at,
	}, playerID)
	if err != nil {
		t.Fatalf("useAbilityAt %s: %v", abID, err)
	}

	return states
}

type stateCases map[string]func(game.ApplyState) bool

// assertStateContains ensures that every predicate matches at least one ApplyState in the slice.
// If a predicate is not satisfied, the test fails with the name of the failed predicate.
func assertStateContains(t *testing.T, states []game.ApplyState, preds stateCases) {
	t.Helper()

iterate:
	for name, assertFn := range preds {
		for _, s := range states {
			if assertFn(s) {
				continue iterate
			}
		}

		t.Fatalf("assert state: predicate %q failed", name)
	}
}
