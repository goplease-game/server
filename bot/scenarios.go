package bot

import (
	"math"

	game "github.com/goplease-game/server"
	"github.com/goplease-game/server/ability"
	"github.com/goplease-game/server/ability/status"
)

// IDs of unite templates.
const (
	BasID    = 1
	GritID   = 2
	FletchID = 3
	SilverID = 4
	MistID   = 5
	JulyID   = 6
)

type scenario func(b *Bot, u *game.Unit) *simAction

var simScenariosByUnit = map[int][]scenario{
	BasID: {
		scenarioBasShieldBash,
		scenarioBasFortify,
		scenarioBasProvokeDefendSquishies,
		scenarioBasProvoke,
	},
	GritID: {
		scenarioGritBattleCry,
		scenarioGritIdolihuSpin,
		scenarioGritPowerPush,
	},
	FletchID: {
		scenarioFletchBestAbility,
	},
	SilverID: {
		scenarioSilverShadowStep,
		scenarioSilverBestAbility,
	},
	MistID: {
		// scenarioMistTranslocationRescueAlly,
		scenarioMistPurge,
		scenarioMistMoveToAlly,
	},
	JulyID: {
		scenarioJulyHeal,
		scenarioJulyEqualize,
		scenarioJulyPurify,
	},
}

func (b *Bot) scenarioAttackPriorityTarget(u *game.Unit) *simAction {
	target := b.priorityTarget(u)
	if target == nil {
		return nil
	}

	return b.simulateAttackTarget(u, target)
}

func (b *Bot) scenarioMoveTowardsPriorityTarget(u *game.Unit) *simAction {
	target := b.priorityTarget(u)
	if target == nil {
		return b.simulateMove(u)
	}

	return b.simulateMoveTowards(u, target.PosVal())
}

func (b *Bot) simulateMove(u *game.Unit) *simAction {
	cell := b.randomReachableCell(u)
	if cell == nil {
		return nil
	}

	b.placeUnitAt(u, *cell)

	return &simAction{
		moveUnit: cell,
	}
}

func (b *Bot) simulateAttackTarget(u *game.Unit, target *game.Unit) *simAction {
	var basicAttack *ability.Ability
	for _, id := range u.Abilities {
		a := ability.ByID(id)
		if a.IsBasicAttack() {
			basicAttack = &a
			break
		}
	}
	if basicAttack == nil {
		return nil
	}

	moveTo, targetPos, ok := findAbilityTarget(b, u, target, basicAttack.ID)
	if !ok {
		return nil
	}

	act := &simAction{
		useAbility: &useAbilityAction{
			abilityID: basicAttack.ID,
			target:    &targetPos,
		},
	}
	if moveTo != u.PosVal() {
		act.moveUnit = &moveTo
	}

	return act
}

// =============================================================================
// Bas — Tank
// =============================================================================

// scenarioBasProvokeDefendSquishies uses Provoke when an enemy is adjacent
// to a high-priority ally (July or Mist) to draw attacks away from them.
func scenarioBasProvokeDefendSquishies(b *Bot, u *game.Unit) *simAction {
	if !u.AbilityReady(ability.Provoke) {
		return nil
	}

	squishyTemplates := map[int]bool{2: true, 3: true, 4: true, 5: true}
	for _, ally := range b.state.queue {
		if !ally.IsAlly(u) || !squishyTemplates[ally.TemplateID] {
			continue
		}
		// Check if any enemy is adjacent to this ally.
		for _, enemy := range b.enemies(u) {
			if ally.PosVal().Distance(enemy.PosVal()) > 1 {
				continue
			}
			// Enemy is threatening a squishy — provoke from current position if possible.
			moveTo, ok := b.findAttackPosition(u, enemy, ability.ByID(ability.Provoke).Range)
			if !ok {
				continue
			}
			_, targetPos, ok := findAbilityTarget(b, u, enemy, ability.Provoke)
			if !ok {
				continue
			}
			return b.simulateMoveAndUseAbility(u, moveTo, ability.Provoke, targetPos)
		}
	}
	return nil
}

// scenarioBasFortify activates Fortify if the unit can reach a position
// where the ability covers 3 or more allies.
func scenarioBasFortify(b *Bot, u *game.Unit) *simAction {
	if !u.AbilityReady(ability.Fortify) {
		return nil
	}

	fortifyRadius := ability.ByID(ability.Fortify).AreaRadius
	bestPos, allyCount := findBestPositionForAOE(b, u, fortifyRadius, countAlliesInRadius)
	if allyCount < 3 {
		return nil
	}

	return b.simulateMoveAndUseAbility(u, bestPos, ability.Fortify, bestPos)
}

