package websocket

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

// agentUpgrader is the WebSocket upgrader for agent connections
var agentUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for agent connections (internal service)
		return true
	},
}

// AgentHandler handles WebSocket upgrade requests from agents
type AgentHandler struct {
	hub    *AgentHub
	logger zerolog.Logger
}

// NewAgentHandler creates a new AgentHandler
func NewAgentHandler(hub *AgentHub, logger zerolog.Logger) *AgentHandler {
	return &AgentHandler{
		hub:    hub,
		logger: logger,
	}
}

// ServeHTTP handles WebSocket upgrade requests from agents (single agent per connection)
func (h *AgentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := agentUpgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to upgrade agent connection")
		return
	}

	// Create new agent client
	client := NewAgentClient(h.hub, conn, h.logger)

	// Register client with hub
	h.hub.register <- client

	// Start client pumps
	client.Start()
}

// ServeMultiplexedHTTP handles WebSocket upgrade requests for multiplexed agent connections
func (h *AgentHandler) ServeMultiplexedHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := agentUpgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to upgrade multiplexed agent connection")
		return
	}

	// Create multiplexed client
	client := NewMultiplexedAgentClient(h.hub, conn, h.logger)

	// Start client pumps (registration happens per-agent via messages)
	client.Start()
}
