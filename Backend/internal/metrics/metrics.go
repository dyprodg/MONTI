package metrics

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/dennisdiepolder/monti/backend/internal/types"
)

// Metrics holds all application metrics
type Metrics struct {
	mu sync.RWMutex

	// Event metrics
	EventsReceivedTotal   int64
	EventsProcessedTotal  int64
	EventProcessingErrors int64

	// WebSocket metrics
	WebSocketConnectionsTotal    int64
	WebSocketDisconnectionsTotal int64
	WebSocketMessagesTotal       int64
	WebSocketErrorsTotal         int64
	activeConnections            int64

	// Aggregation metrics
	AggregationCyclesTotal  int64
	WidgetsBroadcastTotal   int64
	AggregationErrorsTotal  int64
	lastAggregationDuration time.Duration

	// Agent metrics
	agentsByState      map[types.AgentState]int
	agentsByDepartment map[types.Department]int
	agentsByLocation   map[types.Location]int
	totalAgents        int

	// HTTP metrics
	httpRequestsTotal    map[string]map[int]int64 // endpoint -> status -> count
	httpRequestDurations map[string][]float64     // endpoint -> durations

	// Timing
	startTime time.Time
}

// Global metrics instance
var instance *Metrics
var once sync.Once

// Get returns the singleton metrics instance
func Get() *Metrics {
	once.Do(func() {
		instance = &Metrics{
			agentsByState:        make(map[types.AgentState]int),
			agentsByDepartment:   make(map[types.Department]int),
			agentsByLocation:     make(map[types.Location]int),
			httpRequestsTotal:    make(map[string]map[int]int64),
			httpRequestDurations: make(map[string][]float64),
			startTime:            time.Now(),
		}
	})
	return instance
}

// RecordEventReceived increments the events received counter
func (m *Metrics) RecordEventReceived() {
	m.mu.Lock()
	m.EventsReceivedTotal++
	m.mu.Unlock()
}

// RecordEventProcessed increments the events processed counter
func (m *Metrics) RecordEventProcessed() {
	m.mu.Lock()
	m.EventsProcessedTotal++
	m.mu.Unlock()
}

// RecordEventError increments the event processing error counter
func (m *Metrics) RecordEventError() {
	m.mu.Lock()
	m.EventProcessingErrors++
	m.mu.Unlock()
}

// RecordWebSocketConnect increments connection counters
func (m *Metrics) RecordWebSocketConnect() {
	m.mu.Lock()
	m.WebSocketConnectionsTotal++
	m.activeConnections++
	m.mu.Unlock()
}

// RecordWebSocketDisconnect increments disconnection counter
func (m *Metrics) RecordWebSocketDisconnect() {
	m.mu.Lock()
	m.WebSocketDisconnectionsTotal++
	m.activeConnections--
	m.mu.Unlock()
}

// RecordWebSocketMessage increments message counter
func (m *Metrics) RecordWebSocketMessage() {
	m.mu.Lock()
	m.WebSocketMessagesTotal++
	m.mu.Unlock()
}

// RecordWebSocketError increments WebSocket error counter
func (m *Metrics) RecordWebSocketError() {
	m.mu.Lock()
	m.WebSocketErrorsTotal++
	m.mu.Unlock()
}

// RecordAggregationCycle records an aggregation cycle
func (m *Metrics) RecordAggregationCycle(duration time.Duration, widgetCount int) {
	m.mu.Lock()
	m.AggregationCyclesTotal++
	m.WidgetsBroadcastTotal += int64(widgetCount)
	m.lastAggregationDuration = duration
	m.mu.Unlock()
}

// RecordAggregationError increments aggregation error counter
func (m *Metrics) RecordAggregationError() {
	m.mu.Lock()
	m.AggregationErrorsTotal++
	m.mu.Unlock()
}

// UpdateAgentStats updates agent distribution metrics
func (m *Metrics) UpdateAgentStats(agents []types.AgentInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Reset counts
	m.agentsByState = make(map[types.AgentState]int)
	m.agentsByDepartment = make(map[types.Department]int)
	m.agentsByLocation = make(map[types.Location]int)
	m.totalAgents = len(agents)

	for _, agent := range agents {
		m.agentsByState[agent.State]++
		m.agentsByDepartment[agent.Department]++
		m.agentsByLocation[agent.Location]++
	}
}

