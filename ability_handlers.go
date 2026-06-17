// Package game ...
package game

import (
	"github.com/goplease-game/server/ability"
	"github.com/goplease-game/server/ability/status"
)

// abilityHandlers maps unique ability identifiers to their corresponding execution logic functions.
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

// abilityUsedEvent encapsulates the context of an ability execution, including the caster, ability data, and target.
type abilityUsedEvent struct {
	By *Unit
	Ab ability.Ability
	At HexCoord
}

// basicAttackHandler processes standard direct single-target attacks using the caster's default attack attribute.
func basicAttackHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)
	state = a.DealDamageToUnit(e.By, target, e.By.CurrentAtk)
	return
}

// basicMeleeAttackHandler routes execution to the shared default weapon attack handler logic.
func basicMeleeAttackHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	return basicAttackHandler(a, e)
}

// basicRangeAttackHandler routes execution to the shared default weapon attack handler logic.
func basicRangeAttackHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	return basicAttackHandler(a, e)
}

// basicMagicAttackHandler routes execution to the shared default weapon attack handler logic.
func basicMagicAttackHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	return basicAttackHandler(a, e)
}

// fortifyHandler grants defensive shield values to all friendly units within the skill's effective radius.
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

// provokeHandler marks the caster as provoking and inflicts the provoked status condition on nearby enemies.
func provokeHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	state.With(ApplyStatusToUnit(a, status.Provoking, e.By, e.By))

	units := a.EnemiesInRange(e.By, e.Ab.AreaRadius)
	for _, u := range units {
		state.With(ApplyStatusToUnit(a, status.Provoked, e.By, u))
	}

	return state, nil
}

// shieldBashHandler delivers a localized strike that inflicts an associated status modifier upon the target.
func shieldBashHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)

	state.With(ApplyStatusToUnit(a, e.Ab.Effect.ApplyStatus, e.By, target))
	return
}

// powerPushHandler knocks a target unit backward into an empty cell or applies a status debuff if the path is blocked.
func powerPushHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)

	dealDmg := e.Ab.Effect.DealDamage
	pos := e.By.PosVal().Opposite(target.PosVal())

	cell, exists := a.Board.Cells[pos]
	if exists && cell.Unit == nil {
		state.With(a.relocateUnit(target, pos))
		state.ToAll(ApplyState{MoveTo: new(pos), ToUnitID: target.ID})
	} else {
		dealDmg = e.Ab.Effect.DealAltDamage
		state.With(ApplyStatusToUnit(a, e.Ab.Effect.ApplyStatus, e.By, target))
	}

	state.With(a.DealDamageToUnit(e.By, target, dealDmg))
	return state, nil
}

// gangUpHandler strikes a target unit, amplifying damage calculations if an ally occupies the space directly opposite.
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

// eliminateHandler performs an offensive strike, refunding action points to the caster if the blow eliminates the target.
func eliminateHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)

	state.With(a.DealDamageToUnit(e.By, target, e.Ab.Effect.DealDamage))
	if target.IsDead {
		ap := e.Ab.Effect.AddAP
		e.By.CurrentAP += ap
		state.ToAll(
			ApplyState{ChangeAP: new(ap), ToUnitID: e.By.ID},
			ApplyState{SetAP: new(e.By.CurrentAP), ToUnitID: e.By.ID},
		)
	}

	return
}

// translocationHandler swaps the physical grid positions of the casting unit and its selected target unit.
func translocationHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)

	from := e.By.PosVal()
	to := target.PosVal()

	// Clear both cells first before re-assigning units.
	a.Board.Cells[from].Unit = nil
	a.Board.Cells[to].Unit = nil

	e.By.Pos = &to
	a.Board.Cells[to].Unit = e.By

	target.Pos = &from
	a.Board.Cells[from].Unit = target

	state.ToOpp(
		ApplyState{MoveTo: new(to), ToUnitID: e.By.ID},
		ApplyState{MoveTo: new(from), ToUnitID: target.ID},
	)

	return
}

