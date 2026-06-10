package match

import (
	"log"
	"sync"
	"time"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/game"
	"github.com/ognev-dev/goplease/game/bot"
)

const matchmakingTimeout = 30 * time.Second

// MatchCallback is called on the searching player's goroutine when a room is ready.
type MatchCallback func(room *game.Arena, playerIndex int)

type queueEntry struct {
	playerID ds.ID
	cb       MatchCallback
	at       time.Time
}

// Matchmaker pairs players or creates a bot opponent after a timeout.
type Matchmaker struct {
	mu     sync.Mutex
	queue  []queueEntry
	arenas map[ds.ID]*game.Arena

	botAI *bot.Bot
}

func New() *Matchmaker {
	mm := &Matchmaker{
		arenas: make(map[ds.ID]*game.Arena),
		botAI:  bot.New(),
	}
	go mm.watchQueue()
	return mm
}

// Enqueue adds a player to the matchmaking queue.
func (mm *Matchmaker) Enqueue(playerID ds.ID, cb MatchCallback) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// Deduplicate — in case the client reconnects and calls new_game again.
	for _, e := range mm.queue {
		if e.playerID == playerID {
			return
		}
	}

	// If there's already someone waiting, pair them immediately.
	if len(mm.queue) > 0 {
		opponent := mm.queue[0]
		mm.queue = mm.queue[1:]

		arena := mm.createArena(opponent.playerID, playerID, false)

		log.Printf("[match] paired %s vs %s in arena %s", opponent.playerID, playerID, arena.ID)

		// Notify both players (callbacks may send WebSocket messages).
		go opponent.cb(arena, 0)
		go cb(arena, 1)
		return
	}

	mm.queue = append(mm.queue, queueEntry{
		playerID: playerID,
		cb:       cb,
		at:       time.Now(),
	})

	log.Printf("[match] player %s queued (%d in queue)", playerID, len(mm.queue))
}

// Cancel removes a player from the queue (e.g. they disconnected).
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

// Arena returns the active room with the given ID, or nil.
func (mm *Matchmaker) Arena(arenaID ds.ID) *game.Arena {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	return mm.arenas[arenaID]
}

// CloseArena removes a finished room from the registry.
func (mm *Matchmaker) CloseArena(id ds.ID) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	delete(mm.arenas, id)
	log.Printf("[match] arena %s closed", id)
}

// MaybeTriggerBot checks if the active player in a room is a bot and, if so,
// runs its turn asynchronously.
func (mm *Matchmaker) MaybeTriggerBot(room *game.Arena) {
	// Peek at the active player without holding the room lock long.
	activeIdx := room.ActivePlayer
	p := room.Players[activeIdx]
	if !p.IsBot {
		return
	}
	go func() {
		// Small delay so the human client can see the "thinking" state.
		time.Sleep(800 * time.Millisecond)
		mm.botAI.TakeTurn(room, p)
	}()
}

// ArenaByPlayerID finds the active arena for a player.
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

// ─── Internal ─────────────────────────────────────────────────────────────────

// watchQueue periodically checks for players who've been waiting too long and
// pairs them with a bot.
func (mm *Matchmaker) watchQueue() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		mm.promoteStaleEntries()
	}
}

func (mm *Matchmaker) promoteStaleEntries() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	now := time.Now()
	remaining := mm.queue[:0]
	for _, e := range mm.queue {
		if now.Sub(e.at) >= matchmakingTimeout {
			room := mm.createArena(e.playerID, ds.NewID(), true)
			log.Printf("[match] timeout — pairing %s with bot in room %s", e.playerID, room.ID)
			go e.cb(room, 0)

			// Immediately trigger the bot's first response if it goes second.
			go mm.botAI.TakeTurn(room, room.Players[1])
		} else {
			remaining = append(remaining, e)
		}
	}
	mm.queue = remaining
}

func (mm *Matchmaker) createArena(p1ID, p2ID ds.ID, p2IsBot bool) *game.Arena {
	deck1 := game.StartingUnits(p1ID)
	deck2 := game.StartingUnits(p2ID)

	p1 := game.NewPlayer(p1ID, "Player 1", 0, false, deck1)
	p2 := game.NewPlayer(p2ID, nameForPlayer(p2IsBot), 1, p2IsBot, deck2)

	arena := game.NewArena(p1, p2)
	mm.arenas[arena.ID] = arena
	return arena
}

func nameForPlayer(isBot bool) string {
	if isBot {
		return "Bot"
	}

	return "Player 2"
}
