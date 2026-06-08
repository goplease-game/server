package game

import (
	"encoding/json"
	"fmt"
	"strconv"
)

const (
	SafeZoneSize = 2 // columns at each end that are "safe zones"
	BoardSize    = 3
)

type HexCoord struct {
	Q int `json:"q"`
	R int `json:"r"`
}

func (h HexCoord) Key() string {
	return strconv.Itoa(h.Q) + ":" + strconv.Itoa(h.R)
}

func (h HexCoord) Distance(to HexCoord) int {
	abs := func(x int) int {
		if x < 0 {
			return -x
		}
		return x
	}

	return (abs(h.Q-to.Q) + abs(h.Q+h.R-to.Q-to.R) + abs(h.R-to.R)) / 2
}

type BoardCell struct {
	Coord          HexCoord `json:"coord"`
	Unit           *Unit    `json:"unit,omitempty"`
	IsSafeZone     bool     `json:"is_safe_zone,omitzero"`
	SafeZonePlayer int      `json:"-"` // 0 or 1
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
	b := newHexBoard(BoardSize)
	return &b
}

func newHexBoard(size int) Board {
	b := Board{
		Cells: make(map[HexCoord]*BoardCell),
	}

	for q := -size; q <= size; q++ {
		for r := -size; r <= size; r++ {
			if s := -q - r; s < -size || s > size {
				continue
			}
			coord := HexCoord{Q: q, R: r}
			cell := &BoardCell{Coord: coord}

			if coord.Q <= -size+SafeZoneSize-1 {
				cell.IsSafeZone = true
				cell.SafeZonePlayer = 0
			} else if coord.Q >= size-SafeZoneSize+1 {
				cell.IsSafeZone = true
				cell.SafeZonePlayer = 1
			}

			b.Cells[coord] = cell
		}
	}

	return b
}

func (b *Board) CellAt(coord HexCoord) *BoardCell {
	return b.Cells[coord]
}

func (b *Board) UnitAt(coord HexCoord) *Unit {
	cell := b.CellAt(coord)
	if cell == nil {
		return nil
	}

	return cell.Unit
}

func (b *Board) PlaceUnit(coord HexCoord, u *Unit) bool {
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
