package bot

import (
	"github.com/goplease-game/server"
	"github.com/goplease-game/server/ability"
	"github.com/goplease-game/server/ability/status"
	"github.com/goplease-game/server/ds"
)

type simAction struct {
	moveUnit   *game.HexCoord
	useAbility *useAbilityAction
}

type useAbilityAction struct {
	abilityID ability.ID
	target    *game.HexCoord
}

func (b *Bot) simulateUnitTurn(u *game.Unit) *simAction {
	if u.CurrentAP < 1 {
		return nil
	}

	scenarios, ok := simScenariosByUnit[u.TemplateID]
	if !ok {
		return nil
	}

	for _, sc := range scenarios {
		if act := sc(b, u); act != nil {
			return act
		}
	}

	// Default: attack or move toward priority target
	if act := b.scenarioAttackPriorityTarget(u); act != nil {
		return act
	}

	return b.scenarioMoveTowardsPriorityTarget(u)
}

// findBestPositionForAOE finds the cell reachable by u (within MovePoints)
// that maximises the score returned by scoreFn(center, radius).
// Returns the best position and its score.
func findBestPositionForAOE(
	b *Bot,
	u *game.Unit,
	radius int,
	scoreFn func(b *Bot, u *game.Unit, center game.HexCoord, radius int) int,
) (game.HexCoord, int) {
	reachable := b.state.board.Cells.InRange(u.PosVal(), u.CurrentMP)

	bestPos := u.PosVal()
	bestScore := scoreFn(b, u, u.PosVal(), radius)

	for _, cell := range reachable {
		if cell.Unit != nil && cell.Unit.ID != u.ID {
			continue
		}
		score := scoreFn(b, u, cell.Coord, radius)
		if score > bestScore {
			bestScore = score
			bestPos = cell.Coord
		}
	}

	return bestPos, bestScore
}

// countAlliesInRadius counts allies of u within radius of center,
// matching the signature expected by findBestPositionForAOE.
func countAlliesInRadius(b *Bot, u *game.Unit, center game.HexCoord, radius int) int {
	count := 0

	cells := b.state.board.Cells.InRange(center, radius)
	for _, c := range cells {
		unit := b.unitAt(c.Coord)
		if unit != nil && unit.IsAlly(u) && unit.ID != u.ID {
			count++
		}
	}

	return count
}

// priorityTarget returns the current priority target for the given unit.
// It prioritizes specific provokers extracted from Meta, then targets with Hunter's Mark,
// and falls back to the closest enemy with the lowest health pool.
func (b *Bot) priorityTarget(u *game.Unit) *game.Unit {
	enemies := b.enemies(u)
	if len(enemies) == 0 {
		return nil
	}

	// 1. Strict Priority: If the unit is Provoked, it must target its specific provoker.
	if provokedStatus, isProvoked := u.Statuses[status.Provoked]; isProvoked {
		if provokedStatus.Meta != nil {
			if provokerID, ok := provokedStatus.Meta["provoker"].(ds.ID); ok {
				provoker := b.unitByID(provokerID)
				// Verify the provoker exists and is alive before committing.
				if provoker != nil && provoker.Alive() {
					return provoker
				}
			}
		}
	}

	// 2. High Priority: Focus down enemies affected by Hunter's Mark.
	for _, e := range enemies {
		if _, hasMark := e.Statuses[status.Marked]; hasMark {
			return e
		}
	}

	// 3. Fallback Priority: Target the closest enemy, breaking ties with lowest current HP.
	var best *game.Unit
	for _, e := range enemies {
		if best == nil {
			best = e
			continue
		}
		distBest := u.PosVal().Distance(best.PosVal())
		distE := u.PosVal().Distance(e.PosVal())
		if distE < distBest || (distE == distBest && e.CurrentHP < best.CurrentHP) {
			best = e
		}
	}

	return best
}

// canReach reports whether the unit can reach the target considering
// both movement range and ability range.
func (b *Bot) canReach(u *game.Unit, target *game.Unit, abilityRange int) bool {
	_, ok := b.findAttackPosition(u, target, abilityRange)
	return ok
}

