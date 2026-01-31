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

// MultiplexedConnection manages a single WebSocket carrying events for N agents.
// Messages include an agentID field for demuxing.
type MultiplexedConnection struct {
	agents          map[string]*types.Agent              // agentID -> agent
	callbacks       map[string]chan types.CallAssignMsg   // agentID -> call assign channel
	forceEndCalls   map[string]chan string                // agentID -> force end call channel
	forceDisconns   map[string]chan struct{}              // agentID -> force disconnect channel
	conn            *websocket.Conn
	send            chan []byte
	logger          zerolog.Logger
	backendURL      string
	mu              sync.Mutex
	connected       bool
	closed          bool

	heartbeatsSent   int64
	stateChangesSent int64
	reconnects       int64
}

// NewMultiplexedConnection creates a multiplexed WS connection for a batch of agents
func NewMultiplexedConnection(agents []*types.Agent, backendURL string, logger zerolog.Logger) *MultiplexedConnection {
	agentMap := make(map[string]*types.Agent, len(agents))
	callbacks := make(map[string]chan types.CallAssignMsg, len(agents))
	forceEndCalls := make(map[string]chan string, len(agents))
	forceDisconns := make(map[string]chan struct{}, len(agents))
	for _, a := range agents {
		agentMap[a.ID] = a
		callbacks[a.ID] = make(chan types.CallAssignMsg, 4)
		forceEndCalls[a.ID] = make(chan string, 1)
		forceDisconns[a.ID] = make(chan struct{}, 1)
	}

	return &MultiplexedConnection{
		agents:        agentMap,
		callbacks:     callbacks,
		forceEndCalls: forceEndCalls,
		forceDisconns: forceDisconns,
		send:          make(chan []byte, 256),
		logger:        logger.With().Int("mux_agents", len(agents)).Logger(),
		backendURL:    backendURL,
	}
}

// GetCallAssignChan returns the channel where call_assign messages arrive for an agent
func (mc *MultiplexedConnection) GetCallAssignChan(agentID string) <-chan types.CallAssignMsg {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return mc.callbacks[agentID]
}

// GetForceEndCallChan returns the channel where force_end_call messages arrive for an agent
func (mc *MultiplexedConnection) GetForceEndCallChan(agentID string) <-chan string {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return mc.forceEndCalls[agentID]
}

// GetForceDisconnectChan returns the channel where force_disconnect signals arrive for an agent
func (mc *MultiplexedConnection) GetForceDisconnectChan(agentID string) <-chan struct{} {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return mc.forceDisconns[agentID]
}

// Run connects and maintains the multiplexed WebSocket
func (mc *MultiplexedConnection) Run(ctx context.Context) {
	reconnectDelay := initialReconnectDelay

	for {
		mc.mu.Lock()
		closed := mc.closed
		mc.mu.Unlock()
		if closed {
			return
		}

		select {
		case <-ctx.Done():
			mc.Close()
			return
		default:
		}

		err := mc.connect()
		if err != nil {
			mc.logger.Debug().Err(err).Dur("retry_in", reconnectDelay).Msg("mux connection failed, retrying")
			select {
			case <-ctx.Done():
				return
			case <-time.After(reconnectDelay):
			}
			reconnectDelay *= 2
			if reconnectDelay > maxReconnectDelay {
				reconnectDelay = maxReconnectDelay
			}
			mc.reconnects++
			continue
		}

		reconnectDelay = initialReconnectDelay

		// Register all agents
		mc.registerAll()

		// Run connection loop
		mc.runLoop(ctx)

		mc.mu.Lock()
		mc.connected = false
		if mc.conn != nil {
			mc.conn.Close()
			mc.conn = nil
		}
		mc.mu.Unlock()
	}
}

func (mc *MultiplexedConnection) connect() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	wsURL := mc.backendURL + "/ws/agent/multiplexed"
	if len(wsURL) > 4 && wsURL[:4] == "http" {
		wsURL = "ws" + wsURL[4:]
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}

	mc.conn = conn
	mc.connected = true
	mc.logger.Debug().Msg("mux websocket connected")
	return nil
}

func (mc *MultiplexedConnection) registerAll() {
	mc.mu.Lock()
	agents := make([]*types.Agent, 0, len(mc.agents))
	for _, a := range mc.agents {
		agents = append(agents, a)
	}
	mc.mu.Unlock()

	for _, agent := range agents {
		reg := types.AgentRegister{
			Type:       "register",
			AgentID:    agent.ID,
			Department: agent.Department,
			Location:   agent.Location,
			Team:       agent.Team,
			State:      agent.State,
			KPIs:       agent.KPIs,
		}
		data, err := json.Marshal(reg)
		if err != nil {
			continue
		}
		mc.writeMessage(data)
	}
}

func (mc *MultiplexedConnection) runLoop(ctx context.Context) {
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			_, message, err := mc.conn.ReadMessage()
			if err != nil {
				return
			}
			mc.handleIncoming(message)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-readDone:
			return
		case <-heartbeatTicker.C:
			mc.sendHeartbeats()
		case msg := <-mc.send:
			mc.writeMessage(msg)
		}
	}
}

