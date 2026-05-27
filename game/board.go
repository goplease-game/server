package game

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ognev-dev/goplease/game/unit"
)

const (
	BoardRows    = 10
	BoardColumns = 16
	SafeZoneSize = 2 // columns at each end that are "safe zones"
)

type HexCoord struct {
	Q int `json:"q"`
	R int `json:"r"`
}

func (h HexCoord) Key() string {
	return strconv.Itoa(h.Q) + ":" + strconv.Itoa(h.R)
}

type BoardCell struct {
	Coord      HexCoord   `json:"coord"`
	Unit       *unit.Unit `json:"unit,omitempty"`
	IsSafeZone bool       `json:"is_safe_zone,omitzero"`
}

type Board struct {
	Cells BoardCells `json:"cells"`
}

type BoardCells map[HexCoord]*BoardCell

func (b BoardCells) MarshalJSON() ([]byte, error) {
	type Alias BoardCell

	out := make(map[string]*BoardCell, len(b))
	for coord, cell := range b {
		if cell == nil {
			continue
		}

		key := fmt.Sprintf("%d:%d", coord.Q, coord.R)

		out[key] = cell
	}

	return json.Marshal(out)
}

func NewBoard() *Board {
	b := &Board{
		Cells: make(map[HexCoord]*BoardCell),
	}

	for r := 0; r < BoardRows; r++ {
		qOffset := r / 2
		for q := -qOffset; q < BoardColumns-qOffset; q++ {
			coord := HexCoord{
				Q: q,
				R: r,
			}

			b.Cells[coord] = &BoardCell{
				Coord: coord,
			}
		}
	}

	return b
}

func (b *Board) CellAt(coord HexCoord) *BoardCell {
	return b.Cells[coord]
}

func (b *Board) UnitAt(coord HexCoord) *unit.Unit {
	cell := b.CellAt(coord)
	if cell == nil {
		return nil
	}

	return cell.Unit
}

func (b *Board) PlaceUnit(coord HexCoord, u *unit.Unit) bool {
	cell := b.CellAt(coord)
	if cell == nil {
		return false
	}

	cell.Unit = u
	return true
}

func (b *Board) ClearUnit(coord HexCoord) {
	cell := b.CellAt(coord)
	if cell != nil {
		cell.Unit = nil
	}
}

func (b *Board) InBounds(row, col int) bool {
	return row >= 0 && row < BoardRows && col >= 0 && col < BoardColumns
}

// EnemySafeZone returns true if the cell belongs to the given playerIndex (0 or 1)
// enemy's safe zone
func EnemySafeZone(row int, ownerIndex int) bool {
	if ownerIndex == 0 {
		return row >= BoardRows-SafeZoneSize
	}
	return row < SafeZoneSize
}

// PlacementZone returns the valid placement rows for a player.
func PlacementZone(playerIndex int) (minRow, maxRow int) {
	if playerIndex == 0 {
		return 0, SafeZoneSize - 1
	}
	return BoardRows - SafeZoneSize, BoardRows - 1
}
