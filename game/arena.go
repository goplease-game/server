package game

import (
	"fmt"
	"math/rand"
	"sync"

	"github.com/google/uuid"
	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/game/ability"
	"github.com/ognev-dev/goplease/game/ability/status"
)

const UnitsPerPlacementPhase = 3

// Arena holds the full state of one match.
type Arena struct {
	mu sync.Mutex

	ID         string
	Board      *Board
	Players    [2]*Player
	UnitsQueue []*Unit

	CurrentRound int
	ActivePlayer int // 0 or 1 whose turn is
	ActiveUnitID ds.ID

	CurrentTurn            int
	Phase                  RoundPhase
	UnitsPerPlacementPhase int
}

func NewArena(p1, p2 *Player) *Arena {
	return &Arena{
		ID:                     uuid.NewString(),
		Players:                [2]*Player{p1, p2},
		UnitsQueue:             []*Unit{},
		CurrentTurn:            0,
		ActivePlayer:           rand.Intn(2),
		Phase:                  PlacementPhase,
		Board:                  NewBoard(),
		UnitsPerPlacementPhase: UnitsPerPlacementPhase,
	}
}

func (a *Arena) playerByID(id ds.ID) (*Player, int) {
	for i, p := range a.Players {
		if p.ID == id {
			return p, i
		}
	}

	return nil, -1
}

func (a *Arena) ActingUnit() *Unit {
	for _, u := range a.UnitsQueue {
		if a.ActiveUnitID == u.ID {
			return u
		}
	}

	return nil
}

func (a *Arena) MarkReady(playerID ds.ID) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	_, idx := a.playerByID(playerID)
	if idx < 0 {
		return false
	}
	a.Players[idx].Ready = true

	return a.IsPlayersReady()
}

func (a *Arena) IsPlayersReady() bool {
	return a.Players[0].Ready && a.Players[1].Ready
}

func (a *Arena) IsPlayerPlacementDone(idx int) bool {
	p := a.Players[idx]
	return p.UnitsPlacedThisRound >= a.UnitsPerPlacementPhase || len(p.Units) == 0
}

func (a *Arena) PlacementActorIndex() int {
	p1 := a.Players[0].UnitsPlacedThisRound
	p2 := a.Players[1].UnitsPlacedThisRound
	if p2 < p1 {
		return 1
	}

	return 0 // tie-breaker: P1
}

func (a *Arena) PlayerByUnitID(unitID ds.ID) *Player {
	for _, u := range a.UnitsQueue {
		if u.ID == unitID {
			for _, p := range a.Players {
				if p.ID == u.OwnerID {
					return p
				}
			}
		}
	}
	return nil
}

func (a *Arena) PlayerByID(id ds.ID) (*Player, int) {
	for i, p := range a.Players {
		if p.ID == id {
			return p, i
		}
	}
	return nil, -1
}

func (a *Arena) PlaceUnitFromHandToBoard(templateID int, at HexCoord, playerID ds.ID) (u *Unit, err error) {
	player, playerIdx := a.PlayerByID(playerID)
	if player == nil {
		return nil, fmt.Errorf("player %q not found", playerID)
	}

	if a.PlacementActorIndex() != playerIdx {
		return nil, fmt.Errorf("not your turn to place")
	}

	cell, ok := a.Board.Cells[at]
	if !ok || cell == nil {
		return nil, fmt.Errorf("cell %q not found", at)
	}
	if cell.Unit != nil {
		return nil, fmt.Errorf("cell %q already has a unit", at)
	}
	if !cell.IsSafeZone || cell.SafeZonePlayer != playerIdx {
		return nil, fmt.Errorf("cell %q is not a placement zone", at)
	}

	u = player.PopUnitFromHand(templateID)
	if u == nil {
		return nil, fmt.Errorf("unit with template %d not found in hand", templateID)
	}

	u.Pos = at
	cell.Unit = u
	player.UnitsPlacedThisRound++
	a.UnitsQueue = append(a.UnitsQueue, u)

	return u, nil
}

func (a *Arena) MoveUnit(unitID ds.ID, to HexCoord, playerID ds.ID) error {
	player, _ := a.PlayerByID(playerID)
	if player == nil {
		return fmt.Errorf("player %q not found", playerID)
	}

	// Проверяем что сейчас ход этого игрока
	if a.ActiveUnitID != unitID {
		return fmt.Errorf("unit %q is not active", unitID)
	}

	u := a.ActingUnit()
	if u == nil {
		return fmt.Errorf("acting unit not found")
	}

	if u.OwnerID != playerID {
		return fmt.Errorf("unit %q does not belong to player %q", unitID, playerID)
	}

	cell, ok := a.Board.Cells[to]
	if !ok || cell == nil {
		return fmt.Errorf("cell %q not found", to)
	}
	if cell.Unit != nil {
		return fmt.Errorf("cell %q is occupied", to)
	}

	dist := u.Pos.Distance(to)
	if dist > u.CurrentMP {
		return fmt.Errorf("not enough MP: need %d, have %d", dist, u.CurrentMP)
	}

	// Обновляем состояние
	a.Board.Cells[u.Pos].Unit = nil
	u.CurrentMP -= dist
	u.Pos = to
	cell.Unit = u

	return nil
}

func (a *Arena) EndTurn(playerID ds.ID) (st ApplyStates, err error) {
	if a.ActiveUnitID.IsNil() {
		return nil, fmt.Errorf("no active unit")
	}

	u := a.ActingUnit()
	if u == nil {
		return nil, fmt.Errorf("acting unit not found")
	}
	if u.OwnerID != playerID {
		return nil, fmt.Errorf("not your turn")
	}

	// Decrease status durations
	for t, sv := range u.Statuses {
		if sv.Duration == status.Permanent {
			continue
		}

		sv.Duration--
		if sv.Duration < 1 {
			delete(u.Statuses, t)
			st.Add(ApplyState{RemoveStatus: &t, ToUnitID: u.ID})
		} else {
			u.Statuses[t] = sv
			st.Add(ApplyState{
				SetStatusDuration: map[status.Type]int{t: sv.Duration},
				ToUnitID:          u.ID,
			})
		}
	}

	// Reduce ability cooldowns
	for abID, cd := range u.Cooldowns {
		if cd > 0 {
			cd--
			u.Cooldowns[abID] = cd
			st.Add(ApplyState{SetCooldown: &map[ability.ID]int{abID: cd}, ToUnitID: u.ID})
		}
	}

	// Shield decays by 1 every turn
	if u.CurrentShield > 0 {
		u.CurrentShield--
		st.Add(ApplyState{SetShield: &u.CurrentShield, ToUnitID: u.ID})
	}

	// Advance to next unit in queue
	a.advanceActiveUnit()

	return st, nil
}

func (a *Arena) advanceActiveUnit() {
	if a.ActiveUnitID.IsNil() {
		if len(a.UnitsQueue) > 0 {
			a.ActiveUnitID = a.UnitsQueue[0].ID
		}
		return
	}

	for i, u := range a.UnitsQueue {
		if u.ID == a.ActiveUnitID {
			if i+1 < len(a.UnitsQueue) {
				a.ActiveUnitID = a.UnitsQueue[i+1].ID
			} else {
				a.ActiveUnitID = ds.NilID // queue exhausted — triggers new round
			}
			return
		}
	}
}
