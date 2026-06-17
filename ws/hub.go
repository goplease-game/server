package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/goplease-game/server/api"
	"github.com/goplease-game/server/ds"
	"github.com/gorilla/websocket"
)

// upgrader configures HTTP → WebSocket upgrades.
// It sets buffer sizes, handshake timeout, and origin policy.
var upgrader = websocket.Upgrader{
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	HandshakeTimeout: 10 * time.Second,
	// In production replace with an origin whitelist check.
	CheckOrigin: func(*http.Request) bool { return true },
}

const (
	// writeWait defines maximum time allowed to write a message to the client.
	writeWait = 10 * time.Second

	// pongWait defines time to wait for a pong response before considering the connection dead.
	pongWait = 60 * time.Second

	// pingPeriod defines how often the server sends ping messages to keep the connection alive.
	pingPeriod = (pongWait * 9) / 10

	// maxMessageSize limits maximum size of incoming WebSocket messages in bytes.
	maxMessageSize = 4096
)

// Client represents a single connected WebSocket client (browser or game instance).
// It maintains connection state, player identity, and outbound message queue.
type Client struct {
	// ID is a unique runtime identifier for this WebSocket connection instance.
	ID string

	PlayerID ds.ID

	// ArenaID represents the game session/room this client is currently connected to.
	ArenaID ds.ID

	Name string

	hub  *Hub
	conn *websocket.Conn

	// send is a buffered channel used as an outbound message queue to the client.
	// Messages are consumed by writePump in a separate goroutine.
	send chan []byte
}

func newClient(hub *Hub, conn *websocket.Conn, playerID ds.ID) *Client {
	return &Client{
		ID:       uuid.NewString(),
		PlayerID: playerID,
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 128),
	}
}

// SetName assigns a human-readable name to the client (used for logging/debugging/UI).
func (c *Client) SetName(name string) {
	c.Name = name
}

// Send enqueues an outbound message to the client.
// It is non-blocking; if the buffer is full, the message is dropped.
func (c *Client) Send(msg api.OutMessage) {
	b, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[ws] marshal error for client %s: %v", c.ID, err)
		return
	}

	select {
	case c.send <- b:
	default:
		// Buffer full — drop and let the write pump detect the dead connection.
		log.Printf("[ws] send buffer full for client %s, dropping message", c.ID)
	}
}

// readPump continuously reads messages from the WebSocket connection.
// It validates input, unmarshals JSON, and forwards messages to the hub dispatcher.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		err := c.conn.Close()
		if err != nil {
			log.Printf("[ws] close client %s: %v", c.ID, err)
		}
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[ws] unexpected close from client %s: %v", c.ID, err)
			}
			break
		}

		var msg api.InMessage
		err = json.Unmarshal(raw, &msg)
		if err != nil {
			log.Printf("[ws] invalid JSON from client %s: %v", c.ID, err)
			c.Send(api.OutMessage{
				Action: api.ErrorAction,
				Data:   map[string]string{"message": "invalid JSON"},
			})
			continue
		}

		c.hub.dispatch <- Envelope{Client: c, Message: msg}
	}
}

// writePump continuously writes outbound messages from the send queue to the WebSocket.
// It also periodically sends ping messages to keep the connection alive.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		err := c.conn.Close()
		if err != nil {
			log.Printf("[ws] close client %s: %v", c.ID, err)
		}
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			err := c.conn.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				log.Printf("[ws] write error for client %s: %v", c.ID, err)
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			err := c.conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				return
			}
		}
	}
}

// EventKind defines the type of lifecycle event occurring in the hub.
type EventKind int

const (
	// EventConnected indicates a new client connection.
	EventConnected EventKind = iota

	// EventDisconnected indicates a client disconnection.
	EventDisconnected

	// EventMessage indicates an incoming message from a client.
	EventMessage
)

// Event represents a high-level websocket event forwarded to the game server.
type Event struct {
	Kind   EventKind
	Client *Client
	Msg    api.InMessage
}

