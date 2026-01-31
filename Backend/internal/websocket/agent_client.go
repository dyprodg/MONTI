package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/dennisdiepolder/monti/backend/internal/types"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

const (
	// Time allowed to write a message to the agent
	agentWriteWait = 10 * time.Second

	// Time allowed to read the next pong message from the agent
	agentPongWait = 30 * time.Second

	// Send pings to agent with this period (must be less than pongWait)
	agentPingPeriod = 20 * time.Second

	// Maximum message size allowed from agent
	agentMaxMessageSize = 4096
)

// AgentClient represents a WebSocket connection from a simulated agent
type AgentClient struct {
	// Agent ID
	agentID string

	// The hub this client belongs to
	hub *AgentHub

	// The websocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan []byte

	// Logger
	logger zerolog.Logger

	// done channel to signal client shutdown
	done chan struct{}

	// closeOnce ensures send channel is closed only once
	closeOnce sync.Once
}

// NewAgentClient creates a new AgentClient
func NewAgentClient(hub *AgentHub, conn *websocket.Conn, logger zerolog.Logger) *AgentClient {
	return &AgentClient{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 64),
		logger: logger,
		done:   make(chan struct{}),
	}
}

// readPump pumps messages from the websocket connection to the hub
func (c *AgentClient) readPump() {
	defer func() {
		close(c.done)
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(agentMaxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(agentPongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(agentPongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Debug().Err(err).Str("agent_id", c.agentID).Msg("agent websocket read error")
			}
			break
		}

		c.handleMessage(message)
	}
}

// handleMessage processes incoming messages from the agent
func (c *AgentClient) handleMessage(message []byte) {
	// Parse message type
	var msgType struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(message, &msgType); err != nil {
		c.logger.Debug().Err(err).Msg("failed to parse message type")
		return
	}

	switch msgType.Type {
	case "register":
		var reg types.AgentRegister
		if err := json.Unmarshal(message, &reg); err != nil {
			c.logger.Debug().Err(err).Msg("failed to parse register message")
			return
		}
		c.agentID = reg.AgentID
		c.logger = c.logger.With().Str("agent_id", c.agentID).Logger()
		c.hub.agentRegister <- &reg

		// Send acknowledgment (non-blocking, safe if client is closing)
		ack := types.ServerAck{Type: "ack", AgentID: c.agentID}
		if data, err := json.Marshal(ack); err == nil {
			c.safeSend(data)
		}

	case "heartbeat":
		var hb types.AgentHeartbeat
		if err := json.Unmarshal(message, &hb); err != nil {
			c.logger.Debug().Err(err).Msg("failed to parse heartbeat message")
			return
		}
		c.hub.heartbeat <- &hb

	case "state_change":
		var sc types.AgentStateChange
		if err := json.Unmarshal(message, &sc); err != nil {
			c.logger.Debug().Err(err).Msg("failed to parse state_change message")
			return
		}
		c.hub.stateChange <- &sc

	case "call_complete":
		var cc types.CallComplete
		if err := json.Unmarshal(message, &cc); err != nil {
			c.logger.Debug().Err(err).Msg("failed to parse call_complete message")
			return
		}
		c.hub.callComplete <- &cc

	default:
		c.logger.Debug().Str("type", msgType.Type).Msg("unknown message type")
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *AgentClient) writePump() {
	ticker := time.NewTicker(agentPingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(agentWriteWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(agentWriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Start starts the client's read and write pumps
func (c *AgentClient) Start() {
	go c.writePump()
	go c.readPump()
}

// Close safely closes the client's send channel (idempotent)
func (c *AgentClient) Close() {
	c.closeOnce.Do(func() {
		defer func() {
			recover() // absorb panic if channel was already closed
		}()
		close(c.send)
	})
}

// safeSend attempts to send a message, recovering from panic if channel is closed
func (c *AgentClient) safeSend(data []byte) (sent bool) {
	defer func() {
		if r := recover(); r != nil {
			sent = false
		}
	}()

	select {
	case c.send <- data:
		return true
	case <-c.done:
		return false
	default:
		return false
	}
}
