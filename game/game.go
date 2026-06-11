package game

const (
	MaxTurns                   = 20
	TurnTimeSeconds            = 999
	UnitsPerPlacementPhase     = 2
	MaxPhantomAPPerUnitPerTurn = 3
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
