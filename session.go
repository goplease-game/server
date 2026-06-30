// Package game provides core game logic
package game

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/goplease-game/server/ability"
	"github.com/goplease-game/server/ability/status"
	"github.com/goplease-game/server/api"
	"github.com/goplease-game/server/ds"
)

// Event is an outbound message from a Session directed at a specific player.
type Event struct {
	PlayerID ds.ID
	Msg      api.OutMessage
}

// Session runs a self-contained game between two players without any
// network, hub, or matchmaking dependencies.
type Session struct {
	Arena      *Arena
	P1Events   chan api.OutMessage
	P2Events   chan api.OutMessage
	unitCache  map[ds.ID]*Unit
	Log        *Log
	timers     map[ds.ID]*time.Timer
	OnGameOver func()
}

// NewSession creates a Session for the given players and starts the
// placement phase by notifying both players of the initial game state.
func NewSession(p1, p2 *Player) *Session {
	s := NewSessionFromSnapshot(NewArena(p1, p2))
	s.Start()
	return s
}

// NewSessionFromSnapshot creates a Session from a pre-configured Arena.
// Unlike NewSession, it does not send new_game automatically — call Start() when ready.
func NewSessionFromSnapshot(arena *Arena) *Session {
	if arena.UnitsPerPlacementPhase == 0 {
		arena.UnitsPerPlacementPhase = UnitsPerPlacementPhase
	}
	if arena.CurrentRound == 0 {
		arena.CurrentRound = 1
	}

	return &Session{
		Arena:     arena,
		P1Events:  make(chan api.OutMessage, 128),
		P2Events:  make(chan api.OutMessage, 128),
		Log:       NewGameLog(),
		unitCache: map[ds.ID]*Unit{},
		timers:    make(map[ds.ID]*time.Timer),
	}
}

// Start sends the initial game state to both players and begins the game loop
// if the arena is already in PlayPhase.
// Start sends the initial game state to both players.
func (s *Session) Start() {
	s.send(s.Arena.Players[0].ID, api.OutMessage{
		Action: api.NewGameAction,
		Data:   newGamePayload(s.Arena, 0),
	})

	s.send(s.Arena.Players[1].ID, api.OutMessage{
		Action: api.NewGameAction,
		Data:   newGamePayload(s.Arena, 1),
	})

	if s.Arena.DisableBot {
		s.Arena.MarkPlayerReady(s.Arena.Players[1].ID)
	}
}

// Handle processes an inbound action from the given player.
func (s *Session) Handle(playerID ds.ID, action api.Action, data json.RawMessage) {
	switch action {
	case api.ReadyToPlay:
		s.handleReadyToPlay(playerID)
	case api.UnitPlacedAction:
		s.handlePlaceUnit(playerID, data)
	case api.UnitMovedAction:
		s.handleUnitMoved(playerID, data)
	case api.EndTurnAction:
		s.handleEndTurn(playerID)
	case api.UseAbilityAction:
		s.handleUseAbility(playerID, data)
	case api.SurrenderAction:
		s.handleSurrender(playerID)
	default:
		s.sendErr(playerID, "unknown action: "+string(action))
	}

	s.flushGameLog()
}

// handleReadyToPlay marks the player ready and starts the game loop
// once both players have confirmed.
func (s *Session) handleReadyToPlay(playerID ds.ID) {
	s.send(playerID, api.OutMessage{Action: api.WaitingForOpponent})

	if !s.Arena.MarkPlayerReady(playerID) {
		return
	}
	log.Printf("[session] both ready, advancing game loop")

	s.Log.LogSystem("Game started.")
	s.Log.LogSystem(roundTag(s.Arena.CurrentRound))

	s.advanceGameLoop()
}