// timeWarpHandler distorts timelines to place an operational status modifier on the targeted unit.
func timeWarpHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)

	state.With(ApplyStatusToUnit(a, e.Ab.Effect.ApplyStatus, e.By, target))
	return
}

// purgeHandler filters, cleanses, and strips away all positive status effects currently active on an enemy target.
func purgeHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)

	for st, v := range target.Statuses {
		if v.IsPositive() {
			state.With(removeStatusFromUnit(a, st, target))
		}
	}

	return
}

// purifyHandler dispels negative debuffs from a target, triggers healing, and applies a positive protection status.
func purifyHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)

	for st, v := range target.Statuses {
		if v.IsNegative() {
			state.With(removeStatusFromUnit(a, st, target))
		}
	}

	state.With(healUnit(target, e.Ab.Effect.HealHP))
	state.With(ApplyStatusToUnit(a, e.Ab.Effect.ApplyStatus, e.By, target))

	return
}

// healHandler provides a targeted single-unit health restoration effect based on ability attributes.
func healHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)

	return healUnit(target, e.Ab.Effect.HealHP), nil
}

// equalizeHandler sums the current health of nearby allies and redistributes it evenly across them.
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
		target := min(eq, u.BaseHP)
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

	// Distribute residual remainders sequentially to non-maxed units.
	if remainder > 0 {
		for i := range remainder {
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
					*v.ChangeHP++
				}

				state.Global[j] = v
			}
		}
	}

	return
}

// idolihuSpinHandler sweeps the adjacent area, dealing direct damage to all caught enemy targets.
func idolihuSpinHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	units := a.EnemiesInRange(e.By, e.Ab.AreaRadius)
	for _, u := range units {
		state.With(a.DealDamageToUnit(e.By, u, e.By.CurrentAtk))
	}

	return
}

// piercingShotHandler processes an axial line-based projectile attack that damages all enemy units along its path.
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

// battleCryHandler motivates surrounding friendly units, granting them a supportive status condition.
func battleCryHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	units := a.AlliesInRange(e.By, e.Ab.AreaRadius)
	for _, u := range units {
		state.With(ApplyStatusToUnit(a, e.Ab.Effect.ApplyStatus, e.By, u))
	}

	return
}

// shadowStepHandler blinks the caster to a designated coordinate, increasing attack attributes if enemies are nearby.
func shadowStepHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	a.relocateUnit(e.By, e.At)

	enemies := a.EnemiesInRange(e.By, e.Ab.AreaRadius)
	if len(enemies) > 0 {
		e.By.CurrentAtk += e.Ab.Effect.AddAtk
		state.ToAll(
			ApplyState{ChangeAtk: new(e.Ab.Effect.AddAtk), ToUnitID: e.By.ID},
			ApplyState{SetAtk: new(e.By.CurrentAtk), ToUnitID: e.By.ID},
		)
	}

	state.ToOpp(ApplyState{MoveTo: new(e.At), ToUnitID: e.By.ID})
	return
}

// huntersMarkHandler puts a targeted mark condition on a chosen unit to amplify oncoming damage.
func huntersMarkHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)

	return ApplyStatusToUnit(a, e.Ab.Effect.ApplyStatus, e.By, target), nil
}

// hamstringShotHandler inflicts single-target damage alongside a movement-impairing status debuff.
func hamstringShotHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target := a.UnitAt(e.At)

	state.With(a.DealDamageToUnit(e.By, target, e.Ab.Effect.DealDamage))
	state.With(ApplyStatusToUnit(a, e.Ab.Effect.ApplyStatus, e.By, target))

	return
}

// healUnit applies health points to a targeted unit up to their maximum base health constraint.
func healUnit(u *Unit, val int) (state ApplyStates) {
	if u.CurrentHP == u.BaseHP {
		return
	}

	u.CurrentHP += val
	if u.CurrentHP > u.BaseHP {
		val -= u.CurrentHP - u.BaseHP
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