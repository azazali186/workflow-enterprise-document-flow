// Package ws implements the real-time event hub from the README. Clients
// authenticate via the Authorization header (no query params, per project
// convention) and receive document lifecycle events pushed by the workers.
package ws

import (
	"encoding/json"
	"sync"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/logger"
	"github.com/hertz-contrib/websocket"
	"go.uber.org/zap"
)

// Event is the wire format pushed to clients.
type Event struct {
	Type       string `json:"type"`
	DocumentID string `json:"document_id,omitempty"`
	Data       any    `json:"data,omitempty"`
	At         string `json:"at,omitempty"`
}

// Conn is the minimal socket surface the hub needs (satisfied by
// *websocket.Conn at runtime; an interface keeps the hub testable).
type Conn interface {
	WriteMessage(messageType int, data []byte) error
	Close() error
}

// Client is one authenticated connection.
type Client struct {
	hub    *Hub
	conn   Conn
	userID string
	send   chan []byte
}

// Hub tracks connections and routes events to their target users.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]bool
	byUser  map[string]map[*Client]bool
}

// NewHub builds an empty hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]bool),
		byUser:  make(map[string]map[*Client]bool),
	}
}

// Register attaches a client and starts its writer pump.
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = true
	if client.userID != "" {
		if h.byUser[client.userID] == nil {
			h.byUser[client.userID] = make(map[*Client]bool)
		}
		h.byUser[client.userID][client] = true
	}
	go client.writePump()
}

// Unregister detaches a client (called once per connection).
func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[client]; !ok {
		return
	}
	delete(h.clients, client)
	if client.userID != "" {
		delete(h.byUser[client.userID], client)
		if len(h.byUser[client.userID]) == 0 {
			delete(h.byUser, client.userID)
		}
	}
	close(client.send)
}

// PublishToUsers delivers an event to every connection of the given users.
func (h *Hub) PublishToUsers(userIDs []string, evt Event) {
	raw, err := json.Marshal(evt)
	if err != nil {
		logger.Warn("ws: event marshal failed", zap.Error(err))
		return
	}
	targets := map[string]bool{}
	for _, uid := range userIDs {
		if uid != "" {
			targets[uid] = true
		}
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		if len(targets) == 0 || targets[client.userID] {
			select {
			case client.send <- raw:
			default: // slow consumer: drop rather than block the pipeline
				logger.Warn("ws: dropping event for slow client", zap.String("user", client.userID))
			}
		}
	}
}

// Broadcast delivers an event to every connected client.
func (h *Hub) Broadcast(evt Event) { h.PublishToUsers(nil, evt) }

// ClientCount reports the number of live connections (metrics/health).
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// writePump flushes queued messages to the socket.
func (c *Client) writePump() {
	defer func() { _ = c.conn.Close() }()
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}