// handlePlaceUnit places a unit from the player's hand onto the board.
func (s *Session) handlePlaceUnit(playerID ds.ID, data json.RawMessage) {
	var req UnitPlacedPayload
	err := json.Unmarshal(data, &req)
	if err != nil {
		s.sendErr(playerID, "invalid unit_placed payload")
		return
	}

	u, err := s.Arena.PlaceUnitFromHandToBoard(req.TemplateID, req.Coord, playerID)
	if err != nil {
		s.sendErr(playerID, err.Error())
		return
	}

	s.unitCache[u.ID] = u

	s.sendToOpponent(playerID, api.OutMessage{
		Action: api.UnitPlacedAction,
		Data:   PlaceUnitPayload{Coord: req.Coord, Unit: u},
	})

	s.Log.LogAction(LogAction{
		Kind:  LogActionPlace,
		Actor: u,
		Coord: &req.Coord,
	})

	s.advanceGameLoop()
}

// handleUnitMoved moves the acting unit to the given coordinate.
func (s *Session) handleUnitMoved(playerID ds.ID, data json.RawMessage) {
	var req UnitMovedPayload
	err := json.Unmarshal(data, &req)
	if err != nil {
		s.sendErr(playerID, "invalid unit_moved payload")
		return
	}

	u := s.Arena.unitByID(req.UnitID)
	if u == nil {
		err = ErrUnitNotFound
		s.sendErr(playerID, err.Error())
		return
	}

	states, err := s.Arena.MoveUnit(req.UnitID, req.Coord, playerID)
	if err != nil {
		s.sendErr(playerID, err.Error())
		return
	}

	s.sendToOpponent(playerID, api.OutMessage{
		Action: api.UnitMovedAction,
		Data:   UnitMovedPayload{UnitID: req.UnitID, Coord: req.Coord},
	})

	s.Log.LogAction(LogAction{
		Kind:  LogActionMove,
		Actor: u,
		Coord: &req.Coord,
	})

	s.broadcastStates(playerID, states)
}

// handleEndTurn ends the current unit's turn and advances the game loop.
func (s *Session) handleEndTurn(playerID ds.ID) {
	s.cancelTimer(s.Arena.ID)

	u := s.Arena.unitByID(s.Arena.ActiveUnitID)
	if u == nil {
		err := ErrNoActingUnit
		s.sendErr(playerID, err.Error())
		return
	}
	s.Log.LogAction(LogAction{
		Kind:  LogActionEndTurn,
		Actor: u,
	})

	states, err := s.Arena.EndTurn(playerID)
	if err != nil {
		s.sendErr(playerID, err.Error())
		return
	}

	s.broadcastStates(playerID, states)

	if s.checkAndHandleGameOver() {
		return
	}

	s.advanceGameLoop()
}

// handleUseAbility executes an ability for the acting unit.
func (s *Session) handleUseAbility(playerID ds.ID, data json.RawMessage) {
	var req UseAbilityPayload
	err := json.Unmarshal(data, &req)
	if err != nil {
		s.sendErr(playerID, "invalid use_ability payload")
		return
	}

	states, err := s.Arena.UseAbility(req, playerID)
	if err != nil {
		s.sendErr(playerID, err.Error())
		return
	}

	u := s.Arena.unitByID(req.UnitID)
	if u == nil {
		err = ErrUnitNotFound
		s.sendErr(playerID, err.Error())
		return
	}

	action := LogAction{
		Kind:      LogActionAbility,
		Actor:     u,
		AbilityID: req.AbilityID,
		Coord:     req.Target,
	}
	if req.Target != nil {
		// should ignore ShadowStep here, because unit moves to selected cell
		// so it becomes: Silver used ShadowStep on Silver
		if target := s.Arena.UnitAt(*req.Target); target != nil && req.AbilityID != ability.ShadowStep {
			action.Target = target
		}
	}
	s.Log.LogAction(action)

	if s.checkAndHandleGameOver() {
		return
	}

	s.broadcastStates(playerID, states)
}

