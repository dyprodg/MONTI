package api

import (
	"encoding/json"
	"net/http"

	"github.com/dennisdiepolder/monti/backend/internal/callqueue"
	"github.com/dennisdiepolder/monti/backend/internal/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// AgentActionsHandler provides REST endpoints for agent control actions
type AgentActionsHandler struct {
	agentHub     *websocket.AgentHub
	callQueueMgr *callqueue.CallQueueManager
	logger       zerolog.Logger
}

// NewAgentActionsHandler creates a new AgentActionsHandler
func NewAgentActionsHandler(agentHub *websocket.AgentHub, callQueueMgr *callqueue.CallQueueManager, logger zerolog.Logger) *AgentActionsHandler {
	return &AgentActionsHandler{
		agentHub:     agentHub,
		callQueueMgr: callQueueMgr,
		logger:       logger.With().Str("component", "agent_actions").Logger(),
	}
}

// ForceEndCall handles POST /api/agents/{agentId}/calls/{callId}/end
func (h *AgentActionsHandler) ForceEndCall(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentId")
	callID := chi.URLParam(r, "callId")

	if agentID == "" || callID == "" {
		http.Error(w, "agentId and callId are required", http.StatusBadRequest)
		return
	}

	// Force-end the call in the queue manager
	foundAgentID, found := h.callQueueMgr.ForceEndCall(callID)
	if !found {
		http.Error(w, "call not found in active queues", http.StatusNotFound)
		return
	}

	// Notify the agent via WebSocket
	h.agentHub.ForceEndCall(foundAgentID, callID)

	h.logger.Info().
		Str("agent_id", agentID).
		Str("call_id", callID).
		Msg("force-ended call via API")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "call ended",
		"agentId": foundAgentID,
		"callId":  callID,
	})
}

// Logout handles POST /api/agents/{agentId}/logout
func (h *AgentActionsHandler) Logout(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentId")
	if agentID == "" {
		http.Error(w, "agentId is required", http.StatusBadRequest)
		return
	}

	// Force-disconnect the agent (hub handles cleanup)
	ok := h.agentHub.ForceDisconnect(agentID)
	if !ok {
		http.Error(w, "agent not connected", http.StatusNotFound)
		return
	}

	h.logger.Info().
		Str("agent_id", agentID).
		Msg("force-disconnected agent via API")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "agent logged out",
		"agentId": agentID,
	})
}
