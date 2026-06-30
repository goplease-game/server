package game

import (
	"fmt"
	"strings"
	"time"

	"github.com/goplease-game/server/ability"
	"github.com/goplease-game/server/ds"
)

// MessageKind identifies the category of a log message.
type MessageKind string

const (
	// KindAction is a player command that mutates game state.
	// These are the only messages consumed during replay.
	KindAction MessageKind = "action"

	// KindEffect describes the result of an action: damage dealt, shield gained, status applied, etc.
	KindEffect MessageKind = "effect"

	// KindSystem marks structural events: game start, new round, game over.
	KindSystem MessageKind = "system"

	// KindError is an error response to a player command, visible only to the recipient.
	KindError MessageKind = "error"

	// KindChat is a player-to-player or player-to-self message (e.g. command responses).
	KindChat MessageKind = "chat"
)

// LogActionKind identifies what kind of player command was taken.
type LogActionKind string

// Log action kinds.
const (
	LogActionPlace     LogActionKind = "place"
	LogActionMove      LogActionKind = "move"
	LogActionAbility   LogActionKind = "ability"
	LogActionStartTurn LogActionKind = "start_turn"
	LogActionEndTurn   LogActionKind = "end_turn"
	LogActionSurrender LogActionKind = "surrender"
)

// LogAction holds the structured data of a player command.
// It is stored in KindAction messages and consumed during replay.
// ActorID is always set. AbilityID and Coord are optional depending on Kind.
type LogAction struct {
	Kind      LogActionKind
	Actor     *Unit
	AbilityID ability.ID // non-empty only for LogActionAbility
	Coord     *HexCoord  // nil for instant abilities, end_turn, surrender
	Target    *Unit
}

// Render returns a tagged string for the message log.
// The client resolves <unit id="..."> colors based on unit ownership.
func (a LogAction) Render() string {
	actorTag := unitTag(a.Actor)

	switch a.Kind {
	case LogActionPlace:
		if a.Coord != nil {
			return actorTag + " deployed at " + a.Coord.String()
		}
		return actorTag + " deployed"
	case LogActionMove:
		if a.Coord != nil {
			return fmt.Sprintf("%s moved to %s", actorTag, a.Coord)
		}
		return actorTag + " moved"
	case LogActionAbility:
		name := ability.ByID(a.AbilityID).Name
		ab := fmt.Sprintf("<ability>%s</ability>", name)
		if a.Target != nil {
			target := unitTag(a.Target)
			return fmt.Sprintf("%s used %s on %s", actorTag, ab, target)
		}
		if a.Coord != nil {
			return fmt.Sprintf("%s used %s at %s", actorTag, ab, a.Coord)
		}
		return fmt.Sprintf("%s used %s", actorTag, ab)
	case LogActionStartTurn:
		return actorTag + " acting..."
	case LogActionEndTurn:
		return actorTag + " ended turn"
	case LogActionSurrender:
		return actorTag + " surrendered"
	default:
		return actorTag + " acted"
	}
}

// unitTag returns a <unit id="..."> tag for use in log message text.
// The client resolves the color based on whether the unit belongs to the reading player.
func unitTag(u *Unit) string {
	return fmt.Sprintf(`<unit id="%s">%s</unit>`, u.ID, u.Name)
}

// return returns a <player id="..."> tag for use in log message text.
func playerTag(p *Player) string {
	return fmt.Sprintf(`<player id="%s">%s</player>`, p.ID, p.Name)
}

func roundTag(round int) string {
	return fmt.Sprintf(`<round>ROUND %d</round>`, round)
}

// Message is a single entry in the game log.
type Message struct {
	Time      time.Time   `json:"time"`
	Kind      MessageKind `json:"kind"`
	Text      string      `json:"text,omitempty"`      // used by KindEffect, KindSystem, KindError, KindChat
	Action    *LogAction  `json:"-"`                   // non-nil only for KindAction
	Sender    string      `json:"sender,omitempty"`    // player name for KindChat, empty otherwise
	Recipient ds.ID       `json:"recipient,omitempty"` // empty = broadcast; non-empty = only this player sees it
}

