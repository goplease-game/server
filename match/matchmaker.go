// Package match ...
package match

import (
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	game "github.com/goplease-game/server"
	"github.com/goplease-game/server/bot"
	"github.com/goplease-game/server/ds"
)

// matchmakingTimeout is how long a player can wait in the queue before a bot is paired with them.
// TODO config.
const matchmakingTimeout = 15 * time.Second
const friendRoomTimeout = 10 * time.Minute

// ErrFriendRoomNotFound is returned when no room matches the given join code.
var ErrFriendRoomNotFound = errors.New("friend room not found")

// ErrFriendRoomExpired is returned when the room's join code has timed out.
var ErrFriendRoomExpired = errors.New("friend room expired")

// Callback is called when a player has been paired and an arena is ready.
type Callback func(arena *game.Arena, playerIndex int)

// friendRoom holds a waiting player in Play With Friend mode.
type friendRoom struct {
	entry queueEntry
	code  string
	at    time.Time
}

// codeAlphabet excludes visually ambiguous characters (0, O, 1, I).
const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// queueEntry holds a waiting player's ID, callback, and the time they joined the queue.
type queueEntry struct {
	playerID ds.ID
	cb       Callback
	at       time.Time
	isBot    bool
}

// Matchmaker pairs players, manages active arenas, and spawns bots for players who wait too long.
type Matchmaker struct {
	mu          sync.Mutex
	queue       []queueEntry
	arenas      sync.Map // ds.ID → *game.Arena
	playerArena sync.Map // ds.ID → *game.Arena
	notify      Callback
	playerCount atomic.Int64
	friendRooms sync.Map // string → friendRoom
}

// New creates a Matchmaker and starts the background queue watcher.
func New(notify Callback) *Matchmaker {
	mm := &Matchmaker{
		notify: notify,
	}
	go mm.watchQueue()
	return mm
}

// Enqueue adds a player to the queue, or pairs them immediately if someone is already waiting.
func (mm *Matchmaker) Enqueue(playerID ds.ID, isBot bool, cb Callback) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// Guard against duplicate queue entries for the same player.
	for _, e := range mm.queue {
		if e.playerID == playerID {
			return
		}
	}

	if len(mm.queue) > 0 {
		opponent := mm.queue[0]
		mm.queue = mm.queue[1:]

		p1 := mm.newPlayer(opponent.playerID, mm.nextPlayerName(), 0)
		p2 := mm.newPlayer(playerID, mm.nameFor(isBot), 1)
		arena := mm.createArena(p1, p2)

		log.Printf("[match] paired %s vs %s in arena %s", opponent.playerID, playerID, arena.ID)

		go opponent.cb(arena, 0)
		go cb(arena, 1)
		return
	}

	mm.queue = append(mm.queue, queueEntry{
		playerID: playerID,
		cb:       cb,
		at:       time.Now(),
		isBot:    isBot,
	})
}

// Cancel removes a player from the queue. No-op if the player is not queued.
func (mm *Matchmaker) Cancel(playerID ds.ID) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	for i, e := range mm.queue {
		if e.playerID == playerID {
			mm.queue = append(mm.queue[:i], mm.queue[i+1:]...)
			log.Printf("[match] player %s removed from queue", playerID)
			return
		}
	}
}

// Arena returns the active arena with the given ID, or nil if not found.
func (mm *Matchmaker) Arena(arenaID ds.ID) *game.Arena {
	v, ok := mm.arenas.Load(arenaID)
	if !ok {
		return nil
	}

	return v.(*game.Arena)
}

// CloseArena removes an arena from memory once the match is over.
func (mm *Matchmaker) CloseArena(id ds.ID) {
	if ar := mm.Arena(id); ar != nil {
		for _, p := range ar.Players {
			mm.playerArena.Delete(p.ID)
		}
	}

	mm.arenas.Delete(id)
	log.Printf("[match] arena %s closed", id)
}

// ArenaByPlayerID returns the arena the given player is currently in, or nil.
func (mm *Matchmaker) ArenaByPlayerID(playerID ds.ID) *game.Arena {
	v, ok := mm.playerArena.Load(playerID)
	if !ok {
		return nil
	}

	return v.(*game.Arena)
}

// CreateFriendRoom creates a waiting room and returns its join code.
func (mm *Matchmaker) CreateFriendRoom(playerID ds.ID, cb Callback) string {
	code := mm.generateCode()
	mm.friendRooms.Store(code, friendRoom{
		entry: queueEntry{playerID: playerID, cb: cb, at: time.Now()},
		code:  code,
		at:    time.Now(),
	})
	log.Printf("[match] friend room created by %s, code %s", playerID, code)
	return code
}

