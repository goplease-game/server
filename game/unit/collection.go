package unit

import (
	"github.com/ognev-dev/goplease/app/ds"
	ab "github.com/ognev-dev/goplease/game/ability"
)

/*
Unit Balance:
Base Profile (0 Points):
All units start with a baseline set of characteristics:
HP: 9
ATK: 3
Range: 1 (Melee)
MP: 2
In short 9/3/1/2

Balance Currency:
To modify the base profile, units must trade "Weight Points." The total balance must always equal 0.

3 HP = 1 Point
1 ATK = 1 Point
1 MP = 1 Point
Ranged (max range of 3) = 1 Point (Fixed cost for switching from Melee to Ranged)

Constraints & Limits:
Min HP: 3
Min ATK: 1
Min MP: 1
Max MP: 3

Examples of balanced archetypes:
Tank : 12/2/1/2 (+1 HP Point, -1 ATK Point)
Ranger: 6/4/3/1 (-1 HP Point, -1 MP Point, +1 ATK Point, +1 Ranged Point)
Rogue: 6/3/1/3 (-1 HP Point, +1 MP Point)
Support: 9/2/3/2 (-1 ATK Point, +1 Ranged Point)
*/

// TODO
//func (t Template) Validate() error {
//	if t.HP < 3 || t.MovePoints < 1 || t.MovePoints > 3 || t.Attack < 1 {
//		return errors.New("stats out of bounds")
//	}
//
//	score := (t.HP-9)/3 + (t.Attack-3) + (t.MovePoints-2)
//	if t.AttackRange > 1 {
//		score += 1
//	}
//
//	if score != 0 {
//		return fmt.Errorf("unit is not balanced: score is %d, must be 0", score)
//	}
//
//	return nil
//}

var DefaultTemplates = []Template{
	{
		ID:          1,
		Name:        "Bas",
		Description: "A immovable wall who refuses to be the first to step back.",
		HP:          12, Attack: 2, AttackRange: 1, MovePoints: 2,
		ActionPoints: 1,
		Abilities: []ab.ID{
			ab.BasicMeleeAttack,
			ab.ShieldWall,
			ab.Provoke,
			ab.ShieldBash,
			ab.LastStand,
		},
	},
	{
		ID:          2,
		Name:        "Grit",
		Description: "Always finds the thickest of the fight and leaves with new scars.",
		HP:          6, Attack: 4, AttackRange: 1, MovePoints: 2,
		ActionPoints: 1,
		Abilities: []ab.ID{
			ab.BasicMeleeAttack,
			ab.BattleCry,
			ab.Cleave,
			ab.Slam,
			ab.Frenzy,
		},
	},
	{
		ID:          3,
		Name:        "Fletch",
		Description: "Finds peace in the whistle of an arrow and the silence after the shot.",
		HP:          6, Attack: 4, AttackRange: 3, MovePoints: 1,
		ActionPoints: 1,
		Abilities: []ab.ID{
			ab.BasicRangeAttack,
			ab.PiercingShot,
			ab.Prey,
			ab.Disengage,
			ab.CoverFire,
		},
	},
	{
		ID:          4,
		Name:        "Silver",
		Description: "Values the glint of a coin and the edge of a blade above all else.",
		HP:          6, Attack: 3, AttackRange: 1, MovePoints: 3,
		ActionPoints: 1,
		Abilities: []ab.ID{
			ab.BasicMeleeAttack,
			ab.ShadowStep,
			ab.Backstab,
			ab.Eliminate,
			ab.Opportunity,
		},
	},
	{
		ID:          5,
		Name:        "Mist",
		Description: "Speaks in riddles and acts from the safety of the haze.",
		HP:          9, Attack: 3, AttackRange: 3, MovePoints: 1,
		ActionPoints: 1,
		Abilities: []ab.ID{
			ab.BasicMagicAttack,
			ab.Translocation,
			ab.TimeWarp,
			ab.Enfeeble,
			ab.Meditation,
		},
	},
	{
		ID:          6,
		Name:        "July",
		Description: "Believes that any wound can close, as long as she is there in time.",
		HP:          9, Attack: 2, AttackRange: 3, MovePoints: 2,
		ActionPoints: 1,
		Abilities: []ab.ID{
			ab.BasicMagicAttack,
			ab.Heal,
			ab.BalanceLife,
			ab.Cleanse,
			ab.Renewal,
		},
	},
}

func StartingUnits(playerID ds.ID) []*Unit {
	units := make([]*Unit, len(DefaultTemplates))

	for i, t := range DefaultTemplates {
		units[i] = NewUnitFromTemplate(t, playerID)
	}

	return units
}
