package hub

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const writeTimeout = 10 * time.Second

// Hub broadcasts raw RL event JSON to all connected browser WebSocket clients.
type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]struct{}
}

func New() *Hub {
	return &Hub{clients: make(map[*websocket.Conn]struct{})}
}

func (h *Hub) Register(c *websocket.Conn) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
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
	clients := make([]*websocket.Conn, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	var dead []*websocket.Conn
	for _, c := range clients {
		c.SetWriteDeadline(time.Now().Add(writeTimeout))
		if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
			dead = append(dead, c)
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