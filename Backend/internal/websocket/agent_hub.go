package websocket

import (
	"sync"

	"github.com/dennisdiepolder/monti/backend/internal/cache"
	"github.com/dennisdiepolder/monti/backend/internal/metrics"
	"github.com/dennisdiepolder/monti/backend/internal/types"
	"github.com/rs/zerolog"
)

// AgentHub maintains the set of active agent WebSocket connections
type AgentHub struct {
	// Registered agent clients
	agents map[string]*AgentClient // agentID -> client

	// Register requests from agent clients
	register chan *AgentClient

	// Unregister requests from agent clients
	unregister chan *AgentClient

	// Heartbeat messages from agents
	heartbeat chan *types.AgentHeartbeat

	// State change messages from agents
	stateChange chan *types.AgentStateChange

	// Agent registration messages
	agentRegister chan *types.AgentRegister

	// Mutex to protect agents map
	mu sync.RWMutex

	// Logger
	logger zerolog.Logger

	// Agent state tracker
	tracker *cache.AgentStateTracker
}

// NewAgentHub creates a new AgentHub
func NewAgentHub(tracker *cache.AgentStateTracker, logger zerolog.Logger) *AgentHub {
	return &AgentHub{
		agents:        make(map[string]*AgentClient),
		register:      make(chan *AgentClient),
		unregister:    make(chan *AgentClient),
		heartbeat:     make(chan *types.AgentHeartbeat, 1000),
		stateChange:   make(chan *types.AgentStateChange, 500),
		agentRegister: make(chan *types.AgentRegister, 100),
		logger:        logger,
		tracker:       tracker,
	}
}

// Run starts the hub's main loop
func (h *AgentHub) Run() {
	m := metrics.Get()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// Remove existing client with same agentID if any
			if existing, ok := h.agents[client.agentID]; ok {
				existing.Close()
				delete(h.agents, client.agentID)
			}
			h.agents[client.agentID] = client
			h.mu.Unlock()

			h.tracker.SetConnected(client.agentID, true)
			m.RecordAgentConnect()

			h.logger.Debug().
				Str("agent_id", client.agentID).
				Int("total_agents", len(h.agents)).
				Msg("agent connected")

		case client := <-h.unregister:
			h.mu.Lock()
			if existing, ok := h.agents[client.agentID]; ok && existing == client {
				delete(h.agents, client.agentID)
				client.Close()
				h.tracker.SetConnected(client.agentID, false)
				m.RecordAgentDisconnect()

				h.logger.Debug().
					Str("agent_id", client.agentID).
					Int("total_agents", len(h.agents)).
					Msg("agent disconnected")
			}
			h.mu.Unlock()

		case reg := <-h.agentRegister:
			h.tracker.RegisterAgent(reg)
			m.RecordAgentRegister()

			h.logger.Debug().
				Str("agent_id", reg.AgentID).
				Str("state", string(reg.State)).
				Msg("agent registered")

		case hb := <-h.heartbeat:
			h.tracker.UpdateFromHeartbeat(hb)
			m.RecordAgentHeartbeat()

		case sc := <-h.stateChange:
			h.tracker.UpdateFromStateChange(sc)
			m.RecordAgentStateChange()

			h.logger.Debug().
				Str("agent_id", sc.AgentID).
				Str("prev_state", string(sc.PreviousState)).
				Str("new_state", string(sc.NewState)).
				Float64("duration", sc.StateDuration).
				Msg("agent state change")
		}
	}
}

// AgentCount returns the number of connected agents
func (h *AgentHub) AgentCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.agents)
}

// SendToAgent sends a message to a specific agent
func (h *AgentHub) SendToAgent(agentID string, message []byte) bool {
	h.mu.RLock()
	client, ok := h.agents[agentID]
	h.mu.RUnlock()

	if !ok {
		return false
	}

	select {
	case client.send <- message:
		return true
	default:
		return false
	}
}
