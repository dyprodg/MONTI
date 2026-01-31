package callqueue

import (
	"sync"
	"time"

	"github.com/dennisdiepolder/monti/backend/internal/cache"
	"github.com/dennisdiepolder/monti/backend/internal/types"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// CallStore is the subset of storage.Store needed by CallQueueManager
type CallStore interface {
	SaveCallRecord(record types.CallRecord) error
}

// CallQueueManager manages all virtual queues and call routing
type CallQueueManager struct {
	queues   map[types.VQName]*VQQueue
	configs  map[types.VQName]VQConfig
	tracker  *cache.AgentStateTracker
	routing  RoutingStrategy
	store    CallStore
	mu       sync.RWMutex
	logger   zerolog.Logger
}

// NewCallQueueManager creates a new call queue manager
func NewCallQueueManager(tracker *cache.AgentStateTracker, logger zerolog.Logger) *CallQueueManager {
	configs := DefaultVQConfigs()
	queues := make(map[types.VQName]*VQQueue, len(configs))
	for name, cfg := range configs {
		queues[name] = NewVQQueue(cfg)
	}

	return &CallQueueManager{
		queues:  queues,
		configs: configs,
		tracker: tracker,
		routing: &LongestIdleFirst{},
		logger:  logger,
	}
}

// SetStore sets the persistence store for call records
func (m *CallQueueManager) SetStore(store CallStore) {
	m.store = store
}

// EnqueueCall adds a new call to the appropriate VQ
func (m *CallQueueManager) EnqueueCall(vq types.VQName, callID string) *types.Call {
	m.mu.Lock()
	defer m.mu.Unlock()

	queue, ok := m.queues[vq]
	if !ok {
		m.logger.Warn().Str("vq", string(vq)).Msg("unknown VQ, ignoring call")
		return nil
	}

	if callID == "" {
		callID = uuid.New().String()
	}

	dept := types.VQDepartmentMapping[vq]
	call := &types.Call{
		CallID:      callID,
		VQ:          vq,
		Department:  dept,
		Status:      types.CallStatusWaiting,
		EnqueueTime: time.Now(),
	}

	queue.Enqueue(call)

	m.logger.Debug().
		Str("call_id", callID).
		Str("vq", string(vq)).
		Str("department", string(dept)).
		Int("queue_depth", len(queue.Waiting)).
		Msg("call enqueued")

	return call
}

// CompleteCall marks a call as completed
func (m *CallQueueManager) CompleteCall(callID string, talkTime, holdTime float64) *types.Call {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Search all queues for the active call
	for _, queue := range m.queues {
		if call := queue.CompleteCall(callID, talkTime, holdTime); call != nil {
			m.logger.Debug().
				Str("call_id", callID).
				Str("agent_id", call.AgentID).
				Float64("talk_time", talkTime).
				Msg("call completed")

			// Persist call record asynchronously
			if m.store != nil {
				record := callToRecord(call)
				go func() {
					if err := m.store.SaveCallRecord(record); err != nil {
						m.logger.Error().Err(err).Str("call_id", callID).Msg("failed to save call record")
					}
				}()
			}
			return call
		}
	}

	m.logger.Debug().Str("call_id", callID).Msg("call not found in active calls")
	return nil
}

// AbandonCall marks a waiting call as abandoned
func (m *CallQueueManager) AbandonCall(callID string) *types.Call {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, queue := range m.queues {
		if call := queue.AbandonCall(callID); call != nil {
			m.logger.Debug().
				Str("call_id", callID).
				Str("vq", string(queue.Name)).
				Msg("call abandoned")
			return call
		}
	}
	return nil
}

// TickRouting tries to match waiting calls to available agents.
// Returns a list of (call, agentID) pairs that were matched.
func (m *CallQueueManager) TickRouting() []RoutingMatch {
	m.mu.Lock()
	defer m.mu.Unlock()

	var matches []RoutingMatch

	// Process each department's VQs
	for dept, vqNames := range types.DepartmentVQs {
		// Get available agents for this department
		available := m.tracker.GetAvailableByDepartment(dept)
		if len(available) == 0 {
			continue
		}

		// Track which agents have been assigned in this tick
		assigned := make(map[string]bool)

		// Round-robin through VQs in the department
		for _, vqName := range vqNames {
			queue := m.queues[vqName]
			for len(queue.Waiting) > 0 {
				// Filter out already-assigned agents
				free := filterUnassigned(available, assigned)
				if len(free) == 0 {
					break
				}

				agent := m.routing.SelectAgent(free)
				if agent == nil {
					break
				}

				call := queue.DequeueNext()
				queue.AssignToAgent(call, agent.AgentID)
				assigned[agent.AgentID] = true

				matches = append(matches, RoutingMatch{
					Call:    call,
					AgentID: agent.AgentID,
				})

				m.logger.Debug().
					Str("call_id", call.CallID).
					Str("agent_id", agent.AgentID).
					Str("vq", string(vqName)).
					Float64("wait_time", call.WaitTime).
					Msg("call routed to agent")
			}
		}
	}

	return matches
}

// RoutingMatch represents a call matched to an agent
type RoutingMatch struct {
	Call    *types.Call
	AgentID string
}

// GetSnapshot returns the snapshot for a specific VQ
func (m *CallQueueManager) GetSnapshot(vq types.VQName) *types.VQSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	queue, ok := m.queues[vq]
	if !ok {
		return nil
	}

	available := m.tracker.GetAvailableByDepartment(queue.Department)
	snapshot := queue.Snapshot(len(available))
	return &snapshot
}

