// Package ws ...
package ws

import (
	"encoding/json"
	"errors"
	"sync"

	game "github.com/goplease-game/server"
	"github.com/goplease-game/server/api"
	"github.com/goplease-game/server/ds"
	"github.com/goplease-game/server/match"
)

// GameServer wires the WebSocket communication layer with the core game domain.
type GameServer struct {
	hub           *Hub
	matchmaker    *match.Matchmaker
	log           *ActionLogger
	sessions      sync.Map // ds.ID → *game.Session
	playerSession sync.Map // ds.ID → *game.Session
}

// NewGameServer instantiates and returns a fully initialized GameServer.
func NewGameServer(hub *Hub) *GameServer {
	gs := &GameServer{
		hub: hub,
		log: NewActionLogger(true),
	}
	gs.matchmaker = match.New(gs.notifyMatchFound)
	return gs
}

// Run boots up the central server message polling thread.
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

// onConnect sends the connected message to a newly joined client.
func (gs *GameServer) onConnect(c *Client) {
	c.Send(api.OutMessage{
		Action: api.ConnectedAction,
		Data:   ConnectedResponse{PlayerID: c.PlayerID},
	})
}

// onDisconnect removes the client from matchmaking and notifies the opponent.
func (gs *GameServer) onDisconnect(c *Client) {
	gs.matchmaker.Cancel(c.PlayerID)

	if !c.ArenaID.IsNil() {
		gs.hub.Broadcast(c.ArenaID, api.OutMessage{
			Action: api.OppDisconnectedAction,
			Data:   DisconnectResponse{PlayerID: c.PlayerID},
		})
	}
}

// onMessage dispatches an inbound client message to the appropriate handler.
func (gs *GameServer) onMessage(c *Client, msg api.InMessage) {
	gs.log.Received(c.PlayerID.String()[:8], string(msg.Action))

	switch msg.Action {
	case api.NewGameAction:
		gs.prepareNewGame(c)

	case api.CancelMatchAction:
		gs.matchmaker.Cancel(c.PlayerID)
		c.Send(api.OutMessage{Action: api.MatchCancelledAction})

	case api.CreateFriendGameAction:
		gs.createFriendGame(c)

	case api.JoinFriendGameAction:
		gs.joinFriendGame(c, msg.Data)

	case api.CancelFriendRoomAction:
		gs.matchmaker.CancelFriendRoom(c.PlayerID)

	default:
		session := gs.sessionByPlayerID(c.PlayerID)
		if session == nil {
			c.Send(errMsg("session not found"))
			return
		}
		session.Handle(c.PlayerID, msg.Action, msg.Data)
	}
}

// prepareNewGame enqueues the client in matchmaking.
func (gs *GameServer) prepareNewGame(c *Client) {
	gs.matchmaker.Enqueue(c.PlayerID, false, gs.notifyMatchFound)
	c.Send(api.OutMessage{Action: api.SearchingOppAction})
}

// notifyMatchFound is the matchmaker callback — creates a Session and wires both players.
func (gs *GameServer) notifyMatchFound(arena *game.Arena, playerIndex int) {
	p := arena.Players[playerIndex]
	client := gs.hub.ClientByPlayerID(p.ID)
	if client == nil {
		return
	}
	gs.startSession(arena, client)
}

// startSession creates a Session for the arena and wires the WebSocket client to it.
func (gs *GameServer) startSession(arena *game.Arena, c *Client) {
	_, loaded := gs.sessions.LoadOrStore(arena.ID, (*game.Session)(nil))
	if !loaded {
		p1 := arena.Players[0]
		p2 := arena.Players[1]
		session := game.NewSessionFromSnapshot(arena)
		session.OnGameOver = func() {
			gs.sessions.Delete(arena.ID)
			gs.playerSession.Delete(p1.ID)
			gs.playerSession.Delete(p2.ID)
			gs.matchmaker.CloseArena(arena.ID)
		}
		gs.sessions.Store(arena.ID, session)
		gs.playerSession.Store(p1.ID, session)
		gs.playerSession.Store(p2.ID, session)

		go gs.pumpEvents(session.P1Events, p1.ID)
		go gs.pumpEvents(session.P2Events, p2.ID)

		session.Start()
	}

	c.ArenaID = arena.ID
}

// pumpEvents forwards Session events to the corresponding WebSocket client.
func (gs *GameServer) pumpEvents(events <-chan api.OutMessage, playerID ds.ID) {
	for msg := range events {
		c := gs.hub.ClientByPlayerID(playerID)
		if c != nil {
			c.Send(msg)
		}
	}
}

// sessionByPlayerID returns the active Session for the given player.
func (gs *GameServer) sessionByPlayerID(playerID ds.ID) *game.Session {
	v, ok := gs.playerSession.Load(playerID)
	if !ok {
		return nil
	}

	return v.(*game.Session)
}

// createFriendGame creates a friend room and returns the join code to the client.
func (gs *GameServer) createFriendGame(c *Client) {
	code := gs.matchmaker.CreateFriendRoom(c.PlayerID, gs.notifyMatchFound)
	c.Send(api.OutMessage{
		Action: api.FriendRoomCreatedAction,
		Data:   FriendRoomCreatedResponse{JoinCode: code},
	})
}

// joinFriendGame finds the room by code and starts the game for both players.
func (gs *GameServer) joinFriendGame(c *Client, data json.RawMessage) {
	var req JoinFriendGameRequest
	err := json.Unmarshal(data, &req)
	if err != nil {
		c.Send(errMsg("invalid request"))
		return
	}

	err = gs.matchmaker.JoinFriendRoom(req.JoinCode, c.PlayerID, gs.notifyMatchFound)
	switch {
	case errors.Is(err, match.ErrFriendRoomNotFound):
		c.Send(api.OutMessage{Action: api.FriendRoomNotFound})
	case errors.Is(err, match.ErrFriendRoomExpired):
		c.Send(api.OutMessage{Action: api.FriendRoomExpiredAction})
	case err != nil:
		c.Send(errMsg(err.Error()))
	}
}

// errMsg builds a standard error OutMessage.
func errMsg(msg string) api.OutMessage {
	return api.OutMessage{Action: api.ErrorAction, Data: map[string]string{"message": msg}}
}

// ConnectedResponse is sent to a client after successfully joining the server.
type ConnectedResponse struct {
	PlayerID ds.ID `json:"player_id"`
}

// DisconnectResponse is broadcast when a participant drops connection mid-game.
type DisconnectResponse struct {
	PlayerID ds.ID `json:"player_id"`
}

// ErrorResponse holds a standardized error payload.
type ErrorResponse struct {
	Message string `json:"message"`
}

// FriendRoomCreatedResponse is sent after successfully creating a friend room.
type FriendRoomCreatedResponse struct {
	JoinCode string `json:"join_code"`
}

// JoinFriendGameRequest is sent by the client to join a friend room.
type JoinFriendGameRequest struct {
	JoinCode string `json:"join_code"`
}
