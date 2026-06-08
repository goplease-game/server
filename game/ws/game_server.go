package ws

import (
	"encoding/json"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/game"
	"github.com/ognev-dev/goplease/game/ability"
	"github.com/ognev-dev/goplease/game/match"
)

type Action string

const (
	ConnectedAction       Action = "connected"
	NewGameAction         Action = "new_game"
	ReadyToPlay           Action = "ready_to_play"
	WaitingForOpponent    Action = "waiting_for_opponent"
	SearchingOppAction    Action = "searching_opp"
	PlaceUnitAction       Action = "place_unit"
	UnitPlacedAction      Action = "unit_placed"
	UnitMovedAction       Action = "unit_moved"
	EndTurnAction         Action = "end_turn"
	PlayUnitAction        Action = "play_unit"
	ApplyStateAction      Action = "apply_state"
	NewRound              Action = "new_round"
	OppDisconnectedAction Action = "opp_disconnected"
	CancelMatchAction     Action = "cancel_match"
	MatchCancelledAction  Action = "match_canceled"
	UseAbility            Action = "use_ability"
	ErrorAction           Action = "error"
)

// GameServer wires the hub to the game layer.
type GameServer struct {
	hub        *Hub
	matchmaker *match.Matchmaker
}

func NewGameServer(hub *Hub, mm *match.Matchmaker) *GameServer {
	return &GameServer{
		hub:        hub,
		matchmaker: mm,
	}
}

type ConnectedResponse struct {
	PlayerID ds.ID `json:"player_id"`
}

