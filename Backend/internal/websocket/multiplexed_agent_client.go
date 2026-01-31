package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/dennisdiepolder/monti/backend/internal/types"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

// MultiplexedAgentClient handles a single WebSocket carrying events for multiple agents.
// It demuxes by agentID and delegates to the same AgentHub channels.
type MultiplexedAgentClient struct {
	hub      *AgentHub
	conn     *websocket.Conn
	send     chan []byte
	agentIDs map[string]bool // registered agentIDs on this connection
	logger   zerolog.Logger
	done     chan struct{}

	closeOnce sync.Once
	mu        sync.Mutex
}

// NewMultiplexedAgentClient creates a new multiplexed agent client
func NewMultiplexedAgentClient(hub *AgentHub, conn *websocket.Conn, logger zerolog.Logger) *MultiplexedAgentClient {
	return &MultiplexedAgentClient{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		agentIDs: make(map[string]bool),
		logger:   logger,
		done:     make(chan struct{}),
	}
}

func (c *MultiplexedAgentClient) readPump() {
	defer func() {
		close(c.done)
		// Unregister all agents on this connection
		c.mu.Lock()
		agentIDs := make([]string, 0, len(c.agentIDs))
		for id := range c.agentIDs {
			agentIDs = append(agentIDs, id)
		}
		c.mu.Unlock()

		for _, id := range agentIDs {
			// Create a temporary AgentClient for unregistration
			tmpClient := &AgentClient{agentID: id, hub: c.hub}
			c.hub.unregister <- tmpClient
		}
		c.conn.Close()
	}()

	c.conn.SetReadLimit(agentMaxMessageSize * 2) // Larger for multiplexed
	c.conn.SetReadDeadline(time.Now().Add(agentPongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(agentPongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Debug().Err(err).Msg("mux agent websocket read error")
			}
			break
		}

		c.handleMessage(message)
	}
}

// safeSend attempts to send on the send channel without panicking if it's closed
func (c *MultiplexedAgentClient) safeSend(data []byte) bool {
	defer func() {
		if r := recover(); r != nil {
			// channel was closed
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

func (c *MultiplexedAgentClient) handleMessage(message []byte) {
	// Don't process messages if client is shutting down
	select {
	case <-c.done:
		return
	default:
	}

	var msgType struct {
		Type    string `json:"type"`
		AgentID string `json:"agentId"`
	}
	if err := json.Unmarshal(message, &msgType); err != nil {
		c.logger.Debug().Err(err).Msg("failed to parse mux message type")
		return
	}

	switch msgType.Type {
	case "register":
		var reg types.AgentRegister
		if err := json.Unmarshal(message, &reg); err != nil {
			c.logger.Debug().Err(err).Msg("failed to parse mux register message")
			return
		}
		c.mu.Lock()
		c.agentIDs[reg.AgentID] = true
		c.mu.Unlock()

		// Register with hub - create a virtual AgentClient that shares this connection's send channel
		virtualClient := &AgentClient{
			agentID: reg.AgentID,
			hub:     c.hub,
			conn:    c.conn,
			send:    c.send, // share send channel
			logger:  c.logger.With().Str("agent_id", reg.AgentID).Logger(),
			done:    c.done,
		}
		c.hub.register <- virtualClient
		c.hub.agentRegister <- &reg

		// Send ack
		ack := types.ServerAck{Type: "ack", AgentID: reg.AgentID}
		if data, err := json.Marshal(ack); err == nil {
			c.safeSend(data)
		}

	case "heartbeat":
		var hb types.AgentHeartbeat
		if err := json.Unmarshal(message, &hb); err != nil {
			return
		}
		c.hub.heartbeat <- &hb

	case "state_change":
		var sc types.AgentStateChange
		if err := json.Unmarshal(message, &sc); err != nil {
			return
		}
		c.hub.stateChange <- &sc

	case "call_complete":
		var cc types.CallComplete
		if err := json.Unmarshal(message, &cc); err != nil {
			return
		}
		c.hub.callComplete <- &cc
	}
}

func (c *MultiplexedAgentClient) writePump() {
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

// Start starts the multiplexed client's read and write pumps
func (c *MultiplexedAgentClient) Start() {
	go c.writePump()
	go c.readPump()
}

// Close safely closes the client's send channel
func (c *MultiplexedAgentClient) Close() {
	c.closeOnce.Do(func() {
		defer func() {
			recover() // absorb panic if channel was already closed
		}()
		close(c.send)
	})
}
