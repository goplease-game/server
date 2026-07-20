package game

import (
	"time"

	"github.com/goplease-game/game-server/ds"
)

// Score weight constants. Tuned so that Kills dominate the score (a kill is a
// discrete, decisive event capped at 6 per side), while volume stats like
// DamageDealt contribute proportionally less per point since they naturally
// accumulate into the hundreds over a game.
const (
	scoreWeightKill            = 150
	scoreWeightDamageDealt     = 1.0
	scoreWeightShieldApplied   = 1.0
	scoreWeightShieldDestroyed = 1.0
	scoreWeightHealingReceived = 1.0
	scoreWeightHealingWasted   = -1.0
	scoreWeightOverkillDamage  = -1.0
	scoreWeightAbilitiesUsed   = 5.0
	scoreWeightCellsTraveled   = 0.5
)

// Stats holds aggregate statistics for a completed or in-progress game,
// keyed by player ID for the summary screen.
type Stats struct {
	StartedAt   time.Time `json:"started_at"`
	EndedAt     time.Time `json:"ended_at"`
	RoundNumber int       `json:"round_number"`

	FirstBlood *FirstBloodStats       `json:"first_blood"`
	Players    map[ds.ID]*PlayerStats `json:"players"`
}

// FirstBloodStats records the first unit death of the game.
type FirstBloodStats struct {
	Round      int    `json:"round"`
	VictimID   ds.ID  `json:"victim_id"`
	VictimName string `json:"victim_name"`
	VictimSide ds.ID  `json:"victim_side"`
	KillerID   ds.ID  `json:"killer_id"`
	KillerName string `json:"killer_name"`
	KillerSide ds.ID  `json:"killer_side"`
}

// PlayerStats holds per-player aggregate combat and support statistics.
type PlayerStats struct {
	DamageDealt     int `json:"damage_dealt"`
	DamageReceived  int `json:"damage_received"`
	OverkillDamage  int `json:"overkill_damage"`
	Kills           int `json:"kills"`
	Deaths          int `json:"deaths"`
	ShieldApplied   int `json:"shield_applied"`
	ShieldDestroyed int `json:"shield_destroyed"`
	HealingReceived int `json:"healing_received"`
	HealingWasted   int `json:"healing_wasted"` // overheal
	CellsTraveled   int `json:"cells_traveled"`
	AbilitiesUsed   int `json:"abilities_used"`

	AbilityUsage map[string]int `json:"ability_usage"`

	Score int `json:"score"`
}

// NewGameStats creates an initialized Stats ready to accumulate events.
func NewGameStats(startedAt time.Time) *Stats {
	return &Stats{
		StartedAt: startedAt,
		Players:   make(map[ds.ID]*PlayerStats),
	}
}

// RecordDamage records damage dealt by attacker and received by target.
// dmg is the full amount that hit after shield absorption but before the
// HP clamp; overkill is the portion of dmg that exceeded the target's
// remaining HP. Both sides record the same dmg value so DamageDealt and
// DamageReceived stay consistent as an invariant across a 1v1 match.
func (gs *Stats) RecordDamage(attackerSide, targetSide ds.ID, dmg, overkill int) {
	gs.playerStats(attackerSide).DamageDealt += dmg
	gs.playerStats(targetSide).DamageReceived += dmg
	if overkill > 0 {
		gs.playerStats(attackerSide).OverkillDamage += overkill
	}
}

// RecordKill records a kill for killerSide and a death for victimSide.
// If this is the first death of the game, it also sets FirstBlood.
func (gs *Stats) RecordKill(round int, killerID, victimID ds.ID, killerName, victimName string, killerSide, victimSide ds.ID) {
	gs.playerStats(killerSide).Kills++
	gs.playerStats(victimSide).Deaths++

	if gs.FirstBlood == nil {
		gs.FirstBlood = &FirstBloodStats{
			Round:      round,
			VictimID:   victimID,
			VictimName: victimName,
			VictimSide: victimSide,
			KillerID:   killerID,
			KillerName: killerName,
			KillerSide: killerSide,
		}
	}
}

// RecordShieldApplied records shield points applied to units on side.
func (gs *Stats) RecordShieldApplied(side ds.ID, amount int) {
	if amount > 0 {
		gs.playerStats(side).ShieldApplied += amount
	}
}

// RecordShieldDestroyed records shield points absorbed/destroyed on side.
func (gs *Stats) RecordShieldDestroyed(side ds.ID, amount int) {
	if amount > 0 {
		gs.playerStats(side).ShieldDestroyed += amount
	}
}

// RecordHealing records healing received by target's side, splitting out
// the overheal (wasted) portion.
func (gs *Stats) RecordHealing(side ds.ID, healed, wasted int) {
	ps := gs.playerStats(side)
	ps.HealingReceived += healed
	ps.HealingWasted += wasted
}

// RecordMovement records cells traveled by a unit belonging to side.
func (gs *Stats) RecordMovement(side ds.ID, cells int) {
	gs.playerStats(side).CellsTraveled += cells
}

// RecordAbilityUse records usage of a named ability by side.
func (gs *Stats) RecordAbilityUse(side ds.ID, abilityName string) {
	ps := gs.playerStats(side)
	ps.AbilitiesUsed++
	ps.AbilityUsage[abilityName]++
}

// Finalize sets EndedAt and the final round number. Call once when the game ends.
func (gs *Stats) Finalize(endedAt time.Time, round int) {
	gs.EndedAt = endedAt
	gs.RoundNumber = round

	for id, p := range gs.Players {
		gs.Players[id].Score = p.CountScore()
	}
}

// Duration returns the total game duration. Returns 0 if not yet finalized.
func (gs *Stats) Duration() time.Duration {
	if gs.EndedAt.IsZero() {
		return 0
	}
	return gs.EndedAt.Sub(gs.StartedAt)
}

// CountScore computes a single "performance rating" number for the post-game
// summary screen. It is a vanity metric meant to feel rewarding and
// replayable, not a balanced or matchmaking-safe rating.
func (ps *PlayerStats) CountScore() int {
	score := 0.0

	score += float64(ps.Kills) * scoreWeightKill
	score += float64(ps.DamageDealt) * scoreWeightDamageDealt

	score += float64(ps.ShieldApplied) * scoreWeightShieldApplied
	score += float64(ps.ShieldDestroyed) * scoreWeightShieldDestroyed
	score += float64(ps.HealingReceived) * scoreWeightHealingReceived

	score += float64(ps.HealingWasted) * scoreWeightHealingWasted
	score += float64(ps.OverkillDamage) * scoreWeightOverkillDamage

	score += float64(ps.AbilitiesUsed) * scoreWeightAbilitiesUsed
	score += float64(ps.CellsTraveled) * scoreWeightCellsTraveled

	if score < 0 {
		score = 0
	}

	return int(score)
}

// playerStats returns the PlayerStats for id, creating it on first access.
func (gs *Stats) playerStats(id ds.ID) *PlayerStats {
	ps, ok := gs.Players[id]
	if !ok {
		ps = &PlayerStats{AbilityUsage: make(map[string]int)}
		gs.Players[id] = ps
	}
	return ps
}
