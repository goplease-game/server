package game

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/game/ability"
)

const (
	SafeZoneSize = 2 // columns at each end that are "safe zones"
	BoardSize    = 4
)

type HexCoord struct {
	Q int `json:"q"`
	R int `json:"r"`
}

func (h HexCoord) Key() string {
	return strconv.Itoa(h.Q) + ":" + strconv.Itoa(h.R)
}

func (h HexCoord) String() string {
	return h.Key()
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

// Opposite returns the hex coordinate directly opposite to the origin relative to the center.
func (h HexCoord) Opposite(center HexCoord) HexCoord {
	return HexCoord{
		Q: 2*center.Q - h.Q,
		R: 2*center.R - h.R,
	}
}

// Neighbors returns the 6 adjacent hex coordinates around from.
// Does not filter by board boundaries or occupancy.
func (h HexCoord) Neighbors() []HexCoord {
	dirs := []HexCoord{
		{Q: 1, R: 0}, {Q: -1, R: 0},
		{Q: 0, R: 1}, {Q: 0, R: -1},
		{Q: 1, R: -1}, {Q: -1, R: 1},
	}

	coords := make([]HexCoord, 0, 6)
	for _, d := range dirs {
		coords = append(coords, HexCoord{Q: h.Q + d.Q, R: h.R + d.R})
	}

	return coords
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

func (b *Board) CellExists(at HexCoord) bool {
	_, ok := b.Cells[at]
	return ok
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

func (b *BoardCells) UnmarshalJSON(data []byte) error {
	tmp := make(map[string]*BoardCell)
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if *b == nil {
		*b = make(BoardCells, len(tmp))
	}

	for k, cell := range tmp {
		parts := strings.Split(k, ":")
		q, _ := strconv.Atoi(parts[0])
		r, _ := strconv.Atoi(parts[1])
		coord := HexCoord{Q: q, R: r}

		(*b)[coord] = cell
	}

	return nil
}

func (b BoardCells) InRange(from HexCoord, rangeN int) []*BoardCell {
	var result []*BoardCell
	for to, cell := range b {
		if from.Distance(to) <= rangeN {
			result = append(result, cell)
		}
	}

	return result
}

func (b BoardCells) InRangeHavingUnit(from HexCoord, rangeN int) []*BoardCell {
	var result []*BoardCell
	for to, cell := range b {
		if cell.Unit == nil {
			continue
		}
		if from.Distance(to) <= rangeN {
			result = append(result, cell)
		}
	}

	return result
}

func (b BoardCells) IsUnitInRange(from HexCoord, rangeN int, unitID ds.ID) bool {
	for to, cell := range b {
		if cell.Unit == nil {
			continue
		}
		if from.Distance(to) <= rangeN {
			if cell.Unit.ID == unitID {
				return true
			}
		}
	}

	return false
}

func (b BoardCells) InRangeHavingUnitAbility(from HexCoord, rangeN int, abID ability.ID) []*BoardCell {
	var result []*BoardCell
	for to, cell := range b {
		if cell.Unit == nil {
			continue
		}
		if !cell.Unit.HasAbility(abID) {
			continue
		}
		if from.Distance(to) <= rangeN {
			result = append(result, cell)
		}
	}

	return result
}

// Line returns cells along a ray from [from] strictly in the direction of targetPos,
// up to radius steps. If targetPos does not lie on any of the 6 hex axes from [from],
// it returns an empty slice.
func (b BoardCells) Line(from, to HexCoord, size int) []*BoardCell {
	if from == to {
		return []*BoardCell{}
	}

	dq := to.Q - from.Q
	dr := to.R - from.R

	// A valid hex axis requires dr==0, dq==0, or dq==-dr.
	// In all valid cases the unit direction is just sign(dq), sign(dr).
	if dr != 0 && dq != 0 && dq != -dr {
		return []*BoardCell{}
	}

	sign := func(x int) int {
		if x > 0 {
			return 1
		}
		if x < 0 {
			return -1
		}
		return 0
	}

	dir := HexCoord{Q: sign(dq), R: sign(dr)}

	var result []*BoardCell
	cur := from
	for range size {
		cur = HexCoord{Q: cur.Q + dir.Q, R: cur.R + dir.R}
		cell, ok := b[cur]
		if !ok {
			break
		}
		result = append(result, cell)
	}

	return result
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

// lineAreaCells returns all board cells in all 6 directions from `from` up to `length` steps.
// Used for AreaLine abilities like PiercingShot.
func lineAreaCells(cells BoardCells, from HexCoord, radius int) []*BoardCell {
	dirs := []HexCoord{
		{Q: 1, R: 0}, {Q: -1, R: 0},
		{Q: 0, R: 1}, {Q: 0, R: -1},
		{Q: 1, R: -1}, {Q: -1, R: 1},
	}

	var result []*BoardCell
	for _, dir := range dirs {
		cur := from
		for i := 0; i < radius; i++ {
			cur = HexCoord{Q: cur.Q + dir.Q, R: cur.R + dir.R}
			cell, ok := cells[cur]
			if !ok {
				break
			}
			result = append(result, cell)
		}
	}
	return result
}

func ReachableCells(from HexCoord, movePoints int, board Board) []HexCoord {
	type node struct {
		pos  HexCoord
		cost int
	}

	visited := make(map[HexCoord]int)
	visited[from] = 0

	queue := []node{{from, 0}}
	result := make([]HexCoord, 0)

	dirs := []HexCoord{
		{+1, 0}, {+1, -1}, {0, -1},
		{-1, 0}, {-1, +1}, {0, +1},
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		for _, d := range dirs {
			next := HexCoord{
				Q: cur.pos.Q + d.Q,
				R: cur.pos.R + d.R,
			}

			cell, ok := board.Cells[next]
			if !ok {
				continue
			}

			if cell.Unit != nil && next != from {
				continue
			}

			newCost := cur.cost + 1
			if newCost > movePoints {
				continue
			}

			prev, seen := visited[next]
			if seen && prev <= newCost {
				continue
			}

			visited[next] = newCost
			queue = append(queue, node{next, newCost})

			if !seen {
				result = append(result, next)
			}
		}
	}

	return result
}
