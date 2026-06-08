package game

import (
	"github.com/ognev-dev/goplease/app/ds"
)

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
