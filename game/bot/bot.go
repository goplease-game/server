package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/game/api"
)

const (
	replyDelay = 800 * time.Millisecond
)

type Bot struct {
	client   *botClient
	url      string
	playerID ds.ID
	state    *gameState
}

func New() *Bot {
	return &Bot{
		client: newBotClient(),
		url:    "ws://localhost:8080/goplease/",
	}
}

func (b *Bot) run() {
	for msg := range b.client.inbox {
		b.handle(msg)
	}

	return
}

func (b *Bot) Connect() (ds.ID, error) {
	err := b.client.connect(b.url)
	if err != nil {
		return ds.NilID, fmt.Errorf("[bot] Failed to connect: %w", err)
	}

	for msg := range b.client.inbox {
		if msg.Action == api.ConnectedAction {
			var payload struct {
				PlayerID ds.ID `json:"player_id"`
			}
			json.Unmarshal(msg.Data, &payload)
			b.playerID = payload.PlayerID

			go b.run()
			return b.playerID, nil
		}
	}

	return ds.NilID, fmt.Errorf("bot: no connected message")
}

func (b *Bot) handle(msg api.InMessage) {
	time.Sleep(replyDelay)

	if len(msg.Data) > 0 {
		fmt.Printf("[BOT] JSON: %s\n", string(msg.Data))
	}

	switch msg.Action {
	case api.ErrorAction:
		log.Printf("[bot] server error: %s", msg.Data)

	case api.NewGameAction:
		b.handleNewGame(msg.Data)

	case api.YouWinAction:
		log.Println("[bot] won!")

	case api.YouLoseAction:
		log.Println("[bot] lost")

	case api.OppSurrenderedAction:
		log.Println("[bot] opponent surrendered")

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
		// TODO
		b.handleNewRound()

	case api.UnitMovedAction:
		b.handleOpponentUnitMoved(msg.Data)

	case api.ApplyStateAction:
		b.handleApplyState(msg.Data)

	default:
		log.Printf("[bot] unhandled action: %s", msg.Action)
	}
}

func (b *Bot) reply(a api.Action, msg any) {
	b.client.send(a, msg)
}

func PlayerName() string {
	return richardAndPerfectFamily[rand.Intn(len(richardAndPerfectFamily))]
}

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
