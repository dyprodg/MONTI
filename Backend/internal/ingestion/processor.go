package ingestion

import (
	"github.com/dennisdiepolder/monti/backend/internal/cache"
	"github.com/dennisdiepolder/monti/backend/internal/metrics"
	"github.com/dennisdiepolder/monti/backend/internal/types"
	"github.com/rs/zerolog"
)

// CallCompleter handles call completion events
type CallCompleter interface {
	CompleteCall(callID string, talkTime, holdTime float64) *types.Call
}

// DefaultProcessor implements EventProcessor by delegating to AgentStateTracker
type DefaultProcessor struct {
	tracker       *cache.AgentStateTracker
	callCompleter CallCompleter
	logger        zerolog.Logger
}

// NewDefaultProcessor creates a new DefaultProcessor
func NewDefaultProcessor(tracker *cache.AgentStateTracker, logger zerolog.Logger) *DefaultProcessor {
	return &DefaultProcessor{
		tracker: tracker,
		logger:  logger,
	}
}

// SetCallCompleter sets the call completer (to avoid circular init)
func (p *DefaultProcessor) SetCallCompleter(cc CallCompleter) {
	p.callCompleter = cc
}

func (p *DefaultProcessor) ProcessRegister(reg *types.AgentRegister) {
	p.tracker.RegisterAgent(reg)
	metrics.Get().RecordAgentRegister()

	p.logger.Debug().
		Str("agent_id", reg.AgentID).
		Str("state", string(reg.State)).
		Msg("agent registered via processor")
}

func (p *DefaultProcessor) ProcessHeartbeat(hb *types.AgentHeartbeat) {
	p.tracker.UpdateFromHeartbeat(hb)
	metrics.Get().RecordAgentHeartbeat()
}

func (p *DefaultProcessor) ProcessStateChange(sc *types.AgentStateChange) {
	p.tracker.UpdateFromStateChange(sc)
	metrics.Get().RecordAgentStateChange()

	p.logger.Debug().
		Str("agent_id", sc.AgentID).
		Str("prev_state", string(sc.PreviousState)).
		Str("new_state", string(sc.NewState)).
		Float64("duration", sc.StateDuration).
		Msg("agent state change via processor")
}

func (p *DefaultProcessor) ProcessCallComplete(cc *types.CallComplete) {
	if p.callCompleter != nil {
		p.callCompleter.CompleteCall(cc.CallID, cc.TalkTime, cc.HoldTime)
	}

	p.logger.Debug().
		Str("agent_id", cc.AgentID).
		Str("call_id", cc.CallID).
		Float64("talk_time", cc.TalkTime).
		Msg("call complete via processor")
}
