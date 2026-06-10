package game

import (
	"log"

	"github.com/ognev-dev/goplease/game/ability/status"
)

type statusHandler struct {
	onApply              func(a *Arena, from, to *Unit, v status.Value) ApplyStates
	onRemove             func(a *Arena, u *Unit, v status.Value) ApplyStates
	onUnitAttacked       func(a *Arena, dmg *int, v status.Value)
	onTurnStart          func(a *Arena, u *Unit, v status.Value) ApplyStates
	onTurnEnd            func(a *Arena, u *Unit, v status.Value) ApplyStates
	onOtherStatusApplied func(a *Arena, from, to *Unit, applied *status.Value, v status.Value) ApplyStates
	mutate               func(a *Arena, v *status.Value, from, to *Unit)
}

var statusHandlers = map[status.Type]*statusHandler{
	status.Provoked:       provokedSH,
	status.Provoking:      nil, // this is just decorative status
	status.Stunned:        stunnedSH,
	status.Rallied:        ralliedSH,
	status.Exposed:        exposedSH,
	status.Hamstrung:      hamstrungSH,
	status.Sharpened:      sharpenedSH,
	status.DebuffWard:     debuffWardSH,
	status.TemporalAnchor: temporalAnchorSH,
	status.Frenzied:       frenziedSH,
}

var provokedSH = &statusHandler{
	mutate: func(_ *Arena, v *status.Value, from, _ *Unit) {
		if v.Meta == nil {
			v.Meta = map[string]any{}
		}
		v.Meta["provoker"] = from.ID
	},
}

var simpleAttackModifierSH = &statusHandler{
	onApply: func(_ *Arena, from, to *Unit, sv status.Value) (state ApplyStates) {
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

var ralliedSH = simpleAttackModifierSH
var sharpenedSH = simpleAttackModifierSH
var frenziedSH = simpleAttackModifierSH

var exposedSH = &statusHandler{
	onUnitAttacked: func(_ *Arena, dmg *int, st status.Value) {
		*dmg += st.Value
	},
}

var stunnedSH = &statusHandler{
	onTurnStart: func(_ *Arena, u *Unit, v status.Value) (state ApplyStates) {
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

var hamstrungSH = &statusHandler{
	onTurnStart: func(_ *Arena, u *Unit, st status.Value) (state ApplyStates) {
		u.CurrentMP = st.Value
		state.ToAll(ApplyState{
			SetMP:    new(st.Value),
			ToUnitID: u.ID,
		})
		return
	},
	onRemove: func(_ *Arena, u *Unit, v status.Value) (state ApplyStates) {
		u.CurrentMP = u.BaseMP
		state.ToAll(ApplyState{
			SetMP:    new(u.CurrentMP),
			ToUnitID: u.ID,
		})
		return
	},
}

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
			"pos":    u.Pos,
		}
		u.Statuses[sv.Status.Type] = current
		return
	},
	onTurnEnd: func(a *Arena, u *Unit, sv status.Value) (state ApplyStates) {
		if sv.Meta == nil {
			return
		}

		if u.CurrentAP != u.BaseAP {
			diff := u.BaseAP - u.CurrentAP
			u.CurrentAP = u.BaseAP
			state.ToAll(
				ApplyState{ChangeAP: new(diff), ToUnitID: u.ID},
				ApplyState{SetAP: new(u.CurrentAP), ToUnitID: u.ID},
			)
		}

		prevHP := sv.Meta["hp"].(int)
		prevShield := sv.Meta["shield"].(int)
		hpDiff := prevHP - u.CurrentHP
		shDiff := prevShield - u.CurrentShield
		prevPos := sv.Meta["pos"].(HexCoord)

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
				ApplyState{ChangeHP: new(shDiff), ToUnitID: u.ID},
				ApplyState{SetHP: new(u.CurrentShield), ToUnitID: u.ID},
			)
		}
		if prevPos != u.Pos {

			u.Pos = prevPos
			state.ToAll(
				ApplyState{MoveTo: new(prevPos), ToUnitID: u.ID},
			)
		}

		return
	},
}

var debuffWardSH = &statusHandler{
	onOtherStatusApplied: func(_ *Arena, from, to *Unit, applied *status.Value, v status.Value) (state ApplyStates) {
		if !applied.IsNegative() {
			return
		}

		applied.Duration = 0
		state.ToAll(ApplyState{ShowText: new("Debuff Ward!"), ToUnitID: to.ID})
		return
	},
}

func applyStatusToUnit(a *Arena, st status.Type, from, to *Unit) (state ApplyStates) {
	inst := status.ByType(st)
	if inst == nil {
		log.Printf("applyStatusToUnit: unknown status type %s", st)
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
