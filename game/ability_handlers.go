package game

import (
	"fmt"

	"github.com/ognev-dev/goplease/game/ability"
	"github.com/ognev-dev/goplease/game/ability/status"
)

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

func basicMeleeAttackHandler(a *Arena, e abilityUsedEvent) (sts ApplyStates, err error) {
	target, err := a.ShouldUnitAt(e.At)
	if err != nil {
		return sts, err
	}

	sts = a.DealDamageToUnit(e.By, target, e.By.CurrentAtk)
	return
}

func basicRangeAttackHandler(a *Arena, e abilityUsedEvent) (ApplyStates, error) {
	return basicMeleeAttackHandler(a, e)
}

func basicMagicAttackHandler(a *Arena, e abilityUsedEvent) (ApplyStates, error) {
	return basicMeleeAttackHandler(a, e)
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

func provokeHandler(a *Arena, e abilityUsedEvent) (sts ApplyStates, err error) {
	sts.With(applyStatusToUnit(status.Provoking, e.By, e.By))

	units := a.EnemiesInRange(e.By, e.Ab.AreaRadius)
	for _, u := range units {
		sts.With(applyStatusToUnit(status.Provoked, e.By, u))
	}

	return sts, nil
}

func shieldBashHandler(a *Arena, e abilityUsedEvent) (sts ApplyStates, err error) {
	target, err := a.ShouldUnitAt(e.At)
	if err != nil {
		return sts, err
	}

	sts.With(applyStatusToUnit(e.Ab.Effect.ApplyStatus, e.By, target))
	return
}

func powerPushHandler(a *Arena, e abilityUsedEvent) (sts ApplyStates, err error) {
	target, err := a.ShouldUnitAt(e.At)
	if err != nil {
		return sts, err
	}

	dealDmg := e.Ab.Effect.DealDamage

	pos := e.By.Pos.Opposite(target.Pos)
	if a.UnitAt(pos) == nil {
		sts.ToAll(ApplyState{MoveTo: new(pos), ToUnitID: target.ID})
		target.Pos = pos
	} else {
		dealDmg = e.Ab.Effect.DealAltDamage
	}

	sts.With(a.DealDamageToUnit(e.By, target, dealDmg))
	return sts, nil
}

func gangUpHandler(a *Arena, e abilityUsedEvent) (sts ApplyStates, err error) {
	target, err := a.ShouldUnitAt(e.At)
	if err != nil {
		return sts, err
	}

	dealDmg := e.By.CurrentAtk

	pos := e.By.Pos.Opposite(target.Pos)
	u := a.UnitAt(pos)
	if u != nil && u.IsAlly(e.By) {
		dealDmg += e.Ab.Effect.BonusDamage
	}

	sts.With(a.DealDamageToUnit(e.By, target, dealDmg))
	return
}

func eliminateHandler(a *Arena, e abilityUsedEvent) (sts ApplyStates, err error) {
	target, err := a.ShouldUnitAt(e.At)
	if err != nil {
		return sts, err
	}

	sts.With(a.DealDamageToUnit(e.By, target, e.Ab.Effect.DealDamage))
	if target.IsDead {
		ap := e.Ab.Effect.AddAP
		sts.ToAll(
			ApplyState{ChangeAP: new(ap), ToUnitID: e.By.ID},
			ApplyState{SetAP: new(ap), ToUnitID: e.By.ID},
		)
	}

	return
}

func translocationHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target, err := a.ShouldUnitAt(e.At)
	if err != nil {
		return state, err
	}

	// Swapping with self is a no-op and likely a bug — abort.
	if target.ID == e.By.ID {
		err = fmt.Errorf("translocation: cannot swap unit with itself")
		return
	}

	from := e.By.Pos
	to := target.Pos

	a.relocateUnit(e.By, to)
	a.relocateUnit(target, from)

	state.ToAll(
		ApplyState{MoveTo: new(to), ToUnitID: e.By.ID},
		ApplyState{MoveTo: new(from), ToUnitID: target.ID},
	)

	return
}

