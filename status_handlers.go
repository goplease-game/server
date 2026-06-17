package game

import (
	"errors"
	"fmt"
	"log"

	"github.com/goplease-game/server/ability"
	"github.com/goplease-game/server/ability/status"
	"github.com/goplease-game/server/ds"
)

var (
	// ErrUnitProvoked indicates that the unit is provoked and must target the provoking unit.
	ErrUnitProvoked = errors.New("unit is provoked and must target")
)

// statusHandler defines the optional hooks a status can implement to react
// to game events: being applied or removed, dealing or receiving damage,
// the start or end of a turn, or another status being applied. It can also
// mutate its own value at application time and restrict valid ability
// targets while active.
type statusHandler struct {
	onApply              func(a *Arena, from, to *Unit, v status.Value) ApplyStates
	onRemove             func(a *Arena, u *Unit, v status.Value) ApplyStates
	onDamageCalculated   func(a *Arena, dmg *int, sv status.Value)
	onDamageReceived     func(a *Arena, from, to *Unit, sv status.Value) ApplyStates
	onDamageDealt        func(a *Arena, from, to *Unit, v status.Value) ApplyStates
	onTurnStart          func(a *Arena, u *Unit, v status.Value) ApplyStates
	onTurnEnd            func(a *Arena, u *Unit, v status.Value) ApplyStates
	onOtherStatusApplied func(a *Arena, from, to *Unit, applied *status.Value, v status.Value) ApplyStates

	// status may mutate its value when being applied
	mutate func(a *Arena, v *status.Value, from, to *Unit)

	// validateAbilityTarget restricts which ability/target the unit may use
	// while this status is active. Returns an error if the choice is disallowed.
	validateAbilityTarget func(a *Arena, caster *Unit, ab ability.Ability, target *HexCoord, sv status.Value) error
}

// statusHandlers maps each status type to its behavior hooks. A nil entry
// (or a missing onX hook) means the status has no special behavior for that event.
var statusHandlers = map[status.Type]*statusHandler{
	status.Impatience:     impatienceSH,
	status.Provoked:       provokedSH,
	status.Provoking:      nil, // this is just decorative status
	status.Stunned:        stunnedSH,
	status.Rallied:        ralliedSH,
	status.Marked:         markedSH,
	status.Hamstrung:      hamstrungSH,
	status.Sharpened:      sharpenedSH,
	status.DebuffWard:     debuffWardSH,
	status.TemporalAnchor: temporalAnchorSH,
	status.Frenzied:       frenziedSH,
}

// provokedSH records which unit provoked this one and forces direct-damage
// abilities to target the provoker while it's alive.
var provokedSH = &statusHandler{
	mutate: func(_ *Arena, v *status.Value, from, _ *Unit) {
		if v.Meta == nil {
			v.Meta = map[string]any{}
		}
		v.Meta["provoker"] = from.ID
	},
	validateAbilityTarget: func(a *Arena, _ *Unit, ab ability.Ability, target *HexCoord, sv status.Value) error {
		if !ab.IsDirectDamage() {
			return nil
		}

		provokerID, ok := sv.Meta["provoker"].(ds.ID)
		if !ok {
			return nil
		}

		provoker := a.unitByID(provokerID)
		if provoker == nil || !provoker.Alive() {
			return nil
		}

		if target == nil || *target != provoker.PosVal() {
			return fmt.Errorf("%w %s", ErrUnitProvoked, provoker.Name)
		}

		return nil
	},
}

