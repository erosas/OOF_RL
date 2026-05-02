package hub

import (
	"sync"

	"github.com/gorilla/websocket"
)

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

func (h *Hub) Broadcast(msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		_ = c.WriteMessage(websocket.TextMessage, msg)
	}
}