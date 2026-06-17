package game

import (
	"github.com/goplease-game/server/ds"
)

// Player represents a participant in a match, tracking their hand of
// units, placement progress, and the Phantom AP carried over from
// unequal unit counts.
type Player struct {
	ID          ds.ID   `json:"id"`
	Name        string  `json:"name"`
	PlayerIndex int     `json:"-"`     // 0 or 1
	Units       []*Unit `json:"units"` // units at hand

	PhantomAP            int `json:"phantom_ap"`
	UnitsPlacedThisRound int `json:"-"`

	Ready bool `json:"-"`
}

// NewPlayer creates a new Player with the given id, name, seat index, and
// starting hand of units.
func NewPlayer(id ds.ID, name string, index int, units []*Unit) *Player {
	p := &Player{
		ID:          id,
		Name:        name,
		PlayerIndex: index,
		Units:       units,
	}

	return p
}

// HasUnits reports whether the player still has units, either in hand or
// placed on the board.
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

// PopUnitFromHand removes and returns the first unit in the player's hand
// matching templateID, or nil if none is found.
func (p *Player) PopUnitFromHand(templateID int) *Unit {
	for i, u := range p.Units {
		if u.TemplateID == templateID {
			p.Units = append(p.Units[:i], p.Units[i+1:]...)
			return u
		}
	}

	return nil
}

// HasUnitsInHand reports whether the player has any units left in hand.
func (p *Player) HasUnitsInHand() bool {
	return len(p.Units) > 0
}