// simpleAttackModifierSH is a generic handler that adds its value to the
// unit's attack on apply and removes it on removal. Shared by statuses
// that are a flat, reversible attack bonus.
var simpleAttackModifierSH = &statusHandler{
	onApply: func(_ *Arena, _, to *Unit, sv status.Value) (state ApplyStates) {
		to.CurrentAtk += sv.Value
		state.ToAll(
			ApplyState{ChangeAtk: new(sv.Value), ToUnitID: to.ID},
			ApplyState{SetAtk: new(to.CurrentAtk), ToUnitID: to.ID},
		)

		return
	},
	onRemove: func(_ *Arena, u *Unit, sv status.Value) (state ApplyStates) {
		u.CurrentAtk -= sv.Value
		state.ToAll(
			ApplyState{ChangeAtk: new(-sv.Value), ToUnitID: u.ID},
			ApplyState{SetAtk: new(u.CurrentAtk), ToUnitID: u.ID},
		)

		return
	},
}

// sharpenedSH reuses simpleAttackModifierSH: a flat, reversible attack bonus.
var sharpenedSH = simpleAttackModifierSH

// frenziedSH reuses simpleAttackModifierSH: a flat, reversible attack bonus.
var frenziedSH = simpleAttackModifierSH

// markedSH increases incoming damage by its value, and grants the killing
// blow's attacker a permanent attack bonus equal to that value.
var markedSH = &statusHandler{
	onDamageCalculated: func(_ *Arena, dmg *int, sv status.Value) {
		*dmg += sv.Value
	},
	onDamageReceived: func(_ *Arena, from, to *Unit, sv status.Value) (state ApplyStates) {
		if !to.Alive() {
			from.CurrentAtk += sv.Value
			state.ToAll(
				ApplyState{ShowText: new("Hunter!"), ToUnitID: to.ID},
				ApplyState{ChangeAtk: new(sv.Value), ToUnitID: from.ID},
				ApplyState{SetAtk: new(from.CurrentAtk), ToUnitID: from.ID},
			)
		}

		return
	},
}

// ralliedSH grants a reversible attack bonus on apply, and removes itself
// (without re-triggering onRemove) once the unit deals damage, leaving the
// bonus in place permanently.
var ralliedSH = &statusHandler{
	onApply: func(_ *Arena, _, to *Unit, sv status.Value) (state ApplyStates) {
		to.CurrentAtk += sv.Value
		state.ToAll(
			ApplyState{ChangeAtk: new(sv.Value), ToUnitID: to.ID},
			ApplyState{SetAtk: new(to.CurrentAtk), ToUnitID: to.ID},
		)

		return
	},
	onRemove: func(_ *Arena, u *Unit, sv status.Value) (state ApplyStates) {
		u.CurrentAtk -= sv.Value
		state.ToAll(
			ApplyState{ChangeAtk: new(-sv.Value), ToUnitID: u.ID},
			ApplyState{SetAtk: new(u.CurrentAtk), ToUnitID: u.ID},
		)

		return
	},
	onDamageDealt: func(_ *Arena, from, _ *Unit, sv status.Value) (state ApplyStates) {
		// just silently remove status without triggering it's onRemove
		// so original bonus will stay
		from.RemoveStatus(status.Rallied)

		state.ToAll(
			ApplyState{RemoveStatus: new(sv.Status.Type), ToUnitID: from.ID},
		)
		return
	},
}

// stunnedSH makes the unit skip its next turn.
var stunnedSH = &statusHandler{
	onTurnStart: func(_ *Arena, u *Unit, _ status.Value) (state ApplyStates) {
		state.ToSelf(ApplyState{
			SkipTurn: true,
			ToUnitID: u.ID,
		})

		state.ToAll(ApplyState{
			ShowText: new("Stunned!"),
			ToUnitID: u.ID,
		})

		return
	},
}

// impatienceSH permanently increases the unit's attack at the start of
// each turn it's active.
var impatienceSH = &statusHandler{
	onTurnStart: func(_ *Arena, u *Unit, sv status.Value) (state ApplyStates) {
		u.CurrentAtk += sv.Value
		u.BaseAtk += sv.Value

		state.ToAll(
			ApplyState{ChangeAtk: new(sv.Value), ToUnitID: u.ID},
			ApplyState{SetAtk: new(u.CurrentAtk), ToUnitID: u.ID},
		)

		return
	},
}

