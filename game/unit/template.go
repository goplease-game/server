package unit

import (
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
