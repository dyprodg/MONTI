package callqueue

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dennisdiepolder/monti/backend/internal/types"
	"github.com/rs/zerolog"
)

// AgentSender sends messages to connected agents via WebSocket
type AgentSender interface {
	SendToAgent(agentID string, message []byte) bool
}

// RoutingLoop periodically matches waiting calls to available agents
type RoutingLoop struct {
	mgr    *CallQueueManager
	sender AgentSender
	logger zerolog.Logger
}

// NewRoutingLoop creates a new RoutingLoop
func NewRoutingLoop(mgr *CallQueueManager, sender AgentSender, logger zerolog.Logger) *RoutingLoop {
	return &RoutingLoop{
		mgr:    mgr,
		sender: sender,
		logger: logger,
	}
}

// Start begins the routing loop, ticking every 1 second until the context is cancelled
func (rl *RoutingLoop) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	rl.logger.Info().Msg("routing loop started")

	for {
		select {
		case <-ctx.Done():
			rl.logger.Info().Msg("routing loop stopped")
			return
		case <-ticker.C:
			rl.tick()
		}
	}
}

// tick performs a single routing pass
func (rl *RoutingLoop) tick() {
	matches := rl.mgr.TickRouting()

	for _, match := range matches {
		msg := types.CallAssign{
			Type:      "call_assign",
			AgentID:   match.AgentID,
			CallID:    match.Call.CallID,
			VQ:        match.Call.VQ,
			Timestamp: time.Now(),
		}

		data, err := json.Marshal(msg)
		if err != nil {
			rl.logger.Error().Err(err).
				Str("call_id", match.Call.CallID).
				Str("agent_id", match.AgentID).
				Msg("failed to marshal call_assign message")
			continue
		}

		if !rl.sender.SendToAgent(match.AgentID, data) {
			rl.logger.Warn().
				Str("call_id", match.Call.CallID).
				Str("agent_id", match.AgentID).
				Msg("failed to send call_assign to agent")
		}
	}
}
