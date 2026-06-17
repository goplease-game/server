package bot

import (
	"errors"
	"log"
	"math"
	"math/rand/v2"

	game "github.com/goplease-game/server"
	"github.com/goplease-game/server/ability/status"
	"github.com/goplease-game/server/ds"
)

var (
	// ErrNoEmptySafeZoneCells indicates that there are no available empty cells in the safe zone.
	ErrNoEmptySafeZoneCells = errors.New("no empty cells in safe zone")
)

type gameState struct {
	board  *game.Board
	player *game.Player
	queue  []*game.Unit
}

func (b *Bot) addUnitToQueue(u *game.Unit) {
	b.state.queue = append(b.state.queue, u)
}

func (b *Bot) pickRandomUnitFromHand() *game.Unit {
	units := b.state.player.Units
	count := len(units)
	if count == 0 {
		return nil
	}

	idx := rand.IntN(count) //nolint:gosec
	pickedUnit := units[idx]

	units[idx] = units[count-1]
	units = units[:count-1]

	b.state.player.Units = units

	return pickedUnit
}

func (b *Bot) placeUnitAt(u *game.Unit, at game.HexCoord) {
	if u.Pos != nil {
		oldCell := b.state.board.Cells[*u.Pos]
		if oldCell != nil {
			oldCell.Unit = nil
		}
	}

	u.Pos = &at
	cell := b.state.board.Cells[at]
	if cell == nil {
		log.Printf("[bot] placeUnitAt: cell not found at %s", at)
		return
	}
	cell.Unit = u
}

func (b *Bot) unitAt(at game.HexCoord) *game.Unit {
	c := b.cellAt(at)
	if c == nil {
		return nil
	}

	return c.Unit
}

func (b *Bot) randomUnoccupiedSafeZonePos() (pos game.HexCoord, err error) {
	var empty []game.HexCoord
	for coord, cell := range b.state.board.Cells {
		if !cell.IsSafeZone {
			continue
		}
		if cell.Unit != nil {
			continue
		}
		empty = append(empty, coord)
	}

	if len(empty) == 0 {
		err = ErrNoEmptySafeZoneCells
		return
	}

	return empty[rand.IntN(len(empty))], nil //nolint:gosec
}

func (b *Bot) unitByID(unitID ds.ID) *game.Unit {
	for _, u := range b.state.queue {
		if u.ID == unitID {
			return u
		}
	}

	return nil
}

func (b *Bot) moveUnit(unitID ds.ID, to game.HexCoord) {
	u := b.unitByID(unitID)
	if u == nil {
		log.Printf("[bot] moveUnit: unit %s not found", unitID)
		return
	}

	b.placeUnitAt(u, to)
}

func (b *Bot) killUnit(unitID ds.ID) {
	u := b.unitByID(unitID)
	if u == nil {
		log.Printf("[bot] killUnit: unit %s not found", unitID)
		return
	}

	u.IsDead = true

	for i, qu := range b.state.queue {
		if qu.ID == unitID {
			b.state.queue = append(b.state.queue[:i], b.state.queue[i+1:]...)
			break
		}
	}
}

func (b *Bot) findAttackPosition(u *game.Unit, target *game.Unit, attackRange int) (game.HexCoord, bool) {
	if u.PosVal().Distance(target.PosVal()) <= attackRange {
		return u.PosVal(), true
	}

	bestDist := math.MaxInt
	var bestPos game.HexCoord

	for coord, cell := range b.state.board.Cells {
		if cell.Unit != nil {
			continue
		}
		moveDist := u.PosVal().Distance(coord)
		if moveDist > u.CurrentMP {
			continue
		}
		if coord.Distance(target.PosVal()) > attackRange {
			continue
		}
		if moveDist < bestDist {
			bestDist = moveDist
			bestPos = coord
		}
	}

	if bestDist == math.MaxInt {
		return game.HexCoord{}, false
	}
	return bestPos, true
}

func (b *Bot) randomReachableCell(u *game.Unit) *game.HexCoord {
	cells := game.ReachableCells(u.PosVal(), u.CurrentMP, *b.state.board)

	if len(cells) == 0 {
		return nil
	}

	cell := cells[rand.IntN(len(cells))] //nolint:gosec
	return &cell
}

func (b *Bot) enemies(of *game.Unit) []*game.Unit {
	enemies := []*game.Unit{}
	for _, u := range b.state.queue {
		if u.IsEnemy(of) {
			enemies = append(enemies, u)
		}
	}

	return enemies
}

func (b *Bot) cellAt(at game.HexCoord) *game.BoardCell {
	return b.state.board.Cells[at]
}

// addUnitStatus adds a status effect to the unit and refreshes its board card.
func (b *Bot) addUnitStatus(u *game.Unit, statusType status.Type, meta map[string]any) {
	st := status.ByType(statusType)
	if st == nil {
		log.Printf("addUnitStatus: unknown status type %s", statusType)
		return
	}
	if u.Statuses == nil {
		u.Statuses = make(map[status.Type]status.Value)
	}

	u.Statuses[statusType] = status.Value{
		UnitID:   u.ID,
		Duration: st.Duration,
		Value:    st.InitialValue,
		Status:   st,
		Meta:     meta,
	}
}

// removeUnitStatus removes a status effect from the unit and refreshes its board card.
func (b *Bot) removeUnitStatus(u *game.Unit, statusType status.Type) {
	st := status.ByType(statusType)
	if st != nil {
		delete(u.Statuses, statusType)
	}
}

func (b *Bot) updateUnitStatusDuration(u *game.Unit, statusDur map[status.Type]int) {
	for st, dur := range statusDur {
		sv, ok := u.Statuses[st]
		if !ok {
			continue
		}

		sv.Duration = dur
		u.Statuses[st] = sv
	}
}
