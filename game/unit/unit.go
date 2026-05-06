package unit

import (
	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/game/ability"
)

type Template struct {
	ID           int
	Name         string
	Description  string
	HP           int
	Attack       int
	AttackRange  int
	Shield       int
	MovePoints   int
	ActionPoints int
	Abilities    []ability.ID
}

type Unit struct {
	ID         ds.ID
	TemplateID int
	OwnerID    ds.ID
	Name       string

	MaxHP         int
	CurrentHP     int
	CurrentShield int

	AP int // Action Points
	MP int // Move Points

	// board position, -1 - in hand
	PosRow int
	PosCol int

	Abilities []ability.ID
	Cooldowns map[ability.ID]int
}

func NewUnitFromTemplate(t Template, playerID ds.ID) *Unit {
	return &Unit{
		ID:            ds.NewID(),
		TemplateID:    t.ID,
		OwnerID:       playerID,
		Name:          t.Name,
		MaxHP:         t.HP,
		CurrentHP:     t.HP,
		CurrentShield: t.Shield,
		AP:            t.ActionPoints,
		MP:            t.MovePoints,
		PosRow:        -1,
		PosCol:        -1,
		Abilities:     t.Abilities,
		Cooldowns:     make(map[ability.ID]int),
	}
}

func (u *Unit) IsAlive() bool { return u.CurrentHP > 0 }
func (u *Unit) InHand() bool  { return u.PosRow == -1 }
