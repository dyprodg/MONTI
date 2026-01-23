package event

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dennisdiepolder/monti/backend/internal/cache"
	"github.com/dennisdiepolder/monti/backend/internal/metrics"
	"github.com/dennisdiepolder/monti/backend/internal/types"
	"github.com/rs/zerolog"
)

// Receiver handles incoming agent events from AgentSim
type Receiver struct {
	cache          *cache.EventCache
	stateTracker   *cache.AgentStateTracker
	logger         zerolog.Logger
	eventsReceived int64
	lastReceived   time.Time
	mu             sync.RWMutex
}

// NewReceiver creates a new event receiver
func NewReceiver(cache *cache.EventCache, stateTracker *cache.AgentStateTracker, logger zerolog.Logger) *Receiver {
	return &Receiver{
		cache:        cache,
		stateTracker: stateTracker,
		logger:       logger,
	}
}

// HandleEvent receives and caches individual agent events
func (r *Receiver) HandleEvent(w http.ResponseWriter, req *http.Request) {
	m := metrics.Get()

	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var event types.AgentEvent
	if err := json.NewDecoder(req.Body).Decode(&event); err != nil {
		r.logger.Error().Err(err).Msg("failed to decode event")
		m.RecordEventError()
		http.Error(w, "invalid event", http.StatusBadRequest)
		return
	}

	// Record metric
	m.RecordEventReceived()

	// Add event to cache
	r.cache.Add(event)

	// Update agent state tracker
	r.stateTracker.Update(event)

	// Record processed
	m.RecordEventProcessed()

	// Update stats
	atomic.AddInt64(&r.eventsReceived, 1)
	r.mu.Lock()
	r.lastReceived = time.Now()
	r.mu.Unlock()

	// Log periodically
	count := atomic.LoadInt64(&r.eventsReceived)
	if count%1000 == 0 {
		r.logger.Info().
			Int64("total_received", count).
			Int("cache_size", r.cache.Size()).
			Msg("events received")
	}

	w.WriteHeader(http.StatusOK)
}

// GetStats returns receiver statistics
func (r *Receiver) GetStats(w http.ResponseWriter, req *http.Request) {
	r.mu.RLock()
	lastReceived := r.lastReceived
	r.mu.RUnlock()

	stats := map[string]interface{}{
		"events_received": atomic.LoadInt64(&r.eventsReceived),
		"last_received":   lastReceived,
		"cache_size":      r.cache.Size(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
