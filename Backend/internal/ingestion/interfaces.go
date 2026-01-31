package ingestion

import (
	"context"

	"github.com/dennisdiepolder/monti/backend/internal/types"
)

// EventProcessor processes events from any source (AgentSim, Genesys, etc.)
type EventProcessor interface {
	ProcessRegister(reg *types.AgentRegister)
	ProcessHeartbeat(hb *types.AgentHeartbeat)
	ProcessStateChange(sc *types.AgentStateChange)
	ProcessCallComplete(cc *types.CallComplete)
}

// EventSource represents a source of agent events (AgentHub, Genesys adapter, etc.)
type EventSource interface {
	// Start begins receiving events and forwarding them to the processor
	Start(ctx context.Context, processor EventProcessor) error

	// SendToAgent sends a message to a specific agent by ID
	SendToAgent(agentID string, message []byte) bool

	// AgentCount returns the number of connected agents
	AgentCount() int
}
