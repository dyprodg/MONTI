package websocket

import (
	"net/http"

	"github.com/dennisdiepolder/monti/backend/internal/config"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for now
		// TODO: Implement proper origin checking based on config
		return true
	},
}

// Handler handles WebSocket upgrade requests
type Handler struct {
	hub    *Hub
	config *config.Config
	logger zerolog.Logger
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub, cfg *config.Config, logger zerolog.Logger) *Handler {
	return &Handler{
		hub:    hub,
		config: cfg,
		logger: logger,
	}
}

// ServeHTTP handles WebSocket upgrade requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to upgrade connection")
		return
	}

	// Create new client
	client := NewClient(h.hub, conn, h.config, h.logger)

	// Register client with hub
	h.hub.register <- client

	// Start client pumps
	client.Start()
}
