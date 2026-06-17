// Package unit ...
package unit

import (
	ab "github.com/goplease-game/server/ability"
)

/*
Unit Balance:
Base Profile (0 Points):
All units start with a baseline set of characteristics:
HP: 15
ATK: 2
Range: 1 (Melee)
MP: 3
In short 15/2/1/3

Balance Currency:
To modify the base profile, units must trade "Weight Points." The total balance must always equal 0.

5 HP = 1 Point
1 ATK = 1 Point
1 MP = 1 Point
Ranged Upgrade (+2 Range) = 1 Point (Fixed cost to switch from Melee to Ranged, increasing base Range from 1 to 3)

Constraints & Limits:
Min HP: 5
Min ATK: 1
Min MP: 1
Max MP: 4
Max Base Range: 3

Examples of balanced archetypes:
Tank: 40/2/1/3 ( +1 HP Point (30->40), -1 ATK Point (3->2))
Ranger: 20/4/3/2 (-1 HP Point (30->20), -1 MP Point (3->2), +1 ATK Point (3->4), +1 Ranged Upgrade (1->3))
Rogue: 20/3/1/4 (-1 HP Point (30->20), +1 MP Point (3->4))
Support: 30/2/3/3 (-1 ATK Point (3->2), +1 Ranged Upgrade (1->3))
*/

// TODO
// func (t Template) Validate() error {
//	if t.HP < 3 || t.Attack < 1 || t.MovePoints < 1 || t.MovePoints > 3 {
//		return errors.New("stats out of bounds")
//	}
//
//	if t.AttackRange != 1 && t.AttackRange != 3 {
//		return errors.New("attack range must be either 1 (Melee) or 3 (Ranged)")
//	}
//
//	if (t.HP-9)%3 != 0 {
//		return fmt.Errorf("HP must be modified in steps of 3 (current HP: %d)", t.HP)
//	}
//
//	score := (t.HP-9)/3 + (t.Attack - 3) + (t.MovePoints - 2)
//	if t.AttackRange == 3 {
//		score += 1
//	}
//
//	if score != 0 {
//		return fmt.Errorf("unit is not balanced: score is %d, must be 0", score)
//	}
//
//	return nil
// }

// DefaultTemplates is the default roster of unit templates available in the game.
var DefaultTemplates = []Template{
	{
		ID:          1,
		Name:        "Bas",
		Description: "An immovable wall who protects allies by absorbing damage, locking down enemies, and holding the front line at all costs.",
		HP:          20, Attack: 1, AttackRange: 1, MovePoints: 3,
		ActionPoints: 1,
		Abilities: []ab.ID{
			ab.BasicMeleeAttack,
			ab.Fortify,
			ab.Provoke,
			ab.ShieldBash,
			ab.UndyingWill,
		},
	},
	{
		ID:          2,
		Name:        "Grit",
		Description: "A fierce frontline brawler who thrives in the thick of battle, dealing heavy area damage and breaking enemy formations.",
		HP:          10, Attack: 3, AttackRange: 1, MovePoints: 3,
		ActionPoints: 1,
		Abilities: []ab.ID{
			ab.BasicMeleeAttack,
			ab.BattleCry,
			ab.IdolihuSpin,
			ab.PowerPush,
			ab.Frenzy,
		},
	},
	{
		ID:          3,
		Name:        "Fletch",
		Description: "A long-range damage dealer specializing in picking off high-priority targets and providing suppressing cover fire from safety.",
		HP:          10, Attack: 3, AttackRange: 3, MovePoints: 2,
		ActionPoints: 1,
		Abilities: []ab.ID{
			ab.BasicRangeAttack,
			ab.PiercingShot,
			ab.HuntersMark,
			ab.HamstringShot,
			ab.CoverFire,
		},
	},
	{
		ID:          4,
		Name:        "Silver",
		Description: "A highly mobile assassin designed to slip behind enemy lines and eliminate vulnerable targets with devastating backstabs.",
		HP:          10, Attack: 2, AttackRange: 1, MovePoints: 4,
		ActionPoints: 1,
		Abilities: []ab.ID{
			ab.BasicMeleeAttack,
			ab.ShadowStep,
			ab.GangUp,
			ab.Eliminate,
			ab.Opportunity,
		},
	},
	{
		ID:          5,
		Name:        "Mist",
		Description: "A tactical spellcaster who manipulates time and space to control the battlefield, weaken enemies, and reposition units.",
		HP:          10, Attack: 2, AttackRange: 3, MovePoints: 3,
		ActionPoints: 1,
		Abilities: []ab.ID{
			ab.BasicMagicAttack,
			ab.Translocation,
			ab.TimeWarp,
			ab.Purge,
			ab.FocusField,
		},
	},
	{
		ID:          6,
		Name:        "July",
		Description: "Loves fixing leaks, cleans up messes, and keeps everyone from dying, sometimes.",
		HP:          15, Attack: 1, AttackRange: 3, MovePoints: 3,
		ActionPoints: 1,
		Abilities: []ab.ID{
			ab.BasicMagicAttack,
			ab.Heal,
			ab.Equalize,
			ab.Purify,
			ab.BottomlessVial,
		},
	},
}