// RecordHTTPRequest records an HTTP request
func (m *Metrics) RecordHTTPRequest(endpoint string, statusCode int, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.httpRequestsTotal[endpoint] == nil {
		m.httpRequestsTotal[endpoint] = make(map[int]int64)
	}
	m.httpRequestsTotal[endpoint][statusCode]++

	// Keep last 100 durations for percentile calculation
	if len(m.httpRequestDurations[endpoint]) >= 100 {
		m.httpRequestDurations[endpoint] = m.httpRequestDurations[endpoint][1:]
	}
	m.httpRequestDurations[endpoint] = append(m.httpRequestDurations[endpoint], duration.Seconds())
}

// GetActiveConnections returns current WebSocket connections
func (m *Metrics) GetActiveConnections() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeConnections
}

// Handler returns an HTTP handler for the /metrics endpoint
func (m *Metrics) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.mu.RLock()
		defer m.mu.RUnlock()

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		// Helper to write metric
		write := func(name string, value interface{}, labels ...string) {
			labelStr := ""
			if len(labels) > 0 {
				labelStr = "{"
				for i := 0; i < len(labels); i += 2 {
					if i > 0 {
						labelStr += ","
					}
					labelStr += labels[i] + "=\"" + labels[i+1] + "\""
				}
				labelStr += "}"
			}

			switch v := value.(type) {
			case int:
				w.Write([]byte(name + labelStr + " " + strconv.Itoa(v) + "\n"))
			case int64:
				w.Write([]byte(name + labelStr + " " + strconv.FormatInt(v, 10) + "\n"))
			case float64:
				w.Write([]byte(name + labelStr + " " + strconv.FormatFloat(v, 'f', 6, 64) + "\n"))
			}
		}

		// System metrics
		write("monti_uptime_seconds", time.Since(m.startTime).Seconds())

		// Event metrics
		write("monti_events_received_total", m.EventsReceivedTotal)
		write("monti_events_processed_total", m.EventsProcessedTotal)
		write("monti_event_processing_errors_total", m.EventProcessingErrors)

		// Calculate events per second
		uptimeSeconds := time.Since(m.startTime).Seconds()
		if uptimeSeconds > 0 {
			write("monti_events_per_second", float64(m.EventsReceivedTotal)/uptimeSeconds)
		}

		// WebSocket metrics
		write("monti_websocket_connections_total", m.WebSocketConnectionsTotal)
		write("monti_websocket_disconnections_total", m.WebSocketDisconnectionsTotal)
		write("monti_websocket_active_connections", m.activeConnections)
		write("monti_websocket_messages_total", m.WebSocketMessagesTotal)
		write("monti_websocket_errors_total", m.WebSocketErrorsTotal)

		// Aggregation metrics
		write("monti_aggregation_cycles_total", m.AggregationCyclesTotal)
		write("monti_widgets_broadcast_total", m.WidgetsBroadcastTotal)
		write("monti_aggregation_errors_total", m.AggregationErrorsTotal)
		write("monti_aggregation_duration_seconds", m.lastAggregationDuration.Seconds())

		// Agent metrics
		write("monti_agents_total", m.totalAgents)

		// Agents by state
		for state, count := range m.agentsByState {
			write("monti_agents_by_state", count, "state", string(state))
		}

		// Agents by department
		for dept, count := range m.agentsByDepartment {
			write("monti_agents_by_department", count, "department", string(dept))
		}

		// Agents by location
		for loc, count := range m.agentsByLocation {
			write("monti_agents_by_location", count, "location", string(loc))
		}

		// HTTP metrics
		for endpoint, statusCodes := range m.httpRequestsTotal {
			for status, count := range statusCodes {
				write("monti_http_requests_total", count, "endpoint", endpoint, "status", strconv.Itoa(status))
			}
		}
	}
}
