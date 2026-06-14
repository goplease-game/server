package game

const (
	MaxTurns                        = 20 // TODO
	TurnTimeSeconds                 = 999
	UnitsPerPlacementPhase          = 3
	MaxPhantomAPPerUnitPerTurn      = 3
	ApplyImpatienceStatusAfterRound = 10
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
