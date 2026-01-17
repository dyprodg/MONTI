package websocket

import (
	"encoding/json"
	"sync"

	"github.com/dennisdiepolder/monti/backend/internal/types"
	"github.com/rs/zerolog"
)

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from the clients
	broadcast chan []byte

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex to protect clients map
	mu sync.RWMutex

	// Logger
	logger zerolog.Logger
}

// NewHub creates a new Hub
func NewHub(logger zerolog.Logger) *Hub {
	return &Hub{
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		logger:     logger,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Info().
				Str("client_id", client.id).
				Int("total_clients", len(h.clients)).
				Msg("client connected")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.logger.Info().
					Str("client_id", client.id).
					Int("total_clients", len(h.clients)).
					Msg("client disconnected")
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			// Try to parse as a Widget for per-client filtering
			var widget types.Widget
			if err := json.Unmarshal(message, &widget); err != nil {
				// Not a widget, broadcast as-is to all clients
				h.broadcastRaw(message)
				continue
			}

			// Broadcast with per-client RBAC filtering
			h.broadcastFiltered(&widget)
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// broadcastRaw sends a raw message to all clients without filtering
func (h *Hub) broadcastRaw(message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			// Client's send buffer is full, close and remove it
			close(client.send)
			delete(h.clients, client)
			h.logger.Warn().
				Str("client_id", client.id).
				Msg("client send buffer full, closing connection")
		}
	}
}

// broadcastFiltered sends a widget to each client after applying RBAC filtering
func (h *Hub) broadcastFiltered(widget *types.Widget) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		// Apply client-specific RBAC filter
		filtered := client.FilterWidget(widget)
		if filtered == nil {
			// Client doesn't have access to any agents in this widget
			continue
		}

		// Marshal the filtered widget
		data, err := json.Marshal(filtered)
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to marshal filtered widget")
			continue
		}

		select {
		case client.send <- data:
		default:
			// Client's send buffer is full, close and remove it
			close(client.send)
			delete(h.clients, client)
			h.logger.Warn().
				Str("client_id", client.id).
				Msg("client send buffer full, closing connection")
		}
	}
}
