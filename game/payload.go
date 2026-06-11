package game

import (
	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/game/ability"
)

type NewGamePayload struct {
	ArenaID                    ds.ID      `json:"arena_id"`
	Phase                      RoundPhase `json:"phase"`
	Board                      *Board     `json:"board"`
	Player                     *Player    `json:"player"`
	Opponent                   string     `json:"opponent"`
	TurnTimeSeconds            int        `json:"turn_time_seconds"`
	MaxPhantomAPPerUnitPerTurn int        `json:"max_phantom_ap_per_unit_per_turn"`
}

type PlaceUnitPayload struct {
	Coord HexCoord `json:"coord"`
	Unit  *Unit    `json:"unit"`
}

type UnitPlacedPayload struct {
	Coord      HexCoord `json:"coord"`
	TemplateID int      `json:"template_id"`
}

type UnitMovedPayload struct {
	Coord  HexCoord `json:"coord"`
	UnitID ds.ID    `json:"unit_id"`
}

type PlayUnitPayload struct {
	UnitID ds.ID `json:"unit_id"`
}

type UseAbilityPayload struct {
	UnitID    ds.ID      `json:"unit_id"`
	AbilityID ability.ID `json:"ability_id"`
	Target    *HexCoord  `json:"target,omitempty"`
}
