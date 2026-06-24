package game

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"github.com/goplease-game/server/ability"
	"github.com/goplease-game/server/ability/status"
	"github.com/goplease-game/server/ds"
)

var (
	// ErrAbilityIsPassive is returned when attempting to activate a passive ability.
	ErrAbilityIsPassive = errors.New("ability is passive and cannot be activated")

	// ErrNotEnoughAP is returned when neither action points nor phantom AP are available.
	ErrNotEnoughAP = errors.New("not enough AP")

	// ErrTargetRequired is returned when an ability requires a target but none was provided.
	ErrTargetRequired = errors.New("ability requires a target")

	// ErrTargetOutOfRange is returned when the target is outside the ability range.
	ErrTargetOutOfRange = errors.New("target out of range")

	// ErrNoUnitAtTarget is returned when there is no unit at the target cell.
	ErrNoUnitAtTarget = errors.New("no unit at target cell")

	// ErrCannotTargetAlly is returned when an ability cannot target allied units.
	ErrCannotTargetAlly = errors.New("cannot target ally")

	// ErrCannotTargetEnemy is returned when an ability cannot target enemy units.
	ErrCannotTargetEnemy = errors.New("cannot target enemy")

	// ErrMustTargetSelf is returned when an ability must target the caster itself.
	ErrMustTargetSelf = errors.New("must target self")

	// ErrNoActingUnit is returned when there is no unit currently acting.
	ErrNoActingUnit = errors.New("no acting unit")

	// ErrNotYourTurn is returned when a player tries to act outside their turn.
	ErrNotYourTurn = errors.New("not your turn")

	// ErrNoAbilityHandler is returned when no handler exists for the ability.
	ErrNoAbilityHandler = errors.New("no handler for ability")

	// ErrNoAPAndNoPhantomAP is returned when the unit has no AP and no Phantom AP available.
	ErrNoAPAndNoPhantomAP = errors.New("no AP or Phantom AP")

	// ErrMaxPhantomAPUsed is returned when the unit has already spent max Phantom AP this turn.
	ErrMaxPhantomAPUsed = errors.New("max Phantom AP already spent this turn")

	// ErrUnitNotFoundAtPosition is returned when no unit exists at the specified position.
	ErrUnitNotFoundAtPosition = errors.New("unit not found at position")

	// ErrActingUnitNotFound is returned when there is no currently acting unit in the arena.
	ErrActingUnitNotFound = errors.New("acting unit not found")

	// ErrPlayerNotFound is returned when the specified player does not exist.
	ErrPlayerNotFound = errors.New("player not found")

	// ErrUnitNotActive is returned when the unit is not the currently active unit.
	ErrUnitNotActive = errors.New("unit is not active")

	// ErrUnitNotFound is returned when the acting unit cannot be found.
	ErrUnitNotFound = errors.New("acting unit not found")

	// ErrUnitNotOwnedByPlayer is returned when a unit does not belong to the player.
	ErrUnitNotOwnedByPlayer = errors.New("unit does not belong to player")

	// ErrCellNotFound is returned when the target cell does not exist on the board.
	ErrCellNotFound = errors.New("cell not found")

	// ErrCellOccupied is returned when the target cell already contains a unit.
	ErrCellOccupied = errors.New("cell is occupied")

	// ErrNotEnoughMovementPoints is returned when unit does not have enough movement points.
	ErrNotEnoughMovementPoints = errors.New("not enough movement points")

	// ErrNotPlacementTurn indicates that it is not the player's turn to place a unit.
	ErrNotPlacementTurn = errors.New("not your turn to place")

	// ErrCellNotInPlacementZone indicates that the cell is not a valid placement zone for the player.
	ErrCellNotInPlacementZone = errors.New("cell is not a placement zone")

	// ErrUnitNotInHand indicates that the requested unit template is not available in the player's hand.
	ErrUnitNotInHand = errors.New("unit not found in hand")
)

// Arena holds the full state, spatial matrix, turn queues, and metadata of a single active game match.
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

	DisableGameOver bool
	DisableBot bool
	DisableTurnTimer bool
}

// NewArena initializes and returns a pointer to a new Arena instance linking two competitive players.
func NewArena(p1, p2 *Player) *Arena {
	return &Arena{
		ID:                     ds.NewID(),
		Players:                [2]*Player{p1, p2},
		UnitsQueue:             []*Unit{},
		ActivePlayer:           rand.IntN(2), //nolint:gosec
		Phase:                  PlacementPhase,
		Board:                  NewBoard(),
		UnitsPerPlacementPhase: UnitsPerPlacementPhase,
	}
}

