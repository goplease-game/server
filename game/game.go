package game

const (
	MaxTurns                        = 20
	TurnTimeSeconds                 = 30
	UnitsPerPlacementPhase          = 3
	MaxPhantomAPPerUnitPerTurn      = 3
	ApplyImpatienceStatusAfterRound = 3
)

type RoundPhase int

const (
	PlayPhase RoundPhase = iota
	PlacementPhase
	GameOverPhase
)

type EndReason string

const (
	EndNoUnits   EndReason = "no_units"
	EndTurnLimit EndReason = "turn_limit"
)
