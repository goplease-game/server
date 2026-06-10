package game

import "github.com/ognev-dev/goplease/app/ds"

const (
	MaxTurns        = 20
	TurnTimeSeconds = 999
)

type RoundPhase int

const (
	PlayPhase RoundPhase = iota
	PlacementPhase
)

type EndReason string

const (
	EndNoUnits   EndReason = "no_units"
	EndTurnLimit EndReason = "turn_limit"
)

type NewGamePayload struct {
	ArenaID         ds.ID      `json:"arena_id"`
	Phase           RoundPhase `json:"phase"`
	IsMyTurn        bool       `json:"is_my_turn"`
	Board           *Board     `json:"board"`
	Player          *Player    `json:"player"`
	Opponent        string     `json:"opponent"`
	TurnTimeSeconds int        `json:"turn_time_seconds"`
}
