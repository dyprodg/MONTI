package websocket

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/dennisdiepolder/monti/backend/internal/types"
	"github.com/rs/zerolog"
)

func TestNewHub(t *testing.T) {
	logger := zerolog.New(&bytes.Buffer{})
	hub := NewHub(logger)

	if hub == nil {
		t.Fatal("expected hub to be created")
	}

	if hub.clients == nil {
		t.Error("expected clients map to be initialized")
	}

	if hub.broadcast == nil {
		t.Error("expected broadcast channel to be initialized")
	}

	if hub.register == nil {
		t.Error("expected register channel to be initialized")
	}

	if hub.unregister == nil {
		t.Error("expected unregister channel to be initialized")
	}
}

func TestHubClientCount(t *testing.T) {
	logger := zerolog.New(&bytes.Buffer{})
	hub := NewHub(logger)

	// Initial count should be 0
	if hub.ClientCount() != 0 {
		t.Errorf("expected 0 clients, got %d", hub.ClientCount())
	}

	// Simulate adding clients
	hub.mu.Lock()
	hub.clients[&Client{id: "test1"}] = true
	hub.clients[&Client{id: "test2"}] = true
	hub.mu.Unlock()

	if hub.ClientCount() != 2 {
		t.Errorf("expected 2 clients, got %d", hub.ClientCount())
	}
}

func TestHubBroadcast(t *testing.T) {
	logger := zerolog.New(&bytes.Buffer{})
	hub := NewHub(logger)

	// Start hub in goroutine
	go hub.Run()

	// Give hub time to start
	time.Sleep(10 * time.Millisecond)

	// Test broadcast
	message := []byte("test message")
	hub.Broadcast(message)

	// The broadcast should succeed without blocking
	select {
	case <-time.After(100 * time.Millisecond):
		t.Error("broadcast blocked unexpectedly")
	default:
		// Broadcast completed
	}
}

func TestHubRegisterUnregister(t *testing.T) {
	logger := zerolog.New(&bytes.Buffer{})
	hub := NewHub(logger)

	// Start hub in goroutine
	go hub.Run()

	// Create mock client
	client := &Client{
		id:   "test-client",
		hub:  hub,
		send: make(chan []byte, 1),
	}

	// Register client
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Errorf("expected 1 client after register, got %d", hub.ClientCount())
	}

	// Unregister client
	hub.unregister <- client
	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Errorf("expected 0 clients after unregister, got %d", hub.ClientCount())
	}
}

func TestHubBroadcastToMultipleClients(t *testing.T) {
	logger := zerolog.New(&bytes.Buffer{})
	hub := NewHub(logger)

	// Start hub
	go hub.Run()

	// Create multiple mock clients
	client1 := &Client{
		id:   "client1",
		hub:  hub,
		send: make(chan []byte, 10),
	}

	client2 := &Client{
		id:   "client2",
		hub:  hub,
		send: make(chan []byte, 10),
	}

	// Register clients
	hub.register <- client1
	hub.register <- client2
	time.Sleep(10 * time.Millisecond)

	// Broadcast message
	message := []byte("test broadcast")
	hub.Broadcast(message)

	// Wait for message to be sent
	time.Sleep(10 * time.Millisecond)

	// Check both clients received the message
	select {
	case msg := <-client1.send:
		if string(msg) != string(message) {
			t.Errorf("client1 expected %s, got %s", message, msg)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("client1 did not receive message")
	}

	select {
	case msg := <-client2.send:
		if string(msg) != string(message) {
			t.Errorf("client2 expected %s, got %s", message, msg)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("client2 did not receive message")
	}
}

func makeSnapshot(id int) *types.Snapshot {
	return &types.Snapshot{
		Type:      "snapshot",
		Timestamp: time.Now(),
		Departments: map[types.Department]*types.DepartmentData{
			types.DeptSales: {
				Agents: []types.AgentInfo{{AgentID: fmt.Sprintf("agent-%d", id)}},
			},
		},
	}
}

func TestAppendSnapshotHistory_FillsUpToMax(t *testing.T) {
	logger := zerolog.New(&bytes.Buffer{})
	hub := NewHub(logger)

	for i := 0; i < maxSnapshotHistory; i++ {
		hub.appendSnapshotHistory(makeSnapshot(i))
	}

	if len(hub.snapshotHistory) != maxSnapshotHistory {
		t.Errorf("expected %d snapshots, got %d", maxSnapshotHistory, len(hub.snapshotHistory))
	}

	// First snapshot should be agent-0
	first := hub.snapshotHistory[0].Departments[types.DeptSales].Agents[0].AgentID
	if first != "agent-0" {
		t.Errorf("expected first snapshot agent-0, got %s", first)
	}
}

func TestAppendSnapshotHistory_EvictsOldest(t *testing.T) {
	logger := zerolog.New(&bytes.Buffer{})
	hub := NewHub(logger)

	// Fill buffer
	for i := 0; i < maxSnapshotHistory; i++ {
		hub.appendSnapshotHistory(makeSnapshot(i))
	}

	// Add 50 more — should evict the oldest 50
	for i := maxSnapshotHistory; i < maxSnapshotHistory+50; i++ {
		hub.appendSnapshotHistory(makeSnapshot(i))
	}

	if len(hub.snapshotHistory) != maxSnapshotHistory {
		t.Errorf("expected %d snapshots, got %d", maxSnapshotHistory, len(hub.snapshotHistory))
	}

	// Oldest should now be agent-50
	first := hub.snapshotHistory[0].Departments[types.DeptSales].Agents[0].AgentID
	if first != "agent-50" {
		t.Errorf("expected oldest snapshot agent-50, got %s", first)
	}

	// Newest should be agent-349
	last := hub.snapshotHistory[maxSnapshotHistory-1].Departments[types.DeptSales].Agents[0].AgentID
	if last != "agent-349" {
		t.Errorf("expected newest snapshot agent-349, got %s", last)
	}
}

func TestAppendSnapshotHistory_OrderPreserved(t *testing.T) {
	logger := zerolog.New(&bytes.Buffer{})
	hub := NewHub(logger)

	// Overfill by 2x to exercise the copy path many times
	total := maxSnapshotHistory * 2
	for i := 0; i < total; i++ {
		hub.appendSnapshotHistory(makeSnapshot(i))
	}

	// Verify all 300 entries are in order
	for i := 0; i < maxSnapshotHistory; i++ {
		expected := fmt.Sprintf("agent-%d", total-maxSnapshotHistory+i)
		got := hub.snapshotHistory[i].Departments[types.DeptSales].Agents[0].AgentID
		if got != expected {
			t.Errorf("index %d: expected %s, got %s", i, expected, got)
		}
	}
}

func TestAppendSnapshotHistory_BackingArrayStaysFixed(t *testing.T) {
	logger := zerolog.New(&bytes.Buffer{})
	hub := NewHub(logger)

	// Fill to capacity
	for i := 0; i < maxSnapshotHistory; i++ {
		hub.appendSnapshotHistory(makeSnapshot(i))
	}

	capBefore := cap(hub.snapshotHistory)

	// Add 500 more — backing array should never grow
	for i := 0; i < 500; i++ {
		hub.appendSnapshotHistory(makeSnapshot(maxSnapshotHistory + i))
	}

	capAfter := cap(hub.snapshotHistory)
	if capAfter != capBefore {
		t.Errorf("backing array grew: cap before=%d, after=%d", capBefore, capAfter)
	}
}