func (mc *MultiplexedConnection) handleIncoming(message []byte) {
	var msgType struct {
		Type    string `json:"type"`
		AgentID string `json:"agentId"`
	}
	if err := json.Unmarshal(message, &msgType); err != nil {
		return
	}

	switch msgType.Type {
	case "call_assign":
		var ca types.CallAssignMsg
		if err := json.Unmarshal(message, &ca); err != nil {
			return
		}
		mc.mu.Lock()
		ch, ok := mc.callbacks[ca.AgentID]
		mc.mu.Unlock()
		if ok {
			select {
			case ch <- ca:
			default:
				mc.logger.Warn().Str("agent_id", ca.AgentID).Msg("call assign channel full, dropping")
			}
		}
	case "force_end_call":
		var msg struct {
			AgentID string `json:"agentId"`
			CallID  string `json:"callId"`
		}
		if err := json.Unmarshal(message, &msg); err != nil {
			return
		}
		mc.mu.Lock()
		ch, ok := mc.forceEndCalls[msg.AgentID]
		mc.mu.Unlock()
		if ok {
			select {
			case ch <- msg.CallID:
			default:
			}
		}
	case "force_disconnect":
		mc.mu.Lock()
		ch, ok := mc.forceDisconns[msgType.AgentID]
		mc.mu.Unlock()
		if ok {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
	case "ack":
		// Ignore acks
	}
}

func (mc *MultiplexedConnection) sendHeartbeats() {
	mc.mu.Lock()
	agents := make([]*types.Agent, 0, len(mc.agents))
	for _, a := range mc.agents {
		agents = append(agents, a)
	}
	mc.mu.Unlock()

	for _, agent := range agents {
		hb := types.AgentHeartbeat{
			Type:      "heartbeat",
			AgentID:   agent.ID,
			State:     agent.State,
			Timestamp: time.Now(),
			KPIs:      agent.KPIs,
		}
		data, err := json.Marshal(hb)
		if err != nil {
			continue
		}
		mc.writeMessage(data)
		mc.heartbeatsSent++
	}
}

// SendStateChange sends a state change for a specific agent
func (mc *MultiplexedConnection) SendStateChange(agentID string, prevState, newState types.AgentState, duration float64) {
	mc.mu.Lock()
	agent, ok := mc.agents[agentID]
	if !ok {
		mc.mu.Unlock()
		return
	}
	agentCopy := *agent
	mc.mu.Unlock()

	msg := types.AgentStateChangeMsg{
		Type:          "state_change",
		AgentID:       agentCopy.ID,
		PreviousState: prevState,
		NewState:      newState,
		Timestamp:     time.Now(),
		StateDuration: duration,
		KPIs:          agentCopy.KPIs,
		Department:    agentCopy.Department,
		Location:      agentCopy.Location,
		Team:          agentCopy.Team,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	select {
	case mc.send <- data:
		mc.stateChangesSent++
	default:
		mc.logger.Warn().Str("agent_id", agentID).Msg("mux send buffer full")
	}
}

// SendCallComplete sends a call_complete message for a specific agent
func (mc *MultiplexedConnection) SendCallComplete(agentID, callID string, talkTime, holdTime float64) {
	msg := types.CallCompleteMsg{
		Type:      "call_complete",
		AgentID:   agentID,
		CallID:    callID,
		TalkTime:  talkTime,
		HoldTime:  holdTime,
		Timestamp: time.Now(),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	select {
	case mc.send <- data:
	default:
	}
}

func (mc *MultiplexedConnection) writeMessage(data []byte) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.conn == nil || !mc.connected {
		return
	}

	mc.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := mc.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		mc.logger.Debug().Err(err).Msg("mux write error")
	}
}

// Close permanently closes the multiplexed connection
func (mc *MultiplexedConnection) Close() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.closed = true
	if mc.conn != nil {
		mc.conn.Close()
		mc.conn = nil
	}
	mc.connected = false
}

// RemoveAgent removes an agent from the connection so it won't be re-registered on reconnect
func (mc *MultiplexedConnection) RemoveAgent(agentID string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	delete(mc.agents, agentID)
	delete(mc.callbacks, agentID)
	delete(mc.forceEndCalls, agentID)
	delete(mc.forceDisconns, agentID)
}

// UpdateAgent updates the agent data in the connection
func (mc *MultiplexedConnection) UpdateAgent(agent *types.Agent) {
	mc.mu.Lock()
	mc.agents[agent.ID] = agent
	mc.mu.Unlock()
}

// IsConnected returns whether the WebSocket is connected
func (mc *MultiplexedConnection) IsConnected() bool {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return mc.connected
}

// GetMetrics returns connection metrics
func (mc *MultiplexedConnection) GetMetrics() (heartbeats, stateChanges, reconnects int64) {
	return mc.heartbeatsSent, mc.stateChangesSent, mc.reconnects
}
