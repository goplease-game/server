package game

import (
	"github.com/ognev-dev/goplease/game/ability"
)

// because of initialization cycle
func init() {
	triggers = &TriggerRegistry{
		onDeath: []onDeathHandler{
			useUndyingWillAbility,
		},
		onMove: []onMoveHandler{
			recalculateFrenzyAbility,
		},
		onDamageReceived: []onDamageReceivedHandler{
			useCoverFireAbility,
			useOpportunityAbility,
			useBottomlessVialAbility,
		},
		onTurnStart: []onTurnStartHandler{
			useFocusFieldAbility,
			recalculateFrenzyAbility,
			handleOnTurnStartStatuses,
		},
	}
}

var triggers *TriggerRegistry

// onDeathHandler defines a function signature for triggers that execute when a unit dies.
type onDeathHandler func(u *Unit) ApplyStates

// onMoveHandler defines a function signature for triggers that execute when a unit moves.
type onMoveHandler func(a *Arena, u *Unit) ApplyStates

// onDamageReceivedHandler defines a function signature for triggers that execute when a unit takes damage.
type onDamageReceivedHandler func(a *Arena, source, target *Unit) ApplyStates

// onTurnStartHandler defines a function signature for triggers that execute at the beginning of a unit's turn.
type onTurnStartHandler func(a *Arena, u *Unit) ApplyStates

// TriggerRegistry manages and routes game event triggers to their respective handlers.
type TriggerRegistry struct {
	onDeath          []onDeathHandler
	onMove           []onMoveHandler
	onDamageReceived []onDamageReceivedHandler
	onTurnStart      []onTurnStartHandler
}

// SomebodyJustExpectedlyDied executes all registered handlers for a unit's death event.
func (r *TriggerRegistry) SomebodyJustExpectedlyDied(unfortunateOne *Unit) (state ApplyStates) {
	for _, takeCareOf := range r.onDeath {
		state.With(takeCareOf(unfortunateOne))
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

func useUndyingWillAbility(u *Unit) (st ApplyStates) {
	id := ability.UndyingWill
	ab := ability.ByID(id)
	if !u.HasAbility(id) {
		return
	}

	if u.Cooldowns[id] > 0 {
		return
	}

	u.CurrentHP = ab.Effect.HealHP
	u.CurrentShield = ab.Effect.AddShield
	u.IsDead = false

	u.SetCooldown(id, ab.Cooldown)

	st.ToAll(
		ApplyState{UseAbility: new(UseAbilityPayload{UnitID: u.ID, AbilityID: id}), ToUnitID: u.ID},
		ApplyState{ChangeHP: new(u.CurrentHP), ToUnitID: u.ID},
		ApplyState{SetHP: new(u.CurrentHP), ToUnitID: u.ID},
		ApplyState{ChangeShield: new(u.CurrentShield), ToUnitID: u.ID},
		ApplyState{SetShield: new(u.CurrentShield), ToUnitID: u.ID},
	)

	return
}

func recalculateFrenzyAbility(a *Arena, _ *Unit) (sts ApplyStates) {
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
			sts.With(removeStatusFromUnit(ab.Effect.ApplyStatus, u))
			continue
		}

		// Add
		if enemies >= 2 && !isFrenzied {
			sts.With(
				applyStatusToUnit(ab.Effect.ApplyStatus, u, u),
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
			Target:    new(source.Pos),
		}), ToUnitID: u.ID})

		st.With(a.DealDamageToUnit(u, source, ab.Effect.DealDamage))
	}

	return
}

func useOpportunityAbility(a *Arena, source, target *Unit) (st ApplyStates) {
	if source.Pos.Distance(target.Pos) > 1 { // only melee attacks
		return
	}

	id := ability.Opportunity
	ab := ability.ByID(id)

	unitsWithOpportunity := a.EnemiesInRangeHavingAbility(source, ab.Range, id)
	for _, u := range unitsWithOpportunity {
		if u.ID == source.ID { // cannot have opportunity for your own attack
			continue
		}
		if !u.AbilityReady(id) {
			continue
		}

		u.SetCooldown(id, ab.Cooldown)
		st.ToAll(ApplyState{UseAbility: new(UseAbilityPayload{
			UnitID:    u.ID,
			AbilityID: id,
			Target:    new(target.Pos),
		}), ToUnitID: u.ID})
		st.With(a.DealDamageToUnit(u, target, u.CurrentAtk))
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
				Target:    new(unit.Pos),
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
			continue // cannot use on self
		}

		u.SetCooldown(id, ab.Cooldown)
		target.BaseHP += ab.Effect.AddHP

		st.ToAll(ApplyState{UseAbility: new(UseAbilityPayload{
			UnitID:    target.ID,
			AbilityID: id,
			Target:    new(target.Pos),
		}), ToUnitID: target.ID})
		st.ToAll(ApplyState{SetBaseHP: new(target.BaseHP), ToUnitID: target.ID})
		st.With(healUnit(target, ab.Effect.HealHP))

		return // apply only once
	}

	return
}
