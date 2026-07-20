// Package unit ...
package unit

import (
	ab "github.com/goplease-game/game-server/ability"
)

// DefaultTemplates is the default roster of unit templates available in the game.
var DefaultTemplates = []Template{
	{
		ID:          1,
		Name:        "Bas",
		Description: "That guy in the hood who never lets you down.",
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
		Description: "Shows up to every fight already swinging and somehow that counts as a strategy.",
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
		Description: "Picks off the scary ones from three rooftops away.",
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
		Description: "Slips past the main entrance, then makes someone's day very short.",
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
		Description: "Folds time and space like laundry. Do not interfere.",
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
		Description: "Loves fixing leaks, cleans up messes, and keeps everyone from dying. Sometimes.",
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
