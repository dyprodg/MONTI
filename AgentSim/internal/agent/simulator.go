package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dennisdiepolder/monti/agentsim/internal/types"
	"github.com/rs/zerolog"
)

// Simulator manages agent state transitions
type Simulator struct {
	agents       []types.Agent
	activeAgents map[string]bool
	mu           sync.RWMutex
	rng          *rand.Rand
	logger       zerolog.Logger
	backendURL   string
	httpClient   *http.Client
	eventsSent   int64
}

// NewSimulator creates a new agent simulator
func NewSimulator(agents []types.Agent, backendURL string, logger zerolog.Logger) *Simulator {
	return &Simulator{
		agents:       agents,
		activeAgents: make(map[string]bool),
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
		logger:       logger,
		backendURL:   backendURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Start begins simulating agent state changes
func (s *Simulator) Start(ctx context.Context, numActive int) {
	// Activate the specified number of agents
	s.activateAgents(numActive)

	// Start goroutine for each active agent
	for id := range s.activeAgents {
		go s.simulateAgent(ctx, id)
	}

	s.logger.Info().Int("active_agents", numActive).Msg("agent simulation started")
}

// activateAgents sets initial agents to available state
func (s *Simulator) activateAgents(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if count > len(s.agents) {
		count = len(s.agents)
	}

	// Randomly select agents to activate
	indices := s.rng.Perm(len(s.agents))[:count]

	for _, idx := range indices {
		agent := &s.agents[idx]
		agent.State = types.StateAvailable
		agent.StateStart = time.Now()
		agent.LastUpdate = time.Now()
		s.activeAgents[agent.ID] = true

		// Send initial state event
		go s.sendEvent(*agent, 0.0)
	}
}

// simulateAgent runs the state machine for a single agent
func (s *Simulator) simulateAgent(ctx context.Context, agentID string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			agent := s.getAgent(agentID)
			if agent == nil {
				return
			}

			// Wait in current state for a duration
			duration := s.getStateDuration(agent.State)
			time.Sleep(duration)

			// Transition to next state
			nextState := s.getNextState(agent.State)
			s.updateAgentState(agentID, nextState)
		}
	}
}

// getAgent safely retrieves an agent by ID
func (s *Simulator) getAgent(id string) *types.Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.agents {
		if s.agents[i].ID == id {
			return &s.agents[i]
		}
	}
	return nil
}

// updateAgentState updates an agent's state and sends event to backend
func (s *Simulator) updateAgentState(agentID string, newState types.AgentState) {
	s.mu.Lock()
	var agent types.Agent
	var stateDuration float64

	for i := range s.agents {
		if s.agents[i].ID == agentID {
			stateDuration = time.Since(s.agents[i].StateStart).Seconds()
			s.agents[i].State = newState
			s.agents[i].StateStart = time.Now()
			s.agents[i].LastUpdate = time.Now()
			agent = s.agents[i]
			break
		}
	}
	s.mu.Unlock()

	// Send event to backend (non-blocking)
	go s.sendEvent(agent, stateDuration)
}

// sendEvent sends an agent event to the backend
func (s *Simulator) sendEvent(agent types.Agent, stateDuration float64) {
	event := types.AgentEvent{
		AgentID:       agent.ID,
		State:         agent.State,
		Department:    agent.Department,
		Location:      agent.Location,
		Team:          agent.Team,
		Timestamp:     time.Now(),
		StateDuration: stateDuration,
	}

	data, err := json.Marshal(event)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to marshal event")
		return
	}

	resp, err := s.httpClient.Post(
		s.backendURL+"/internal/event",
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		s.logger.Debug().Err(err).Str("agent_id", agent.ID).Msg("failed to send event")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.Debug().Int("status", resp.StatusCode).Str("agent_id", agent.ID).Msg("backend returned non-200 status")
		return
	}

	atomic.AddInt64(&s.eventsSent, 1)
}

// getStateDuration returns how long an agent should stay in a state
func (s *Simulator) getStateDuration(state types.AgentState) time.Duration {
	base := time.Duration(0)

	switch state {
	case types.StateAvailable:
		base = time.Duration(3+s.rng.Intn(10)) * time.Second
	case types.StateOnCall:
		base = time.Duration(30+s.rng.Intn(180)) * time.Second // 30s-3.5min
	case types.StateAfterCallWork:
		base = time.Duration(10+s.rng.Intn(20)) * time.Second // 10-30s
	case types.StateBreak:
		base = time.Duration(300+s.rng.Intn(300)) * time.Second // 5-10min
	case types.StateLunch:
		base = time.Duration(1800+s.rng.Intn(1800)) * time.Second // 30-60min
	case types.StateMeeting:
		base = time.Duration(600+s.rng.Intn(1800)) * time.Second // 10-40min
	case types.StateTraining:
		base = time.Duration(1800+s.rng.Intn(3600)) * time.Second // 30-90min
	case types.StateOnHold:
		base = time.Duration(10+s.rng.Intn(30)) * time.Second // 10-40s
	case types.StateTransferring:
		base = time.Duration(5+s.rng.Intn(10)) * time.Second // 5-15s
	case types.StateConference:
		base = time.Duration(60+s.rng.Intn(240)) * time.Second // 1-5min
	default:
		base = time.Duration(5+s.rng.Intn(10)) * time.Second
	}

	return base
}

// getNextState determines the next state based on current state and probabilities
func (s *Simulator) getNextState(current types.AgentState) types.AgentState {
	roll := s.rng.Float64()

	switch current {
	case types.StateAvailable:
		if roll < 0.7 {
			return types.StateOnCall
		} else if roll < 0.85 {
			return types.StateBreak
		} else if roll < 0.95 {
			return types.StateMeeting
		}
		return types.StateTraining

	case types.StateOnCall:
		if roll < 0.05 {
			return types.StateOnHold
		} else if roll < 0.10 {
			return types.StateTransferring
		} else if roll < 0.12 {
			return types.StateConference
		}
		return types.StateAfterCallWork

	case types.StateAfterCallWork:
		if roll < 0.80 {
			return types.StateAvailable
		} else if roll < 0.95 {
			return types.StateBreak
		}
		return types.StateLunch

	case types.StateOnHold:
		return types.StateOnCall

	case types.StateTransferring:
		return types.StateAfterCallWork

	case types.StateConference:
		return types.StateAfterCallWork

	case types.StateBreak:
		return types.StateAvailable

	case types.StateLunch:
		return types.StateAvailable

	case types.StateMeeting:
		return types.StateAvailable

	case types.StateTraining:
		return types.StateAvailable

	default:
		return types.StateAvailable
	}
}

// GetAllAgents returns a snapshot of all agents
func (s *Simulator) GetAllAgents() []types.Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := make([]types.Agent, len(s.agents))
	copy(snapshot, s.agents)
	return snapshot
}

// GetActiveCount returns the number of active agents
func (s *Simulator) GetActiveCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.activeAgents)
}

// GetEventsSent returns the total number of events sent to backend
func (s *Simulator) GetEventsSent() int64 {
	return atomic.LoadInt64(&s.eventsSent)
}
