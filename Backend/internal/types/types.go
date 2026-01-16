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

// AgentKPIs contains performance metrics for an agent
type AgentKPIs struct {
	TotalCalls           int     `json:"totalCalls"`
	AvgCallDuration      float64 `json:"avgCallDuration"`      // seconds
	AcwTime              float64 `json:"acwTime"`              // seconds
	AcwCount             int     `json:"acwCount"`
	HoldCount            int     `json:"holdCount"`
	HoldTime             float64 `json:"holdTime"`             // seconds
	TransferCount        int     `json:"transferCount"`
	ConferenceCount      int     `json:"conferenceCount"`
	BreakTime            float64 `json:"breakTime"`            // seconds
	LoginTime            float64 `json:"loginTime"`            // seconds since login
	Occupancy            float64 `json:"occupancy"`            // 0-100%
	Adherence            float64 `json:"adherence"`            // 0-100%
	AvgHandleTime        float64 `json:"avgHandleTime"`        // seconds
	FirstCallResolution  float64 `json:"firstCallResolution"`  // 0-100%
	CustomerSatisfaction float64 `json:"customerSatisfaction"` // 1-5
}

// AgentEvent represents an individual agent state event from AgentSim
type AgentEvent struct {
	AgentID       string     `json:"agentId"`
	State         AgentState `json:"state"`
	Department    Department `json:"department"`
	Location      Location   `json:"location"`
	Team          string     `json:"team"`
	Timestamp     time.Time  `json:"timestamp"`
	StateDuration float64    `json:"stateDuration"` // seconds in current state
	KPIs          AgentKPIs  `json:"kpis"`
}

// AgentInfo represents the current state of an agent
type AgentInfo struct {
	AgentID    string     `json:"agentId"`
	State      AgentState `json:"state"`
	Department Department `json:"department"`
	Location   Location   `json:"location"`
	Team       string     `json:"team"`
	StateStart time.Time  `json:"stateStart"` // when current state started
	LastUpdate time.Time  `json:"lastUpdate"` // last event received
	KPIs       AgentKPIs  `json:"kpis"`
}

// Widget represents aggregated data for a single widget
type Widget struct {
	Type       string        `json:"type"` // "global_overview" or "department_overview"
	Department Department    `json:"department,omitempty"`
	Timestamp  time.Time     `json:"timestamp"`
	Summary    WidgetSummary `json:"summary"`
	Events     []AgentEvent  `json:"events,omitempty"` // Recent events for this widget (deprecated, use Agents instead)
	Agents     []AgentInfo   `json:"agents,omitempty"` // All agents in this widget
}

// WidgetSummary contains aggregated counts
type WidgetSummary struct {
	TotalAgents         int                    `json:"totalAgents"` // Total number of agents
	TotalEvents         int                    `json:"totalEvents,omitempty"` // Total events (deprecated)
	StateBreakdown      map[AgentState]int     `json:"stateBreakdown"`
	DepartmentBreakdown map[Department]int     `json:"departmentBreakdown,omitempty"`
	LocationBreakdown   map[Location]int       `json:"locationBreakdown,omitempty"`
}