// findClosestReachableEnemy returns the nearest enemy reachable
// within movement + abilityRange, or nil if none can be reached.
func findClosestReachableEnemy(b *Bot, u *game.Unit, abilityRange int) *game.Unit {
	enemies := b.enemies(u)

	var best *game.Unit
	for _, e := range enemies {
		_, ok := b.findAttackPosition(u, e, abilityRange)
		if !ok {
			continue
		}
		if best == nil {
			best = e
			continue
		}
		if u.PosVal().Distance(e.PosVal()) < u.PosVal().Distance(best.PosVal()) {
			best = e
		}
	}
	return best
}

// findAbilityTarget returns the position to pass as Target to HandleAbility,
// and the position to move to before using the ability.
// Returns false if the ability cannot be used against the given target from current position.
func findAbilityTarget(b *Bot, u *game.Unit, target *game.Unit, abilityID ability.ID) (moveTo game.HexCoord, targetPos game.HexCoord, ok bool) {
	a := ability.ByID(abilityID)

	switch a.Activation {
	case ability.SelectFreeCell:
		// Target is a free cell within range, not a unit.
		// Find a free cell adjacent to the priority target within ability range.
		targetPos, ok = b.findFreeCellAdjacentTo(u, target, a.Range)
		if !ok {
			return
		}
		moveTo = u.PosVal() // no walking — ability itself handles repositioning
		return

	default:
		// Target is a unit — find a position from which u can hit it.
		moveTo, ok = b.findAttackPosition(u, target, a.Range)
		if !ok {
			return
		}
		targetPos = target.PosVal()
		return
	}
}

// findFreeCellAdjacentTo returns a free board cell adjacent to target
// reachable within stepRange hex steps from u's current position.
func (b *Bot) findFreeCellAdjacentTo(u *game.Unit, target *game.Unit, stepRange int) (game.HexCoord, bool) {
	neighbors := target.PosVal().Neighbors()
	for _, pos := range neighbors {
		unit := b.unitAt(pos)
		if unit != nil {
			continue
		}
		if u.PosVal().Distance(pos) <= stepRange {
			return pos, true
		}
	}
	return game.HexCoord{}, false
}

// countEnemiesInRangeFrom counts enemies of u within radius of a given position.
// Used to evaluate AoE value before committing to a move.
func countEnemiesInRangeFrom(b *Bot, center game.HexCoord, u *game.Unit, radius int) int {
	count := 0
	cells := b.state.board.Cells.InRange(center, radius)
	for _, c := range cells {
		if c.Unit != nil && c.Unit.IsEnemy(u) {
			count++
		}
	}

	return count
}

// findClosestEnemyWithBuffs returns the nearest reachable enemy
// that has at least one positive status effect, or nil.
func findClosestEnemyWithBuffs(b *Bot, u *game.Unit, abilityRange int) *game.Unit {
	enemies := b.enemies(u)

	var best *game.Unit
	for _, e := range enemies {
		if !hasPositiveStatus(e) {
			continue
		}
		_, ok := b.findAttackPosition(u, e, abilityRange)
		if !ok {
			continue
		}
		if best == nil {
			best = e
			continue
		}
		if u.PosVal().Distance(e.PosVal()) < u.PosVal().Distance(best.PosVal()) {
			best = e
		}
	}
	return best
}

// hasPositiveStatus reports whether the unit has any active positive status effect.
func hasPositiveStatus(u *game.Unit) bool {
	for _, v := range u.Statuses {
		if v.IsPositive() {
			return true
		}
	}
	return false
}

// hasNegativeStatus reports whether the unit has any active negative status effect.
func hasNegativeStatus(u *game.Unit) bool {
	for _, v := range u.Statuses {
		if v.IsNegative() {
			return true
		}
	}
	return false
}

// closestAlly returns the nearest living ally, excluding self. Returns nil if none.
func (b *Bot) closestAlly(u *game.Unit) *game.Unit {
	var best *game.Unit
	for _, other := range b.state.queue {
		if other.ID == u.ID || !other.IsAlly(u) {
			continue
		}
		if best == nil {
			best = other
			continue
		}
		if u.PosVal().Distance(other.PosVal()) < u.PosVal().Distance(best.PosVal()) {
			best = other
		}
	}

	return best
}