// hamstrungSH zeroes the unit's move points while active and restores
// them to base on removal.
var hamstrungSH = &statusHandler{
	onTurnStart: func(_ *Arena, u *Unit, st status.Value) (state ApplyStates) {
		u.CurrentMP = 0
		state.ToAll(ApplyState{
			SetMP:    new(st.Value),
			ToUnitID: u.ID,
		})
		return
	},
	onRemove: func(_ *Arena, u *Unit, _ status.Value) (state ApplyStates) {
		u.CurrentMP = u.BaseMP
		state.ToAll(ApplyState{
			SetMP:    new(u.CurrentMP),
			ToUnitID: u.ID,
		})
		return
	},
}

// temporalAnchorSH grants bonus AP at the start of the turn, snapshots HP,
// shield, and position, then restores all three (plus the AP) at the end
// of the turn.
var temporalAnchorSH = &statusHandler{
	onTurnStart: func(_ *Arena, u *Unit, sv status.Value) (state ApplyStates) {
		u.CurrentAP += sv.Value
		state.ToAll(
			ApplyState{ChangeAP: new(sv.Value), ToUnitID: u.ID},
			ApplyState{SetAP: new(u.CurrentAP), ToUnitID: u.ID},
		)

		current := u.Statuses[sv.Status.Type]
		current.Meta = map[string]any{
			"hp":     u.CurrentHP,
			"shield": u.CurrentShield,
			"pos":    u.PosVal(),
		}
		u.Statuses[sv.Status.Type] = current
		return
	},
	onTurnEnd: func(a *Arena, u *Unit, sv status.Value) (state ApplyStates) {
		if sv.Meta == nil {
			return
		}
		if u.CurrentAP != u.BaseAP {
			u.CurrentAP = u.BaseAP
			state.ToAll(
				ApplyState{SetAP: new(u.CurrentAP), ToUnitID: u.ID},
			)
		}

		prevHP := sv.Meta["hp"].(int) //nolint:forcetypeassert
		prevShield := sv.Meta["shield"].(int) //nolint:forcetypeassert
		hpDiff := prevHP - u.CurrentHP
		shDiff := prevShield - u.CurrentShield
		prevPos := sv.Meta["pos"].(HexCoord) //nolint:forcetypeassert

		if hpDiff != 0 {
			u.CurrentHP = prevHP
			state.ToAll(
				ApplyState{ChangeHP: new(hpDiff), ToUnitID: u.ID},
				ApplyState{SetHP: new(u.CurrentHP), ToUnitID: u.ID},
			)
		}
		if shDiff != 0 {
			u.CurrentShield = prevShield
			state.ToAll(
				ApplyState{ChangeShield: new(shDiff), ToUnitID: u.ID},
				ApplyState{SetShield: new(u.CurrentShield), ToUnitID: u.ID},
			)
		}
		if prevPos != u.PosVal() {
			a.Board.Cells[u.PosVal()].Unit = nil
			u.Pos = &prevPos
			a.Board.Cells[prevPos].Unit = u
			state.ToAll(ApplyState{MoveTo: new(prevPos), ToUnitID: u.ID})
		}

		return
	},
}

// debuffWardSH cancels any negative status applied to the unit while it's
// active, by zeroing the incoming status's duration before it's stored.
var debuffWardSH = &statusHandler{
	onOtherStatusApplied: func(_ *Arena, _, to *Unit, applied *status.Value, _ status.Value) (state ApplyStates) {
		if !applied.IsNegative() {
			return
		}

		applied.Duration = 0
		state.ToAll(ApplyState{ShowText: new("Debuff Ward!"), ToUnitID: to.ID})
		return
	},
}