// scenarioBasShieldBash uses Shield Bash on any reachable enemy
// when the priority target is out of range.
func scenarioBasShieldBash(b *Bot, u *game.Unit) *simAction {
	if !u.AbilityReady(ability.ShieldBash) {
		return nil
	}

	target := b.priorityTarget(u)
	if target != nil && b.canReach(u, target, ability.ByID(ability.BasicMeleeAttack).Range) {
		// Priority target is reachable — prefer normal attack.
		return nil
	}

	shieldBashRange := ability.ByID(ability.ShieldBash).Range
	enemy := findClosestReachableEnemy(b, u, shieldBashRange)
	if enemy == nil {
		return nil
	}

	moveTo, targetPos, ok := findAbilityTarget(b, u, enemy, ability.ShieldBash)
	if !ok {
		return nil
	}

	return b.simulateMoveAndUseAbility(u, moveTo, ability.ShieldBash, targetPos)
}

// scenarioBasProvoke uses Provoke when the priority target is unreachable,
// other abilities are on cooldown, and the ability hits at least one enemy.
func scenarioBasProvoke(b *Bot, u *game.Unit) *simAction {
	if !u.AbilityReady(ability.Provoke) {
		return nil
	}

	target := b.priorityTarget(u)
	if target != nil && b.canReach(u, target, ability.ByID(ability.BasicMeleeAttack).Range) {
		return nil
	}

	// Only use Provoke as a last resort when other abilities are unavailable.
	if u.AbilityReady(ability.Fortify) || u.AbilityReady(ability.ShieldBash) {
		return nil
	}

	provokeRange := ability.ByID(ability.Provoke).Range
	enemy := findClosestReachableEnemy(b, u, provokeRange)
	if enemy == nil {
		return nil
	}

	moveTo, targetPos, ok := findAbilityTarget(b, u, enemy, ability.Provoke)
	if !ok {
		return nil
	}

	return b.simulateMoveAndUseAbility(u, moveTo, ability.Provoke, targetPos)
}

// =============================================================================
// Grit — Warrior
// =============================================================================

// scenarioGritPowerPush uses Power Push, preferring targets that cannot be pushed
// (adjacent to a wall or board edge) to guarantee the bonus damage.
func scenarioGritPowerPush(b *Bot, u *game.Unit) *simAction {
	if !u.AbilityReady(ability.PowerPush) {
		return nil
	}

	target := b.priorityTarget(u)
	if target == nil {
		return nil
	}

	moveTo, ok := b.findAttackPosition(u, target, ability.ByID(ability.PowerPush).Range)
	if !ok {
		return nil
	}

	// Prefer using PowerPush when the target is blocked (alt damage triggers).
	pushDest := u.PosVal().Opposite(target.PosVal())
	cell, exists := b.state.board.Cells[pushDest]
	blocked := !exists || (cell.Unit != nil)
	if !blocked {
		return nil // save cooldown — only 2 damage, not worth it
	}

	return b.simulateMoveAndUseAbility(u, moveTo, ability.PowerPush, target.PosVal())
}

// scenarioGritBattleCry uses Battle Cry if an enemy is within 2 cells
// and at least one ally (including self or others) is affected by the AoE.
func scenarioGritBattleCry(b *Bot, u *game.Unit) *simAction {
	if !u.AbilityReady(ability.BattleCry) {
		return nil
	}

	// Check if any enemy is within 2 cells from Grit's current position
	enemyClose := false
	for _, enemy := range b.enemies(u) {
		if u.PosVal().Distance(enemy.PosVal()) <= 2 {
			enemyClose = true
			break
		}
	}

	if !enemyClose {
		return nil
	}

	battleCryRadius := ability.ByID(ability.BattleCry).AreaRadius
	bestPos, allyCount := findBestPositionForAOE(b, u, battleCryRadius, countAlliesInRadius)

	if allyCount < 2 {
		return nil
	}

	return b.simulateMoveAndUseAbility(u, bestPos, ability.BattleCry, bestPos)
}

// scenarioGritIdolihuSpin uses IDOLIHU! Spin when the priority target
// falls within the spin's area of effect and at least 2 enemies are in range.
// With only one target, a basic attack is preferred to avoid wasting the cooldown.
func scenarioGritIdolihuSpin(b *Bot, u *game.Unit) *simAction {
	if !u.AbilityReady(ability.IdolihuSpin) {
		return nil
	}

	target := b.priorityTarget(u)
	if target == nil {
		return nil
	}

	spinRadius := ability.ByID(ability.IdolihuSpin).AreaRadius
	moveTo, ok := b.findAttackPosition(u, target, spinRadius)
	if !ok {
		return nil
	}

	// Count enemies reachable from the candidate position, not current position,
	// since we may move before spinning.
	enemyCount := countEnemiesInRangeFrom(b, moveTo, u, spinRadius)
	if enemyCount < 2 {
		return nil
	}

	return b.simulateMoveAndUseAbility(u, moveTo, ability.IdolihuSpin, moveTo)
}

