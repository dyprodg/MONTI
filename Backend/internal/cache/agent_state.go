package cache

import (
	"sync"

	"github.com/dennisdiepolder/monti/backend/internal/types"
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

// Update updates or adds an agent's state
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

	t.agents[event.AgentID] = &types.AgentInfo{
		AgentID:    event.AgentID,
		State:      event.State,
		Department: event.Department,
		Location:   event.Location,
		Team:       event.Team,
		StateStart: stateStart,
		LastUpdate: event.Timestamp,
		KPIs:       event.KPIs,
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