// Envelope wraps an incoming client message with its sender metadata.
type Envelope struct {
	Client  *Client
	Message api.InMessage
}

// RoomBroadcast represents a message sent to all clients within a specific room/arena.
type RoomBroadcast struct {
	RoomID  ds.ID
	Message api.OutMessage
}

// Hub is the central websocket connection manager.
//
// It is responsible for:
//   - tracking connected clients
//   - routing incoming messages to game logic
//   - broadcasting messages to arenas/rooms
//
// It does NOT contain game logic.
type Hub struct {
	mu             sync.RWMutex
	clients        map[string]*Client
	clientByPlayer map[ds.ID]*Client

	register   chan *Client
	unregister chan *Client
	dispatch   chan Envelope
	broadcast  chan RoomBroadcast

	// Events is read by GameServer.Run(). Buffered to decouple the two loops.
	Events chan Event
}

// NewHub creates and initializes a new Hub instance.
// It sets up internal channels and empty client registries.
// The hub is not started automatically; Run must be called separately.
func NewHub() *Hub {
	return &Hub{
		clients:        make(map[string]*Client),
		clientByPlayer: make(map[ds.ID]*Client),
		register:       make(chan *Client, 16),
		unregister:     make(chan *Client, 16),
		dispatch:       make(chan Envelope, 256),
		broadcast:      make(chan RoomBroadcast, 256),
		Events:         make(chan Event, 256),
	}
}

// Run starts hub internal loops (registry and broadcast processing).
// Must be executed once, typically in a goroutine.
func (h *Hub) Run() {
	go h.broadcastLoop()
	h.registryLoop()
}

// Broadcast sends a message to all clients in the specified room.
func (h *Hub) Broadcast(roomID ds.ID, msg api.OutMessage) {
	h.broadcast <- RoomBroadcast{RoomID: roomID, Message: msg}
}

// ClientByPlayerID returns the active websocket client for a given player ID.
// Returns nil if no client is connected.
func (h *Hub) ClientByPlayerID(playerID ds.ID) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.clientByPlayer[playerID]
}

// ServeWS upgrades an HTTP request to a WebSocket connection,
// creates a client instance, and registers it in the hub.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	playerID := ds.NewID()
	idString := r.URL.Query().Get("player_id")
	if idString != "" {
		var err error
		playerID, err = ds.ParseID(idString)
		if err != nil {
			log.Printf("invalid UUID: %v", err)
			return
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws] upgrade failed for player %q: %v", playerID, err) //nolint:gosec
		return
	}

	c := newClient(h, conn, playerID)
	h.register <- c

	go c.writePump()
	go c.readPump()
}

// broadcastLoop delivers room-wide messages to connected clients.
// It runs independently to avoid blocking registry operations.
func (h *Hub) broadcastLoop() {
	for bc := range h.broadcast {
		h.mu.RLock()
		for _, c := range h.clients {
			if c.ArenaID == bc.RoomID {
				c.Send(bc.Message)
			}
		}
		h.mu.RUnlock()
	}
}

// registryLoop manages client lifecycle (register/unregister) and message dispatching.
// It is the single writer to the clients map.
func (h *Hub) registryLoop() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c.ID] = c
			h.clientByPlayer[c.PlayerID] = c
			h.mu.Unlock()
			log.Printf("[hub] connected: %s (player %s)", c.ID, c.PlayerID)
			h.Events <- Event{Kind: EventConnected, Client: c}

		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c.ID]; ok {
				delete(h.clients, c.ID)
				delete(h.clientByPlayer, c.PlayerID)
				close(c.send)
			}
			h.mu.Unlock()
			log.Printf("[hub] disconnected: %s (player %s)", c.ID, c.PlayerID)
			h.Events <- Event{Kind: EventDisconnected, Client: c}

		case env := <-h.dispatch:
			h.Events <- Event{Kind: EventMessage, Client: env.Client, Msg: env.Message}
		}
	}
}