// =============================================================================
// Fletch — Ranger
// =============================================================================

// scenarioFletchBestAbility tries each ability in priority order.
// Uses Hunter's Mark if ready and the target isn't already marked.
func scenarioFletchBestAbility(b *Bot, u *game.Unit) *simAction {
	target := b.priorityTarget(u)
	if target == nil {
		return nil
	}

	// Special check for Hunter's Mark: do not cast if target already has it
	if u.AbilityReady(ability.HuntersMark) {
		if _, alreadyMarked := target.Statuses[status.Marked]; !alreadyMarked {
			moveTo, targetPos, ok := findAbilityTarget(b, u, target, ability.HuntersMark)
			if ok {
				return b.simulateMoveAndUseAbility(u, moveTo, ability.HuntersMark, targetPos)
			}
		}
	}

	// Fallback to other regular attacks if Mark is on CD or already applied
	prioritized := []ability.ID{
		ability.HamstringShot,
		ability.PiercingShot,
		ability.BasicRangeAttack,
	}

	for _, abilityID := range prioritized {
		if !u.AbilityReady(abilityID) {
			continue
		}
		moveTo, targetPos, ok := findAbilityTarget(b, u, target, abilityID)
		if !ok {
			continue
		}
		return b.simulateMoveAndUseAbility(u, moveTo, abilityID, targetPos)
	}

	return nil
}

// =============================================================================
// Silver — Rogue
// =============================================================================

// scenarioSilverShadowStep always uses Shadow Step if ready,
// ensuring landing on a cell adjacent to the priority target.
func scenarioSilverShadowStep(b *Bot, u *game.Unit) *simAction {
	if !u.AbilityReady(ability.ShadowStep) {
		return nil
	}

	target := b.priorityTarget(u)
	if target == nil {
		return nil
	}

	// Find any free cell adjacent to the priority target within Shadow Step range
	shadowStepRange := ability.ByID(ability.ShadowStep).Range
	dest, ok := b.findFreeCellAdjacentTo(u, target, shadowStepRange)
	if !ok {
		return nil
	}

	// Teleport always happens from current position (u.PosVal()) to the destination adjacent to enemy
	return b.simulateMoveAndUseAbility(u, u.PosVal(), ability.ShadowStep, dest)
}

// scenarioSilverBestAbility tries each ability in priority order and uses
// the first one that can reach the priority target.
// Priority: Eliminate > Gang Up > Shadow Step > basic attack.
func scenarioSilverBestAbility(b *Bot, u *game.Unit) *simAction {
	target := b.priorityTarget(u)
	if target == nil {
		return nil
	}

	prioritized := []ability.ID{
		ability.Eliminate,
		ability.GangUp,
		ability.ShadowStep,
		ability.BasicMeleeAttack,
	}

	for _, abilityID := range prioritized {
		if !u.AbilityReady(abilityID) {
			continue
		}
		moveTo, targetPos, ok := findAbilityTarget(b, u, target, abilityID)
		if !ok {
			continue
		}
		return b.simulateMoveAndUseAbility(u, moveTo, abilityID, targetPos)
	}

	return nil
}

// =============================================================================
// Mist — Mage
// =============================================================================

// scenarioMistTranslocationRescueAlly swaps a threatened ally (adjacent to an enemy)
// with Mist itself to pull them to safety.
func scenarioMistTranslocationRescueAlly(b *Bot, u *game.Unit) *simAction {
	if !u.AbilityReady(ability.Translocation) {
		return nil
	}

	transRange := ability.ByID(ability.Translocation).Range

	priority := []int{6, 5, 1, 2, 3, 4}
	for _, templateID := range priority {
		for _, ally := range b.state.queue {
			if !ally.IsAlly(u) || ally.TemplateID != templateID {
				continue
			}
			// Cannot swap with self.
			if ally.ID == u.ID {
				continue
			}
			if u.PosVal().Distance(ally.PosVal()) > transRange {
				continue
			}
			threatened := false
			for _, enemy := range b.enemies(u) {
				if enemy.PosVal().Distance(ally.PosVal()) <= 1 {
					threatened = true
					break
				}
			}
			if !threatened {
				continue
			}
			return b.simulateMoveAndUseAbility(u, u.PosVal(), ability.Translocation, ally.PosVal())
		}
	}

	return nil
}