// CheckGameOver returns the loser's array index if one side has no living units left, or -1 if the game continues.
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

// ActingUnit finds and returns the pointer to the unit that currently possesses the active turn token.
func (a *Arena) ActingUnit() *Unit {
	for _, u := range a.UnitsQueue {
		if a.ActiveUnitID == u.ID {
			return u
		}
	}

	return nil
}

// MarkPlayerReady flags the designated player as ready for the next round and returns if both players are ready.
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

// IsPlayersReady evaluates if both players in the match have toggled their ready status flags.
func (a *Arena) IsPlayersReady() bool {
	return a.Players[0].Ready && a.Players[1].Ready
}

// IsPlayerPlacementDone verifies if a player has exhausted their available units or filled their placement cap.
func (a *Arena) IsPlayerPlacementDone(idx int) bool {
	p := a.Players[idx]
	return p.UnitsPlacedThisRound >= a.UnitsPerPlacementPhase || len(p.Units) == 0
}

// PlacementActorIndex determines which player's turn it is to deploy a unit during the placement phase.
func (a *Arena) PlacementActorIndex() int {
	p1 := a.Players[0].UnitsPlacedThisRound
	p2 := a.Players[1].UnitsPlacedThisRound
	if p2 < p1 {
		return 1
	}

	return 0 // tie-breaker: P1
}

// PlayerByUnitID retrieves the player instance owning the specified unit ID along with their match index.
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

// PlayerByID locates and returns a player object based on their personal unique identity string.
func (a *Arena) PlayerByID(id ds.ID) (*Player, int) {
	for i, p := range a.Players {
		if p.ID == id {
			return p, i
		}
	}
	return nil, -1
}

// PlaceUnitFromHandToBoard binds a chosen unit from a player's hand onto a safe zone grid cell on the game map.
func (a *Arena) PlaceUnitFromHandToBoard(templateID int, at HexCoord, playerID ds.ID) (*Unit, error) {
	player, playerIdx := a.PlayerByID(playerID)
	if player == nil {
		return nil, ErrPlayerNotFound
	}

	if a.PlacementActorIndex() != playerIdx {
		return nil, ErrNotPlacementTurn
	}

	cell, ok := a.Board.Cells[at]
	if !ok || cell == nil {
		return nil, ErrCellNotFound
	}
	if cell.Unit != nil {
		return nil, ErrCellOccupied
	}
	if !cell.IsSafeZone || cell.SafeZonePlayer != playerIdx {
		return nil, ErrCellNotInPlacementZone
	}

	u := player.PopUnitFromHand(templateID)
	if u == nil {
		return nil, ErrUnitNotInHand
	}

	u.Pos = &at
	cell.Unit = u
	player.UnitsPlacedThisRound++
	a.UnitsQueue = append(a.UnitsQueue, u)

	return u, nil
}

// MoveUnit validates movement cost, consumes movement points, and transitions a unit to a target cell.
func (a *Arena) MoveUnit(unitID ds.ID, to HexCoord, playerID ds.ID) (sts ApplyStates, err error) {
	player, _ := a.PlayerByID(playerID)
	if player == nil {
		return ApplyStates{}, ErrPlayerNotFound
	}

	if a.ActiveUnitID != unitID {
		return ApplyStates{}, ErrUnitNotActive
	}

	u := a.ActingUnit()
	if u == nil {
		return ApplyStates{}, ErrUnitNotFound
	}

	if u.OwnerID != playerID {
		return ApplyStates{}, ErrUnitNotOwnedByPlayer
	}

	cell, ok := a.Board.Cells[to]
	if !ok || cell == nil {
		return ApplyStates{}, ErrCellNotFound
	}
	if cell.Unit != nil {
		return ApplyStates{}, ErrCellOccupied
	}

	dist := u.Pos.Distance(to)
	if dist > u.CurrentMP {
		return ApplyStates{}, ErrNotEnoughMovementPoints
	}

	u.CurrentMP -= dist
	sts.With(a.relocateUnit(u, to))

	return sts, nil
}

