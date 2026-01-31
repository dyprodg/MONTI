package types

import "time"

// CallStatus represents the lifecycle state of a call
type CallStatus string

const (
	CallStatusWaiting   CallStatus = "waiting"   // In queue, not yet assigned
	CallStatusActive    CallStatus = "active"    // Currently being handled by an agent
	CallStatusCompleted CallStatus = "completed" // Successfully completed
	CallStatusAbandoned CallStatus = "abandoned" // Caller hung up while waiting
)

// Call represents an active or queued call in the system
type Call struct {
	CallID      string     `json:"callId"`
	VQ          VQName     `json:"vq"`
	Department  Department `json:"department"`
	Status      CallStatus `json:"status"`
	EnqueueTime time.Time  `json:"enqueueTime"`
	AssignTime  *time.Time `json:"assignTime,omitempty"`
	CompleteTime *time.Time `json:"completeTime,omitempty"`
	AgentID     string     `json:"agentId,omitempty"`
	TalkTime    float64    `json:"talkTime,omitempty"`    // seconds
	HoldTime    float64    `json:"holdTime,omitempty"`    // seconds
	WrapTime    float64    `json:"wrapTime,omitempty"`    // seconds
	WaitTime    float64    `json:"waitTime,omitempty"`    // seconds in queue
}

// ServiceLevel tracks SL metrics for a VQ
type ServiceLevel struct {
	Target          int     `json:"target"`          // target percentage (e.g., 80)
	ThresholdSecs   int     `json:"thresholdSecs"`   // threshold in seconds (e.g., 20)
	AnsweredInSL    int     `json:"answeredInSL"`    // calls answered within threshold
	TotalAnswered   int     `json:"totalAnswered"`   // total calls answered
	CurrentSL       float64 `json:"currentSL"`       // calculated SL percentage
}

// VQSnapshot represents the current state of a virtual queue
type VQSnapshot struct {
	VQ              VQName     `json:"vq"`
	Department      Department `json:"department"`
	WaitingCount    int        `json:"waitingCount"`
	ActiveCount     int        `json:"activeCount"`
	CompletedCount  int        `json:"completedCount"`
	AbandonedCount  int        `json:"abandonedCount"`
	LongestWaitSecs float64    `json:"longestWaitSecs"`
	AvailableAgents int        `json:"availableAgents"`
	ServiceLevel    ServiceLevel `json:"serviceLevel"`
}

// VQWidget contains all VQ snapshots for a department
type VQWidget struct {
	Type       string       `json:"type"` // "vq_overview"
	Department Department   `json:"department"`
	Timestamp  time.Time    `json:"timestamp"`
	Queues     []VQSnapshot `json:"queues"`
}