// scenarioMistPurge uses Purge to cleanse Rallied or TemporalAnchor buffs from enemies.
func scenarioMistPurge(b *Bot, u *game.Unit) *simAction {
	if !u.AbilityReady(ability.Purge) {
		return nil
	}

	purgeRange := ability.ByID(ability.Purge).Range
	enemies := b.enemies(u)

	var target *game.Unit
	for _, e := range enemies {
		// Check if enemy has either Rallied (from Grit) or Temporal Anchor
		_, hasRallied := e.Statuses[status.Rallied]
		_, hasAnchor := e.Statuses[status.TemporalAnchor]

		if hasRallied || hasAnchor {
			if u.PosVal().Distance(e.PosVal()) <= u.CurrentMP+purgeRange {
				target = e
				break
			}
		}
	}

	if target == nil {
		return nil
	}

	moveTo, targetPos, ok := findAbilityTarget(b, u, target, ability.Purge)
	if !ok {
		return nil
	}

	return b.simulateMoveAndUseAbility(u, moveTo, ability.Purge, targetPos)
}

// scenarioMistMoveToAlly moves Mist adjacent to the nearest ally to activate Focus Field.
// Skipped if moving would cause Mist to lose line of sight to the priority target.
func scenarioMistMoveToAlly(b *Bot, u *game.Unit) *simAction {
	target := b.priorityTarget(u)

	// Check if the priority target is reachable before considering movement.
	ptReachableBefore := target != nil && b.canReach(u, target, ability.ByID(ability.BasicMagicAttack).Range)

	ally := b.closestAlly(u)
	if ally == nil {
		return nil
	}

	moveTo, ok := b.adjacentPosition(u, ally)
	if !ok {
		return nil
	}

	// Do not reposition if it would give up a reachable priority target.
	if ptReachableBefore {
		if !canReachFrom(moveTo, target, ability.ByID(ability.BasicMagicAttack).Range) {
			return nil
		}
	}

	b.placeUnitAt(u, moveTo)

	return &simAction{
		moveUnit: &moveTo,
	}
}

// =============================================================================
// July — Support
// =============================================================================

// scenarioJulyEqualize uses Equalize when it would benefit the most wounded ally
// more than a regular Heal would (i.e. average HP in range > wounded HP + healAmount).
func scenarioJulyEqualize(b *Bot, u *game.Unit) *simAction {
	if !u.AbilityReady(ability.Equalize) {
		return nil
	}

	equalizeRadius := ability.ByID(ability.Equalize).AreaRadius
	allies := append(b.alliesInRange(u, equalizeRadius), u)
	if len(allies) < 2 {
		return nil
	}

	var sumHP, minHP int
	minHP = math.MaxInt
	for _, a := range allies {
		sumHP += a.CurrentHP
		if a.CurrentHP < minHP {
			minHP = a.CurrentHP
		}
	}

	avg := sumHP / len(allies)
	healGain := avg - minHP

	// Only use Equalize if it heals the worst-off ally more than a regular Heal.
	if healGain <= ability.ByID(ability.Heal).Effect.HealHP {
		return nil
	}

	return b.simulateUseAbility(u, ability.Equalize, u.PosVal())
}

// scenarioJulyHeal heals the most wounded ally (or self) within range.
// Skipped if all friendly units are at full HP.
func scenarioJulyHeal(b *Bot, u *game.Unit) *simAction {
	if !u.AbilityReady(ability.Heal) {
		return nil
	}

	healRange := ability.ByID(ability.Heal).Range
	target := b.mostWoundedAllyInRange(u, healRange)
	if target == nil {
		return nil
	}

	return b.simulateUseAbility(u, ability.Heal, target.PosVal())
}

// scenarioJulyPurify prioritizes cleansing Hunter's Mark from teammates.
func scenarioJulyPurify(b *Bot, u *game.Unit) *simAction {
	if !u.AbilityReady(ability.Purify) {
		return nil
	}

	purifyRange := ability.ByID(ability.Purify).Range
	candidates := append(b.alliesInRange(u, purifyRange), u)

	var target *game.Unit
	for _, ally := range candidates {
		// High priority: remove the deadly Hunter's Mark
		if _, hasMark := ally.Statuses[status.Marked]; hasMark {
			target = ally
			break
		}
	}

	// Fallback to any other debuff if no one is marked
	if target == nil {
		target = b.allyWithDebuffInRange(u, purifyRange)
	}

	if target == nil {
		return nil
	}

	return b.simulateUseAbility(u, ability.Purify, target.PosVal())
}
