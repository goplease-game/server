package game

import "github.com/ognev-dev/goplease/game/unit"

const MaxTurns = 20

type Phase string

const (
	PhaseUnitPlacement Phase = "unit_placement" // current player places units
	PhaseUnitActing    Phase = "unit_acting"    // current player playing with unit
	PhaseGameOver      Phase = "game_over"
)

type EndReason string

const (
	EndNoUnits   EndReason = "no_units"
	EndTurnLimit EndReason = "turn_limit"
)

type NewGamePayload struct {
	RoomID     string      `json:"room_id"`
	IsYourTurn bool        `json:"is_your_turn"`
	Board      Board       `json:"board"`
	Units      []unit.Unit `json:"units"`
	Opponent   *Player     `json:"opponent"`
}
