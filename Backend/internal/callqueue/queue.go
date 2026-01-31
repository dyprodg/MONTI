package callqueue

import (
	"time"

	"github.com/dennisdiepolder/monti/backend/internal/types"
)

// VQQueue represents a per-VQ FIFO queue
type VQQueue struct {
	Name       types.VQName
	Department types.Department
	Waiting    []*types.Call            // FIFO queue of waiting calls
	Active     map[string]*types.Call   // callID -> active call
	Completed  int
	Abandoned  int
	SL         *SLTracker
}

// NewVQQueue creates a new per-VQ queue
func NewVQQueue(config VQConfig) *VQQueue {
	return &VQQueue{
		Name:       config.Name,
		Department: config.Department,
		Waiting:    make([]*types.Call, 0),
		Active:     make(map[string]*types.Call),
		SL:         NewSLTracker(config.SLTarget, config.SLSeconds),
	}
}

// Enqueue adds a call to the waiting queue
func (q *VQQueue) Enqueue(call *types.Call) {
	call.Status = types.CallStatusWaiting
	q.Waiting = append(q.Waiting, call)
}

// DequeueNext removes and returns the next waiting call (FIFO)
func (q *VQQueue) DequeueNext() *types.Call {
	if len(q.Waiting) == 0 {
		return nil
	}
	call := q.Waiting[0]
	q.Waiting = q.Waiting[1:]
	return call
}

// AssignToAgent moves a call from waiting to active
func (q *VQQueue) AssignToAgent(call *types.Call, agentID string) {
	now := time.Now()
	call.Status = types.CallStatusActive
	call.AgentID = agentID
	call.AssignTime = &now
	call.WaitTime = now.Sub(call.EnqueueTime).Seconds()
	q.Active[call.CallID] = call

	// Record SL
	q.SL.RecordAnswer(call.WaitTime)
}

// CompleteCall marks a call as completed and removes from active
func (q *VQQueue) CompleteCall(callID string, talkTime, holdTime float64) *types.Call {
	call, ok := q.Active[callID]
	if !ok {
		return nil
	}
	now := time.Now()
	call.Status = types.CallStatusCompleted
	call.CompleteTime = &now
	call.TalkTime = talkTime
	call.HoldTime = holdTime
	delete(q.Active, callID)
	q.Completed++
	return call
}

// AbandonCall marks the first waiting call as abandoned (or specific by ID)
func (q *VQQueue) AbandonCall(callID string) *types.Call {
	for i, call := range q.Waiting {
		if call.CallID == callID {
			q.Waiting = append(q.Waiting[:i], q.Waiting[i+1:]...)
			now := time.Now()
			call.Status = types.CallStatusAbandoned
			call.CompleteTime = &now
			call.WaitTime = now.Sub(call.EnqueueTime).Seconds()
			q.Abandoned++
			return call
		}
	}
	return nil
}

// LongestWaitSecs returns the wait time of the oldest waiting call
func (q *VQQueue) LongestWaitSecs() float64 {
	if len(q.Waiting) == 0 {
		return 0
	}
	return time.Since(q.Waiting[0].EnqueueTime).Seconds()
}

// Wipe clears all waiting and active calls, returning the count of cleared calls
func (q *VQQueue) Wipe() int {
	count := len(q.Waiting) + len(q.Active)
	q.Waiting = nil
	q.Active = make(map[string]*types.Call)
	return count
}

// Snapshot returns a VQSnapshot of the current queue state
func (q *VQQueue) Snapshot(availableAgents int) types.VQSnapshot {
	return types.VQSnapshot{
		VQ:              q.Name,
		Department:      q.Department,
		WaitingCount:    len(q.Waiting),
		ActiveCount:     len(q.Active),
		CompletedCount:  q.Completed,
		AbandonedCount:  q.Abandoned,
		LongestWaitSecs: q.LongestWaitSecs(),
		AvailableAgents: availableAgents,
		ServiceLevel:    q.SL.Snapshot(),
	}
}