const logOutBufSize = 64

// Log is the ordered message log for a session.
// It accumulates game actions (for replay), effects, system events, errors, and chat.
// Out is a channel drained by Session.flushGameLog after each Handle call.
type Log struct {
	messages []Message
	Out      chan Message
}

// NewGameLog creates a Log with a buffered output channel.
func NewGameLog() *Log {
	return &Log{
		Out: make(chan Message, logOutBufSize),
	}
}

// LogAction records a player command. KindAction messages are the only ones
// used during replay — pass them back through Session.Handle to reproduce the game.
func (g *Log) LogAction(action LogAction) {
	g.emit(Message{
		Time:   time.Now(),
		Kind:   KindAction,
		Text:   action.Render(),
		Action: &action,
	})
}

// LogEffect records an effect broadcast to all players.
// text should use semantic tags: <unit id="...">, <ability>, <damage>, <shield>, <round>, <hp>, <ap>.
func (g *Log) LogEffect(text string) {
	g.emit(Message{
		Time: time.Now(),
		Kind: KindEffect,
		Text: text,
	})
}

// LogSystem records a structural event broadcast to all players.
func (g *Log) LogSystem(text string, args ...any) {
	if len(args) > 0 {
		text = fmt.Sprintf(text, args...)
	}

	g.emit(Message{
		Time: time.Now(),
		Kind: KindSystem,
		Text: text,
	})
}

// LogError records an error message visible only to the given player.
func (g *Log) LogError(recipient ds.ID, text string) {
	g.emit(Message{
		Time:      time.Now(),
		Kind:      KindError,
		Recipient: recipient,
		Text:      text,
	})
}

// LogChat records a command response or future chat message.
// Set recipient to a player ID for a private response (e.g. /stats).
// Set recipient to ds.NilID for broadcast.
func (g *Log) LogChat(sender string, recipient ds.ID, text string) {
	g.emit(Message{
		Time:      time.Now(),
		Kind:      KindChat,
		Sender:    sender,
		Recipient: recipient,
		Text:      text,
	})
}

// All returns a copy of all messages in the log.
func (g *Log) All() []Message {
	out := make([]Message, len(g.messages))
	copy(out, g.messages)
	return out
}

// ForPlayer returns all messages the given player is allowed to see.
func (g *Log) ForPlayer(playerID ds.ID) []Message {
	out := make([]Message, 0, len(g.messages))
	for _, m := range g.messages {
		if m.Recipient == ds.NilID || m.Recipient == playerID {
			out = append(out, m)
		}
	}
	return out
}

// ReplayActions returns only KindAction entries in order.
// Pass each LogAction back through Session.Handle to reproduce the game.
func (g *Log) ReplayActions() []LogAction {
	out := make([]LogAction, 0)
	for _, m := range g.messages {
		if m.Kind == KindAction && m.Action != nil {
			out = append(out, *m.Action)
		}
	}
	return out
}

// emit appends a message to the log and queues it for delivery via Out.
func (g *Log) emit(m Message) {
	g.messages = append(g.messages, m)
	g.Out <- m
}

// knownTags lists all simple semantic tags used in log text.
// The client maps each to a display color; the server never decides colors.
var knownTags = []string{
	"ability", "damage", "shield", "round", "hp", "ap",
}

// StripTags removes all semantic tags from a string, leaving plain text.
// Useful for server-side logging to stdout or writing to a plain text file.
func StripTags(s string) string {
	for _, tag := range knownTags {
		s = strings.ReplaceAll(s, "<"+tag+">", "")
		s = strings.ReplaceAll(s, "</"+tag+">", "")
	}
	for strings.Contains(s, "<unit") {
		start := strings.Index(s, "<unit")
		end := strings.Index(s[start:], ">")
		if end < 0 {
			break
		}
		s = s[:start] + s[start+end+1:]
	}
	s = strings.ReplaceAll(s, "</unit>", "")
	return s
}