// handleSurrender ends the match with the surrendering player losing.
func (s *Session) handleSurrender(playerID ds.ID) {
	s.send(playerID, api.OutMessage{Action: api.YouLoseAction})
	s.sendToOpponent(playerID, api.OutMessage{Action: api.OppSurrenderedAction})

	u := s.Arena.ActingUnit()
	if u == nil {
		err := ErrNoActingUnit
		s.sendErr(playerID, err.Error())
		return
	}

	s.Log.LogAction(LogAction{
		Kind:  LogActionSurrender,
		Actor: u,
	})

	s.Arena.Phase = GameOverPhase
	s.cancelTimer(s.Arena.ID)
	if s.OnGameOver != nil {
		s.OnGameOver()
	}
}

// advanceGameLoop steps through placement and play phases.
func (s *Session) advanceGameLoop() {
	switch s.Arena.Phase {
	case PlacementPhase:
		s.runPlacementPhase()
	case PlayPhase:
		s.advancePlayPhase()
	case GameOverPhase:
		return
	}
}

// advancePlayPhase activates the next unit or transitions to placement
// if no units remain in the queue.
func (s *Session) advancePlayPhase() {
	activeUnit := s.Arena.ActingUnit()
	if activeUnit == nil {
		if s.Arena.Players[0].HasUnitsInHand() || s.Arena.Players[1].HasUnitsInHand() {
			s.Arena.Phase = PlacementPhase
			s.runPlacementPhase()
		} else {
			s.startNewRound()
		}
		return
	}

	s.broadcast(api.OutMessage{
		Action: api.ActiveUnitChangedAction,
		Data:   ActiveUnitChangedPayload{UnitID: activeUnit.ID},
	})
	s.Log.LogAction(LogAction{
		Kind:  LogActionStartTurn,
		Actor: activeUnit,
	})

	owner, ownerIdx := s.Arena.PlayerByUnitID(activeUnit.ID)
	log.Printf("[session] active unit %s owner: %v", activeUnit.ID, owner)
	if owner == nil {
		log.Printf("[session] owner is nil, returning")
		return
	}

	s.Arena.ActivePlayer = ownerIdx

	states := OnTurnStart(s.Arena, activeUnit)

	if states.HasSkipTurn() {
		s.broadcastStates(owner.ID, states)
		endStates, err := s.Arena.EndTurn(owner.ID)
		if err != nil {
			log.Printf("[session] EndTurn error on skip: %v", err)
			return
		}
		s.broadcastStates(owner.ID, endStates)
		s.advanceGameLoop()
		return
	}

	s.send(owner.ID, api.OutMessage{
		Action: api.PlayUnitAction,
		Data:   PlayUnitPayload{UnitID: activeUnit.ID},
	})
	s.broadcastStates(owner.ID, states)
	s.sendToOpponent(owner.ID, api.OutMessage{Action: api.WaitingForOpponent})
	s.scheduleTimer(s.Arena, activeUnit.ID)
}

// runPlacementPhase directs each player to place units in turn.
func (s *Session) runPlacementPhase() {
	p1Done := s.Arena.IsPlayerPlacementDone(0)
	p2Done := s.Arena.IsPlayerPlacementDone(1)

	if p1Done && p2Done {
		s.startNewRound()
		return
	}

	actorIdx := s.Arena.PlacementActorIndex()
	actor := s.Arena.Players[actorIdx]
	other := s.Arena.Players[1-actorIdx]

	s.send(actor.ID, api.OutMessage{Action: api.PlaceUnitAction})
	s.send(other.ID, api.OutMessage{Action: api.WaitingForOpponent})
}

// startNewRound resets per-round state and begins the next round.
func (s *Session) startNewRound() {
	ar := s.Arena
	ar.CurrentRound++
	ar.Phase = PlayPhase

	s.Log.LogSystem(roundTag(s.Arena.CurrentRound))

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

	s.broadcast(api.OutMessage{Action: api.NewRound})
	s.broadcastStates(ar.Players[ar.ActivePlayer].ID, ar.RecalculatePhantomAP())
	s.advanceGameLoop()
}

