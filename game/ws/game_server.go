package ws

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/game"
	"github.com/ognev-dev/goplease/game/api"
	"github.com/ognev-dev/goplease/game/match"
)

// GameServer wires the hub to the game layer.
type GameServer struct {
	hub        *Hub
	matchmaker *match.Matchmaker
	log        *ActionLogger
}

func NewGameServer(hub *Hub) *GameServer {
	gs := &GameServer{
		hub: hub,
		log: NewActionLogger(true),
	}
	gs.matchmaker = match.New(gs.notifyMatchFound)
	return gs
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

func (gs *GameServer) notifyMatchFound(arena *game.Arena, playerIndex int) {
	p := arena.Players[playerIndex]
	client := gs.hub.ClientByPlayerID(p.ID)
	if client == nil {
		return
	}
	client.ArenaID = arena.ID
	client.Send(api.OutMessage{
		Action: api.NewGameAction,
		Data:   NewGamePayload(arena, playerIndex),
	})
}

func (gs *GameServer) onConnect(c *Client) {
	c.Send(api.OutMessage{
		Action: api.ConnectedAction,
		Data:   ConnectedResponse{PlayerID: c.PlayerID},
	})
}

func (gs *GameServer) onDisconnect(c *Client) {
	gs.matchmaker.Cancel(c.PlayerID)
	if !c.ArenaID.IsNil() {
		gs.hub.Broadcast(c.ArenaID, api.OutMessage{
			Action: api.OppDisconnectedAction,
			Data:   DisconnectResponse{PlayerID: c.PlayerID},
		})
	}
}

func (gs *GameServer) onMessage(c *Client, msg api.InMessage) {
	ar := gs.matchmaker.Arena(c.ArenaID)
	gs.log.Received(gs.playerName(ar, c.PlayerID), string(msg.Action))

	switch msg.Action {
	case api.NewGameAction:
		gs.prepareNewGame(c)

	case api.ReadyToPlay:
		gs.handleReadyToPlay(c)

	case api.CancelMatchAction:
		gs.matchmaker.Cancel(c.PlayerID)
		c.Send(api.OutMessage{Action: api.MatchCancelledAction, Data: nil})

	case api.UnitPlacedAction:
		gs.handlePlaceUnit(c, msg.Data)

	case api.UnitMovedAction:
		gs.handleUnitMoved(c, msg.Data)

	case api.EndTurnAction:
		gs.handleEndTurn(c)

	case api.UseAbilityAction:
		gs.handleUseAbility(c, msg.Data)

	case api.SurrenderAction:
		gs.handleSurrender(c)

	default:
		c.Send(api.OutMessage{
			Action: api.ErrorAction,
			Data: ErrorResponse{
				Message: "[server] unknown action: " + string(msg.Action),
			},
		})
	}
}

func (gs *GameServer) prepareNewGame(c *Client) {
	gs.matchmaker.Enqueue(c.PlayerID, false, func(room *game.Arena, playerIndex int) {
		p := room.Players[playerIndex]
		client := gs.hub.ClientByPlayerID(p.ID)
		if client == nil {
			return
		}

		client.ArenaID = room.ID
		client.Send(api.OutMessage{
			Action: api.NewGameAction,
			Data:   NewGamePayload(room, playerIndex),
		})
	})

	c.Send(api.OutMessage{Action: api.SearchingOppAction, Data: nil})
}

func (gs *GameServer) handleReadyToPlay(c *Client) {
	ar := gs.matchmaker.ArenaByPlayerID(c.PlayerID)
	if ar == nil {
		c.Send(errMsg("arena not found"))
		return
	}

	c.Send(api.OutMessage{Action: api.WaitingForOpponent})

	if !ar.MarkPlayerReady(c.PlayerID) {
		return
	}

	gs.advanceGameLoop(ar)
}

func (gs *GameServer) advanceGameLoop(ar *game.Arena) {
	switch ar.Phase {
	case game.PlacementPhase:
		gs.runPlacementPhase(ar)
	case game.PlayPhase:
		gs.advancePlayPhase(ar)
	case game.GameOverPhase:
		return
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

	gs.hub.Broadcast(ar.ID, api.OutMessage{
		Action: api.ActiveUnitChangedAction,
		Data:   game.ActiveUnitChangedPayload{UnitID: activeUnit.ID},
	})

	owner, ownerIdx := ar.PlayerByUnitID(activeUnit.ID)
	if owner == nil {
		return
	}

	ar.ActivePlayer = ownerIdx

	states := game.OnTurnStart(ar, activeUnit)

	if states.HasSkipTurn() {
		gs.broadcastStates(ar, owner.ID, states)
		endStates, err := ar.EndTurn(owner.ID)
		if err != nil {
			log.Printf("[gameloop] EndTurn error on skip: %v", err)
			return
		}
		gs.broadcastStates(ar, owner.ID, endStates)
		gs.advanceGameLoop(ar)
		return
	}

	gs.sendToPlayer(ar, owner.ID, api.OutMessage{
		Action: api.PlayUnitAction,
		Data:   game.PlayUnitPayload{UnitID: activeUnit.ID},
	})
	gs.broadcastStates(ar, owner.ID, states)
	gs.sendToOpponent(ar, owner.ID, api.OutMessage{Action: api.WaitingForOpponent})
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

	gs.sendToPlayer(ar, actor.ID, api.OutMessage{Action: api.PlaceUnitAction})
	gs.sendToPlayer(ar, other.ID, api.OutMessage{Action: api.WaitingForOpponent})
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

	gs.hub.Broadcast(ar.ID, api.OutMessage{Action: api.NewRound})
	gs.broadcastStates(ar, ar.Players[ar.ActivePlayer].ID, ar.RecalculatePhantomAP())

	gs.advanceGameLoop(ar)
}

func (gs *GameServer) sendToPlayer(ar *game.Arena, playerID ds.ID, msg api.OutMessage) {
	gs.log.Sent(gs.playerName(ar, playerID), string(msg.Action))
	c := gs.hub.ClientByPlayerID(playerID)
	if c != nil {
		c.Send(msg)
	}
}

func (gs *GameServer) sendToOpponent(ar *game.Arena, playerID ds.ID, msg api.OutMessage) {
	for _, p := range ar.Players {
		if p.ID != playerID {
			gs.sendToPlayer(ar, p.ID, msg)
			return
		}
	}
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

	gs.log.Event(gs.playerName(ar, c.PlayerID),
		fmt.Sprintf("placed %s at %s", u.Name, req.Coord))

	gs.sendToOpponent(ar, c.PlayerID, api.OutMessage{
		Action: api.UnitPlacedAction,
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

	pl, _ := ar.PlayerByID(c.PlayerID)
	prevPos := "null"
	uName := "null"
	u := ar.ActingUnit()
	if u != nil {
		prevPos = u.PosVal().String()
		uName = u.Name
	}

	gs.log.Event(pl.Name, fmt.Sprintf("%s moved from %s to %s", uName, prevPos, req.Coord))

	states, err := ar.MoveUnit(req.UnitID, req.Coord, c.PlayerID)
	if err != nil {
		c.Send(errMsg(err.Error()))
		return
	}

	gs.sendToOpponent(ar, c.PlayerID, api.OutMessage{
		Action: api.UnitMovedAction,
		Data: game.UnitMovedPayload{
			UnitID: req.UnitID,
			Coord:  req.Coord,
		},
	})

	gs.broadcastStates(ar, c.PlayerID, states)
}

func (gs *GameServer) handleUseAbility(c *Client, raw json.RawMessage) {
	var req game.UseAbilityPayload
	if err := json.Unmarshal(raw, &req); err != nil {
		c.Send(errMsg("invalid use_ability payload"))
		return
	}

	ar := gs.matchmaker.Arena(c.ArenaID)
	if ar == nil {
		c.Send(errMsg("arena not found"))
		return
	}

	gs.logAbilityUse(c, req)

	states, err := ar.UseAbility(req, c.PlayerID)
	if err != nil {
		c.Send(errMsg(err.Error()))
		return
	}

	if gs.checkAndHandleGameOver(ar) {
		return
	}

	gs.broadcastStates(ar, c.PlayerID, states)
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

	gs.log.Event(gs.playerName(ar, c.PlayerID),
		fmt.Sprintf("ended turn, next active=%s", ar.ActiveUnitID))

	gs.broadcastStates(ar, c.PlayerID, states)

	if gs.checkAndHandleGameOver(ar) {
		return
	}

	gs.advanceGameLoop(ar)
}

func errMsg(msg string) api.OutMessage {
	return api.OutMessage{Action: "error", Data: map[string]string{"message": msg}}
}

func NewGamePayload(arena *game.Arena, myIndex int) game.NewGamePayload {
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
		TurnTimeSeconds:            game.TurnTimeSeconds,
		MaxPhantomAPPerUnitPerTurn: game.MaxPhantomAPPerUnitPerTurn,
		ArenaID:                    arena.ID,
		Phase:                      arena.Phase,
		Board:                      &game.Board{Cells: cells},
		Player:                     arena.Players[myIndex],
		Opponent:                   arena.Players[1-myIndex].Name,
	}
}

func (gs *GameServer) broadcastStates(ar *game.Arena, playerID ds.ID, states game.ApplyStates) {
	if len(states.Global) > 0 {
		gs.hub.Broadcast(ar.ID, api.OutMessage{Action: api.ApplyStateAction, Data: states.Global})
	}
	if len(states.Self) > 0 {
		gs.sendToPlayer(ar, playerID, api.OutMessage{Action: api.ApplyStateAction, Data: states.Self})
	}
	if len(states.Opponent) > 0 {
		gs.sendToOpponent(ar, playerID, api.OutMessage{Action: api.ApplyStateAction, Data: states.Opponent})
	}
}

func (gs *GameServer) playerName(ar *game.Arena, playerID ds.ID) string {
	if ar != nil {
		p, _ := ar.PlayerByID(playerID)
		if p != nil {
			return p.Name
		}
	}
	return playerID.String()[:8]
}

func (gs *GameServer) checkAndHandleGameOver(ar *game.Arena) bool {
	loserIdx := ar.CheckGameOver()
	if loserIdx < 0 {
		return false
	}

	loser := ar.Players[loserIdx]
	winner := ar.Players[1-loserIdx]

	gs.sendToPlayer(ar, loser.ID, api.OutMessage{Action: api.YouLoseAction})
	gs.sendToPlayer(ar, winner.ID, api.OutMessage{Action: api.YouWinAction})

	ar.Phase = game.GameOverPhase
	gs.matchmaker.CloseArena(ar.ID)

	return true
}

func (gs *GameServer) handleSurrender(c *Client) {
	ar := gs.matchmaker.ArenaByPlayerID(c.PlayerID)
	if ar == nil {
		return
	}

	gs.sendToPlayer(ar, c.PlayerID, api.OutMessage{Action: api.YouLoseAction})
	gs.sendToOpponent(ar, c.PlayerID, api.OutMessage{Action: api.OppSurrenderedAction})

	ar.Phase = game.GameOverPhase
	gs.matchmaker.CloseArena(ar.ID)
}

func (gs *GameServer) logAbilityUse(c *Client, req game.UseAbilityPayload) {
	ar := gs.matchmaker.Arena(c.ArenaID)

	var at string
	if req.Target != nil {
		at = " at " + req.Target.String()
		if u := ar.UnitAt(*req.Target); u != nil {
			at = fmt.Sprintf(" on unit %s%s", u.Name, at)
		}
	}

	pl, _ := ar.PlayerByID(c.PlayerID)
	uName := "null"
	pos := "null"
	if u := ar.ActingUnit(); u != nil {
		uName = u.Name
		pos = u.PosVal().String()
	}

	gs.log.Event(pl.Name, fmt.Sprintf("%s (%s) used ability '%s'"+at, uName, pos, req.AbilityID))

}
