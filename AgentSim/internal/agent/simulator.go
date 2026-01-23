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
	agents        []types.Agent
	activeAgents  map[string]bool
	agentCancels  map[string]context.CancelFunc
	mu            sync.RWMutex
	rng           *rand.Rand
	logger        zerolog.Logger
	backendURL    string
	httpClient    *http.Client
	eventsSent    int64
	backendErrors int64
	running       bool
	ctx           context.Context
	cancel        context.CancelFunc

	// Additional metrics
	startTime          time.Time
	stateTransitions   int64
	stateChangeCounts  map[types.AgentState]int64
	stateMu            sync.RWMutex
}

// NewSimulator creates a new agent simulator
func NewSimulator(agents []types.Agent, backendURL string, logger zerolog.Logger) *Simulator {
	// Optimize HTTP transport for high concurrency (2000 agents)
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     200,
		IdleConnTimeout:     90 * time.Second,
	}

	return &Simulator{
		agents:            agents,
		activeAgents:      make(map[string]bool),
		agentCancels:      make(map[string]context.CancelFunc),
		rng:               rand.New(rand.NewSource(time.Now().UnixNano())),
		logger:            logger,
		backendURL:        backendURL,
		startTime:         time.Now(),
		stateChangeCounts: make(map[types.AgentState]int64),
		httpClient: &http.Client{
			Timeout:   5 * time.Second,
			Transport: transport,
		},
	}
}

// Start begins simulating agent state changes
func (s *Simulator) Start(ctx context.Context, numActive int) {
	s.mu.Lock()
	s.running = true
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.mu.Unlock()

	// Activate the specified number of agents
	s.activateAgents(numActive)

	// Start goroutine for each active agent
	s.mu.Lock()
	for id := range s.activeAgents {
		agentCtx, agentCancel := context.WithCancel(s.ctx)
		s.agentCancels[id] = agentCancel
		go s.simulateAgent(agentCtx, id)
	}
	s.mu.Unlock()

	s.logger.Info().Int("active_agents", numActive).Msg("agent simulation started")
}

// Stop stops all active agents
func (s *Simulator) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.running = false

	// Cancel all agent goroutines
	for id, cancel := range s.agentCancels {
		cancel()
		delete(s.agentCancels, id)
	}

	// Clear active agents
	s.activeAgents = make(map[string]bool)

	if s.cancel != nil {
		s.cancel()
	}

	s.logger.Info().Msg("all agents stopped")
}

// Scale dynamically adjusts the number of active agents
func (s *Simulator) Scale(ctx context.Context, targetAgents int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if targetAgents > len(s.agents) {
		targetAgents = len(s.agents)
	}
	if targetAgents < 0 {
		targetAgents = 0
	}

	currentCount := len(s.activeAgents)
	s.logger.Info().
		Int("current", currentCount).
		Int("target", targetAgents).
		Msg("scaling agents")

	if targetAgents > currentCount {
		// Scale up: add more agents
		needed := targetAgents - currentCount

		// Get inactive agents
		var inactiveIndices []int
		for i := range s.agents {
			if !s.activeAgents[s.agents[i].ID] {
				inactiveIndices = append(inactiveIndices, i)
			}
		}

		// Shuffle and take needed
		s.rng.Shuffle(len(inactiveIndices), func(i, j int) {
			inactiveIndices[i], inactiveIndices[j] = inactiveIndices[j], inactiveIndices[i]
		})

		if needed > len(inactiveIndices) {
			needed = len(inactiveIndices)
		}

		// Ensure we have a valid context
		if s.ctx == nil {
			s.ctx, s.cancel = context.WithCancel(ctx)
			s.running = true
		}

		for i := 0; i < needed; i++ {
			idx := inactiveIndices[i]
			agent := &s.agents[idx]
			agent.State = types.StateAvailable
			agent.StateStart = time.Now()
			agent.LastUpdate = time.Now()
			agent.LoginTime = time.Now()
			agent.KPIs = s.generateInitialKPIs()
			s.activeAgents[agent.ID] = true

			// Start agent goroutine
			agentCtx, agentCancel := context.WithCancel(s.ctx)
			s.agentCancels[agent.ID] = agentCancel
			go s.simulateAgent(agentCtx, agent.ID)

			// Send initial state event
			go s.sendEvent(*agent, 0.0)
		}

	} else if targetAgents < currentCount {
		// Scale down: remove agents
		toRemove := currentCount - targetAgents

		// Get list of active agent IDs
		var activeIDs []string
		for id := range s.activeAgents {
			activeIDs = append(activeIDs, id)
		}

		// Randomly select agents to deactivate
		s.rng.Shuffle(len(activeIDs), func(i, j int) {
			activeIDs[i], activeIDs[j] = activeIDs[j], activeIDs[i]
		})

		for i := 0; i < toRemove && i < len(activeIDs); i++ {
			id := activeIDs[i]
			if cancel, ok := s.agentCancels[id]; ok {
				cancel()
				delete(s.agentCancels, id)
			}
			delete(s.activeAgents, id)
		}
	}

	s.logger.Info().
		Int("active_agents", len(s.activeAgents)).
		Msg("scaling complete")

	return nil
}

