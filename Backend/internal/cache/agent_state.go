package cache

import (
	"sync"
	"time"

	"github.com/dennisdiepolder/monti/backend/internal/types"
)

const (
	// StaleThreshold is the duration after which an agent is considered stale (3 missed heartbeats)
	StaleThreshold = 6 * time.Second
)

// AgentStateTracker maintains the current state of all agents
type AgentStateTracker struct {
	agents map[string]*types.AgentInfo // agentID -> current state
	mu     sync.RWMutex
}

// NewAgentStateTracker creates a new agent state tracker
func NewAgentStateTracker() *AgentStateTracker {
	return &AgentStateTracker{
		agents: make(map[string]*types.AgentInfo),
	}
}

// Update updates or adds an agent's state (from HTTP POST event - legacy)
func (t *AgentStateTracker) Update(event types.AgentEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()

	existing, exists := t.agents[event.AgentID]

	// If agent exists and state changed, update state start time
	// Otherwise, keep the existing state start time
	stateStart := event.Timestamp
	if exists && existing.State == event.State {
		stateStart = existing.StateStart
	}

	// Preserve connection status if exists
	connectionStatus := types.StatusConnected
	if exists {
		connectionStatus = existing.ConnectionStatus
	}

	t.agents[event.AgentID] = &types.AgentInfo{
		AgentID:          event.AgentID,
		State:            event.State,
		Department:       event.Department,
		Location:         event.Location,
		Team:             event.Team,
		StateStart:       stateStart,
		LastUpdate:       event.Timestamp,
		LastHeartbeat:    time.Now(),
		ConnectionStatus: connectionStatus,
		KPIs:             event.KPIs,
	}
}

// UpdateFromHeartbeat updates an agent's state from a WebSocket heartbeat
func (t *AgentStateTracker) UpdateFromHeartbeat(hb *types.AgentHeartbeat) {
	t.mu.Lock()
	defer t.mu.Unlock()

	existing, exists := t.agents[hb.AgentID]
	if !exists {
		// Agent not registered yet, ignore heartbeat
		return
	}

	// Update state if changed
	stateStart := existing.StateStart
	if existing.State != hb.State {
		stateStart = time.Now()
	}

	existing.State = hb.State
	existing.KPIs = hb.KPIs
	existing.LastHeartbeat = time.Now()
	existing.LastUpdate = time.Now()
	existing.ConnectionStatus = types.StatusConnected
	existing.StateStart = stateStart
}

// UpdateFromStateChange updates an agent's state from a WebSocket state change message
func (t *AgentStateTracker) UpdateFromStateChange(sc *types.AgentStateChange) {
	t.mu.Lock()
	defer t.mu.Unlock()

	existing, exists := t.agents[sc.AgentID]
	if !exists {
		// Agent not registered yet, create new entry
		t.agents[sc.AgentID] = &types.AgentInfo{
			AgentID:          sc.AgentID,
			State:            sc.NewState,
			Department:       sc.Department,
			Location:         sc.Location,
			Team:             sc.Team,
			StateStart:       time.Now(),
			LastUpdate:       time.Now(),
			LastHeartbeat:    time.Now(),
			ConnectionStatus: types.StatusConnected,
			KPIs:             sc.KPIs,
		}
		return
	}

	existing.State = sc.NewState
	existing.KPIs = sc.KPIs
	existing.LastHeartbeat = time.Now()
	existing.LastUpdate = time.Now()
	existing.ConnectionStatus = types.StatusConnected
	existing.StateStart = time.Now()
}

// RegisterAgent registers a new agent connection
func (t *AgentStateTracker) RegisterAgent(reg *types.AgentRegister) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.agents[reg.AgentID] = &types.AgentInfo{
		AgentID:          reg.AgentID,
		State:            reg.State,
		Department:       reg.Department,
		Location:         reg.Location,
		Team:             reg.Team,
		StateStart:       now,
		LastUpdate:       now,
		LastHeartbeat:    now,
		ConnectionStatus: types.StatusConnected,
		KPIs:             reg.KPIs,
	}
}

// SetConnected updates the connection status of an agent
func (t *AgentStateTracker) SetConnected(agentID string, connected bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if agent, exists := t.agents[agentID]; exists {
		if connected {
			agent.ConnectionStatus = types.StatusConnected
			agent.LastHeartbeat = time.Now()
		} else {
			agent.ConnectionStatus = types.StatusDisconnected
			agent.LastHeartbeat = time.Now() // Track when disconnection happened for cleanup
		}
	}
}

// CheckStaleAgents marks agents as stale if no heartbeat received within threshold
func (t *AgentStateTracker) CheckStaleAgents() {
	t.mu.Lock()
	defer t.mu.Unlock()

	threshold := time.Now().Add(-StaleThreshold)
	for _, agent := range t.agents {
		if agent.ConnectionStatus == types.StatusConnected &&
			agent.LastHeartbeat.Before(threshold) {
			agent.ConnectionStatus = types.StatusStale
		}
	}
}

// GetAll returns all agents' current states
func (t *AgentStateTracker) GetAll() []types.AgentInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

	states := make([]types.AgentInfo, 0, len(t.agents))
	for _, state := range t.agents {
		states = append(states, *state)
	}
	return states
}

// GetConnectedAgents returns only agents that are currently connected
func (t *AgentStateTracker) GetConnectedAgents() []types.AgentInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

	states := make([]types.AgentInfo, 0, len(t.agents))
	for _, state := range t.agents {
		if state.ConnectionStatus == types.StatusConnected {
			states = append(states, *state)
		}
	}
	return states
}

// RemoveDisconnected removes agents that have been disconnected for longer than maxAge
func (t *AgentStateTracker) RemoveDisconnected(maxAge time.Duration) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	threshold := time.Now().Add(-maxAge)
	removed := 0
	for id, agent := range t.agents {
		if agent.ConnectionStatus == types.StatusDisconnected &&
			agent.LastHeartbeat.Before(threshold) {
			delete(t.agents, id)
			removed++
		}
	}
	return removed
}

// GetByDepartment returns all agents in a specific department
func (t *AgentStateTracker) GetByDepartment(dept types.Department) []types.AgentInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

	states := make([]types.AgentInfo, 0)
	for _, state := range t.agents {
		if state.Department == dept {
			states = append(states, *state)
		}
	}
	return states
}

// Count returns the total number of tracked agents
func (t *AgentStateTracker) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.agents)
}

// GetConnectionStats returns connection statistics
func (t *AgentStateTracker) GetConnectionStats() (connected, stale, disconnected int) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, agent := range t.agents {
		switch agent.ConnectionStatus {
		case types.StatusConnected:
			connected++
		case types.StatusStale:
			stale++
		case types.StatusDisconnected:
			disconnected++
		}
	}
	return
}
