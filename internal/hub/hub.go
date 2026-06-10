package hub

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const writeTimeout = 10 * time.Second

// Hub broadcasts raw RL event JSON to all connected browser WebSocket clients.
//
// Broadcast is called concurrently — from the RL client read loop, from each
// WASM plugin's event worker, and from HTTP handlers. gorilla/websocket
// forbids concurrent writers on one connection, so each client carries its
// own write mutex.
type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]*client
}

type client struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
}

func New() *Hub {
	return &Hub{clients: make(map[*websocket.Conn]*client)}
}

func (h *Hub) Register(c *websocket.Conn) {
	h.mu.Lock()
	h.clients[c] = &client{conn: c}
	h.mu.Unlock()
}

func (h *Hub) Unregister(c *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

// Broadcast sends msg to every registered client. Clients that fail to accept
// the write within writeTimeout are removed from the hub so one slow or dead
// client cannot block delivery to the rest.
func (h *Hub) Broadcast(msg []byte) {
	h.mu.RLock()
	clients := make([]*client, 0, len(h.clients))
	for _, c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	var dead []*websocket.Conn
	for _, c := range clients {
		c.writeMu.Lock()
		c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		err := c.conn.WriteMessage(websocket.TextMessage, msg)
		c.writeMu.Unlock()
		if err != nil {
			dead = append(dead, c.conn)
		}
	}
	if len(dead) == 0 {
		return
	}
	h.mu.Lock()
	for _, c := range dead {
		delete(h.clients, c)
	}
	h.mu.Unlock()
}