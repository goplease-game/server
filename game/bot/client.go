package bot

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ognev-dev/goplease/game/api"
)

type client struct {
	inbox    chan api.InMessage
	outbox   chan []byte
	stop     chan struct{}
	stopOnce sync.Once
	mu       sync.Mutex
	conn     *websocket.Conn
}

func newBotClient() *client {
	return &client{
		inbox:  make(chan api.InMessage, 128),
		outbox: make(chan []byte, 128),
		stop:   make(chan struct{}),
	}
}

func (c *client) connect(url string) error {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	go c.readLoop(conn)
	go c.writeLoop(conn)
	return nil
}

func (c *client) send(action api.Action, data any) {
	b, err := json.Marshal(api.OutMessage{Action: action, Data: data})
	if err != nil {
		log.Printf("[bot] marshal error: %v", err)
		return
	}
	select {
	case c.outbox <- b:
	default:
		log.Println("[bot] outbox full, dropping message")
	}
}

func (c *client) readLoop(conn *websocket.Conn) {
	defer func() {
		c.close()
		close(c.inbox)
	}()
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[bot] read error: %v", err)
			return
		}
		var msg api.InMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Printf("[bot] bad JSON: %v", err)
			continue
		}
		select {
		case c.inbox <- msg:
		default:
			log.Println("[bot] inbox full, dropping message")
		}
	}
}

func (c *client) writeLoop(conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case data := <-c.outbox:
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("[bot] write error: %v", err)
				c.close()
				return
			}
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.close()
				return
			}
		case <-c.stop:
			_ = conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		}
	}
}

func (c *client) close() {
	c.stopOnce.Do(func() {
		c.mu.Lock()
		if c.conn != nil {
			_ = c.conn.Close()
		}
		c.mu.Unlock()
		close(c.stop)
	})
}