// EndTurn decrements active statuses, cycles skill cooldowns, decays shield layers, and prompts the unit queue forward.
func (a *Arena) EndTurn(playerID ds.ID) (state ApplyStates, err error) {
	if a.ActiveUnitID.IsNil() {
		if a.Players[a.ActivePlayer].ID != playerID {
			err= ErrNotYourTurn
			return
		}
		return
	}

	u := a.ActingUnit()
	if u == nil {
		err = ErrActingUnitNotFound
		return
	}
	if u.OwnerID != playerID {
		return
	}

	// Decrease status durations
	for t, sv := range u.Statuses {
		h, ok := statusHandlers[t]
		if ok && h != nil && h.onTurnEnd != nil {
			state.With(h.onTurnEnd(a, u, sv))
		}

		if sv.Duration == status.Permanent {
			continue
		}

		sv.Duration--
		if sv.Duration < 1 {
			if h != nil && h.onRemove != nil {
				state.With(h.onRemove(a, u, sv))
			}
			u.RemoveStatus(t)
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

	a.advanceActiveUnit()

	return state, nil
}

// UnitAt queries the board coordinates and returns the occupant unit pointer if present.
func (a *Arena) UnitAt(pos HexCoord) *Unit {
	c := a.Board.Cells[pos]
	if c != nil {
		return c.Unit
	}

	return nil
}

// ShouldUnitAt extracts the unit sitting on a tile or throws an descriptive error message if empty.
func (a *Arena) ShouldUnitAt(pos HexCoord) (*Unit, error) {
	unit := a.UnitAt(pos)
	if unit == nil {
		return nil, ErrUnitNotFoundAtPosition
	}

	return unit, nil
}

// UseAbility validates action point reserves, runs validation mechanics, and executes custom ability handler functions.
func (a *Arena) UseAbility(req UseAbilityPayload, playerID ds.ID) (state ApplyStates, err error) {
	u := a.ActingUnit()
	if u == nil {
		return ApplyStates{}, ErrNoActingUnit
	}
	if u.OwnerID != playerID {
		return ApplyStates{}, ErrNotYourTurn
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
				return ApplyStates{}, ErrNoAPAndNoPhantomAP
			}
			if u.PhantomAPUsedThisTurn >= MaxPhantomAPPerUnitPerTurn {
				return ApplyStates{}, ErrMaxPhantomAPUsed
			}

			player.PhantomAP--
			u.PhantomAPUsedThisTurn++

			state.ToSelf(ApplyState{SetPhantomAP: &player.PhantomAP})
		}
	}

	handler, ok := abilityHandlers[req.AbilityID]
	if !ok {
		return ApplyStates{}, ErrNoAbilityHandler
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
		ToUnitID:   u.ID,
		UseAbility: new(req),
	})

	u.SetCooldown(ab.ID, ab.Cooldown)

	return state, nil
}

// DealDamageToUnit processes an incoming damage event against a unit, reducing shields first before harming health points.
func (a *Arena) DealDamageToUnit(source, target *Unit, val int) (state ApplyStates) {
	defer func() {
		if !state.IsEmpty() {
			state.With(triggers.DamageReceived(a, source, target))
			state.With(triggers.DamageDealt(a, source, target))
		}
	}()

	triggerStatusOnDamageCalculated(a, target, &val)

	var shieldRemoved int
	if target.CurrentShield > 0 {
		if target.CurrentShield < val {
			shieldRemoved = target.CurrentShield
			target.CurrentShield = 0
			val -= shieldRemoved
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

	// Shield fully absorbed the damage
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

			state.With(a.RecalculatePhantomAP())
		}
	}

	return
}

// RemoveUnitFromQueue extracts a dead or missing unit out of the initiative loop array and shifts indices if needed.
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

// RecalculatePhantomAP dynamically calculates the compensation action points assigned due to asymmetrical team populations.
func (a *Arena) RecalculatePhantomAP() (state ApplyStates) {
	counts := [2]int{}
	for i, p := range a.Players {
		for _, u := range a.UnitsQueue {
			if u.OwnerID == p.ID {
				counts[i]++ //nolint:gosec
			}
		}
	}

	for i := range a.Players {
		delta := counts[1-i] - counts[i] //nolint:gosec
		delta = max(0, delta)
		a.Players[i].PhantomAP = delta
	}

	state.ToSelf(ApplyState{SetPhantomAP: new(a.Players[a.ActivePlayer].PhantomAP)})
	state.ToOpp(ApplyState{SetPhantomAP: new(a.Players[1-a.ActivePlayer].PhantomAP)})

	return state
}

// AlliesInRange locates friendly units sitting within a radial threshold relative to a certain coordinate source.
func (a *Arena) AlliesInRange(u *Unit, radius int) []*Unit {
	units := []*Unit{}
	cells := a.Board.Cells.InRangeHavingUnit(u.PosVal(), radius)
	for _, c := range cells {
		if !c.Unit.Alive() || c.Unit.IsEnemy(u) {
			continue
		}

		units = append(units, c.Unit)
	}

	return units
}

