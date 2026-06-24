// Package bot ...
package bot

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	game "github.com/goplease-game/server"
	"github.com/goplease-game/server/api"
	"github.com/goplease-game/server/config"
	"github.com/goplease-game/server/ds"
)

var (
	// ErrBotNoConnectedMessage indicates that the bot has no connected message available.
	ErrBotNoConnectedMessage = errors.New("bot: no connected message")
)

// replyDelay sets the artificial pause duration before a bot transmits its reaction to simulate human pacing.
const replyDelay = 500 * time.Millisecond

// Bot represents an automated agent instance that connects to the game server and acts as an opponent.
type Bot struct {
	t        transport
	playerID ds.ID
	state    *gameState
}

// New instantiates and returns a pre-configured Bot runner referencing default connection addresses.
func New() *Bot {
	port := config.Get().Port

	botAddr := "localhost:" + port

	// ws://localhost:port/play/.
	wsPlayEndpoint := fmt.Sprintf("ws://%s/play/", botAddr)

	return &Bot{t: newWSTransport(wsPlayEndpoint)}
}

// NewWithSession creates a Bot that plays directly against a Session
// without any network transport.
func NewWithSession(playerID ds.ID, session *game.Session) *Bot {
	return &Bot{
		playerID: playerID,
		t:        newChanTransport(playerID, session),
	}
}

// Connect opens a network pipeline toward the server, polls for authentication handshake tokens, and spawns processing threads.
func (b *Bot) Connect() (ds.ID, error) {
	ws := b.t.(*wsTransport)
	err := ws.c.connect(ws.url)
	if err != nil {
		return ds.NilID, fmt.Errorf("[bot] Failed to connect: %w", err)
	}

	connected := false
	// Drain the synchronous inbox channel to intercept and evaluate authorization responses.
	for msg := range b.t.Inbox() {
		if msg.Action == api.ConnectedAction {
			var payload struct {
				PlayerID ds.ID `json:"player_id"`
			}
			err := json.Unmarshal(msg.Data, &payload)
			if err != nil {
				return ds.NilID, fmt.Errorf("[bot] Failed to unmarshal connected message: %w", err)
			}
			b.playerID = payload.PlayerID

			connected = true
			break
		}
	}

	if !connected {
		return ds.NilID, ErrBotNoConnectedMessage
	}

	go b.Run()

	return b.playerID, nil
}

// Run starts the bot message processing loop.
func (b *Bot) Run() {
	for msg := range b.t.Inbox() {
		b.handle(msg)
	}
}

// PlayerName samples and yields a randomized aesthetic nickname chosen out of the predefined Richard catalog.
func PlayerName() string {
	return richardAndPerfectFamily[rand.Intn(len(richardAndPerfectFamily))] //nolint:gosec
}

// handle intercepts structural server frames and channels them over into automated strategic routine updates.
func (b *Bot) handle(msg api.InMessage) {
	time.Sleep(replyDelay)

	if len(msg.Data) > 0 {
		fmt.Printf("[BOT] %s: %s\n", msg.Action, string(msg.Data))
	}

	switch msg.Action {
	case api.ErrorAction:
		log.Printf("[bot] server error: %s", msg.Data)

	case api.NewGameAction:
		b.handleNewGame(msg.Data)

	case api.YouWinAction, api.YouLoseAction, api.OppSurrenderedAction:
		b.handleGameOver(msg.Action)

	case api.PlaceUnitAction:
		b.handlePlaceUnit()

	case api.PlayUnitAction:
		b.handlePlayUnit(msg.Data)

	case api.WaitingForOpponent:
		log.Println("[bot] waiting for opponent...")
		// nothing to do

	case api.UnitPlacedAction:
		b.handleOpponentUnitPlaced(msg.Data)

	case api.NewRound:
		b.handleNewRound()

	case api.UnitMovedAction:
		b.handleOpponentUnitMoved(msg.Data)

	case api.ApplyStateAction:
		b.handleApplyState(msg.Data)
	case api.OppDisconnectedAction:
		b.handleOppDisconnected()
	case api.ActiveUnitChangedAction:
		// active unit is only for visuals now

	default:
		log.Printf("[bot] unhandled action: %s", msg.Action)
	}
}