// ApplyStatusToUnit applies status st to unit to (caused by from), giving
// to's existing statuses (e.g. Debuff Ward) a chance to intercept it via
// onOtherStatusApplied, refreshing the duration if the status is already
// active, and triggering onApply only on the status's first application.
func ApplyStatusToUnit(a *Arena, st status.Type, from, to *Unit) (state ApplyStates) {
	inst := status.ByType(st)
	if inst == nil {
		log.Printf("ApplyStatusToUnit: unknown status type %s", st)
		return
	}

	sv := status.Value{
		UnitID:   to.ID,
		Duration: inst.Duration,
		Value:    inst.InitialValue,
		Status:   inst,
	}

	statusH := statusHandlers[st]
	if statusH != nil && statusH.mutate != nil {
		statusH.mutate(a, &sv, from, to)
	}

	for t, v := range to.Statuses {
		if t == st {
			continue
		}
		h := statusHandlers[t]
		if h == nil || h.onOtherStatusApplied == nil {
			continue
		}
		state.With(h.onOtherStatusApplied(a, from, to, &sv, v))
		if sv.Duration == 0 {
			return
		}
	}

	// If status already exists — just refresh duration, do not call onApply again.
	_, alreadyActive := to.Statuses[st]

	to.AddStatus(sv)
	state.ToAll(ApplyState{
		AddStatus:     new(st),
		AddStatusMeta: sv.Meta,
		ToUnitID:      to.ID,
	})

	if !alreadyActive && statusH != nil && statusH.onApply != nil {
		state.With(statusH.onApply(a, from, to, sv))
	}

	return state
}

// removeStatusFromUnit removes status st from unit u and runs its
// onRemove hook, if any.
func removeStatusFromUnit(a *Arena, st status.Type, u *Unit) (state ApplyStates) {
	sv, ok := u.Statuses[st]
	if !ok {
		log.Printf("removeStatusFromUnit: unit missing status: %s", st)
		return
	}

	u.RemoveStatus(st)

	h := statusHandlers[st]
	if h != nil && h.onRemove != nil {
		state.With(h.onRemove(a, u, sv))
	}

	state.ToAll(ApplyState{
		RemoveStatus: new(st),
		ToUnitID:     u.ID,
	})

	return
}

// handleOnTurnStartStatuses runs the onTurnStart hook for each of unit's
// active statuses.
func handleOnTurnStartStatuses(a *Arena, unit *Unit) (state ApplyStates) {
	for t, v := range unit.Statuses {
		h, ok := statusHandlers[t]
		if !ok || h == nil {
			continue
		}

		if h.onTurnStart == nil {
			continue
		}

		state.With(h.onTurnStart(a, unit, v))
	}

	return
}

// handleOnDamageDealtStatuses runs the onDamageDealt hook for each of
// from's active statuses after from deals damage to to.
func handleOnDamageDealtStatuses(a *Arena, from, to *Unit) (state ApplyStates) {
	for t, v := range from.Statuses {
		h, ok := statusHandlers[t]
		if !ok || h == nil {
			continue
		}

		if h.onDamageDealt == nil {
			continue
		}

		state.With(h.onDamageDealt(a, from, to, v))
	}

	return
}

// handleOnDamageReceivedStatuses runs the onDamageReceived hook for each
// of to's active statuses after to takes damage from from.
func handleOnDamageReceivedStatuses(a *Arena, from, to *Unit) (state ApplyStates) {
	for t, v := range to.Statuses {
		h, ok := statusHandlers[t]
		if !ok || h == nil {
			continue
		}

		if h.onDamageReceived == nil {
			continue
		}

		state.With(h.onDamageReceived(a, from, to, v))
	}

	return
}

// triggerStatusOnDamageCalculated runs the onDamageCalculated hook for
// each of u's active statuses, letting them adjust dmg before it's applied.
func triggerStatusOnDamageCalculated(a *Arena, u *Unit, dmg *int) {
	for t, v := range u.Statuses {
		h, ok := statusHandlers[t]
		if !ok || h == nil {
			continue
		}

		if h.onDamageCalculated == nil {
			continue
		}

		h.onDamageCalculated(a, dmg, v)
	}
}