type DisconnectResponse struct {
	PlayerID ds.ID `json:"player_id"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

// Run reads from hub.Events and dispatches to game handlers.
// Call once in a goroutine: go gs.Run()
func (gs *GameServer) Run() {
	for event := range gs.hub.Events {
		switch event.Kind {
		case EventConnected:
			gs.onConnect(event.Client)
		case EventDisconnected:
			gs.onDisconnect(event.Client)
		case EventMessage:
			gs.onMessage(event.Client, event.Msg)
		}
	}
}

func (gs *GameServer) onConnect(c *Client) {
	c.Send(OutMessage{
		Action: ConnectedAction,
		Data:   ConnectedResponse{PlayerID: c.PlayerID},
	})
}

func (gs *GameServer) onDisconnect(c *Client) {
	gs.matchmaker.Cancel(c.PlayerID)
	if c.ArenaID != "" {
		gs.hub.Broadcast(c.ArenaID, OutMessage{
			Action: OppDisconnectedAction,
			Data:   DisconnectResponse{PlayerID: c.PlayerID},
		})
	}
}

func (gs *GameServer) onMessage(c *Client, msg InMessage) {
	switch msg.Action {

	case NewGameAction:
		gs.prepareNewGame(c)

	case ReadyToPlay:
		gs.handleReadyToPlay(c)

	case CancelMatchAction:
		gs.matchmaker.Cancel(c.PlayerID)
		c.Send(OutMessage{Action: MatchCancelledAction, Data: nil})

	case UnitPlacedAction:
		gs.handlePlaceUnit(c, msg.Data)

	case UnitMovedAction:
		gs.handleUnitMoved(c, msg.Data)

	case EndTurnAction:
		gs.handleEndTurn(c)

	case UseAbility:
		gs.useAbility(c, msg.Data)

	default:
		c.Send(OutMessage{
			Action: ErrorAction,
			Data: ErrorResponse{
				Message: "[server] unknown action: " + string(msg.Action),
			},
		})
	}
}

func (gs *GameServer) prepareNewGame(c *Client) {
	gs.matchmaker.Enqueue(c.PlayerID, func(room *game.Arena, playerIndex int) {
		p := room.Players[playerIndex]
		client := gs.hub.ClientByPlayerID(p.ID)
		if client == nil {
			return
		}

		client.ArenaID = room.ID
		client.Send(OutMessage{
			Action: NewGameAction,
			Data:   newGamePayload(room, playerIndex),
		})
	})

	c.Send(OutMessage{Action: SearchingOppAction, Data: nil})
}

func (gs *GameServer) handleReadyToPlay(c *Client) {
	ar := gs.matchmaker.ArenaByPlayerID(c.PlayerID)
	if ar == nil {
		c.Send(errMsg("arena not found"))
		return
	}

	c.Send(OutMessage{Action: WaitingForOpponent})

	if !ar.MarkReady(c.PlayerID) {
		return
	}

	gs.advanceGameLoop(ar)
}

// advanceGameLoop determines what happens next in the arena.
func (gs *GameServer) advanceGameLoop(ar *game.Arena) {
	switch ar.Phase {
	case game.PlacementPhase:
		gs.runPlacementPhase(ar)
	case game.PlayPhase:
		gs.advancePlayPhase(ar)
	}
}

func (gs *GameServer) advancePlayPhase(ar *game.Arena) {
	activeUnit := ar.ActingUnit()
	if activeUnit == nil {
		// Queue exhausted — check if players have units in hand.
		if ar.Players[0].HasUnitsInHand() || ar.Players[1].HasUnitsInHand() {
			ar.Phase = game.PlacementPhase
			gs.runPlacementPhase(ar)
		} else {
			gs.startNewRound(ar)
		}
		return
	}

	owner := ar.PlayerByUnitID(activeUnit.ID)
	if owner == nil {
		return
	}

	gs.sendToPlayer(ar, owner.ID, OutMessage{
		Action: PlayUnitAction,
		Data:   game.PlayUnitPayload{UnitID: activeUnit.ID},
	})
	gs.sendToOpponent(ar, owner.ID, OutMessage{Action: WaitingForOpponent})
}

func (gs *GameServer) runPlacementPhase(ar *game.Arena) {
	p1Done := ar.IsPlayerPlacementDone(0)
	p2Done := ar.IsPlayerPlacementDone(1)

	if p1Done && p2Done {
		gs.startNewRound(ar)
		return
	}

	actorIdx := ar.PlacementActorIndex()
	actor := ar.Players[actorIdx]
	other := ar.Players[1-actorIdx]

	gs.sendToPlayer(ar, actor.ID, OutMessage{Action: PlaceUnitAction})
	gs.sendToPlayer(ar, other.ID, OutMessage{Action: WaitingForOpponent})
}

func (gs *GameServer) startNewRound(ar *game.Arena) {
	ar.CurrentRound++
	ar.Phase = game.PlayPhase

	for _, u := range ar.UnitsQueue {
		u.CurrentAP = u.BaseAP
		u.CurrentMP = u.BaseMP
	}
	if len(ar.UnitsQueue) > 0 {
		ar.ActiveUnitID = ar.UnitsQueue[0].ID
	} else {
		ar.ActiveUnitID = ds.NilID
	}

	if ar.UnitsPerPlacementPhase > 1 {
		ar.UnitsPerPlacementPhase--
	}
	ar.Players[0].UnitsPlacedThisRound = 0
	ar.Players[1].UnitsPlacedThisRound = 0

	gs.hub.Broadcast(ar.ID, OutMessage{Action: NewRound})
	gs.advanceGameLoop(ar)
}

func (gs *GameServer) sendToPlayer(ar *game.Arena, playerID ds.ID, msg OutMessage) {
	c := gs.hub.ClientByPlayerID(playerID)
	if c != nil {
		c.Send(msg)
	}
}

func (gs *GameServer) sendToOpponent(ar *game.Arena, playerID ds.ID, msg OutMessage) {
	for _, p := range ar.Players {
		if p.ID != playerID {
			gs.sendToPlayer(ar, p.ID, msg)
			return
		}
	}
}

type useAbilityReq struct {
	UnitID    ds.ID          `json:"unit_id"`
	AbilityID ability.ID     `json:"ability_id"`
	Target    *game.HexCoord `json:"target,omitempty"`
}

func (gs *GameServer) handlePlaceUnit(c *Client, raw json.RawMessage) {
	var req game.UnitPlacedPayload
	if err := json.Unmarshal(raw, &req); err != nil {
		c.Send(errMsg("invalid unit_placed payload"))
		return
	}

	ar := gs.matchmaker.Arena(c.ArenaID)
	if ar == nil {
		c.Send(errMsg("arena not found"))
		return
	}

	u, err := ar.PlaceUnitFromHandToBoard(req.TemplateID, req.Coord, c.PlayerID)
	if err != nil {
		c.Send(errMsg(err.Error()))
		return
	}

	gs.sendToOpponent(ar, c.PlayerID, OutMessage{
		Action: UnitPlacedAction,
		Data: game.PlaceUnitPayload{
			Coord: req.Coord,
			Unit:  u,
		},
	})

	gs.advanceGameLoop(ar)
}

func (gs *GameServer) handleUnitMoved(c *Client, raw json.RawMessage) {
	var req game.UnitMovedPayload
	if err := json.Unmarshal(raw, &req); err != nil {
		c.Send(errMsg("invalid unit_moved payload"))
		return
	}

	ar := gs.matchmaker.Arena(c.ArenaID)
	if ar == nil {
		c.Send(errMsg("arena not found"))
		return
	}

	err := ar.MoveUnit(req.UnitID, req.Coord, c.PlayerID)
	if err != nil {
		c.Send(errMsg(err.Error()))
		return
	}

	gs.sendToOpponent(ar, c.PlayerID, OutMessage{
		Action: UnitMovedAction,
		Data: game.UnitMovedPayload{
			UnitID: req.UnitID,
			Coord:  req.Coord,
		},
	})
}

func (gs *GameServer) useAbility(c *Client, raw json.RawMessage) {
	var req useAbilityReq
	if err := json.Unmarshal(raw, &req); err != nil {
		c.Send(errMsg("invalid use_ability payload"))
		return
	}

	ar := gs.matchmaker.Arena(c.ArenaID)
	if ar == nil {
		c.Send(errMsg("arena not found"))
		return
	}

	unit := ar.ActingUnit()
	if unit == nil {
		c.Send(errMsg("unit not found"))
		return
	}

	err := unit.ValidateAbilityUse(req.AbilityID)
	if err != nil {
		c.Send(errMsg(err.Error()))
		return
	}

	// find ability
	ab, ok := ability.Abilities[req.AbilityID]
	if !ok {
		c.Send(errMsg("unknown ability"))
		return
	}

	_ = ab

	// pass ability down to ability execution pipeline
}

func (gs *GameServer) handleEndTurn(c *Client) {
	ar := gs.matchmaker.Arena(c.ArenaID)
	if ar == nil {
		c.Send(errMsg("arena not found"))
		return
	}

	states, err := ar.EndTurn(c.PlayerID)
	if err != nil {
		c.Send(errMsg(err.Error()))
		return
	}

	if len(states) > 0 {
		data, _ := json.Marshal(states)
		gs.hub.Broadcast(ar.ID, OutMessage{Action: ApplyStateAction, Data: data})
	}

	gs.advanceGameLoop(ar)
}

func errMsg(msg string) OutMessage {
	return OutMessage{Action: "error", Data: map[string]string{"message": msg}}
}

func newGamePayload(arena *game.Arena, myIndex int) game.NewGamePayload {
	cells := make(game.BoardCells, len(arena.Board.Cells))

	for coord, cell := range arena.Board.Cells {
		if cell == nil {
			continue
		}
		c := *cell
		if cell.IsSafeZone && cell.SafeZonePlayer != myIndex {
			c.IsSafeZone = false
		}
		cells[coord] = &c
	}

	return game.NewGamePayload{
		TurnTimeSeconds: game.TurnTimeSeconds,
		ArenaID:         arena.ID,
		Phase:           arena.Phase,
		IsMyTurn:        arena.ActivePlayer == myIndex,
		Board:           &game.Board{Cells: cells},
		Player:          arena.Players[myIndex],
		Opponent:        arena.Players[1-myIndex].Name,
	}
}