func timeWarpHandler(a *Arena, e abilityUsedEvent) (sts ApplyStates, err error) {
	target, err := a.ShouldUnitAt(e.At)
	if err != nil {
		return sts, err
	}

	sts.With(applyStatusToUnit(e.Ab.Effect.ApplyStatus, e.By, target))
	return
}

func purgeHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target, err := a.ShouldUnitAt(e.At)
	if err != nil {
		return state, err
	}

	for st, v := range target.Statuses {
		if v.IsPositive() {
			state.With(removeStatusFromUnit(st, target))
		}
	}

	return
}

func purifyHandler(a *Arena, e abilityUsedEvent) (state ApplyStates, err error) {
	target, err := a.ShouldUnitAt(e.At)
	if err != nil {
		return state, err
	}

	for st, v := range target.Statuses {
		if v.IsNegative() {
			state.With(removeStatusFromUnit(st, target))
		}
	}

	state.With(healUnit(target, e.Ab.Effect.HealHP))
	state.With(applyStatusToUnit(e.Ab.Effect.ApplyStatus, e.By, target))

	return
}

func healHandler(a *Arena, e abilityUsedEvent) (sts ApplyStates, err error) {
	target, err := a.ShouldUnitAt(e.At)
	if err != nil {
		return sts, err
	}

	st := healUnit(target, e.Ab.Effect.HealHP)
	return st, nil
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

func idolihuSpinHandler(a *Arena, e abilityUsedEvent) (sts ApplyStates, err error) {
	units := a.EnemiesInRange(e.By, e.Ab.AreaRadius)
	for _, u := range units {
		sts.With(a.DealDamageToUnit(e.By, u, e.By.CurrentAtk))
	}

	return
}

func piercingShotHandler(a *Arena, e abilityUsedEvent) (sts ApplyStates, err error) {
	cells := a.Board.Cells.Line(e.By.Pos, e.At, e.Ab.AreaRadius)
	for _, c := range cells {
		unit := a.UnitAt(c.Coord)
		if unit != nil && unit.IsEnemy(e.By) {
			sts.With(a.DealDamageToUnit(e.By, unit, e.Ab.Effect.DealDamage))
		}
	}

	return
}

func battleCryHandler(a *Arena, e abilityUsedEvent) (sts ApplyStates, err error) {
	units := a.AlliesInRange(e.By, e.Ab.AreaRadius)
	for _, u := range units {
		sts.With(applyStatusToUnit(e.Ab.Effect.ApplyStatus, e.By, u))
	}

	return
}

func shadowStepHandler(a *Arena, e abilityUsedEvent) (sts ApplyStates, err error) {
	if a.CellOccupied(e.At) {
		err = fmt.Errorf("shadowStep: target cell %s is occupied", e.At)
		return
	}

	a.relocateUnit(e.By, e.At)

	sts.ToOpp(ApplyState{MoveTo: new(e.At), ToUnitID: e.By.ID})

	sts.With(applyStatusToUnit(e.Ab.Effect.ApplyStatus, e.By, e.By))

	return
}

func huntersMarkHandler(a *Arena, e abilityUsedEvent) (sts ApplyStates, err error) {
	target, err := a.ShouldUnitAt(e.At)
	if err != nil {
		return sts, err
	}

	sts.With(applyStatusToUnit(e.Ab.Effect.ApplyStatus, e.By, target))

	return
}

func hamstringShotHandler(a *Arena, e abilityUsedEvent) (sts ApplyStates, err error) {
	target, err := a.ShouldUnitAt(e.At)
	if err != nil {
		return sts, err
	}

	sts.With(a.DealDamageToUnit(e.By, target, e.Ab.Effect.DealDamage))
	sts.With(applyStatusToUnit(e.Ab.Effect.ApplyStatus, e.By, target))

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
