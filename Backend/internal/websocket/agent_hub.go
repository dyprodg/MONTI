package websocket

import (
	"encoding/json"
	"sync"

	"github.com/dennisdiepolder/monti/backend/internal/cache"
	"github.com/dennisdiepolder/monti/backend/internal/ingestion"
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

	// Call complete messages from agents
	callComplete chan *types.CallComplete

	// Mutex to protect agents map
	mu sync.RWMutex

	// Logger
	logger zerolog.Logger

	// Agent state tracker (for connection status management)
	tracker *cache.AgentStateTracker

	// Event processor (for processing agent events)
	processor ingestion.EventProcessor
}

// NewAgentHub creates a new AgentHub
func NewAgentHub(tracker *cache.AgentStateTracker, processor ingestion.EventProcessor, logger zerolog.Logger) *AgentHub {
	return &AgentHub{
		agents:        make(map[string]*AgentClient),
		register:      make(chan *AgentClient),
		unregister:    make(chan *AgentClient),
		heartbeat:     make(chan *types.AgentHeartbeat, 1000),
		stateChange:   make(chan *types.AgentStateChange, 500),
		agentRegister: make(chan *types.AgentRegister, 100),
		callComplete:  make(chan *types.CallComplete, 500),
		logger:        logger,
		tracker:       tracker,
		processor:     processor,
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
				h.tracker.SetDisconnected(client.agentID)
				m.RecordAgentDisconnect()

				h.logger.Debug().
					Str("agent_id", client.agentID).
					Int("total_agents", len(h.agents)).
					Msg("agent disconnected")
			}
			h.mu.Unlock()

		case reg := <-h.agentRegister:
			h.processor.ProcessRegister(reg)

		case hb := <-h.heartbeat:
			h.processor.ProcessHeartbeat(hb)

		case sc := <-h.stateChange:
			h.processor.ProcessStateChange(sc)

		case cc := <-h.callComplete:
			h.processor.ProcessCallComplete(cc)
		}
	}
}

// ForceEndCall sends a force_end_call message to the specified agent
func (h *AgentHub) ForceEndCall(agentID, callID string) bool {
	msg := types.ForceEndCall{
		Type:    "force_end_call",
		CallID:  callID,
		AgentID: agentID,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to marshal force_end_call")
		return false
	}
	return h.SendToAgent(agentID, data)
}

// ForceDisconnect sends a force_disconnect message to the agent, then closes the connection
func (h *AgentHub) ForceDisconnect(agentID string) bool {
	msg := types.ForceDisconnect{
		Type:    "force_disconnect",
		AgentID: agentID,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to marshal force_disconnect")
		return false
	}

	// Send the message first
	h.SendToAgent(agentID, data)

	// Then close the connection
	h.mu.Lock()
	client, ok := h.agents[agentID]
	if ok {
		delete(h.agents, agentID)
		client.Close()
		h.tracker.SetDisconnected(agentID)
		metrics.Get().RecordAgentDisconnect()
		h.logger.Info().Str("agent_id", agentID).Msg("agent force-disconnected")
	}
	h.mu.Unlock()

	return ok
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

	return client.safeSend(message)
}
