package game

import (
	"github.com/goplease-game/server/ability"
	"github.com/goplease-game/server/ds"
)

// NewGamePayload is sent to the client when a new match starts, containing
// the initial board, player, and match configuration.
type NewGamePayload struct {
	ArenaID                    ds.ID      `json:"arena_id"`
	Phase                      RoundPhase `json:"phase"`
	Board                      *Board     `json:"board"`
	Player                     *Player    `json:"player"`
	Opponent                   string     `json:"opponent"`
	TurnTimeSeconds            int        `json:"turn_time_seconds"`
	MaxPhantomAPPerUnitPerTurn int        `json:"max_phantom_ap_per_unit_per_turn"`
}

// PlaceUnitPayload is the payload for placing a unit on the board at a specific coordinate.
type PlaceUnitPayload struct {
	Coord HexCoord `json:"coord"`
	Unit  *Unit    `json:"unit"`
}

// UnitPlacedPayload is the payload broadcast when a unit is placed on the board.
type UnitPlacedPayload struct {
	Coord      HexCoord `json:"coord"`
	TemplateID int      `json:"template_id"`
}

// UnitMovedPayload is the payload broadcast when a unit moves to a new coordinate.
type UnitMovedPayload struct {
	Coord  HexCoord `json:"coord"`
	UnitID ds.ID    `json:"unit_id"`
}

// PlayUnitPayload is the payload for playing a unit from hand onto the board.
type PlayUnitPayload struct {
	UnitID ds.ID `json:"unit_id"`
}

// UseAbilityPayload is the payload for requesting that a unit use an ability,
// optionally targeting a specific hex coordinate.
type UseAbilityPayload struct {
	UnitID    ds.ID      `json:"unit_id"`
	AbilityID ability.ID `json:"ability_id"`
	Target    *HexCoord  `json:"target,omitempty"`
}

// ActiveUnitChangedPayload is the payload broadcast when the active unit changes.
type ActiveUnitChangedPayload struct {
	UnitID ds.ID `json:"unit_id"`
}
