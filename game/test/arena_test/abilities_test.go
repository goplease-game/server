package arena_test

import (
	"testing"

	"github.com/ognev-dev/goplease/game"
	"github.com/ognev-dev/goplease/game/ability"
)

func TestBasicAttack(t *testing.T) {
	ar, p1, p2 := setupGame(t)

	u1 := placeUnit(t, ar, p1.ID, BasID, 0, 0)
	u2 := placeUnit(t, ar, p2.ID, JulyID, 0, 1)

	ar.ActiveUnitID = u1.ID
	ar.ActivePlayer = 0

	expectedHP := u2.CurrentHP - u1.CurrentAtk
	states := useAbilityAt(t, ar, p1.ID, ability.BasicMeleeAttack, u2.Pos)

	if u2.CurrentHP != expectedHP {
		t.Errorf("expected hp %d, got %d", expectedHP, u2.CurrentHP)
	}

	assertStateContains(t, states.Global, func(s game.ApplyState) bool {
		return s.ToUnitID == u2.ID && s.SetHP != nil && *s.SetHP == expectedHP
	})
}