// findAdjacentPosition returns the closest free cell adjacent to target
// that u can reach within its movement range, or false if none exists.
func (b *Bot) adjacentPosition(u *game.Unit, target *game.Unit) (game.HexCoord, bool) {
	neighbors := target.PosVal().Neighbors()

	var best game.HexCoord
	bestDist := -1

	for _, pos := range neighbors {
		unit := b.unitAt(pos)
		if unit != nil {
			continue
		}
		dist := u.PosVal().Distance(pos)
		if dist > u.CurrentMP {
			continue
		}
		if bestDist < 0 || dist < bestDist {
			best = pos
			bestDist = dist
		}
	}

	return best, bestDist >= 0
}

// canReachFrom reports whether a unit standing at fromPos could reach
// the target given abilityRange (ignores movement — assumes unit is already at fromPos).
func canReachFrom(from game.HexCoord, target *game.Unit, abilityRange int) bool {
	return from.Distance(target.PosVal()) <= abilityRange
}

// simulateMoveTowards moves unit `u` along a valid path toward `targetPos`.
// It leverages game.ReachableCells to fully avoid walls and occupied hexes.
func (b *Bot) simulateMoveTowards(u *game.Unit, targetPos game.HexCoord) *simAction {
	// Gather all hexes reachable within the unit's remaining Movement Points.
	reachable := game.ReachableCells(u.PosVal(), u.CurrentMP, *b.state.board)
	if len(reachable) == 0 {
		return nil
	}

	bestPos := u.PosVal()
	bestDist := u.PosVal().Distance(targetPos)

	for _, coord := range reachable {
		// Never path through or end a turn on an occupied cell.
		if cell := b.cellAt(coord); cell != nil && cell.Unit != nil {
			continue
		}

		d := coord.Distance(targetPos)
		if d < bestDist {
			bestDist = d
			bestPos = coord
		}
	}

	// If no progress can be made toward the target, stay stationary.
	if bestPos == u.PosVal() {
		return nil
	}

	b.placeUnitAt(u, bestPos)

	return &simAction{
		moveUnit: &bestPos,
	}
}

func (b *Bot) alliesInRange(u *game.Unit, radius int) []*game.Unit {
	units := []*game.Unit{}

	cells := b.state.board.Cells.InRange(u.PosVal(), radius)

	for _, c := range cells {
		if c.Unit != nil && !c.Unit.IsDead && c.Unit.IsAlly(u) {
			units = append(units, c.Unit)
		}
	}

	return units
}

// simulateMoveAndUseAbility moves the unit to moveTo (if different from current position)
// and then applies the given ability at targetPos.
func (b *Bot) simulateMoveAndUseAbility(u *game.Unit, moveTo game.HexCoord, abilityID ability.ID, targetPos game.HexCoord) *simAction {
	sim := &simAction{}

	if moveTo != u.PosVal() {
		b.placeUnitAt(u, moveTo)
		sim.moveUnit = &moveTo
	}

	sim.useAbility = &useAbilityAction{
		abilityID: abilityID,
		target:    &targetPos,
	}

	return sim
}

// simulateUseAbility applies an ability to the given target position
// and returns the resulting actions.
func (b *Bot) simulateUseAbility(_ *game.Unit, abilityID ability.ID, targetPos game.HexCoord) *simAction {
	return &simAction{
		useAbility: &useAbilityAction{
			abilityID: abilityID,
			target:    &targetPos,
		},
	}
}

// findMostWoundedAllyInRange returns the ally (or self) within healRange
// with the lowest CurrentHP relative to BaseHP.
// Returns nil if all units in range are at full HP.
func (b *Bot) mostWoundedAllyInRange(u *game.Unit, healRange int) *game.Unit {
	candidates := append(b.alliesInRange(u, healRange), u)

	var best *game.Unit
	for _, ally := range candidates {
		if ally.CurrentHP >= ally.BaseHP {
			continue
		}
		if best == nil {
			best = ally
			continue
		}
		if ally.CurrentHP < best.CurrentHP {
			best = ally
		}
	}

	return best
}

// findAllyWithDebuffInRange returns the first ally (or self) within range
// that has at least one negative status effect, or nil.
func (b *Bot) allyWithDebuffInRange(u *game.Unit, abilityRange int) *game.Unit {
	candidates := append(b.alliesInRange(u, abilityRange), u)

	for _, ally := range candidates {
		if hasNegativeStatus(ally) {
			return ally
		}
	}

	return nil
}
