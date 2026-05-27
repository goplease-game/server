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
MP: 3
In short 9/3/1/3

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
Max MP: 4

Examples of balanced archetypes:
Tank : 12/2/1/3 (+1 HP Point, -1 ATK Point)
Ranger: 6/4/3/2 (-1 HP Point, -1 MP Point, +1 ATK Point, +1 Ranged Point)
Rogue: 6/3/1/4 (-1 HP Point, +1 MP Point)
Support: 9/2/3/3 (-1 ATK Point, +1 Ranged Point)
*/

// TODO
//func (t Template) Validate() error {
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
//}

var DefaultTemplates = []Template{
	{
		ID:          1,
		Name:        "Bas",
		Description: "An immovable wall who protects allies by absorbing damage, locking down enemies, and holding the front line at all costs.",
		HP:          12, Attack: 2, AttackRange: 1, MovePoints: 3,
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
		HP:          6, Attack: 4, AttackRange: 1, MovePoints: 3,
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
		HP:          6, Attack: 4, AttackRange: 3, MovePoints: 2,
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
		HP:          6, Attack: 3, AttackRange: 1, MovePoints: 4,
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
		HP:          9, Attack: 3, AttackRange: 3, MovePoints: 2,
		ActionPoints: 1,
		Abilities: []ab.ID{
			ab.BasicMagicAttack,
			ab.Translocation,
			ab.TimeWarp,
			ab.Purge,
			ab.ArcaneChaos,
		},
	},
	{
		ID:          6,
		Name:        "July",
		Description: "A support and healer focused on keeping the team alive, removing negative effects, and sustaining allies through long battles.",
		HP:          9, Attack: 2, AttackRange: 3, MovePoints: 3,
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

func StartingUnits(playerID ds.ID) []*Unit {
	units := make([]*Unit, len(DefaultTemplates))

	for i, t := range DefaultTemplates {
		units[i] = NewUnitFromTemplate(t, playerID)
	}

	return units
}
