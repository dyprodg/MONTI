package api

import (
	"encoding/json"
	"net/http"

	"github.com/dennisdiepolder/monti/backend/internal/storage"
	"github.com/dennisdiepolder/monti/backend/internal/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// AgentHistoryHandler provides REST endpoints for agent history data
type AgentHistoryHandler struct {
	store  storage.Store
	logger zerolog.Logger
}

// NewAgentHistoryHandler creates a new AgentHistoryHandler
func NewAgentHistoryHandler(store storage.Store, logger zerolog.Logger) *AgentHistoryHandler {
	return &AgentHistoryHandler{
		store:  store,
		logger: logger.With().Str("component", "agent_history_handler").Logger(),
	}
}

// GetHistory returns agent daily stats for the given agent
// GET /api/agents/{agentId}/history
func (h *AgentHistoryHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentId")
	if agentID == "" {
		http.Error(w, "agentId is required", http.StatusBadRequest)
		return
	}

	stats, err := h.store.GetAgentDailyStats(agentID)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", agentID).Msg("failed to get agent daily stats")
		http.Error(w, "failed to retrieve history", http.StatusInternalServerError)
		return
	}

	if stats == nil {
		stats = []types.AgentDailyStats{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetCalls returns call records for the given agent on a specific date
// GET /api/agents/{agentId}/calls?date=YYYY-MM-DD
func (h *AgentHistoryHandler) GetCalls(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentId")
	if agentID == "" {
		http.Error(w, "agentId is required", http.StatusBadRequest)
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		http.Error(w, "date query parameter is required (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	records, err := h.store.GetAgentCallsByDate(agentID, date)
	if err != nil {
		h.logger.Error().Err(err).
			Str("agent_id", agentID).
			Str("date", date).
			Msg("failed to get agent calls")
		http.Error(w, "failed to retrieve calls", http.StatusInternalServerError)
		return
	}

	if records == nil {
		records = []types.CallRecord{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}
