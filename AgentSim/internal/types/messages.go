package types

import "time"

// AgentHeartbeat is sent from agent to backend periodically
type AgentHeartbeat struct {
	Type      string     `json:"type"` // "heartbeat"
	AgentID   string     `json:"agentId"`
	State     AgentState `json:"state"`
	Timestamp time.Time  `json:"timestamp"`
	KPIs      AgentKPIs  `json:"kpis"`
}

// AgentStateChangeMsg is sent from agent to backend on state transitions
type AgentStateChangeMsg struct {
	Type          string     `json:"type"` // "state_change"
	AgentID       string     `json:"agentId"`
	PreviousState AgentState `json:"previousState"`
	NewState      AgentState `json:"newState"`
	Timestamp     time.Time  `json:"timestamp"`
	StateDuration float64    `json:"stateDuration"`
	KPIs          AgentKPIs  `json:"kpis"`
	Department    Department `json:"department"`
	Location      Location   `json:"location"`
	Team          string     `json:"team"`
}

// AgentRegister is sent when an agent first connects
type AgentRegister struct {
	Type       string     `json:"type"` // "register"
	AgentID    string     `json:"agentId"`
	Department Department `json:"department"`
	Location   Location   `json:"location"`
	Team       string     `json:"team"`
	State      AgentState `json:"state"`
	KPIs       AgentKPIs  `json:"kpis"`
}

// ServerAck is sent from backend to agent as acknowledgment
type ServerAck struct {
	Type    string `json:"type"` // "ack"
	AgentID string `json:"agentId"`
}