// GetAllSnapshots returns snapshots for all VQs grouped by department
func (m *CallQueueManager) GetAllSnapshots() map[types.Department][]types.VQSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[types.Department][]types.VQSnapshot)

	for dept, vqNames := range types.DepartmentVQs {
		available := m.tracker.GetAvailableByDepartment(dept)
		availCount := len(available)

		snapshots := make([]types.VQSnapshot, 0, len(vqNames))
		for _, vqName := range vqNames {
			queue := m.queues[vqName]
			snapshots = append(snapshots, queue.Snapshot(availCount))
		}
		result[dept] = snapshots
	}

	return result
}

// WipeAllCalls clears all waiting and active calls from every queue
func (m *CallQueueManager) WipeAllCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	total := 0
	for _, queue := range m.queues {
		total += queue.Wipe()
	}

	m.logger.Info().Int("cleared", total).Msg("wiped all calls from all queues")
	return total
}

// ForceEndCall finds an active call by ID, marks it completed, and returns the agentID
func (m *CallQueueManager) ForceEndCall(callID string) (agentID string, found bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, queue := range m.queues {
		call, ok := queue.Active[callID]
		if !ok {
			continue
		}

		talkTime := 0.0
		if call.AssignTime != nil {
			talkTime = time.Since(*call.AssignTime).Seconds()
		}

		completed := queue.CompleteCall(callID, talkTime, 0)
		if completed == nil {
			continue
		}

		m.logger.Info().
			Str("call_id", callID).
			Str("agent_id", completed.AgentID).
			Float64("talk_time", talkTime).
			Msg("call force-ended")

		if m.store != nil {
			record := callToRecord(completed)
			go func() {
				if err := m.store.SaveCallRecord(record); err != nil {
					m.logger.Error().Err(err).Str("call_id", callID).Msg("failed to save force-ended call record")
				}
			}()
		}

		return completed.AgentID, true
	}

	return "", false
}

// filterUnassigned returns agents not in the assigned map
func filterUnassigned(agents []types.AgentInfo, assigned map[string]bool) []types.AgentInfo {
	result := make([]types.AgentInfo, 0, len(agents))
	for _, a := range agents {
		if !assigned[a.AgentID] {
			result = append(result, a)
		}
	}
	return result
}

// callToRecord converts a completed Call to a CallRecord for persistence
func callToRecord(call *types.Call) types.CallRecord {
	record := types.CallRecord{
		CallID:     call.CallID,
		VQ:         call.VQ,
		Department: string(call.Department),
		AgentID:    call.AgentID,
		WaitTime:   call.WaitTime,
		TalkTime:   call.TalkTime,
		HoldTime:   call.HoldTime,
		WrapTime:   call.WrapTime,
		HandleTime: call.TalkTime + call.HoldTime + call.WrapTime,
		Abandoned:  call.Status == types.CallStatusAbandoned,
	}

	record.DateKey = call.EnqueueTime.Format("2006-01-02")
	record.EnqueueTime = call.EnqueueTime.Format(time.RFC3339)
	if call.AssignTime != nil {
		record.AssignTime = call.AssignTime.Format(time.RFC3339)
		record.AnsweredInSL = call.WaitTime <= 20.0
	}
	if call.CompleteTime != nil {
		record.CompleteTime = call.CompleteTime.Format(time.RFC3339)
	}

	return record
}
