package types

import "time"

// AgentState represents the current state of an agent
type AgentState string

const (
	// Basic states
	StateAvailable AgentState = "available"
	StateBusy      AgentState = "busy"
	StateOnCall    AgentState = "on_call"
	StateBreak     AgentState = "break"
	StateOffline   AgentState = "offline"

	// Extended states
	StateAfterCallWork AgentState = "after_call_work"
	StateTraining      AgentState = "training"
	StateMeeting       AgentState = "meeting"
	StateLunch         AgentState = "lunch"

	// Call-specific states
	StateOnHold       AgentState = "on_hold"
	StateTransferring AgentState = "transferring"
	StateConference   AgentState = "conference"
)

// Department represents different call center departments
type Department string

const (
	DeptSales     Department = "sales"
	DeptSupport   Department = "support"
	DeptTechnical Department = "technical"
	DeptRetention Department = "retention"
)

// Location represents physical locations
type Location string

const (
	LocationBerlin    Location = "berlin"
	LocationMunich    Location = "munich"
	LocationHamburg   Location = "hamburg"
	LocationFrankfurt Location = "frankfurt"
	LocationRemote    Location = "remote"
)

// AgentEvent represents an individual agent state event from AgentSim
type AgentEvent struct {
	AgentID       string     `json:"agentId"`
	State         AgentState `json:"state"`
	Department    Department `json:"department"`
	Location      Location   `json:"location"`
	Team          string     `json:"team"`
	Timestamp     time.Time  `json:"timestamp"`
	StateDuration float64    `json:"stateDuration"` // seconds in current state
}

// Widget represents aggregated data for a single widget
type Widget struct {
	Type       string        `json:"type"` // "global_overview" or "department_overview"
	Department Department    `json:"department,omitempty"`
	Timestamp  time.Time     `json:"timestamp"`
	Summary    WidgetSummary `json:"summary"`
	Events     []AgentEvent  `json:"events"` // Recent events for this widget
}

// WidgetSummary contains aggregated counts
type WidgetSummary struct {
	TotalEvents         int                    `json:"totalEvents"`
	StateBreakdown      map[AgentState]int     `json:"stateBreakdown"`
	DepartmentBreakdown map[Department]int     `json:"departmentBreakdown,omitempty"`
	LocationBreakdown   map[Location]int       `json:"locationBreakdown,omitempty"`
}
