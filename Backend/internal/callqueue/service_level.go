package callqueue

import "github.com/dennisdiepolder/monti/backend/internal/types"

// SLTracker tracks service level metrics for a VQ
type SLTracker struct {
	Target        int // target percentage (e.g., 80)
	ThresholdSecs int // threshold in seconds (e.g., 20)
	AnsweredInSL  int // calls answered within threshold
	TotalAnswered int // total calls answered
}

// NewSLTracker creates a new SL tracker with the given target
func NewSLTracker(target, thresholdSecs int) *SLTracker {
	return &SLTracker{
		Target:        target,
		ThresholdSecs: thresholdSecs,
	}
}

// RecordAnswer records a call being answered
func (s *SLTracker) RecordAnswer(waitTimeSecs float64) {
	s.TotalAnswered++
	if waitTimeSecs <= float64(s.ThresholdSecs) {
		s.AnsweredInSL++
	}
}

// CurrentSL returns the current service level percentage
func (s *SLTracker) CurrentSL() float64 {
	if s.TotalAnswered == 0 {
		return 100.0 // No calls answered yet, SL is 100%
	}
	return float64(s.AnsweredInSL) / float64(s.TotalAnswered) * 100.0
}

// Snapshot returns a ServiceLevel snapshot
func (s *SLTracker) Snapshot() types.ServiceLevel {
	return types.ServiceLevel{
		Target:        s.Target,
		ThresholdSecs: s.ThresholdSecs,
		AnsweredInSL:  s.AnsweredInSL,
		TotalAnswered: s.TotalAnswered,
		CurrentSL:     s.CurrentSL(),
	}
}
