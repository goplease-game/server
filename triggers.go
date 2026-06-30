package game

import (
	"github.com/goplease-game/server/ability"
	"github.com/goplease-game/server/ability/status"
)

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

// triggers is the global runtime instance of the event dispatch registry.
var triggers *TriggerRegistry

// onDeathHandler executes logic reacting to a unit's death on the arena.
type onDeathHandler func(*Arena, *Unit) ApplyStates

// onMoveHandler executes logic reacting to a unit's repositioning.
type onMoveHandler func(*Arena, *Unit) ApplyStates

// onDamageReceivedHandler executes logic reacting to a target unit taking damage from a source unit.
type onDamageReceivedHandler func(a *Arena, source, target *Unit) ApplyStates

// onDamageDealtHandler executes logic reacting to a source unit inflicting damage upon a target unit.
type onDamageDealtHandler func(a *Arena, source, target *Unit) ApplyStates

// onTurnStartHandler executes logic reacting to the initiation of a unit's active turn.
type onTurnStartHandler func(*Arena, *Unit) ApplyStates

// TriggerRegistry manages groups of sequential event handlers hooked into core game-loop actions.
type TriggerRegistry struct {
	onDeath          []onDeathHandler
	onMove           []onMoveHandler
	onDamageReceived []onDamageReceivedHandler
	onDamageDealt    []onDamageDealtHandler
	onTurnStart      []onTurnStartHandler
}

// SomebodyJustExpectedlyDied dispatches the death event to all registered onDeath handlers.
func (r *TriggerRegistry) SomebodyJustExpectedlyDied(a *Arena, unfortunateOne *Unit) (state ApplyStates) {
	for _, takeCareOf := range r.onDeath {
		state.With(takeCareOf(a, unfortunateOne))
	}

	return
}

// UnitMoved dispatches the movement event to all registered onMove handlers.
func (r *TriggerRegistry) UnitMoved(a *Arena, u *Unit) (state ApplyStates) {
	for _, handler := range r.onMove {
		state.With(handler(a, u))
	}

	return
}

// DamageReceived dispatches the defensive damage event to all registered onDamageReceived handlers.
func (r *TriggerRegistry) DamageReceived(a *Arena, source, target *Unit) (st ApplyStates) {
	for _, handler := range r.onDamageReceived {
		st.With(handler(a, source, target))
	}

	return
}

// DamageDealt dispatches the offensive damage event to all registered onDamageDealt handlers.
func (r *TriggerRegistry) DamageDealt(a *Arena, from, to *Unit) (st ApplyStates) {
	for _, handler := range r.onDamageDealt {
		st.With(handler(a, from, to))
	}

	return
}

// TurnStarted resets round-specific variables and triggers all registered start-of-turn event handlers.
func (r *TriggerRegistry) TurnStarted(a *Arena, u *Unit) (state ApplyStates) {
	u.PhantomAPUsedThisTurn = 0
	u.CurrentMP = u.BaseMP

	state.ToSelf(ApplyState{SetMP: new(u.CurrentMP), ToUnitID: u.ID})

	for _, handler := range r.onTurnStart {
		state.With(handler(a, u))
	}

	return
}

// OnTurnStart acts as a public entrypoint to evaluate and route turn start events via the global trigger registry.
func OnTurnStart(a *Arena, u *Unit) (sts ApplyStates) {
	return triggers.TurnStarted(a, u)
}

// useUndyingWillAbility processes the passive cheat-death mechanic for tanks, restoring 1 HP and applying a shield.
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

// recalculateFrenzyAbility updates the Frenzy status condition on all units depending on the current proximity of enemies.
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

// useCoverFireAbility checks and fires ranger counter-attacks at enemies who dared attack their adjacent allies.
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

// useOpportunityAbility triggers a free melee follow-up strike for rogues when an ally hits an adjacent enemy.
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

// useFocusFieldAbility decreases non-passive ability cooldowns for friendly targets positioned next to the Mist area.
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

// useBottomlessVialAbility expands maximum HP parameters and heals an ally unit when they receive a hit.
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
			UnitID:    u.ID,
			AbilityID: id,
			Target:    target.Pos,
		}), ToUnitID: target.ID})
		st.ToAll(ApplyState{SetBaseHP: new(target.BaseHP), ToUnitID: target.ID})
		st.With(healUnit(target, ab.Effect.HealHP))

		return // apply only once
	}

	return
}

// handleImpatience checks round thresholds to forcefully inflict an Impatience debuff if the game stalls too long.
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
