package cache

import (
	"sync"

	"github.com/dennisdiepolder/monti/backend/internal/types"
)

// EventCache stores agent events in memory
type EventCache struct {
	events []types.AgentEvent
	mu     sync.RWMutex
}

// NewEventCache creates a new event cache
func NewEventCache() *EventCache {
	return &EventCache{
		events: make([]types.AgentEvent, 0, 2000),
	}
}

// Add appends an event to the cache
func (c *EventCache) Add(event types.AgentEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, event)
}

// GetAndClear returns all events and clears the cache
func (c *EventCache) GetAndClear() []types.AgentEvent {
	c.mu.Lock()
	defer c.mu.Unlock()

	events := c.events
	c.events = make([]types.AgentEvent, 0, 2000) // pre-allocate for next second
	return events
}

// Size returns the current number of cached events
func (c *EventCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.events)
}
