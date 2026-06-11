package match

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/game"
	"github.com/ognev-dev/goplease/game/bot"
)

const matchmakingTimeout = 2 * time.Second

type MatchCallback func(arena *game.Arena, playerIndex int)

type queueEntry struct {
	playerID ds.ID
	cb       MatchCallback
	at       time.Time
	isBot    bool
}

type Matchmaker struct {
	mu          sync.Mutex
	queue       []queueEntry
	arenas      map[ds.ID]*game.Arena
	notify      MatchCallback
	playerCount int
}

func New(notify MatchCallback) *Matchmaker {
	mm := &Matchmaker{
		notify: notify,
		arenas: make(map[ds.ID]*game.Arena),
	}
	go mm.watchQueue()
	return mm
}

func (mm *Matchmaker) nextPlayerName() string {
	mm.playerCount++
	return fmt.Sprintf("Player %d", mm.playerCount)
}

func (mm *Matchmaker) Enqueue(playerID ds.ID, isBot bool, cb MatchCallback) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

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

func (mm *Matchmaker) nameFor(isBot bool) string {
	if isBot {
		return bot.PlayerName()
	}
	return mm.nextPlayerName()
}

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

func (mm *Matchmaker) Arena(arenaID ds.ID) *game.Arena {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	return mm.arenas[arenaID]
}

func (mm *Matchmaker) CloseArena(id ds.ID) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	delete(mm.arenas, id)
	log.Printf("[match] arena %s closed", id)
}

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

func (mm *Matchmaker) watchQueue() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		mm.promoteStaleEntries()
	}
}

func (mm *Matchmaker) promoteStaleEntries() {
	mm.mu.Lock()

	now := time.Now()
	remaining := mm.queue[:0]
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

		mm.mu.Lock()
		mm.queue = append(mm.queue, e)
		mm.mu.Unlock()

		mm.Enqueue(botID, true, mm.notify)
	}
}

func (mm *Matchmaker) createArena(p1, p2 *game.Player) *game.Arena {
	arena := game.NewArena(p1, p2)
	mm.arenas[arena.ID] = arena
	return arena
}

func (mm *Matchmaker) newPlayer(playerID ds.ID, name string, index int) *game.Player {
	deck := game.StartingUnits(playerID)
	return game.NewPlayer(playerID, name, index, deck)
}
