package game

import (
	"github.com/ognev-dev/goplease/game/ability"
	"github.com/ognev-dev/goplease/game/ability/status"
)

// because of initialization cycle
func init() {
	triggers = &TriggerRegistry{
		onDeath: []onDeathHandler{
			useUndyingWillAbility,
			recalculateFrenzyAbility,
		},
		onMove: []onMoveHandler{
			recalculateFrenzyAbility,
		},
		onDamageReceived: []onDamageReceivedHandler{
			useCoverFireAbility,
			useOpportunityAbility,
			useBottomlessVialAbility,
			handleOnDamageReceivedStatuses,
		},
		onDamageDealt: []onDamageDealtHandler{
			handleOnDamageDealtStatuses,
		},
		onTurnStart: []onTurnStartHandler{
			useFocusFieldAbility,
			recalculateFrenzyAbility,
			handleOnTurnStartStatuses,
			handleImpatience,
		},
	}
}

var triggers *TriggerRegistry

// onDeathHandler defines a function signature for triggers that execute when a unit dies.
type onDeathHandler func(*Arena, *Unit) ApplyStates

// onMoveHandler defines a function signature for triggers that execute when a unit moves.
type onMoveHandler func(*Arena, *Unit) ApplyStates

// onDamageReceivedHandler defines a function signature for triggers that execute when a unit takes damage.
type onDamageReceivedHandler func(a *Arena, source, target *Unit) ApplyStates

// onDamageDealtHandler defines a function signature for triggers that execute when a unit deal damage.
type onDamageDealtHandler func(a *Arena, source, target *Unit) ApplyStates

// onTurnStartHandler defines a function signature for triggers that execute at the beginning of a unit's turn.
type onTurnStartHandler func(*Arena, *Unit) ApplyStates

// TriggerRegistry manages and routes game event triggers to their respective handlers.
type TriggerRegistry struct {
	onDeath          []onDeathHandler
	onMove           []onMoveHandler
	onDamageReceived []onDamageReceivedHandler
	onDamageDealt    []onDamageDealtHandler
	onTurnStart      []onTurnStartHandler
}

// SomebodyJustExpectedlyDied executes all registered handlers for a unit's death event.
func (r *TriggerRegistry) SomebodyJustExpectedlyDied(a *Arena, unfortunateOne *Unit) (state ApplyStates) {
	for _, takeCareOf := range r.onDeath {
		state.With(takeCareOf(a, unfortunateOne))
	}

	return
}

// UnitMoved executes all registered handlers for a unit's movement event.
func (r *TriggerRegistry) UnitMoved(a *Arena, u *Unit) (state ApplyStates) {
	for _, handler := range r.onMove {
		state.With(handler(a, u))
	}

	return
}

// DamageReceived executes all registered handlers for an event where a target unit takes damage from a source.
func (r *TriggerRegistry) DamageReceived(a *Arena, source, target *Unit) (st ApplyStates) {
	for _, handler := range r.onDamageReceived {
		st.With(handler(a, source, target))
	}

	return
}

// DamageDealt executes all registered handlers for an event where a target unit deals damage.
func (r *TriggerRegistry) DamageDealt(a *Arena, from, to *Unit) (st ApplyStates) {
	for _, handler := range r.onDamageDealt {
		st.With(handler(a, from, to))
	}

	return
}

// TurnStarted executes all registered handlers when a unit's turn begins and resets turn-specific variables.
func (r *TriggerRegistry) TurnStarted(a *Arena, u *Unit) (state ApplyStates) {
	u.PhantomAPUsedThisTurn = 0

	for _, handler := range r.onTurnStart {
		state.With(handler(a, u))
	}

	return
}

func OnTurnStart(a *Arena, u *Unit) (sts ApplyStates) {
	return triggers.TurnStarted(a, u)
}

func useUndyingWillAbility(_ *Arena, u *Unit) (state ApplyStates) {
	id := ability.UndyingWill
	ab := ability.ByID(id)
	if !u.HasAbility(id) {
		return
	}

	if !u.AbilityReady(id) {
		return
	}

	u.CurrentHP = ab.Effect.HealHP
	u.CurrentShield = ab.Effect.AddShield
	u.IsDead = false

	u.SetCooldown(id, ab.Cooldown)

	state.ToAll(
		ApplyState{UseAbility: new(UseAbilityPayload{UnitID: u.ID, AbilityID: id}), ToUnitID: u.ID},
		ApplyState{ChangeHP: new(u.CurrentHP), ToUnitID: u.ID},
		ApplyState{SetHP: new(u.CurrentHP), ToUnitID: u.ID},
		ApplyState{ChangeShield: new(u.CurrentShield), ToUnitID: u.ID},
		ApplyState{SetShield: new(u.CurrentShield), ToUnitID: u.ID},
	)

	return
}

