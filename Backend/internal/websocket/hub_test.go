package websocket

import (
	"bytes"
	"testing"
	"time"

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
