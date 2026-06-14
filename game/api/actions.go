package api

type Action string

const (
	ConnectedAction         Action = "connected"
	NewGameAction           Action = "new_game"
	ReadyToPlay             Action = "ready_to_play"
	WaitingForOpponent      Action = "waiting_for_opponent"
	SearchingOppAction      Action = "searching_opp"
	PlaceUnitAction         Action = "place_unit"
	UnitPlacedAction        Action = "unit_placed"
	UnitMovedAction         Action = "unit_moved"
	EndTurnAction           Action = "end_turn"
	PlayUnitAction          Action = "play_unit"
	ApplyStateAction        Action = "apply_state"
	NewRound                Action = "new_round"
	YouWinAction            Action = "you_win"
	YouLoseAction           Action = "you_lose"
	SurrenderAction         Action = "surrender"
	OppSurrenderedAction    Action = "opponent_surrendered"
	OppDisconnectedAction   Action = "opp_disconnected"
	CancelMatchAction       Action = "cancel_match"
	MatchCancelledAction    Action = "match_canceled"
	UseAbilityAction        Action = "use_ability"
	ErrorAction             Action = "error"
	ActiveUnitChangedAction Action = "active_unit_changed"
)
