package bot

import (
	"encoding/json"
	"fmt"

	game "github.com/goplease-game/server"
	"github.com/goplease-game/server/api"
	"github.com/goplease-game/server/ds"
)

// transport abstracts message delivery for the bot.
type transport interface {
	send(action api.Action, data any)
	Inbox() <-chan api.InMessage
	close()
}

// wsTransport delivers messages over a real WebSocket connection.
type wsTransport struct {
	c   *client
	url string
}

func newWSTransport(url string) *wsTransport {
	return &wsTransport{c: newBotClient(), url: url}
}

func (t *wsTransport) Inbox() <-chan api.InMessage { return t.c.inbox }

func (t *wsTransport) send(action api.Action, data any) { t.c.send(action, data) }
func (t *wsTransport) close()                           { t.c.close() }

// chanTransport delivers messages directly via Session channels.
type chanTransport struct {
	in     <-chan api.OutMessage
	handle func(api.Action, json.RawMessage)
	inbox  chan api.InMessage
	stop   chan struct{}
}

func newChanTransport(playerID ds.ID, session *game.Session) *chanTransport {
	t := &chanTransport{
		in:    session.P2Events,
		inbox: make(chan api.InMessage, 128),
		stop:  make(chan struct{}),
	}
	t.handle = func(action api.Action, data json.RawMessage) {
		session.Handle(playerID, action, data)
	}
	go t.readLoop()
	return t
}

func (t *chanTransport) Inbox() <-chan api.InMessage { return t.inbox }

func (t *chanTransport) readLoop() {
	for {
		select {
		case msg, ok := <-t.in:
			if !ok {
				close(t.inbox)
				return
			}
			data, err := json.Marshal(msg.Data)
			if err != nil {
				fmt.Printf("[readLoop] unmarshal: %s\n", err)
				return
			}
			t.inbox <- api.InMessage{Action: msg.Action, Data: data}
		case <-t.stop:
			close(t.inbox)
			return
		}
	}
}

func (t *chanTransport) send(action api.Action, data any) {
	b, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("[send] unmarshal: %s\n", err)
		return
	}
	t.handle(action, b)
}

func (t *chanTransport) close() { close(t.stop) }