// checkAndHandleGameOver checks for a winner and sends game-over events.
func (s *Session) checkAndHandleGameOver() bool {
	if s.Arena.DisableGameOver {
		return false
	}

	loserIdx := s.Arena.CheckGameOver()
	if loserIdx < 0 {
		return false
	}

	loser := s.Arena.Players[loserIdx]
	winner := s.Arena.Players[1-loserIdx]

	s.Log.LogSystem(`%s wins`, playerTag(winner))

	s.send(loser.ID, api.OutMessage{Action: api.YouLoseAction})
	s.send(winner.ID, api.OutMessage{Action: api.YouWinAction})
	s.Arena.Phase = GameOverPhase
	s.cancelTimer(s.Arena.ID)
	if s.OnGameOver != nil {
		s.OnGameOver()
	}

	return true
}

// send enqueues an outbound event directed at a specific player.
func (s *Session) send(playerID ds.ID, msg api.OutMessage) {
	log.Printf("[session] send to %s: %s\n", playerID, msg.Action)
	if playerID == s.Arena.Players[0].ID {
		s.P1Events <- msg
	} else {
		s.P2Events <- msg
	}
}

// sendToOpponent enqueues an outbound event directed at the opponent of playerID.
func (s *Session) sendToOpponent(playerID ds.ID, msg api.OutMessage) {
	for _, p := range s.Arena.Players {
		if p.ID != playerID {
			s.send(p.ID, msg)
			return
		}
	}
}

// broadcast enqueues an outbound event directed at all players in the session.
func (s *Session) broadcast(msg api.OutMessage) {
	for _, p := range s.Arena.Players {
		s.send(p.ID, msg)
	}
}

// broadcastStates routes ApplyStates to Self, Opponent, and Global channels,
// and logs any loggable effects from the states.
func (s *Session) broadcastStates(playerID ds.ID, states ApplyStates) {
	s.logStates(states) // log before sending so flush picks them up together

	if len(states.Global) > 0 {
		s.broadcast(api.OutMessage{Action: api.ApplyStateAction, Data: states.Global})
	}
	if len(states.Self) > 0 {
		s.send(playerID, api.OutMessage{Action: api.ApplyStateAction, Data: states.Self})
	}
	if len(states.Opponent) > 0 {
		s.sendToOpponent(playerID, api.OutMessage{Action: api.ApplyStateAction, Data: states.Opponent})
	}
}

// sendErr sends an error message to the given player.
func (s *Session) sendErr(playerID ds.ID, msg string) {
	s.Log.LogError(playerID, msg)
	s.send(playerID, api.OutMessage{
		Action: api.ErrorAction,
		Data:   map[string]string{"message": msg},
	})
}

// scheduleTimer sets a turn timer that force-ends the turn on expiry.
func (s *Session) scheduleTimer(ar *Arena, unitID ds.ID) {
	if ar.DisableTurnTimer {
		return
	}

	s.cancelTimer(ar.ID)
	t := time.AfterFunc(TurnTimeSeconds*time.Second, func() {
		if ar.ActiveUnitID != unitID {
			return
		}
		owner := ar.Players[ar.ActivePlayer]
		states, err := ar.EndTurn(owner.ID)
		if err != nil {
			log.Printf("[session] EndTurn error on timeout: %v", err)
			return
		}
		s.broadcastStates(owner.ID, states)
		s.advanceGameLoop()
	})
	s.timers[ar.ID] = t
}

// cancelTimer stops and removes the active turn timer for the given arena.
func (s *Session) cancelTimer(arenaID ds.ID) {
	if t, ok := s.timers[arenaID]; ok {
		t.Stop()
		delete(s.timers, arenaID)
	}
}

