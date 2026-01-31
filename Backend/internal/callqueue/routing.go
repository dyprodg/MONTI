package callqueue

import (
	"github.com/dennisdiepolder/monti/backend/internal/types"
)

// RoutingStrategy selects the best agent to handle a call
type RoutingStrategy interface {
	SelectAgent(available []types.AgentInfo) *types.AgentInfo
}

// LongestIdleFirst selects the agent who has been available the longest
type LongestIdleFirst struct{}

// SelectAgent picks the available agent with the oldest StateStart time
func (l *LongestIdleFirst) SelectAgent(available []types.AgentInfo) *types.AgentInfo {
	if len(available) == 0 {
		return nil
	}

	oldest := &available[0]
	for i := 1; i < len(available); i++ {
		if available[i].StateStart.Before(oldest.StateStart) {
			oldest = &available[i]
		}
	}
	return oldest
}
