package types

import "time"

// CallAssign is sent from backend to agent when a call is routed
type CallAssign struct {
	Type     string    `json:"type"` // "call_assign"
	AgentID  string    `json:"agentId"`
	CallID   string    `json:"callId"`
	VQ       VQName    `json:"vq"`
	Timestamp time.Time `json:"timestamp"`
}

// CallComplete is sent from agent to backend when a call is finished
type CallComplete struct {
	Type      string    `json:"type"` // "call_complete"
	AgentID   string    `json:"agentId"`
	CallID    string    `json:"callId"`
	TalkTime  float64   `json:"talkTime"`  // seconds
	HoldTime  float64   `json:"holdTime"`  // seconds
	Timestamp time.Time `json:"timestamp"`
}

// ForceEndCall is sent from backend to agent to end an active call
type ForceEndCall struct {
	Type    string `json:"type"`    // "force_end_call"
	CallID  string `json:"callId"`
	AgentID string `json:"agentId"`
}

// ForceDisconnect is sent from backend to agent to force logout
type ForceDisconnect struct {
	Type    string `json:"type"`    // "force_disconnect"
	AgentID string `json:"agentId"`
}

// IncomingCall represents a new call entering the system
type IncomingCall struct {
	Type       string     `json:"type"` // "incoming_call"
	CallID     string     `json:"callId"`
	VQ         VQName     `json:"vq"`
	Department Department `json:"department"`
	Timestamp  time.Time  `json:"timestamp"`
}
