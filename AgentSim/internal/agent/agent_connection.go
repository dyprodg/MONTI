package agent

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/dennisdiepolder/monti/agentsim/internal/types"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

const (
	// Heartbeat interval
	heartbeatInterval = 2 * time.Second

	// Write timeout
	writeTimeout = 10 * time.Second

	// Reconnect backoff
	initialReconnectDelay = 1 * time.Second
	maxReconnectDelay     = 30 * time.Second
)

// AgentConnection manages the WebSocket connection for a single agent
type AgentConnection struct {
	agent      *types.Agent
	conn       *websocket.Conn
	send       chan []byte
	done       chan struct{}
	logger     zerolog.Logger
	backendURL string
	mu         sync.Mutex
	connected  bool

	// Metrics
	heartbeatsSent   int64
	stateChangesSent int64
	reconnects       int64
}

// NewAgentConnection creates a new agent connection
func NewAgentConnection(agent *types.Agent, backendURL string, logger zerolog.Logger) *AgentConnection {
	return &AgentConnection{
		agent:      agent,
		send:       make(chan []byte, 64),
		done:       make(chan struct{}),
		logger:     logger.With().Str("agent_id", agent.ID).Logger(),
		backendURL: backendURL,
	}
}

// Run starts the connection and maintains it
func (ac *AgentConnection) Run(ctx context.Context) {
	reconnectDelay := initialReconnectDelay

	for {
		select {
		case <-ctx.Done():
			ac.close()
			return
		default:
		}

		err := ac.connect()
		if err != nil {
			ac.logger.Debug().Err(err).Dur("retry_in", reconnectDelay).Msg("connection failed, retrying")
			select {
			case <-ctx.Done():
				return
			case <-time.After(reconnectDelay):
			}
			// Exponential backoff
			reconnectDelay *= 2
			if reconnectDelay > maxReconnectDelay {
				reconnectDelay = maxReconnectDelay
			}
			ac.reconnects++
			continue
		}

		// Reset backoff on successful connection
		reconnectDelay = initialReconnectDelay

		// Register agent
		ac.sendRegister()

		// Run connection loop
		ac.runLoop(ctx)

		// Connection lost, try to reconnect
		ac.mu.Lock()
		ac.connected = false
		if ac.conn != nil {
			ac.conn.Close()
			ac.conn = nil
		}
		ac.mu.Unlock()
	}
}

// connect establishes the WebSocket connection
func (ac *AgentConnection) connect() error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	wsURL := ac.backendURL + "/ws/agent"
	// Convert http:// to ws:// or https:// to wss://
	if len(wsURL) > 4 && wsURL[:4] == "http" {
		wsURL = "ws" + wsURL[4:]
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}

	ac.conn = conn
	ac.connected = true
	ac.logger.Debug().Msg("websocket connected")
	return nil
}

// close closes the connection
func (ac *AgentConnection) close() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if ac.conn != nil {
		ac.conn.Close()
		ac.conn = nil
	}
	ac.connected = false
}

// runLoop handles sending heartbeats and receiving messages
func (ac *AgentConnection) runLoop(ctx context.Context) {
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()

	// Start read goroutine
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			_, _, err := ac.conn.ReadMessage()
			if err != nil {
				return
			}
			// We don't really process incoming messages, just acks
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-readDone:
			return
		case <-heartbeatTicker.C:
			ac.sendHeartbeat()
		case msg := <-ac.send:
			ac.writeMessage(msg)
		}
	}
}

// sendRegister sends the initial registration message
func (ac *AgentConnection) sendRegister() {
	reg := types.AgentRegister{
		Type:       "register",
		AgentID:    ac.agent.ID,
		Department: ac.agent.Department,
		Location:   ac.agent.Location,
		Team:       ac.agent.Team,
		State:      ac.agent.State,
		KPIs:       ac.agent.KPIs,
	}
	data, err := json.Marshal(reg)
	if err != nil {
		ac.logger.Error().Err(err).Msg("failed to marshal register message")
		return
	}
	ac.writeMessage(data)
}

// sendHeartbeat sends a heartbeat message
func (ac *AgentConnection) sendHeartbeat() {
	ac.mu.Lock()
	agent := *ac.agent
	ac.mu.Unlock()

	hb := types.AgentHeartbeat{
		Type:      "heartbeat",
		AgentID:   agent.ID,
		State:     agent.State,
		Timestamp: time.Now(),
		KPIs:      agent.KPIs,
	}
	data, err := json.Marshal(hb)
	if err != nil {
		ac.logger.Error().Err(err).Msg("failed to marshal heartbeat")
		return
	}
	ac.writeMessage(data)
	ac.heartbeatsSent++
}

// SendStateChange sends a state change message
func (ac *AgentConnection) SendStateChange(prevState, newState types.AgentState, duration float64) {
	ac.mu.Lock()
	agent := *ac.agent
	ac.mu.Unlock()

	msg := types.AgentStateChangeMsg{
		Type:          "state_change",
		AgentID:       agent.ID,
		PreviousState: prevState,
		NewState:      newState,
		Timestamp:     time.Now(),
		StateDuration: duration,
		KPIs:          agent.KPIs,
		Department:    agent.Department,
		Location:      agent.Location,
		Team:          agent.Team,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		ac.logger.Error().Err(err).Msg("failed to marshal state change")
		return
	}

	select {
	case ac.send <- data:
		ac.stateChangesSent++
	default:
		ac.logger.Warn().Msg("send buffer full, dropping state change")
	}
}

// writeMessage writes a message to the WebSocket
func (ac *AgentConnection) writeMessage(data []byte) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if ac.conn == nil || !ac.connected {
		return
	}

	ac.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := ac.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		ac.logger.Debug().Err(err).Msg("write error")
	}
}

// UpdateAgent updates the agent pointer (called when state changes)
func (ac *AgentConnection) UpdateAgent(agent *types.Agent) {
	ac.mu.Lock()
	ac.agent = agent
	ac.mu.Unlock()
}

// GetMetrics returns connection metrics
func (ac *AgentConnection) GetMetrics() (heartbeats, stateChanges, reconnects int64) {
	return ac.heartbeatsSent, ac.stateChangesSent, ac.reconnects
}

// IsConnected returns whether the connection is established
func (ac *AgentConnection) IsConnected() bool {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	return ac.connected
}
