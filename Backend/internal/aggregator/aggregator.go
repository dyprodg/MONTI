package aggregator

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dennisdiepolder/monti/backend/internal/cache"
	"github.com/dennisdiepolder/monti/backend/internal/metrics"
	"github.com/dennisdiepolder/monti/backend/internal/types"
	"github.com/dennisdiepolder/monti/backend/internal/websocket"
	"github.com/rs/zerolog"
)

// Aggregator collects events and creates widgets
type Aggregator struct {
	cache        *cache.EventCache
	stateTracker *cache.AgentStateTracker
	hub          *websocket.Hub
	logger       zerolog.Logger
}

// NewAggregator creates a new aggregator
func NewAggregator(cache *cache.EventCache, stateTracker *cache.AgentStateTracker, hub *websocket.Hub, logger zerolog.Logger) *Aggregator {
	return &Aggregator{
		cache:        cache,
		stateTracker: stateTracker,
		hub:          hub,
		logger:       logger,
	}
}

// Start begins aggregating events and broadcasting widgets
func (a *Aggregator) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	m := metrics.Get()
	a.logger.Info().Msg("aggregator started")

	for {
		select {
		case <-ctx.Done():
			a.logger.Info().Msg("aggregator stopped")
			return

		case <-ticker.C:
			cycleStart := time.Now()

			// Clear recent events (we don't need them anymore)
			events := a.cache.GetAndClear()

			// Get only connected agents (excludes disconnected/stale)
			allAgents := a.stateTracker.GetConnectedAgents()
			if len(allAgents) == 0 {
				continue
			}

			// Update agent metrics
			m.UpdateAgentStats(allAgents)

			// Create widgets from all agent states
			widgets := a.createWidgetsFromStates(allAgents)
			widgetCount := 0

			for _, widget := range widgets {
				data, err := json.Marshal(widget)
				if err != nil {
					a.logger.Error().Err(err).Msg("failed to marshal widget")
					m.RecordAggregationError()
					continue
				}

				a.logger.Debug().
					Str("widget_type", widget.Type).
					Str("department", string(widget.Department)).
					Int("agent_count", len(widget.Agents)).
					Int("recent_events", len(events)).
					Msg("broadcasting widget")

				a.hub.Broadcast(data)
				widgetCount++
			}

			// Record aggregation cycle metrics
			m.RecordAggregationCycle(time.Since(cycleStart), widgetCount)

			a.logger.Debug().
				Int("events_processed", len(events)).
				Int("total_agents", len(allAgents)).
				Int("widgets_created", len(widgets)).
				Int("clients", a.hub.ClientCount()).
				Msg("widgets broadcasted")
		}
	}
}

// createWidgetsFromStates generates widgets from current agent states
func (a *Aggregator) createWidgetsFromStates(agents []types.AgentInfo) []types.Widget {
	// Group agents by department
	deptAgents := make(map[types.Department][]types.AgentInfo)
	for _, agent := range agents {
		deptAgents[agent.Department] = append(deptAgents[agent.Department], agent)
	}

	widgets := make([]types.Widget, 0, 5)

	// 1. Global overview widget
	widgets = append(widgets, a.createGlobalWidgetFromStates(agents))

	// 2-5. Department widgets
	for dept, deptAgs := range deptAgents {
		widgets = append(widgets, a.createDepartmentWidgetFromStates(dept, deptAgs))
	}

	return widgets
}

// createGlobalWidgetFromStates creates a global overview widget from agent states
func (a *Aggregator) createGlobalWidgetFromStates(agents []types.AgentInfo) types.Widget {
	summary := types.WidgetSummary{
		TotalAgents:         len(agents),
		StateBreakdown:      make(map[types.AgentState]int),
		DepartmentBreakdown: make(map[types.Department]int),
	}

	for _, agent := range agents {
		summary.StateBreakdown[agent.State]++
		summary.DepartmentBreakdown[agent.Department]++
	}

	return types.Widget{
		Type:      "global_overview",
		Timestamp: time.Now(),
		Summary:   summary,
		Agents:    agents,
	}
}

// createDepartmentWidgetFromStates creates a department-specific widget from agent states
func (a *Aggregator) createDepartmentWidgetFromStates(dept types.Department, agents []types.AgentInfo) types.Widget {
	summary := types.WidgetSummary{
		TotalAgents:       len(agents),
		StateBreakdown:    make(map[types.AgentState]int),
		LocationBreakdown: make(map[types.Location]int),
	}

	for _, agent := range agents {
		summary.StateBreakdown[agent.State]++
		summary.LocationBreakdown[agent.Location]++
	}

	return types.Widget{
		Type:       "department_overview",
		Department: dept,
		Timestamp:  time.Now(),
		Summary:    summary,
		Agents:     agents,
	}
}
