package game

import "github.com/ognev-dev/goplease/game/unit"

const (
	BoardCols    = 12 // X axis (columns)
	BoardRows    = 12 // Y axis (rows), split equally between players
	SafeZoneRows = 2  // rows at each end that are "safe zones"
)

// Board is a 2D grid. board[col][row] = *Unit or nil.
type Board [BoardCols][BoardRows]*unit.Unit

func (b *Board) At(col, row int) *unit.Unit     { return b[col][row] }
func (b *Board) Set(col, row int, u *unit.Unit) { b[col][row] = u }
func (b *Board) Clear(col, row int)             { b[col][row] = nil }
func (b *Board) InBounds(col, row int) bool {
	return col >= 0 && col < BoardCols && row >= 0 && row < BoardRows
}

// EnemySafeZone returns true if the cell belongs to the given playerIndex (0 or 1)
// enemy's safe zone
func EnemySafeZone(row int, ownerIndex int) bool {
	if ownerIndex == 0 {
		return row >= BoardRows-SafeZoneRows
	}
	return row < SafeZoneRows
}

// PlacementZone returns the valid placement rows for a player.
func PlacementZone(playerIndex int) (minRow, maxRow int) {
	if playerIndex == 0 {
		return 0, SafeZoneRows - 1
	}
	return BoardRows - SafeZoneRows, BoardRows - 1
}