// JoinFriendRoom finds the room by code and starts the game.
// Returns an error if the code is not found or has expired.
func (mm *Matchmaker) JoinFriendRoom(code string, playerID ds.ID, cb Callback) error {
	v, ok := mm.friendRooms.Load(code)
	if !ok {
		return ErrFriendRoomNotFound
	}

	room := v.(friendRoom)
	if time.Since(room.at) > friendRoomTimeout {
		mm.friendRooms.Delete(code)
		return ErrFriendRoomExpired
	}

	mm.friendRooms.Delete(code)

	p1 := mm.newPlayer(room.entry.playerID, mm.nextPlayerName(), 0)
	p2 := mm.newPlayer(playerID, mm.nextPlayerName(), 1)
	arena := mm.createArena(p1, p2)

	log.Printf("[match] friend game started: %s vs %s in arena %s (code %s)",
		room.entry.playerID, playerID, arena.ID, code)

	go room.entry.cb(arena, 0)
	go cb(arena, 1)
	return nil
}

// CancelFriendRoom removes the friend room created by playerID.
func (mm *Matchmaker) CancelFriendRoom(playerID ds.ID) {
	mm.friendRooms.Range(func(key, value any) bool {
		room := value.(friendRoom)
		if room.entry.playerID == playerID {
			mm.friendRooms.Delete(key)
			log.Printf("[match] friend room %s cancelled by %s", room.code, playerID)
			return false
		}
		return true
	})
}

// watchQueue periodically checks for players who have been waiting too long and pairs them with a bot.
func (mm *Matchmaker) watchQueue() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		mm.promoteStaleEntries()
		mm.cleanStaleFriendRooms()
	}
}

// promoteStaleEntries spawns bots for any players who have been waiting longer than matchmakingTimeout.
func (mm *Matchmaker) promoteStaleEntries() {
	mm.mu.Lock()

	now := time.Now()
	remaining := mm.queue[:0] // Re-use queue memory to perform an in-place filter.
	var toSpawn []queueEntry

	for _, e := range mm.queue {
		if now.Sub(e.at) >= matchmakingTimeout && !e.isBot {
			toSpawn = append(toSpawn, e)
		} else {
			remaining = append(remaining, e)
		}
	}
	mm.queue = remaining
	mm.mu.Unlock()

	for _, e := range toSpawn {
		b := bot.New()
		botID, err := b.Connect()
		if err != nil {
			log.Printf("[match] failed to spawn bot: %v", err)
			mm.mu.Lock()
			mm.queue = append(mm.queue, e)
			mm.mu.Unlock()
			continue
		}
		log.Printf("[match] spawned bot %s for player %s", botID, e.playerID)

		// Place the user back into the system so that the next Enqueue pass links them with the new bot instance.
		mm.mu.Lock()
		mm.queue = append(mm.queue, e)
		mm.mu.Unlock()

		mm.Enqueue(botID, true, mm.notify)
	}
}

// createArena creates a new arena, registering it and both players in the lookup maps.
func (mm *Matchmaker) createArena(p1, p2 *game.Player) *game.Arena {
	arena := game.NewArena(p1, p2)
	mm.arenas.Store(arena.ID, arena)
	mm.playerArena.Store(p1.ID, arena)
	mm.playerArena.Store(p2.ID, arena)

	return arena
}

// newPlayer creates a player with the given ID, name, and starting deck.
func (mm *Matchmaker) newPlayer(playerID ds.ID, name string, index int) *game.Player {
	deck := game.StartingUnits(playerID)
	return game.NewPlayer(playerID, name, index, deck)
}

// nameFor returns a bot name or the next sequential player name.
func (mm *Matchmaker) nameFor(isBot bool) string {
	if isBot {
		return bot.PlayerName()
	}
	return mm.nextPlayerName()
}

// nextPlayerName returns the next sequential guest name, e.g. "Player 1", "Player 2".
func (mm *Matchmaker) nextPlayerName() string {
	return fmt.Sprintf("Player %d", mm.playerCount.Add(1))
}

// generateCode returns a random 6-character uppercase code from codeAlphabet.
func (mm *Matchmaker) generateCode() string {
	b := make([]byte, 6) //nolint:mnd
	for i := range b {
		b[i] = codeAlphabet[rand.IntN(len(codeAlphabet))] //nolint:gosec
	}
	return string(b)
}

// cleanStaleFriendRooms removes friend rooms that have exceeded friendRoomTimeout.
func (mm *Matchmaker) cleanStaleFriendRooms() {
	mm.friendRooms.Range(func(key, value any) bool {
		room := value.(friendRoom)
		if time.Since(room.at) > friendRoomTimeout {
			mm.friendRooms.Delete(key)
			log.Printf("[match] friend room %s expired", room.code)
		}
		return true
	})
}
