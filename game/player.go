package game

import (
	"github.com/ognev-dev/goplease/app/ds"
)

type Player struct {
	ID          ds.ID   `json:"id"`
	Name        string  `json:"name"`
	PlayerIndex int     `json:"-"`     // 0 or 1
	Units       []*Unit `json:"units"` // units at hand

	PhantomAP            int `json:"phantom_ap"`
	UnitsPlacedThisRound int `json:"-"`

	Ready bool `json:"-"`
}

func NewPlayer(id ds.ID, name string, index int, units []*Unit) *Player {
	p := &Player{
		ID:          id,
		Name:        name,
		PlayerIndex: index,
		Units:       units,
	}

	return p
}

func (p *Player) HasUnits(board *Board) bool {
	if len(p.Units) > 0 {
		return true
	}

	for _, cell := range board.Cells {
		if cell != nil && cell.Unit != nil && cell.Unit.OwnerID == p.ID {
			return true
		}
	}

	return false
}

func (p *Player) PopUnitFromHand(templateID int) *Unit {
	for i, u := range p.Units {
		if u.TemplateID == templateID {
			p.Units = append(p.Units[:i], p.Units[i+1:]...)
			return u
		}
	}

	return nil
}

func (p *Player) HasUnitsInHand() bool {
	return len(p.Units) > 0
}
