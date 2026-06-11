package game

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/game/ability"
	"github.com/ognev-dev/goplease/game/ability/status"
)

// Arena holds the full state of one match.
type Arena struct {
	mu sync.Mutex

	ID         ds.ID
	Board      *Board
	Players    [2]*Player
	UnitsQueue []*Unit

	CurrentRound int
	ActivePlayer int // 0 or 1 whose turn is
	ActiveUnitID ds.ID

	Phase                  RoundPhase
	UnitsPerPlacementPhase int
}

func NewArena(p1, p2 *Player) *Arena {
	return &Arena{
		ID:                     ds.NewID(),
		Players:                [2]*Player{p1, p2},
		UnitsQueue:             []*Unit{},
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

// CheckGameOver returns the loser's index if one side has no living units, or -1 if game continues.
func (a *Arena) CheckGameOver() int {
	for i, p := range a.Players {
		hasAlive := false
		for _, u := range a.UnitsQueue {
			if u.OwnerID == p.ID && !u.IsDead {
				hasAlive = true
				break
			}
		}
		if !hasAlive && len(a.UnitsQueue) > 0 {
			return i
		}
	}

	return -1
}

func (a *Arena) ActingUnit() *Unit {
	for _, u := range a.UnitsQueue {
		if a.ActiveUnitID == u.ID {
			return u
		}
	}

	return nil
}

func (a *Arena) MarkPlayerReady(playerID ds.ID) (bothReady bool) {
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

func (a *Arena) PlayerByUnitID(unitID ds.ID) (*Player, int) {
	for _, u := range a.UnitsQueue {
		if u.ID == unitID {
			for i, p := range a.Players {
				if p.ID == u.OwnerID {
					return p, i
				}
			}
		}
	}

	return nil, -1
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

	u.Pos = &at
	cell.Unit = u
	player.UnitsPlacedThisRound++
	a.UnitsQueue = append(a.UnitsQueue, u)

	return u, nil
}

func (a *Arena) MoveUnit(unitID ds.ID, to HexCoord, playerID ds.ID) (sts ApplyStates, err error) {
	player, _ := a.PlayerByID(playerID)
	if player == nil {
		err = fmt.Errorf("player %q not found", playerID)
		return
	}

	if a.ActiveUnitID != unitID {
		err =  fmt.Errorf("unit %q is not active", unitID)
		return
	}

	u := a.ActingUnit()
	if u == nil {
		err = fmt.Errorf("acting unit not found")
		return
	}

	if u.OwnerID != playerID {
		err =  fmt.Errorf("unit %q does not belong to player %q", unitID, playerID)
		return
	}

	cell, ok := a.Board.Cells[to]
	if !ok || cell == nil {
		err = fmt.Errorf("cell %q not found", to)
		return
	}
	if cell.Unit != nil {
		err = fmt.Errorf("cell %q is occupied", to)
		return
	}

	dist := u.Pos.Distance(to)
	if dist > u.CurrentMP {
		err = fmt.Errorf("not enough MP: need %d, have %d", dist, u.CurrentMP)
		return
	}

	u.CurrentMP -= dist
	sts.With(a.relocateUnit(u, to))

	return sts, nil
}

// relocateUnit moves a unit to a new cell and applies onMove triggers.
// No validation — caller is responsible.
func (a *Arena) relocateUnit(u *Unit, to HexCoord) (sts ApplyStates) {
	a.Board.Cells[u.PosVal()].Unit = nil
	u.Pos = &to
	a.Board.Cells[to].Unit = u

	return triggers.UnitMoved(a, u)
}

func (a *Arena) EndTurn(playerID ds.ID) (state ApplyStates, err error) {
	if a.ActiveUnitID.IsNil() {
		if a.Players[a.ActivePlayer].ID != playerID {
			err = fmt.Errorf("not your turn")
			return
		}
		return // queue exhausted — advanceGameLoop will handle it
	}

	u := a.ActingUnit()
	if u == nil {
		err = errors.New("acting unit not found")
		return
	}
	if u.OwnerID != playerID {
		err = fmt.Errorf("not your turn")
		return
	}



	// Decrease status durations
	for t, sv := range u.Statuses {
		// before status removed, trigger onTurnEnd
		h, ok := statusHandlers[t]
		if ok && h != nil && h.onTurnEnd != nil {
			state.With(h.onTurnEnd(a, u, sv))
		}

		if sv.Duration == status.Permanent {
			continue
		}

		sv.Duration--
		if sv.Duration < 1 {
			delete(u.Statuses, t)
			state.ToAll(ApplyState{RemoveStatus: &t, ToUnitID: u.ID})
		} else {
			u.Statuses[t] = sv
			state.ToAll(ApplyState{
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
			state.ToAll(ApplyState{SetCooldown: &map[ability.ID]int{abID: cd}, ToUnitID: u.ID})
		}
	}

	// Shield decays by 1 every turn
	if u.CurrentShield > 0 {
		u.CurrentShield--
		state.ToAll(ApplyState{SetShield: &u.CurrentShield, ToUnitID: u.ID})
	}

	// Advance to next unit in queue
	a.advanceActiveUnit()

	return state, nil
}

func (a *Arena) advanceActiveUnit() {
	for i, u := range a.UnitsQueue {
		if u.ID == a.ActiveUnitID {
			if i+1 < len(a.UnitsQueue) {
				a.ActiveUnitID = a.UnitsQueue[i+1].ID
			} else {
				a.ActiveUnitID = ds.NilID
			}
			return
		}
	}
}

func (a *Arena) UnitAt(pos HexCoord) *Unit {
	c := a.Board.Cells[pos]
	if c != nil {
		return c.Unit
	}

	return nil
}

func (a *Arena) ShouldUnitAt(pos HexCoord) (*Unit, error) {
	unit := a.UnitAt(pos)
	if unit == nil {
		return nil, fmt.Errorf("unit not found at: %s", pos)
	}

	return unit, nil
}

func (a *Arena) UseAbility(req UseAbilityPayload, playerID ds.ID) (state ApplyStates, err error) {
	u := a.ActingUnit()
	if u == nil {
		return ApplyStates{}, fmt.Errorf("no acting unit")
	}
	if u.OwnerID != playerID {
		return ApplyStates{}, fmt.Errorf("not your turn")
	}

	ab := ability.ByID(req.AbilityID)
	err = a.ValidateAbilityUse(u, ab, req.Target)
	if err != nil {
		return
	}

	var target HexCoord
	if req.Target != nil {
		target = *req.Target
	}

	// Passive abilities do not consume AP.
	if !ab.IsPassive {
		if u.CurrentAP > 0 {
			u.CurrentAP--
			state.ToAll(ApplyState{SetAP: &u.CurrentAP, ToUnitID: u.ID})
		} else {
			player, _ := a.PlayerByID(playerID)
			if player == nil || player.PhantomAP < 1 {
				err = fmt.Errorf("unit %s has no AP or Phantom AP", u.ID)
				return
			}
			if u.PhantomAPUsedThisTurn >= MaxPhantomAPPerUnitPerTurn {
				err = fmt.Errorf("unit %s already spent max Phantom AP this turn", u.ID)
				return
			}
			player.PhantomAP--
			u.PhantomAPUsedThisTurn++
			state.ToSelf(ApplyState{SetPhantomAP: &player.PhantomAP})
		}
	}

	handler, ok := abilityHandlers[req.AbilityID]
	if !ok {
		err = fmt.Errorf("no handler for ability: %s", req.AbilityID)
		return
	}

	event := abilityUsedEvent{
		By: u,
		Ab: ab,
		At: target,
	}
	handlerStates, err := handler(a, event)
	if err != nil {
		return state, err
	}
	state.With(handlerStates)
	state.ToOpp(ApplyState{
		ToUnitID: u.ID,
		UseAbility: new(req),
	})

	u.SetCooldown(ab.ID, ab.Cooldown)

	return state, nil
}

// DealDamageToUnit applies val damage to the unit, accounting for shield absorption.
// Shield absorbs damage first and any excess damage carries over to HP.
// If HP reaches zero, the unit is marked as dead and removed from the queue.
// Returns a slice of ApplyState mutations to be sent to the client for visual feedback.
func (a *Arena) DealDamageToUnit(source, target *Unit, val int) (state ApplyStates) {
	defer func() {
		if !state.IsEmpty() {
			state.With(triggers.DamageReceived(a, source, target))
		}
	}()

	eff, ok := target.Statuses[status.Exposed]
	if ok {
		val += eff.Value
	}

	var shieldRemoved int
	if target.CurrentShield > 0 {
		if target.CurrentShield < val {
			shieldRemoved = target.CurrentShield
			target.CurrentShield = 0
			val = val - shieldRemoved
		} else {
			shieldRemoved = val
			target.CurrentShield -= val
			val = 0
		}
	}

	if shieldRemoved > 0 {
		state.ToAll(
			ApplyState{ChangeShield: new(-shieldRemoved), ToUnitID: target.ID},
			ApplyState{SetShield: new(target.CurrentShield), ToUnitID: target.ID},
		)
	}

	// shield fully absorbed the damage
	if val == 0 {
		return state
	}

	if target.CurrentHP < val {
		val = target.CurrentHP
	}

	target.CurrentHP -= val
	state.ToAll(
		ApplyState{ChangeHP: new(-val), ToUnitID: target.ID},
		ApplyState{SetHP: new(target.CurrentHP), ToUnitID: target.ID},
	)

	if target.CurrentHP <= 0 {
		target.IsDead = true
		state.With(triggers.SomebodyJustExpectedlyDied(a, target))

		if target.IsDead {
			a.RemoveUnitFromQueue(target.ID)
			state.ToAll(ApplyState{IsDead: true, ToUnitID: target.ID})

			// Recalculate after unit is removed from queue so counts are correct.
			state.With(a.RecalculatePhantomAP())
		}
	}

	return
}

// RemoveUnitFromQueue removes a dead unit and advances ActiveUnitID if needed.
func (a *Arena) RemoveUnitFromQueue(unitID ds.ID) {
	removedIdx := -1
	for i, u := range a.UnitsQueue {
		if u.ID == unitID {
			removedIdx = i
			a.UnitsQueue = append(a.UnitsQueue[:i], a.UnitsQueue[i+1:]...)
			break
		}
	}

	if removedIdx < 0 {
		return
	}

	if a.ActiveUnitID != unitID {
		return
	}

	if removedIdx < len(a.UnitsQueue) {
		a.ActiveUnitID = a.UnitsQueue[removedIdx].ID
	} else {
		a.ActiveUnitID = ds.NilID
	}
}

// RecalculatePhantomAP recalculates the Phantom AP pool for both players.
// Phantom AP = max(0, enemy living units - friendly living units).
func (a *Arena) RecalculatePhantomAP() (state ApplyStates) {
	counts := [2]int{}
	for i, p := range a.Players {
		for _, u := range a.UnitsQueue {
			if u.OwnerID == p.ID {
				counts[i]++
			}
		}
	}

	for i := range a.Players {
		delta := counts[1-i] - counts[i]
		if delta < 0 {
			delta = 0
		}
		a.Players[i].PhantomAP = delta
	}

	state.ToSelf(ApplyState{SetPhantomAP: new(a.Players[a.ActivePlayer].PhantomAP)})
	state.ToOpp(ApplyState{SetPhantomAP: new(a.Players[1-a.ActivePlayer].PhantomAP)})

	return state
}

func (a *Arena) AlliesInRange(u *Unit, radius int) []*Unit {
	units := []*Unit{}
	cells := a.Board.Cells.InRangeHavingUnit(u.PosVal(), radius)
	for _, c := range cells {
		if !c.Unit.Alive() || c.Unit.IsEnemy(u){
			continue
		}

		units = append(units, c.Unit)
	}

	return units
}

func (a *Arena) EnemiesInRange(u *Unit, radius int) []*Unit {
	units := []*Unit{}
	cells := a.Board.Cells.InRangeHavingUnit(u.PosVal(), radius)
	for _, c := range cells {
		if !c.Unit.Alive() || c.Unit.IsAlly(u){
			continue
		}

		units = append(units, c.Unit)
	}

	return units
}

func (a *Arena) AlliesInRangeHavingAbility(u *Unit, radius int, abID ability.ID) []*Unit {
	units := []*Unit{}
	cells := a.Board.Cells.InRangeHavingUnitAbility(u.PosVal(), radius, abID)
	for _, c := range cells {
		if !c.Unit.Alive() || c.Unit.IsEnemy(u) {
			continue
		}

		units = append(units, c.Unit)
	}

	return units
}

func (a *Arena) EnemiesInRangeHavingAbility(u *Unit, radius int, abID ability.ID) []*Unit {
	units := []*Unit{}
	cells := a.Board.Cells.InRangeHavingUnitAbility(u.PosVal(), radius, abID)
	for _, c := range cells {
		if !c.Unit.Alive() || c.Unit.IsAlly(u) {
			continue
		}

		units = append(units, c.Unit)
	}

	return units
}

func (a *Arena) CountEnemiesInRange(u *Unit, rangeN int, atLeastOpt ...int) (count int) {
	var atLeast int
	if len(atLeastOpt) > 0 {
		atLeast = atLeastOpt[0]
	}

	for to, cell := range a.Board.Cells {
		if cell.Unit == nil {
			continue
		}
		if cell.Unit.IsAlly(u) {
			continue
		}

		if u.Pos.Distance(to) <= rangeN {
			count++
			if atLeast > 0 && count >= atLeast {
				break
			}
		}
	}

	return
}

func (a *Arena) CellOccupied(at HexCoord) (ok bool) {
	c := a.Board.Cells[at]
	if c == nil {
		return
	}

	return c.Unit != nil
}

func (a *Arena) ValidateAbilityUse(caster *Unit, ab ability.Ability, targetAt *HexCoord) error {
	 err := caster.ValidateAbilityUse(ab.ID)
	 if err != nil {
		return err
	}

	if ab.IsPassive {
		return fmt.Errorf("ability %s is passive and cannot be activated", ab.ID)
	}

	if !ab.IsPassive && caster.CurrentAP < 1 {
		player, _ := a.PlayerByID(caster.OwnerID)
		if player == nil || player.PhantomAP < 1 {
			return fmt.Errorf("not enough AP")
		}
	}

	if ab.Activation != ability.Instant && targetAt == nil {
		return fmt.Errorf("ability %s requires a target", ab.ID)
	}

	if targetAt == nil || ab.Activation == ability.Instant {
		return nil
	}

	rangeN := ab.Range
	if ab.Area != "" {
		rangeN = ab.AreaRadius
	}

	log.Printf("[validate] caster pos: %v, target: %v, distance: %d, range: %d",
		caster.PosVal(), *targetAt, caster.PosVal().Distance(*targetAt), rangeN)

	if caster.Pos.Distance(*targetAt) > rangeN {
		return fmt.Errorf("target out of range: distance %d, range %d",
			caster.Pos.Distance(*targetAt), rangeN)
	}

	target := a.UnitAt(*targetAt)

	switch ab.TargetMode {
	case ability.TargetEnemies, ability.TargetEnemiesAndSelf:
		if target == nil {
			return fmt.Errorf("ability %s: no unit at target cell", ab.ID)
		}
		if target.OwnerID == caster.OwnerID && ab.TargetMode == ability.TargetEnemies {
			return fmt.Errorf("ability %s: cannot target ally", ab.ID)
		}

	case ability.TargetAllies, ability.TargetAlliesAndSelf:
		if target == nil {
			return fmt.Errorf("ability %s: no unit at target cell", ab.ID)
		}
		if target.OwnerID != caster.OwnerID {
			return fmt.Errorf("ability %s: cannot target enemy", ab.ID)
		}

	case ability.TargetSelf:
		if target == nil || target.ID != caster.ID {
			return fmt.Errorf("ability %s: must target self", ab.ID)
		}

	case ability.TargetAny:
		if target == nil {
			return fmt.Errorf("ability %s: no unit at target", ab.ID)
		}

	case "": // no target mode — free cell or AOE, no unit required
	}

	// Activation-specific checks
	switch ab.Activation {
	case ability.SelectFreeCell:
		if target != nil {
			return fmt.Errorf("ability %s: target cell is occupied", ab.ID)
		}
	}

	return nil
}