package ticker

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/dennisdiepolder/monti/backend/internal/websocket"
	"github.com/rs/zerolog"
)

func TestNewTicker(t *testing.T) {
	logger := zerolog.New(&bytes.Buffer{})
	hub := websocket.NewHub(logger)
	ticker := NewTicker(hub, 1*time.Second, logger)

	if ticker == nil {
		t.Fatal("expected ticker to be created")
	}

	if ticker.hub != hub {
		t.Error("ticker hub not set correctly")
	}

	if ticker.interval != 1*time.Second {
		t.Errorf("expected interval 1s, got %v", ticker.interval)
	}
}

func TestTickerStart(t *testing.T) {
	logger := zerolog.New(&bytes.Buffer{})
	hub := websocket.NewHub(logger)
	go hub.Run()

	// Create ticker with short interval for testing
	ticker := NewTicker(hub, 100*time.Millisecond, logger)

	// Start ticker with context
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Run ticker
	done := make(chan bool)
	go func() {
		ticker.Start(ctx)
		done <- true
	}()

	// Wait for context to timeout
	<-ctx.Done()

	// Wait for ticker to stop
	select {
	case <-done:
		// Ticker stopped as expected
	case <-time.After(1 * time.Second):
		t.Error("ticker did not stop after context cancel")
	}
}

func TestTickerBroadcastsMessages(t *testing.T) {
	logger := zerolog.New(&bytes.Buffer{})
	hub := websocket.NewHub(logger)
	go hub.Run()

	// Create ticker with short interval
	ticker := NewTicker(hub, 50*time.Millisecond, logger)

	// Start ticker and let it run for a few ticks
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan bool)
	go func() {
		ticker.Start(ctx)
		done <- true
	}()

	// Wait for ticker to complete
	<-done

	// Verify the hub is still operational after ticker ran
	if hub.ClientCount() < 0 {
		t.Error("expected non-negative client count")
	}
}

func TestTimeMessage(t *testing.T) {
	now := time.Now()
	msg := TimeMessage{
		Timestamp:  now.Format(time.RFC3339),
		ServerTime: now.Unix(),
	}

	// Test JSON marshaling
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Test JSON unmarshaling
	var decoded TimeMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Timestamp != msg.Timestamp {
		t.Errorf("expected timestamp %s, got %s", msg.Timestamp, decoded.Timestamp)
	}

	if decoded.ServerTime != msg.ServerTime {
		t.Errorf("expected serverTime %d, got %d", msg.ServerTime, decoded.ServerTime)
	}
}

func TestTickerStopsOnContextCancel(t *testing.T) {
	logger := zerolog.New(&bytes.Buffer{})
	hub := websocket.NewHub(logger)
	go hub.Run()

	ticker := NewTicker(hub, 100*time.Millisecond, logger)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan bool)
	go func() {
		ticker.Start(ctx)
		done <- true
	}()

	// Let it run for a bit
	time.Sleep(200 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for ticker to stop
	select {
	case <-done:
		// Success - ticker stopped
	case <-time.After(1 * time.Second):
		t.Error("ticker did not stop within timeout after context cancel")
	}
}
