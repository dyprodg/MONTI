package aggregator

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dennisdiepolder/monti/backend/internal/cache"
	"github.com/dennisdiepolder/monti/backend/internal/types"
	"github.com/dennisdiepolder/monti/backend/internal/websocket"
	"github.com/rs/zerolog"
)

// Aggregator collects events and creates widgets
type Aggregator struct {
	cache  *cache.EventCache
	hub    *websocket.Hub
	logger zerolog.Logger
}

// NewAggregator creates a new aggregator
func NewAggregator(cache *cache.EventCache, hub *websocket.Hub, logger zerolog.Logger) *Aggregator {
	return &Aggregator{
		cache:  cache,
		hub:    hub,
		logger: logger,
	}
}

// Start begins aggregating events and broadcasting widgets
func (a *Aggregator) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	a.logger.Info().Msg("aggregator started")

	for {
		select {
		case <-ctx.Done():
			a.logger.Info().Msg("aggregator stopped")
			return

		case <-ticker.C:
			events := a.cache.GetAndClear()
			if len(events) == 0 {
				continue
			}

			widgets := a.createWidgets(events)

			for _, widget := range widgets {
				data, err := json.Marshal(widget)
				if err != nil {
					a.logger.Error().Err(err).Msg("failed to marshal widget")
					continue
				}

				a.logger.Debug().
					Str("widget_type", widget.Type).
					Str("department", string(widget.Department)).
					Int("event_count", len(widget.Events)).
					RawJSON("widget_json", data).
					Msg("broadcasting widget")

				a.hub.Broadcast(data)
			}

			a.logger.Debug().
				Int("events_processed", len(events)).
				Int("widgets_created", len(widgets)).
				Int("clients", a.hub.ClientCount()).
				Msg("widgets broadcasted")
		}
	}
}

// createWidgets generates widgets from events
func (a *Aggregator) createWidgets(events []types.AgentEvent) []types.Widget {
	// Group events by department
	deptEvents := make(map[types.Department][]types.AgentEvent)
	for _, event := range events {
		deptEvents[event.Department] = append(deptEvents[event.Department], event)
	}

	widgets := make([]types.Widget, 0, 5)

	// 1. Global overview widget
	widgets = append(widgets, a.createGlobalWidget(events))

	// 2-5. Department widgets
	for dept, deptEvs := range deptEvents {
		widgets = append(widgets, a.createDepartmentWidget(dept, deptEvs))
	}

	return widgets
}

// createGlobalWidget creates a global overview widget
func (a *Aggregator) createGlobalWidget(events []types.AgentEvent) types.Widget {
	summary := types.WidgetSummary{
		TotalEvents:         len(events),
		StateBreakdown:      make(map[types.AgentState]int),
		DepartmentBreakdown: make(map[types.Department]int),
	}

	for _, event := range events {
		summary.StateBreakdown[event.State]++
		summary.DepartmentBreakdown[event.Department]++
	}

	return types.Widget{
		Type:      "global_overview",
		Timestamp: time.Now(),
		Summary:   summary,
		Events:    events,
	}
}

// createDepartmentWidget creates a department-specific widget
func (a *Aggregator) createDepartmentWidget(dept types.Department, events []types.AgentEvent) types.Widget {
	summary := types.WidgetSummary{
		TotalEvents:       len(events),
		StateBreakdown:    make(map[types.AgentState]int),
		LocationBreakdown: make(map[types.Location]int),
	}

	for _, event := range events {
		summary.StateBreakdown[event.State]++
		summary.LocationBreakdown[event.Location]++
	}

	return types.Widget{
		Type:       "department_overview",
		Department: dept,
		Timestamp:  time.Now(),
		Summary:    summary,
		Events:     events,
	}
}
