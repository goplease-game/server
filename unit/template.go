package unit

import (
	"github.com/goplease-game/server/ability"
)

// Template defines the static base stats and ability set for a unit type,
// used to construct individual unit instances when placed on the board.
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