// EnemiesInRange locates opposing force units positioned inside a radial zone extending from the target host.
func (a *Arena) EnemiesInRange(u *Unit, radius int) []*Unit {
	units := []*Unit{}
	cells := a.Board.Cells.InRangeHavingUnit(u.PosVal(), radius)
	for _, c := range cells {
		if !c.Unit.Alive() || c.Unit.IsAlly(u) {
			continue
		}

		units = append(units, c.Unit)
	}

	return units
}

// AlliesInRangeHavingAbility filters and returns friendly units possessing a specific ability within range.
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

// EnemiesInRangeHavingAbility filters and returns opposing units possessing a specific ability within range.
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

// CountEnemiesInRange aggregates the raw summation of nearby enemy units, optimization breaks early if requested.
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

// CellOccupied checks the coordinate cells mapping index to confirm whether any living unit occupies it.
func (a *Arena) CellOccupied(at HexCoord) (ok bool) {
	c := a.Board.Cells[at]
	if c == nil {
		return
	}

	return c.Unit != nil
}

// ValidateAbilityUse guarantees target coordinates match structural requirements and rules of active execution.
func (a *Arena) ValidateAbilityUse(caster *Unit, ab ability.Ability, targetAt *HexCoord) error {
	err := caster.ValidateAbilityUse(ab.ID)
	if err != nil {
		return err
	}

	if ab.IsPassive {
		return abilityErr(ab.ID, ErrAbilityIsPassive)
	}

	for t, sv := range caster.Statuses {
		h := statusHandlers[t]
		if h == nil || h.validateAbilityTarget == nil {
			continue
		}

		err := h.validateAbilityTarget(a, caster, ab, targetAt, sv)
		if err != nil {
			return err
		}
	}

	if !ab.IsPassive && caster.CurrentAP < 1 {
		player, _ := a.PlayerByID(caster.OwnerID)
		if player == nil || player.PhantomAP < 1 {
			return abilityErr(ab.ID, ErrNotEnoughAP)
		}
	}

	if ab.Activation != ability.Instant && targetAt == nil {
		return abilityErr(ab.ID, ErrTargetRequired)
	}

	if targetAt == nil || ab.Activation == ability.Instant {
		return nil
	}

	rangeN := ab.Range
	if ab.Area != "" {
		rangeN = ab.AreaRadius
	}

	if caster.Pos.Distance(*targetAt) > rangeN {
		return abilityErr(ab.ID, ErrTargetOutOfRange)
	}

	target := a.UnitAt(*targetAt)

	switch ab.TargetMode {
	case ability.TargetEnemies, ability.TargetEnemiesAndSelf:
		if target == nil {
			return abilityErr(ab.ID, ErrNoUnitAtTarget)
		}
		if target.OwnerID == caster.OwnerID && ab.TargetMode == ability.TargetEnemies {
			return abilityErr(ab.ID, ErrCannotTargetAlly)
		}

	case ability.TargetAllies, ability.TargetAlliesAndSelf:
		if target == nil {
			return abilityErr(ab.ID, ErrNoUnitAtTarget)
		}
		if target.OwnerID != caster.OwnerID {
			return abilityErr(ab.ID, ErrCannotTargetEnemy)
		}

	case ability.TargetSelf:
		if target == nil || target.ID != caster.ID {
			return abilityErr(ab.ID, ErrMustTargetSelf)
		}

	case ability.TargetAny:
		if target == nil {
			return abilityErr(ab.ID, ErrNoUnitAtTarget)
		}
	}

	return nil
}

// unitByID resolves and yields a matching internal unit object pointer using its unique structural identity key.
func (a *Arena) unitByID(unitID ds.ID) *Unit {
	for _, u := range a.UnitsQueue {
		if u.ID == unitID {
			return u
		}
	}

	return nil
}

// advanceActiveUnit shifts the turn ownership pointer onto the next sequential unit index inside the initiative list.
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

// relocateUnit updates spatial matrix references and executes corresponding trigger registrations for movement.
func (a *Arena) relocateUnit(u *Unit, to HexCoord) (sts ApplyStates) {
	a.Board.Cells[u.PosVal()].Unit = nil
	u.Pos = &to
	a.Board.Cells[to].Unit = u

	return triggers.UnitMoved(a, u)
}

// playerByID resolves and returns a player reference and their index position from a given unique ID.
func (a *Arena) playerByID(id ds.ID) (*Player, int) {
	for i, p := range a.Players {
		if p.ID == id {
			return p, i
		}
	}

	return nil, -1
}

func abilityErr(id ability.ID, err error) error {
	return fmt.Errorf("ability %s: %w", string(id), err)
}