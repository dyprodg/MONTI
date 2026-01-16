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

// Agent represents a call center agent
type Agent struct {
	ID         string     `json:"id"`
	Department Department `json:"department"`
	Location   Location   `json:"location"`
	Team       string     `json:"team"`
	State      AgentState `json:"state"`
	StateStart time.Time  `json:"stateStart"`
	LastUpdate time.Time  `json:"lastUpdate"`
	LoginTime  time.Time  `json:"loginTime"`
	KPIs       AgentKPIs  `json:"kpis"`
}

// AgentEvent represents an individual agent state event sent to Backend
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

// SimulationConfig holds configuration for the simulation
type SimulationConfig struct {
	TotalAgents int `json:"totalAgents"`
	ActiveAgents      int           `json:"activeAgents"`
}

// SimulationStatus represents current simulation state
type SimulationStatus struct {
	Running      bool       `json:"running"`
	TotalAgents  int        `json:"totalAgents"`
	ActiveAgents int        `json:"activeAgents"`
	EventsSent   int64      `json:"eventsSent"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
}
