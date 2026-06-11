package game

import (
	"github.com/ognev-dev/goplease/game/ability"
	"github.com/ognev-dev/goplease/game/ability/status"
)

// Validation is handled by Arena.ValidateAbilityUse before dispatch,
// so handlers may assume all preconditions are satisfied.

var abilityHandlers = map[ability.ID]func(a *Arena, e abilityUsedEvent) (ApplyStates, error){
	ability.BasicMeleeAttack: basicMeleeAttackHandler,
	ability.BasicRangeAttack: basicRangeAttackHandler,
	ability.BasicMagicAttack: basicMagicAttackHandler,

	ability.Fortify:    fortifyHandler,
	ability.Provoke:    provokeHandler,
	ability.ShieldBash: shieldBashHandler,

	ability.BattleCry:   battleCryHandler,
	ability.IdolihuSpin: idolihuSpinHandler,
	ability.PowerPush:   powerPushHandler,

	ability.PiercingShot:  piercingShotHandler,
	ability.HuntersMark:   huntersMarkHandler,
	ability.HamstringShot: hamstringShotHandler,

	ability.ShadowStep: shadowStepHandler,
	ability.GangUp:     gangUpHandler,
	ability.Eliminate:  eliminateHandler,

	ability.Translocation: translocationHandler,
	ability.TimeWarp:      timeWarpHandler,
	ability.Purge:         purgeHandler,

	ability.Heal:     healHandler,
	ability.Equalize: equalizeHandler,
	ability.Purify:   purifyHandler,
}

type abilityUsedEvent struct {
	By *Unit
	Ab ability.Ability
	At HexCoord
}

func basicAttackHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)
	state = a.DealDamageToUnit(e.By, target, e.By.CurrentAtk)
	return
}

func basicMeleeAttackHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	return basicAttackHandler(a, e)
}

func basicRangeAttackHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	return basicAttackHandler(a, e)
}

func basicMagicAttackHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	return basicAttackHandler(a, e)
}

func fortifyHandler(a *Arena, e abilityUsedEvent) (sts ApplyStates, err error) {
	units := a.AlliesInRange(e.By, e.Ab.AreaRadius)
	val := e.Ab.Effect.AddShield
	for _, u := range units {
		u.CurrentShield += val
		sts.ToAll(
			ApplyState{ChangeShield: new(val), ToUnitID: u.ID},
			ApplyState{SetShield: new(u.CurrentShield), ToUnitID: u.ID},
		)
	}

	return sts, nil
}

// TODO test this for both sides + client behaviour
func provokeHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	state.With(applyStatusToUnit(a, status.Provoking, e.By, e.By))

	units := a.EnemiesInRange(e.By, e.Ab.AreaRadius)
	for _, u := range units {
		state.With(applyStatusToUnit(a, status.Provoked, e.By, u))
	}

	return state, nil
}

func shieldBashHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)

	state.With(applyStatusToUnit(a, e.Ab.Effect.ApplyStatus, e.By, target))
	return
}

func powerPushHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)

	dealDmg := e.Ab.Effect.DealDamage
	pos := e.By.PosVal().Opposite(target.PosVal())

	if a.UnitAt(pos) == nil {
		state.With(a.relocateUnit(target, pos))
		state.ToAll(ApplyState{MoveTo: new(pos), ToUnitID: target.ID})
		target.Pos = &pos
	} else {
		dealDmg = e.Ab.Effect.DealAltDamage
	}

	state.With(a.DealDamageToUnit(e.By, target, dealDmg))
	return state, nil
}

func gangUpHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)

	dealDmg := e.By.CurrentAtk
	pos := e.By.PosVal().Opposite(target.PosVal())
	u := a.UnitAt(pos)
	if u != nil && u.IsAlly(e.By) {
		dealDmg += e.Ab.Effect.BonusDamage
	}

	state.With(a.DealDamageToUnit(e.By, target, dealDmg))
	return
}

func eliminateHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)

	state.With(a.DealDamageToUnit(e.By, target, e.Ab.Effect.DealDamage))
	if target.IsDead {
		ap := e.Ab.Effect.AddAP
		e.By.CurrentAP += ap
		state.ToAll(
			ApplyState{ChangeAP: new(ap), ToUnitID: e.By.ID},
			ApplyState{SetAP: new(ap), ToUnitID: e.By.ID},
		)
	}

	return
}

func translocationHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)

	from := e.By.PosVal()
	to := target.PosVal()

	a.relocateUnit(e.By, to)
	a.relocateUnit(target, from)

	state.ToAll(
		ApplyState{MoveTo: new(to), ToUnitID: e.By.ID},
		ApplyState{MoveTo: new(from), ToUnitID: target.ID},
	)

	return
}

func timeWarpHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)

	state.With(applyStatusToUnit(a, e.Ab.Effect.ApplyStatus, e.By, target))
	return
}

func purgeHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)

	for st, v := range target.Statuses {
		if v.IsPositive() {
			state.With(removeStatusFromUnit(a, st, target))
		}
	}

	return
}

func purifyHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)

	for st, v := range target.Statuses {
		if v.IsNegative() {
			state.With(removeStatusFromUnit(a, st, target))
		}
	}

	state.With(healUnit(target, e.Ab.Effect.HealHP))
	state.With(applyStatusToUnit(a, e.Ab.Effect.ApplyStatus, e.By, target))

	return
}

func healHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)

	return healUnit(target, e.Ab.Effect.HealHP), nil
}

func equalizeHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	var sumHP int
	units := a.AlliesInRange(e.By, e.Ab.AreaRadius)
	for _, u := range units {
		sumHP += u.CurrentHP
	}

	count := len(units)
	if count <= 1 {
		return
	}

	eq := sumHP / count
	remainder := sumHP - eq*count

	for _, u := range units {
		// Clamp target HP to unit's max to prevent overheal.
		target := eq
		if target > u.BaseHP {
			target = u.BaseHP
		}

		if u.CurrentHP == target {
			continue
		}

		changeBy := target - u.CurrentHP
		u.CurrentHP = target

		state.ToAll(
			ApplyState{ChangeHP: new(changeBy), ToUnitID: u.ID},
			ApplyState{SetHP: new(u.CurrentHP), ToUnitID: u.ID},
		)
	}

	if remainder > 0 {
		for i := 0; i < remainder; i++ {
			u := units[i%count]
			if u.CurrentHP >= u.BaseHP {
				continue
			}
			u.CurrentHP++

			for j, v := range state.Global {
				if v.ToUnitID != u.ID {
					continue
				}

				if v.SetHP != nil {
					v.SetHP = new(u.CurrentHP)
				}
				if v.ChangeHP != nil {
					*v.ChangeHP += 1
				}

				state.Global[j] = v
			}
		}
	}

	return
}

func idolihuSpinHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	units := a.EnemiesInRange(e.By, e.Ab.AreaRadius)
	for _, u := range units {
		state.With(a.DealDamageToUnit(e.By, u, e.By.CurrentAtk))
	}

	return
}

func piercingShotHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	cells := a.Board.Cells.Line(e.By.PosVal(), e.At, e.Ab.AreaRadius)
	for _, c := range cells {
		unit := a.UnitAt(c.Coord)
		if unit != nil && unit.IsEnemy(e.By) {
			state.With(a.DealDamageToUnit(e.By, unit, e.Ab.Effect.DealDamage))
		}
	}

	return
}

func battleCryHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	units := a.AlliesInRange(e.By, e.Ab.AreaRadius)
	for _, u := range units {
		state.With(applyStatusToUnit(a, e.Ab.Effect.ApplyStatus, e.By, u))
	}

	return
}

func shadowStepHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	a.relocateUnit(e.By, e.At)

	state.ToOpp(ApplyState{MoveTo: new(e.At), ToUnitID: e.By.ID})
	state.With(applyStatusToUnit(a, e.Ab.Effect.ApplyStatus, e.By, e.By))

	return
}

func huntersMarkHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)

	return applyStatusToUnit(a, e.Ab.Effect.ApplyStatus, e.By, target), nil
}

func hamstringShotHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)

	state.With(a.DealDamageToUnit(e.By, target, e.Ab.Effect.DealDamage))
	state.With(applyStatusToUnit(a, e.Ab.Effect.ApplyStatus, e.By, target))

	return
}

func healUnit(u *Unit, val int) (state ApplyStates) {
	if u.CurrentHP == u.BaseHP {
		return
	}

	u.CurrentHP += val
	if u.CurrentHP > u.BaseHP {
		val = val - (u.CurrentHP - u.BaseHP)
		u.CurrentHP = u.BaseHP
	}

	if val == 0 {
		return
	}

	state.ToAll(
		ApplyState{ChangeHP: new(val), ToUnitID: u.ID},
		ApplyState{SetHP: new(u.CurrentHP), ToUnitID: u.ID},
	)

	return
}
