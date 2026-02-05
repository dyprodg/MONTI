package websocket

import (
	"encoding/json"
	"sync"

	"github.com/dennisdiepolder/monti/backend/internal/metrics"
	"github.com/dennisdiepolder/monti/backend/internal/types"
	"github.com/rs/zerolog"
)

const maxSnapshotHistory = 300

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

	// Ring buffer of recent snapshots (max maxSnapshotHistory)
	snapshotHistory []*types.Snapshot

	// Logger
	logger zerolog.Logger
}

// NewHub creates a new Hub
func NewHub(logger zerolog.Logger) *Hub {
	return &Hub{
		broadcast:       make(chan []byte, 256),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		clients:         make(map[*Client]bool),
		snapshotHistory: make([]*types.Snapshot, 0, maxSnapshotHistory),
		logger:          logger,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	m := metrics.Get()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			m.RecordWebSocketConnect()
			h.logger.Info().
				Str("client_id", client.id).
				Int("total_clients", len(h.clients)).
				Msg("client connected")

			// Send snapshot history to newly connected client
			h.sendSnapshotHistory(client)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				m.RecordWebSocketDisconnect()
				h.logger.Info().
					Str("client_id", client.id).
					Int("total_clients", len(h.clients)).
					Msg("client disconnected")
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			m.RecordWebSocketMessage()

			// Check message type
			var msgType struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(message, &msgType); err != nil {
				h.broadcastRaw(message)
				continue
			}

			switch msgType.Type {
			case "snapshot":
				// Single snapshot with all agents + all queues — apply per-client RBAC
				var snapshot types.Snapshot
				if err := json.Unmarshal(message, &snapshot); err != nil {
					h.broadcastRaw(message)
					continue
				}
				h.appendSnapshotHistory(&snapshot)
			h.broadcastSnapshot(&snapshot)

			default:
				h.broadcastRaw(message)
			}
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

// appendSnapshotHistory adds a snapshot to the ring buffer, evicting the oldest if full
func (h *Hub) appendSnapshotHistory(snapshot *types.Snapshot) {
	if len(h.snapshotHistory) < maxSnapshotHistory {
		h.snapshotHistory = append(h.snapshotHistory, snapshot)
		return
	}
	// Shift left and overwrite last slot — avoids re-slicing which leaks
	// old pointers in the backing array and prevents GC from collecting them
	copy(h.snapshotHistory, h.snapshotHistory[1:])
	h.snapshotHistory[maxSnapshotHistory-1] = snapshot
}

// sendSnapshotHistory sends the buffered snapshot history to a newly connected client
func (h *Hub) sendSnapshotHistory(client *Client) {
	if len(h.snapshotHistory) == 0 {
		return
	}

	// Build RBAC-filtered history for this client
	filtered := make([]*types.Snapshot, 0, len(h.snapshotHistory))
	for _, snap := range h.snapshotHistory {
		filtered = append(filtered, client.FilterSnapshot(snap))
	}

	msg := struct {
		Type      string            `json:"type"`
		Snapshots []*types.Snapshot `json:"snapshots"`
	}{
		Type:      "snapshot_history",
		Snapshots: filtered,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to marshal snapshot history")
		return
	}

	select {
	case client.send <- data:
		h.logger.Info().
			Str("client_id", client.id).
			Int("history_size", len(filtered)).
			Msg("sent snapshot history to client")
	default:
		h.logger.Warn().
			Str("client_id", client.id).
			Msg("client send buffer full, skipping history")
	}
}

// broadcastSnapshot sends the snapshot to each client after applying RBAC filtering
func (h *Hub) broadcastSnapshot(snapshot *types.Snapshot) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		// Apply client-specific RBAC filter
		filtered := client.FilterSnapshot(snapshot)

		data, err := json.Marshal(filtered)
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to marshal filtered snapshot")
			continue
		}

		select {
		case client.send <- data:
		default:
			close(client.send)
			delete(h.clients, client)
			h.logger.Warn().
				Str("client_id", client.id).
				Msg("client send buffer full, closing connection")
		}
	}
}
