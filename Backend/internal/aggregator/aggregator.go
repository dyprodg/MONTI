package aggregator

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dennisdiepolder/monti/backend/internal/alerts"
	"github.com/dennisdiepolder/monti/backend/internal/cache"
	"github.com/dennisdiepolder/monti/backend/internal/metrics"
	"github.com/dennisdiepolder/monti/backend/internal/types"
	"github.com/dennisdiepolder/monti/backend/internal/websocket"
	"github.com/rs/zerolog"
)

// VQSnapshotProvider provides VQ snapshots grouped by department
type VQSnapshotProvider interface {
	GetAllSnapshots() map[types.Department][]types.VQSnapshot
}

// Aggregator collects events and creates widgets
type Aggregator struct {
	cache        *cache.EventCache
	stateTracker *cache.AgentStateTracker
	hub          *websocket.Hub
	callQueue    VQSnapshotProvider
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

// SetCallQueue sets the VQ snapshot provider
func (a *Aggregator) SetCallQueue(cq VQSnapshotProvider) {
	a.callQueue = cq
}

// Start begins aggregating events and broadcasting a single snapshot every tick
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

			// Clear recent events
			a.cache.GetAndClear()

			// Get ALL agents (including offline/disconnected)
			allAgents := a.stateTracker.GetAllAgents()

			// Only connected agents for metrics and alerts
			connectedAgents := a.stateTracker.GetConnectedAgents()
			if len(connectedAgents) > 0 {
				m.UpdateAgentStats(connectedAgents)
				alerts.CheckAgentAlerts(connectedAgents)
			}

			// Get VQ snapshots
			var vqSnapshots map[types.Department][]types.VQSnapshot
			if a.callQueue != nil {
				vqSnapshots = a.callQueue.GetAllSnapshots()
			}

			// Build one snapshot with everything
			snapshot := a.buildSnapshot(allAgents, vqSnapshots)

			data, err := json.Marshal(snapshot)
			if err != nil {
				a.logger.Error().Err(err).Msg("failed to marshal snapshot")
				m.RecordAggregationError()
				continue
			}

			a.hub.Broadcast(data)

			// Record aggregation cycle metrics
			m.RecordAggregationCycle(time.Since(cycleStart), 1)

			a.logger.Debug().
				Int("total_agents", len(allAgents)).
				Int("payload_bytes", len(data)).
				Int("clients", a.hub.ClientCount()).
				Msg("snapshot broadcasted")
		}
	}
}

// buildSnapshot creates a single Snapshot containing all agents and all queues
func (a *Aggregator) buildSnapshot(agents []types.AgentInfo, vqSnapshots map[types.Department][]types.VQSnapshot) types.Snapshot {
	// Group agents by department
	deptAgents := make(map[types.Department][]types.AgentInfo)
	for _, agent := range agents {
		deptAgents[agent.Department] = append(deptAgents[agent.Department], agent)
	}

	// Build department data (ensure all 4 departments are always present)
	departments := make(map[types.Department]*types.DepartmentData, 4)
	for _, dept := range []types.Department{types.DeptSales, types.DeptSupport, types.DeptTechnical, types.DeptRetention} {
		departments[dept] = &types.DepartmentData{
			Agents: deptAgents[dept],
			Queues: vqSnapshots[dept],
		}
		// Ensure non-nil slices for clean JSON
		if departments[dept].Agents == nil {
			departments[dept].Agents = []types.AgentInfo{}
		}
		if departments[dept].Queues == nil {
			departments[dept].Queues = []types.VQSnapshot{}
		}
	}

	return types.Snapshot{
		Type:        "snapshot",
		Timestamp:   time.Now(),
		Departments: departments,
	}
}