func recalculateFrenzyAbility(a *Arena, _ *Unit) (state ApplyStates) {
	id := ability.Frenzy
	ab := ability.ByID(id)

	for _, u := range a.UnitsQueue {
		if !u.HasAbility(id) {
			continue
		}

		enemies := a.CountEnemiesInRange(u, ab.AreaRadius, 2)
		isFrenzied := u.HasStatus(ab.Effect.ApplyStatus)

		// Remove
		if enemies < 2 && isFrenzied {
			state.With(removeStatusFromUnit(a, ab.Effect.ApplyStatus, u))
			continue
		}

		// Add
		if enemies >= 2 && !isFrenzied {
			state.With(
				ApplyStatusToUnit(a, ab.Effect.ApplyStatus, u, u),
			)
		}
	}

	return
}

func useCoverFireAbility(a *Arena, source, target *Unit) (st ApplyStates) {
	if source.IsAlly(target) {
		return
	}

	id := ability.CoverFire
	ab := ability.ByID(id)

	unitsWithCoverFire := a.EnemiesInRangeHavingAbility(source, ab.Range, id)
	for _, u := range unitsWithCoverFire {
		if !u.AbilityReady(id) {
			continue
		}

		if target.ID == u.ID {
			continue // cannot apply CF from self
		}

		u.SetCooldown(id, ab.Cooldown)
		st.ToAll(ApplyState{UseAbility: new(UseAbilityPayload{
			UnitID:    u.ID,
			AbilityID: id,
			Target:    source.Pos,
		}), ToUnitID: u.ID})

		st.With(a.DealDamageToUnit(u, source, ab.Effect.DealDamage))
	}

	return
}

func useOpportunityAbility(a *Arena, source, target *Unit) (state ApplyStates) {
	if source.PosVal().Distance(target.PosVal()) > 1 { // only melee attacks
		return
	}

	id := ability.Opportunity
	ab := ability.ByID(id)

	cells := a.Board.Cells.InRangeHavingUnitAbility(target.PosVal(), ab.Range, id)
	for _, c := range cells {
		u := c.Unit
		if !u.Alive() || u.IsEnemy(source) {
			continue
		}
		if u.ID == source.ID { // cannot have opportunity for your own attack
			continue
		}
		if !u.AbilityReady(id) {
			continue
		}

		u.SetCooldown(id, ab.Cooldown)
		state.ToAll(ApplyState{UseAbility: new(UseAbilityPayload{
			UnitID:    u.ID,
			AbilityID: id,
			Target:    target.Pos,
		}), ToUnitID: u.ID})
		state.With(a.DealDamageToUnit(u, target, u.CurrentAtk))
	}

	return
}

func useFocusFieldAbility(a *Arena, unit *Unit) (st ApplyStates) {
	id := ability.FocusField
	ab := ability.ByID(id)

	unitsWithFocusField := a.AlliesInRangeHavingAbility(unit, ab.Range, id)
	for _, u := range unitsWithFocusField {
		if u.ID == unit.ID { // cannot have FF to yourself
			continue
		}

		var abUsed bool
		for abID, cd := range unit.Cooldowns {
			if ability.ByID(abID).IsPassive {
				continue
			}

			if cd > 0 {
				cd--
				unit.SetCooldown(abID, cd)
				abUsed = true
				st.ToSelf(ApplyState{SetCooldown: new(map[ability.ID]int{abID: cd}), ToUnitID: unit.ID})
			}
		}

		if abUsed {
			st.ToAll(ApplyState{UseAbility: new(UseAbilityPayload{
				UnitID:    u.ID,
				AbilityID: id,
				Target:    unit.Pos,
			}), ToUnitID: unit.ID})
		}

		return // trigger only once
	}

	return
}

// TODO apply status to display how much max HP increased
func useBottomlessVialAbility(a *Arena, _, target *Unit) (st ApplyStates) {
	id := ability.BottomlessVial
	ab := ability.ByID(id)

	units := a.AlliesInRangeHavingAbility(target, ab.AreaRadius, id)
	for _, u := range units {
		if !u.AbilityReady(id) {
			continue
		}

		if u.ID == target.ID {
			continue // cannot trigger on the unit that has the ability
		}

		u.SetCooldown(id, ab.Cooldown)
		target.BaseHP += ab.Effect.AddHP

		st.ToAll(ApplyState{UseAbility: new(UseAbilityPayload{
			UnitID:    target.ID,
			AbilityID: id,
			Target:    target.Pos,
		}), ToUnitID: target.ID})
		st.ToAll(ApplyState{SetBaseHP: new(target.BaseHP), ToUnitID: target.ID})
		st.With(healUnit(target, ab.Effect.HealHP))

		return // apply only once
	}

	return
}

func handleImpatience(a *Arena, unit *Unit) (state ApplyStates) {
	if a.CurrentRound < ApplyImpatienceStatusAfterRound {
		return
	}

	st := status.Impatience
	if !unit.HasStatus(st) {
		return ApplyStatusToUnit(a, st, unit, unit)
	}

	return
}
