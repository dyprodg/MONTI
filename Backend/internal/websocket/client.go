package websocket

import (
	"time"

	"github.com/dennisdiepolder/monti/backend/internal/auth"
	"github.com/dennisdiepolder/monti/backend/internal/config"
	"github.com/dennisdiepolder/monti/backend/internal/types"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

// Client is a middleman between the websocket connection and the hub
type Client struct {
	// Unique client ID
	id string

	// The hub this client belongs to
	hub *Hub

	// The websocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan []byte

	// Configuration
	config *config.Config

	// Logger
	logger zerolog.Logger

	// User claims with allowed locations for RBAC filtering
	claims *auth.Claims
}

// NewClient creates a new Client
func NewClient(hub *Hub, conn *websocket.Conn, cfg *config.Config, logger zerolog.Logger, claims *auth.Claims) *Client {
	clientID := uuid.New().String()
	return &Client{
		id:     clientID,
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		config: cfg,
		logger: logger.With().Str("client_id", clientID).Logger(),
		claims: claims,
	}
}

// readPump pumps messages from the websocket connection to the hub
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(c.config.PongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.config.PongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error().Err(err).Msg("websocket read error")
			}
			break
		}
		c.logger.Debug().Str("message", string(message)).Msg("received message from client")
	}
}

// writePump pumps messages from the hub to the websocket connection
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(c.config.PingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Start starts the client's read and write pumps
func (c *Client) Start() {
	go c.writePump()
	go c.readPump()
}

// FilterWidget filters a widget's agents based on the client's allowed locations
// Returns nil if no agents are visible to this client
func (c *Client) FilterWidget(widget *types.Widget) *types.Widget {
	// If no claims or no agents, return as-is
	if c.claims == nil || len(widget.Agents) == 0 {
		return widget
	}

	// If user has all locations (admin), return original widget
	if len(c.claims.AllowedLocations) == len(types.AllLocations) {
		return widget
	}

	// Filter agents by allowed locations
	var filteredAgents []types.AgentInfo
	for _, agent := range widget.Agents {
		if c.claims.IsLocationAllowed(agent.Location) {
			filteredAgents = append(filteredAgents, agent)
		}
	}

	// If no agents visible, return nil (don't send this widget)
	if len(filteredAgents) == 0 {
		return nil
	}

	// Recalculate summary stats for filtered agents
	stateBreakdown := make(map[types.AgentState]int)
	locationBreakdown := make(map[types.Location]int)

	for _, agent := range filteredAgents {
		stateBreakdown[agent.State]++
		locationBreakdown[agent.Location]++
	}

	// Create filtered widget copy
	filteredWidget := &types.Widget{
		Type:       widget.Type,
		Department: widget.Department,
		Timestamp:  widget.Timestamp,
		Summary: types.WidgetSummary{
			TotalAgents:         len(filteredAgents),
			StateBreakdown:      stateBreakdown,
			DepartmentBreakdown: widget.Summary.DepartmentBreakdown,
			LocationBreakdown:   locationBreakdown,
		},
		Agents: filteredAgents,
	}

	return filteredWidget
}
