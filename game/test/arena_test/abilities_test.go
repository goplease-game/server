package arena_test

import (
	"testing"

	"github.com/ognev-dev/goplease/game"
	"github.com/ognev-dev/goplease/game/ability"
	"github.com/ognev-dev/goplease/game/ability/status"
)

func TestBasicAttack(t *testing.T) {
	ar, p1, p2 := setupGame(t)

	u1 := placeUnit(t, ar, p1.ID, BasID, 0, 0)
	u2 := placeUnit(t, ar, p2.ID, JulyID, 0, 1)

	ar.ActiveUnitID = u1.ID
	ar.ActivePlayer = 0

	expectedHP := u2.CurrentHP - u1.CurrentAtk
	states := useAbilityAt(t, ar, p1.ID, ability.BasicMeleeAttack, u2.PosVal())

	if u2.CurrentHP != expectedHP {
		t.Errorf("expected hp %d, got %d", expectedHP, u2.CurrentHP)
	}

	assertStateContains(t, states.Global, stateCases{
		"hp_reduced": func(s game.ApplyState) bool {
			return s.ToUnitID == u2.ID && s.SetHP != nil && *s.SetHP == expectedHP
		},
	})
}

func TestBonusAttackFromHunterMark(t *testing.T) {
	ar, p1, p2 := setupGame(t)

	u1 := placeUnit(t, ar, p1.ID, SilverID, 0, 0)
	u2 := placeUnit(t, ar, p2.ID, JulyID, 0, 3)

	ar.ActiveUnitID = u2.ID
	ar.ActivePlayer = 1

	u1.CurrentHP = 3
	game.ApplyStatusToUnit(ar, status.Marked, u2, u1)
	// +1 from Marked should kill u1 and must give +1 to u2 permanently
	u2.CurrentAtk = 2

	expectedDmg := 3
	expectedNewAtk := 3

	states := useAbilityAt(t, ar, p2.ID, ability.BasicMagicAttack, u1.PosVal())

	assertStateContains(t, states.Global, stateCases{
		"hp_changed": func(s game.ApplyState) bool {
			return s.ToUnitID == u1.ID && s.ChangeHP != nil && *s.ChangeHP == -expectedDmg
		},
		"hp_reduced_to_0": func(s game.ApplyState) bool {
			return s.ToUnitID == u1.ID && s.SetHP != nil && *s.SetHP == 0
		},
		"dead": func(s game.ApplyState) bool {
			return s.ToUnitID == u1.ID && s.IsDead
		},
		"attack_increased": func(s game.ApplyState) bool {
			return s.ToUnitID == u2.ID && s.SetAtk != nil && *s.SetAtk == expectedNewAtk
		},
	})

}