// flushGameLog drains all pending log messages and delivers each to the
// appropriate player(s). Call at the end of every Handle case.
func (s *Session) flushGameLog() {
	for {
		select {
		case msg := <-s.Log.Out:
			out := api.OutMessage{Action: api.GameLogAction, Data: msg}
			if msg.Recipient != ds.NilID {
				s.send(msg.Recipient, out)
			} else {
				s.broadcast(out)
			}
		default:
			return
		}
	}
}

func (s *Session) logStates(states ApplyStates) {
	all := append(append(states.Global, states.Self...), states.Opponent...)
	for _, st := range all {
		s.logState(st)
	}
}

func (s *Session) logState(st ApplyState) {
	if st.ToUnitID.IsNil() {
		return
	}
	unit := unitTag(s.unitCache[st.ToUnitID])

	if st.ChangeHP != nil {
		if *st.ChangeHP < 0 {
			s.Log.LogEffect(fmt.Sprintf("%s took <damage>%d</damage> damage", unit, -*st.ChangeHP))
		} else {
			s.Log.LogEffect(fmt.Sprintf("%s healed <hp>+%d</hp>", unit, *st.ChangeHP))
		}
	}
	if st.ChangeShield != nil {
		if *st.ChangeShield > 0 {
			s.Log.LogEffect(fmt.Sprintf("%s gained <shield>+%d</shield> shield", unit, *st.ChangeShield))
		} else if *st.ChangeShield < 0 {
			s.Log.LogEffect(fmt.Sprintf("%s lost <damage>%d</damage> shield", unit, -*st.ChangeShield))
		}
	}
	if st.AddStatus != nil {
		name := status.ByType(*st.AddStatus).Name
		s.Log.LogEffect(fmt.Sprintf("%s received status <ability>%s</ability>", unit, name))
	}
	if st.RemoveStatus != nil {
		name := status.ByType(*st.RemoveStatus).Name
		s.Log.LogEffect(fmt.Sprintf("%s lost status <ability>%s</ability>", unit, name))
	}
	if st.ChangeAP != nil {
		s.Log.LogEffect(fmt.Sprintf("%s received <ap>%d</ap> AP", unit, *st.ChangeAP))
	}
	if st.IsDead {
		s.Log.LogEffect(unit + " died")
	}
	if st.UseAbility != nil {
		ab := ability.ByID(st.UseAbility.AbilityID)
		unitID := st.UseAbility.UnitID
		unit := unitTag(s.unitCache[unitID])

		if ab.IsPassive { // active abilities will be logged by Handler
			name := fmt.Sprintf("<ability>%s</ability>", ab.Name)
			if st.UseAbility.Target != nil {
				target := s.Arena.UnitAt(*st.UseAbility.Target)
				if target != nil {
					targetTag := unitTag(target)
					s.Log.LogEffect(fmt.Sprintf("%s triggered %s on %s", unit, name, targetTag))
				} else {
					s.Log.LogEffect(fmt.Sprintf("%s triggered %s at %s", unit, name, st.UseAbility.Target))
				}
			} else {
				s.Log.LogEffect(fmt.Sprintf("%s triggered %s", unit, name))
			}
		}
	}
}

// newGamePayload builds the initial game state payload for the given player slot.
func newGamePayload(arena *Arena, myIndex int) NewGamePayload {
	cells := make(BoardCells, len(arena.Board.Cells))
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

	turnTime := TurnTimeSeconds
	if arena.DisableTurnTimer {
		turnTime = 0
	}

	return NewGamePayload{
		TurnTimeSeconds:            turnTime,
		MaxPhantomAPPerUnitPerTurn: MaxPhantomAPPerUnitPerTurn,
		ArenaID:                    arena.ID,
		Phase:                      arena.Phase,
		Board:                      &Board{Cells: cells},
		Queue:                      arena.UnitsQueue,
		Player:                     arena.Players[myIndex],
		Opponent:                   arena.Players[1-myIndex].Name,
		Round:                      arena.CurrentRound,
	}
}
