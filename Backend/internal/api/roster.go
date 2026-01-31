package api

import (
	"encoding/json"
	"net/http"

	"github.com/dennisdiepolder/monti/backend/internal/cache"
	"github.com/dennisdiepolder/monti/backend/internal/types"
	"github.com/rs/zerolog"
)

// RosterEntry represents a single agent in the roster payload
type RosterEntry struct {
	AgentID    string          `json:"agentId"`
	Department types.Department `json:"department"`
	Location   types.Location   `json:"location"`
	Team       string          `json:"team"`
}

// RosterHandler handles the roster registration endpoint
type RosterHandler struct {
	tracker *cache.AgentStateTracker
	logger  zerolog.Logger
}

// NewRosterHandler creates a new RosterHandler
func NewRosterHandler(tracker *cache.AgentStateTracker, logger zerolog.Logger) *RosterHandler {
	return &RosterHandler{
		tracker: tracker,
		logger:  logger.With().Str("component", "roster").Logger(),
	}
}

// HandleRoster handles POST /internal/agents/roster
func (h *RosterHandler) HandleRoster(w http.ResponseWriter, r *http.Request) {
	var roster []RosterEntry
	if err := json.NewDecoder(r.Body).Decode(&roster); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	registered := 0
	for _, entry := range roster {
		h.tracker.RegisterOfflineAgent(entry.AgentID, entry.Department, entry.Location, entry.Team)
		registered++
	}

	h.logger.Info().Int("registered", registered).Msg("roster received")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]int{"registered": registered})
}
