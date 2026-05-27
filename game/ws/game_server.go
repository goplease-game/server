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
	SearchingOppAction    Action = "searching_opp"
	PlaceUnitAction       Action = "place_unit"
	UnitPlacedAction      Action = "unit_placed"
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
	if c.RoomID != "" {
		gs.hub.Broadcast(c.RoomID, OutMessage{
			Action: OppDisconnectedAction,
			Data:   DisconnectResponse{PlayerID: c.PlayerID},
		})
	}
}

func (gs *GameServer) onMessage(c *Client, msg InMessage) {
	switch msg.Action {

	case NewGameAction:
		gs.prepareNewGame(c)

	case CancelMatchAction:
		gs.matchmaker.Cancel(c.PlayerID)
		c.Send(OutMessage{Action: MatchCancelledAction, Data: nil})

	case PlaceUnitAction:
		gs.handlePlaceUnit(c, msg.Data)

	case UseAbility:
		gs.useAbility(c, msg.Data)

	case "end_turn": // TODO
		gs.handleEndTurn(c)

	default:
		c.Send(OutMessage{
			Action: ErrorAction,
			Data: ErrorResponse{
				Message: "unknown action: " + string(msg.Action),
			},
		})
	}
}

func (gs *GameServer) prepareNewGame(c *Client) {
	gs.matchmaker.Enqueue(c.PlayerID, func(room *game.Room, playerIndex int) {
		for idx, p := range room.Players {
			client := gs.hub.ClientByPlayerID(p.ID)
			if client == nil {
				continue
			}

			client.RoomID = room.ID
			client.Send(OutMessage{
				Action: NewGameAction,
				Data:   newGamePayload(room, idx),
			})
		}
	})

	c.Send(OutMessage{Action: SearchingOppAction, Data: nil})
}

type placeUnitReq struct {
	UnitID ds.ID `json:"unit_id"`
	Col    int   `json:"col"`
	Row    int   `json:"row"`
}

type useAbilityReq struct {
	UnitID    ds.ID          `json:"unit_id"`
	AbilityID ability.ID     `json:"ability_id"`
	Target    *game.HexCoord `json:"target,omitempty"`
}

func (gs *GameServer) handlePlaceUnit(c *Client, raw json.RawMessage) {
	var req placeUnitReq
	if err := json.Unmarshal(raw, &req); err != nil {
		c.Send(errMsg("invalid place_unit payload"))
		return
	}

	room := gs.matchmaker.Room(c.RoomID)
	if room == nil {
		c.Send(errMsg("room not found"))
		return
	}

	if err := room.PlaceUnit(c.PlayerID, game.PlacementAction{
		UnitID: req.UnitID,
		Col:    req.Col,
		Row:    req.Row,
	}); err != nil {
		c.Send(errMsg(err.Error()))
		return
	}

	// TODO response
}

func (gs *GameServer) useAbility(c *Client, raw json.RawMessage) {
	var req useAbilityReq
	if err := json.Unmarshal(raw, &req); err != nil {
		c.Send(errMsg("invalid use_ability payload"))
		return
	}

	room := gs.matchmaker.Room(c.RoomID)
	if room == nil {
		c.Send(errMsg("room not found"))
		return
	}

	unit, err := room.ActingUnit()
	if err != nil {
		c.Send(errMsg(err.Error()))
		return
	}

	err = unit.ValidateAbilityUse(req.AbilityID)
	if err != nil {
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
	room := gs.matchmaker.Room(c.RoomID)
	if room == nil {
		c.Send(errMsg("room not found"))
		return
	}

	result, err := room.EndTurn(c.PlayerID)
	if err != nil {
		c.Send(errMsg(err.Error()))
		return
	}

	// Broadcast simulation events to both players.
	gs.hub.Broadcast(room.ID, OutMessage{
		Action: "turn_result",
		Data:   result,
	})

	if result.IsOver {
		gs.hub.Broadcast(room.ID, OutMessage{
			Action: "game_over",
			Data: map[string]any{
				"winner": result.Winner,
				"reason": result.Reason,
			},
		})
		gs.matchmaker.CloseRoom(room.ID)
		return
	}

	// TODO

	// If the next active player is a bot, trigger its turn automatically.
	gs.matchmaker.MaybeTriggerBot(room)
}

func errMsg(msg string) OutMessage {
	return OutMessage{Action: "error", Data: map[string]string{"message": msg}}
}

func newGamePayload(room *game.Room, myIndex int) game.NewGamePayload {
	preparedBoard := game.NewBoard()

	for coord, cell := range room.Board.Cells {
		if cell == nil {
			continue
		}

		c := *cell

		// Convert axial -> visual column
		col := coord.Q + coord.R/2
		if myIndex == 0 {
			c.IsSafeZone = col < game.SafeZoneSize
		} else {
			c.IsSafeZone = col >= (game.BoardColumns - game.SafeZoneSize)
		}

		preparedBoard.Cells[coord] = &c
	}

	return game.NewGamePayload{
		RoomID:   room.ID,
		Phase:    room.Phase,
		IsMyTurn: room.ActivePlayer == myIndex,
		Board:    preparedBoard,
		Player:   room.Players[myIndex],
		Opponent: room.Players[1-myIndex].Name,
	}
}
