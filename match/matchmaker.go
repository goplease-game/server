// Package match ...
package match

import (
	"fmt"
	"log"
	"sync"
	"time"

	game "github.com/goplease-game/server"
	"github.com/goplease-game/server/bot"
	"github.com/goplease-game/server/ds"
)

// matchmakingTimeout defines the duration a player can remain in the queue before a computer opponent is subbed in.
const matchmakingTimeout = 2 * time.Second

// Callback represents the signaling function triggered once an arena allocation or player pairing resolves.
type Callback func(arena *game.Arena, playerIndex int)

// queueEntry stores metadata, targeting parameters, and timestamps for an active matchmaking participant.
type queueEntry struct {
	playerID ds.ID
	cb       Callback
	at       time.Time
	isBot    bool
}

// Matchmaker coordinates player aggregation, pairing synchronization, and automated companion injection routines.
type Matchmaker struct {
	mu          sync.Mutex
	queue       []queueEntry
	arenas      map[ds.ID]*game.Arena
	notify      Callback
	playerCount int
}

// New instantiates and provisions a fresh Matchmaker entity, launching the background queue supervisor thread.
func New(notify Callback) *Matchmaker {
	mm := &Matchmaker{
		notify: notify,
		arenas: make(map[ds.ID]*game.Arena),
	}
	go mm.watchQueue()
	return mm
}

// Enqueue checks identity keys to either merge matching profiles into live arenas or buffer them in the active search pool.
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

// Cancel safely excises an active player tracking entry from the matchmaking buffer arrays.
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

// Arena extracts and retrieves the reference pointer of an active match instance based on its domain signature key.
func (mm *Matchmaker) Arena(arenaID ds.ID) *game.Arena {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	return mm.arenas[arenaID]
}

// CloseArena removes an active game field instance from memory tracking maps once its structural phase terminates.
func (mm *Matchmaker) CloseArena(id ds.ID) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	delete(mm.arenas, id)
	log.Printf("[match] arena %s closed", id)
}

// ArenaByPlayerID crawls active maps to resolve which active instance currently holds a reference to the target user.
func (mm *Matchmaker) ArenaByPlayerID(playerID ds.ID) *game.Arena {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	for _, ar := range mm.arenas {
		for _, p := range ar.Players {
			if p.ID == playerID {
				return ar
			}
		}
	}
	return nil
}

// watchQueue runs a recurring loop to monitor, update, and advance wait timers for queued items.
func (mm *Matchmaker) watchQueue() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		mm.promoteStaleEntries()
	}
}

// promoteStaleEntries filters waiting entries to spin up automated bot opponents for connections exceeding timeouts.
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

// createArena configures a new combat instance, mapping its signature values into memory registries.
func (mm *Matchmaker) createArena(p1, p2 *game.Player) *game.Arena {
	arena := game.NewArena(p1, p2)
	mm.arenas[arena.ID] = arena
	return arena
}

// newPlayer instantiates a fully operational structural player representation, complete with initial starting card decks.
func (mm *Matchmaker) newPlayer(playerID ds.ID, name string, index int) *game.Player {
	deck := game.StartingUnits(playerID)
	return game.NewPlayer(playerID, name, index, deck)
}

// nameFor routes structural triggers to fetch randomized bot identifiers or default player sequence strings.
func (mm *Matchmaker) nameFor(isBot bool) string {
	if isBot {
		return bot.PlayerName()
	}
	return mm.nextPlayerName()
}

// nextPlayerName generates a sequential string name for unauthenticated guest or client connections.
func (mm *Matchmaker) nextPlayerName() string {
	mm.playerCount++
	return fmt.Sprintf("Player %d", mm.playerCount)
}
