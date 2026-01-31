package types

import "time"

// VQName represents a virtual queue identifier
type VQName string

// Sales VQs
const (
	VQSalesInbound    VQName = "sales_inbound"
	VQSalesOutbound   VQName = "sales_outbound"
	VQSalesCallback   VQName = "sales_callback"
	VQSalesChat       VQName = "sales_chat"
)

// Support VQs
const (
	VQSupportGeneral  VQName = "support_general"
	VQSupportBilling  VQName = "support_billing"
	VQSupportCallback VQName = "support_callback"
	VQSupportChat     VQName = "support_chat"
)

// Technical VQs
const (
	VQTechL1          VQName = "tech_l1"
	VQTechL2          VQName = "tech_l2"
	VQTechCallback    VQName = "tech_callback"
	VQTechChat        VQName = "tech_chat"
)

// Retention VQs
const (
	VQRetentionSave   VQName = "retention_save"
	VQRetentionCancel VQName = "retention_cancel"
	VQRetentionCallback VQName = "retention_callback"
	VQRetentionChat   VQName = "retention_chat"
)

// CallAssignMsg is received from backend when a call is routed to this agent
type CallAssignMsg struct {
	Type      string    `json:"type"` // "call_assign"
	AgentID   string    `json:"agentId"`
	CallID    string    `json:"callId"`
	VQ        VQName    `json:"vq"`
	Timestamp time.Time `json:"timestamp"`
}

// CallCompleteMsg is sent to backend when a call is finished
type CallCompleteMsg struct {
	Type      string    `json:"type"` // "call_complete"
	AgentID   string    `json:"agentId"`
	CallID    string    `json:"callId"`
	TalkTime  float64   `json:"talkTime"`  // seconds
	HoldTime  float64   `json:"holdTime"`  // seconds
	Timestamp time.Time `json:"timestamp"`
}