// IsRunning returns whether the simulation is running
func (s *Simulator) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
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
		agent.LoginTime = time.Now()
		agent.KPIs = s.generateInitialKPIs()
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
			previousState := s.agents[i].State
			stateDuration = time.Since(s.agents[i].StateStart).Seconds()

			// Update KPIs before changing state
			s.updateKPIs(&s.agents[i], previousState, stateDuration)

			s.agents[i].State = newState
			s.agents[i].StateStart = time.Now()
			s.agents[i].LastUpdate = time.Now()
			agent = s.agents[i]
			break
		}
	}
	s.mu.Unlock()

	// Track state transition metrics
	atomic.AddInt64(&s.stateTransitions, 1)
	s.stateMu.Lock()
	s.stateChangeCounts[newState]++
	s.stateMu.Unlock()

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
		KPIs:          agent.KPIs,
	}

	data, err := json.Marshal(event)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to marshal event")
		atomic.AddInt64(&s.backendErrors, 1)
		return
	}

	resp, err := s.httpClient.Post(
		s.backendURL+"/internal/event",
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		s.logger.Debug().Err(err).Str("agent_id", agent.ID).Msg("failed to send event")
		atomic.AddInt64(&s.backendErrors, 1)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.Debug().Int("status", resp.StatusCode).Str("agent_id", agent.ID).Msg("backend returned non-200 status")
		atomic.AddInt64(&s.backendErrors, 1)
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

// GetBackendErrors returns the total number of backend errors
func (s *Simulator) GetBackendErrors() int64 {
	return atomic.LoadInt64(&s.backendErrors)
}

// GetMetrics returns Prometheus-compatible metrics
func (s *Simulator) GetMetrics() map[string]interface{} {
	s.mu.RLock()
	activeCount := len(s.activeAgents)
	totalAgents := len(s.agents)

	// Count agents by state, department, and location
	stateCount := make(map[types.AgentState]int)
	deptCount := make(map[types.Department]int)
	locCount := make(map[types.Location]int)

	for _, agent := range s.agents {
		if s.activeAgents[agent.ID] {
			stateCount[agent.State]++
			deptCount[agent.Department]++
			locCount[agent.Location]++
		}
	}
	s.mu.RUnlock()

	// Calculate events per second
	uptime := time.Since(s.startTime).Seconds()
	eventsPerSecond := float64(0)
	if uptime > 0 {
		eventsPerSecond = float64(atomic.LoadInt64(&s.eventsSent)) / uptime
	}

	metrics := map[string]interface{}{
		"agentsim_active_agents":        activeCount,
		"agentsim_total_agents":         totalAgents,
		"agentsim_events_sent_total":    atomic.LoadInt64(&s.eventsSent),
		"agentsim_backend_errors_total": atomic.LoadInt64(&s.backendErrors),
		"agentsim_state_transitions":    atomic.LoadInt64(&s.stateTransitions),
		"agentsim_events_per_second":    eventsPerSecond,
		"agentsim_uptime_seconds":       uptime,
		"agentsim_running":              s.IsRunning(),
	}

	// Add state breakdown
	for state, count := range stateCount {
		metrics["agentsim_agents_by_state{state=\""+string(state)+"\"}"] = count
	}

	// Add department breakdown
	for dept, count := range deptCount {
		metrics["agentsim_agents_by_department{department=\""+string(dept)+"\"}"] = count
	}

	// Add location breakdown
	for loc, count := range locCount {
		metrics["agentsim_agents_by_location{location=\""+string(loc)+"\"}"] = count
	}

	return metrics
}

// generateInitialKPIs creates realistic initial KPI values for a newly logged-in agent
func (s *Simulator) generateInitialKPIs() types.AgentKPIs {
	return types.AgentKPIs{
		TotalCalls:           0,
		AvgCallDuration:      0,
		AcwTime:              0,
		AcwCount:             0,
		HoldCount:            0,
		HoldTime:             0,
		TransferCount:        0,
		ConferenceCount:      0,
		BreakTime:            0,
		LoginTime:            0,
		Occupancy:            0,
		Adherence:            85 + s.rng.Float64()*15, // 85-100% starting adherence
		AvgHandleTime:        0,
		FirstCallResolution:  70 + s.rng.Float64()*25, // 70-95% FCR
		CustomerSatisfaction: 3.5 + s.rng.Float64()*1.5, // 3.5-5.0 CSAT
	}
}

// updateKPIs updates agent KPIs based on current state and duration
func (s *Simulator) updateKPIs(agent *types.Agent, previousState types.AgentState, stateDuration float64) {
	now := time.Now()
	agent.KPIs.LoginTime = now.Sub(agent.LoginTime).Seconds()

	switch previousState {
	case types.StateOnCall:
		agent.KPIs.TotalCalls++
		// Update average call duration
		if agent.KPIs.TotalCalls == 1 {
			agent.KPIs.AvgCallDuration = stateDuration
		} else {
			agent.KPIs.AvgCallDuration =
				(agent.KPIs.AvgCallDuration*float64(agent.KPIs.TotalCalls-1) + stateDuration) / float64(agent.KPIs.TotalCalls)
		}
		// Update average handle time (simplified: same as call duration for now)
		agent.KPIs.AvgHandleTime = agent.KPIs.AvgCallDuration

		// Randomly adjust FCR and CSAT slightly
		agent.KPIs.FirstCallResolution = clamp(agent.KPIs.FirstCallResolution+(s.rng.Float64()-0.5)*2, 60, 100)
		agent.KPIs.CustomerSatisfaction = clamp(agent.KPIs.CustomerSatisfaction+(s.rng.Float64()-0.5)*0.2, 1, 5)

	case types.StateAfterCallWork:
		agent.KPIs.AcwCount++
		agent.KPIs.AcwTime += stateDuration

	case types.StateOnHold:
		agent.KPIs.HoldCount++
		agent.KPIs.HoldTime += stateDuration

	case types.StateTransferring:
		agent.KPIs.TransferCount++

	case types.StateConference:
		agent.KPIs.ConferenceCount++

	case types.StateBreak, types.StateLunch:
		agent.KPIs.BreakTime += stateDuration
	}

	// Calculate occupancy: (call time + ACW time) / (login time - break time) * 100
	productiveTime := agent.KPIs.AvgCallDuration*float64(agent.KPIs.TotalCalls) + agent.KPIs.AcwTime
	availableTime := agent.KPIs.LoginTime - agent.KPIs.BreakTime
	if availableTime > 0 {
		agent.KPIs.Occupancy = clamp((productiveTime/availableTime)*100, 0, 100)
	}

	// Adherence fluctuates slightly
	agent.KPIs.Adherence = clamp(agent.KPIs.Adherence+(s.rng.Float64()-0.5)*1, 70, 100)
}

// clamp restricts a value to a min/max range
func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
