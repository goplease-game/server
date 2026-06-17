package game

// Game loop rules, phase limitations, and turn constraints.
const (
	// MaxTurns defines the maximum number of rounds allowed before the match ends.
	MaxTurns = 20 // TODO
	// TurnTimeSeconds specifies the total time limit allocated for a unit's turn.
	TurnTimeSeconds = 45
	// UnitsPerPlacementPhase sets how many units players can place during a single deployment stage.
	UnitsPerPlacementPhase = 3
	// MaxPhantomAPPerUnitPerTurn caps the temporary action points a unit can spend in a turn.
	MaxPhantomAPPerUnitPerTurn = 3
	// ApplyImpatienceStatusAfterRound dictates the round threshold after which a fatigue status is forced.
	ApplyImpatienceStatusAfterRound = 10
)

// RoundPhase represents the current state or stage of the game match loop.
type RoundPhase int

// Game match loop lifecycle phases.
const (
	// PlayPhase indicates active combat rounds where units execute actions.
	PlayPhase RoundPhase = iota
	// PlacementPhase indicates the initial deployment setup state for positioning units.
	PlacementPhase
	// GameOverPhase indicates the match has concluded and results are ready.
	GameOverPhase
)