// reply encapsulates formatting behaviors around shipping action tokens out toward the server pipe.
func (b *Bot) reply(a api.Action, msg any) {
	fmt.Printf("[BOT REPLY] %s: %v\n", a, msg)
	b.t.send(a, msg)
}

// richardAndPerfectFamily lists the collection of stylized persona name patterns applied onto automated mock users.
var richardAndPerfectFamily = []string{
	"Richard Go Please",
	"Richard Never Let You Go",
	"Richard The Tireless",
	"Richard Lion Poop",
	"Richard Too Slow",
	"Richard Asking For Trouble",
	"Richard Out Of Steps",
	"Richard Standing Still",
	"Richard One Cell Left",
	"Richard The Miscalculator",
	"Richard Thinking Long",
	"Richard Wrong Turn",
	"Richard No Move No Cry",
	"Richard Chicken Heart",
	"Richard Pigeon Brain",
	"Richard The Wet Pants",
	"Richard Missed Again",
	"Richard Took An Arrow To The Knee",
	"Richard Looks Rather Pale",
	"Richard Shout A Lot",
	"Richard Not A Choom",
	"Richard Never Lags",
	"Richard Cornered Himself",
	"Richard The Meatshield",
	"Richard No Plan B",
	"Richard Goat Simulator",
	"Richard The Soft Taco",
	"Richard Please Do Not Report",
	"Richard Would You Kindly",
	"Richard Praised The Sun",
	"Richard Git Gud",
	"Richard Hey You Awake",
	"Richard The Cake Is Real",
	"Richard Needs More Vespene Gas",
	"Richard Wololo",
	"Richard Finish Him Later",
	"Richard Still Alive",
	"Richard The Chicken King",
	"Richard Companion Cube",
	"Richard MissingNo",
	"Richard Pressing F",
	"Richard Trying His Best",
	"Richard Completed Tutorial",
	"Richard Estimated Poorly",
	"Richard Confidently Incorrect",
	"Richard Forgot The Objective",
	"Richard Accidentally Heroic",
	"Richard Slightly Lost",
	"Richard Looking Busy",
	"Richard Hero By Accident",
	"Richard Critical Failure",
	"Richard Failed Perception",
	"Richard Lost His Tadpole",
	"Richard Tomb Invader",
	"Richard Looking For The Exit",
	"Richard Cyberpsycho",
	"Richard Konpeki Survivor",
	"Richard No Silver In Hand",
	"Richard Night City Sleeper",
	"Richard Johnny Silverfoot",
	"Richard No Eddies No Cry",
	"Richard Ripperdoc Wannabe",
	"Richard Fuss Roo Dad",
	"Richard Creeper Hugger",
	"Richard Punches Trees",
	"Richard Lost In Mineshaft",
	"Richard Five Stars Instantly",
	"Richard Mission Failed Again",
	"Richard Trevor Approved",
	"Richard The Cake Is Real",
	"Richard Companion Cube Holder",
	"Richard Test Subject Error",
	"Richard Crowbar Enthusiast",
	"Richard Toss A Coin",
	"Richard Potion Misuse",
	"Richard Contract Declined",
	"Richard Dig Straight Down",
	"Richard Shotgun Diplomacy",
	"Richard Too Angry To Die",
	"Richard Found The Keycard",
	"Richard Rocket Jumped Wrong",
	"Richard Arena Survivor",
	"Richard Quad Damage Confused",
	"Richard Mephisto Farmer",
	"Richard No Town Portal",
	"Richard Needs More Minerals",
	"Richard Supply Blocked",
	"Richard Zerg Rush Victim",
	"Richard Vault Dweller",
	"Richard Critical Radiation",
	"Richard Pip Boy Broken",
}
