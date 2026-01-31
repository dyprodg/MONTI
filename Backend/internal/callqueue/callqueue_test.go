package callqueue

import (
	"testing"
	"time"

	"github.com/dennisdiepolder/monti/backend/internal/cache"
	"github.com/dennisdiepolder/monti/backend/internal/types"
	"github.com/rs/zerolog"
)

func TestVQQueueFIFOOrdering(t *testing.T) {
	cfg := VQConfig{Name: types.VQSalesInbound, Department: types.DeptSales, SLTarget: 80, SLSeconds: 20}
	q := NewVQQueue(cfg)

	// Enqueue 3 calls
	c1 := &types.Call{CallID: "call-1", EnqueueTime: time.Now()}
	c2 := &types.Call{CallID: "call-2", EnqueueTime: time.Now()}
	c3 := &types.Call{CallID: "call-3", EnqueueTime: time.Now()}

	q.Enqueue(c1)
	q.Enqueue(c2)
	q.Enqueue(c3)

	if len(q.Waiting) != 3 {
		t.Fatalf("expected 3 waiting, got %d", len(q.Waiting))
	}

	// Dequeue should return in FIFO order
	got := q.DequeueNext()
	if got.CallID != "call-1" {
		t.Errorf("expected call-1 first, got %s", got.CallID)
	}
	got = q.DequeueNext()
	if got.CallID != "call-2" {
		t.Errorf("expected call-2 second, got %s", got.CallID)
	}
	got = q.DequeueNext()
	if got.CallID != "call-3" {
		t.Errorf("expected call-3 third, got %s", got.CallID)
	}

	if q.DequeueNext() != nil {
		t.Error("expected nil from empty queue")
	}
}

func TestLongestIdleFirstSelection(t *testing.T) {
	strategy := &LongestIdleFirst{}

	now := time.Now()
	agents := []types.AgentInfo{
		{AgentID: "agent-1", StateStart: now.Add(-5 * time.Minute)},
		{AgentID: "agent-2", StateStart: now.Add(-10 * time.Minute)}, // longest idle
		{AgentID: "agent-3", StateStart: now.Add(-2 * time.Minute)},
	}

	selected := strategy.SelectAgent(agents)
	if selected == nil {
		t.Fatal("expected agent to be selected")
	}
	if selected.AgentID != "agent-2" {
		t.Errorf("expected agent-2 (longest idle), got %s", selected.AgentID)
	}
}

func TestLongestIdleFirstEmpty(t *testing.T) {
	strategy := &LongestIdleFirst{}
	if strategy.SelectAgent(nil) != nil {
		t.Error("expected nil for empty list")
	}
}

func TestServiceLevelCalculation(t *testing.T) {
	sl := NewSLTracker(80, 20)

	// No calls yet - SL should be 100%
	if sl.CurrentSL() != 100.0 {
		t.Errorf("expected 100%% SL with no calls, got %.1f%%", sl.CurrentSL())
	}

	// 4 calls answered in SL, 1 outside
	sl.RecordAnswer(10) // in SL
	sl.RecordAnswer(15) // in SL
	sl.RecordAnswer(19) // in SL
	sl.RecordAnswer(20) // exactly at threshold, counts as in SL
	sl.RecordAnswer(25) // outside SL

	// 4/5 = 80%
	if sl.CurrentSL() != 80.0 {
		t.Errorf("expected 80%% SL, got %.1f%%", sl.CurrentSL())
	}

	snapshot := sl.Snapshot()
	if snapshot.AnsweredInSL != 4 {
		t.Errorf("expected 4 answered in SL, got %d", snapshot.AnsweredInSL)
	}
	if snapshot.TotalAnswered != 5 {
		t.Errorf("expected 5 total answered, got %d", snapshot.TotalAnswered)
	}
}

func TestCallQueueManagerEnqueueRouteComplete(t *testing.T) {
	tracker := cache.NewAgentStateTracker()
	logger := zerolog.Nop()
	mgr := NewCallQueueManager(tracker, logger)

	// Register an available agent in sales
	tracker.RegisterAgent(&types.AgentRegister{
		AgentID:    "agent-1",
		Department: types.DeptSales,
		Location:   types.LocationBerlin,
		Team:       "Team A",
		State:      types.StateAvailable,
	})

	// Enqueue a call
	call := mgr.EnqueueCall(types.VQSalesInbound, "call-1")
	if call == nil {
		t.Fatal("expected call to be enqueued")
	}
	if call.Status != types.CallStatusWaiting {
		t.Errorf("expected waiting status, got %s", call.Status)
	}

	// Route
	matches := mgr.TickRouting()
	if len(matches) != 1 {
		t.Fatalf("expected 1 routing match, got %d", len(matches))
	}
	if matches[0].AgentID != "agent-1" {
		t.Errorf("expected agent-1, got %s", matches[0].AgentID)
	}
	if matches[0].Call.Status != types.CallStatusActive {
		t.Errorf("expected active status, got %s", matches[0].Call.Status)
	}

	// Complete
	completed := mgr.CompleteCall("call-1", 120.0, 5.0)
	if completed == nil {
		t.Fatal("expected call to be completed")
	}
	if completed.TalkTime != 120.0 {
		t.Errorf("expected 120s talk time, got %.1f", completed.TalkTime)
	}

	// Check SL was updated
	snapshot := mgr.GetSnapshot(types.VQSalesInbound)
	if snapshot == nil {
		t.Fatal("expected snapshot")
	}
	if snapshot.ServiceLevel.TotalAnswered != 1 {
		t.Errorf("expected 1 total answered, got %d", snapshot.ServiceLevel.TotalAnswered)
	}
}

func TestCallQueueManagerNoAvailableAgent(t *testing.T) {
	tracker := cache.NewAgentStateTracker()
	logger := zerolog.Nop()
	mgr := NewCallQueueManager(tracker, logger)

	// No agents registered
	mgr.EnqueueCall(types.VQSalesInbound, "call-1")
	matches := mgr.TickRouting()
	if len(matches) != 0 {
		t.Errorf("expected 0 matches with no agents, got %d", len(matches))
	}

	// Call should still be waiting
	snapshot := mgr.GetSnapshot(types.VQSalesInbound)
	if snapshot.WaitingCount != 1 {
		t.Errorf("expected 1 waiting, got %d", snapshot.WaitingCount)
	}
}

func TestCallQueueManagerAbandon(t *testing.T) {
	tracker := cache.NewAgentStateTracker()
	logger := zerolog.Nop()
	mgr := NewCallQueueManager(tracker, logger)

	mgr.EnqueueCall(types.VQSupportGeneral, "call-1")
	abandoned := mgr.AbandonCall("call-1")
	if abandoned == nil {
		t.Fatal("expected call to be abandoned")
	}
	if abandoned.Status != types.CallStatusAbandoned {
		t.Errorf("expected abandoned status, got %s", abandoned.Status)
	}

	snapshot := mgr.GetSnapshot(types.VQSupportGeneral)
	if snapshot.AbandonedCount != 1 {
		t.Errorf("expected 1 abandoned, got %d", snapshot.AbandonedCount)
	}
}

func TestGetAllSnapshots(t *testing.T) {
	tracker := cache.NewAgentStateTracker()
	logger := zerolog.Nop()
	mgr := NewCallQueueManager(tracker, logger)

	snapshots := mgr.GetAllSnapshots()
	if len(snapshots) != 4 {
		t.Errorf("expected 4 departments, got %d", len(snapshots))
	}
	for dept, qs := range snapshots {
		if len(qs) != 4 {
			t.Errorf("expected 4 VQs for %s, got %d", dept, len(qs))
		}
	}
}
