package callqueue

import (
	"encoding/json"
	"net/http"

	"github.com/dennisdiepolder/monti/backend/internal/types"
	"github.com/rs/zerolog"
)

// CallHandler handles HTTP requests for call queue operations
type CallHandler struct {
	mgr    *CallQueueManager
	logger zerolog.Logger
}

// NewCallHandler creates a new CallHandler
func NewCallHandler(mgr *CallQueueManager, logger zerolog.Logger) *CallHandler {
	return &CallHandler{
		mgr:    mgr,
		logger: logger,
	}
}

// enqueueRequest is the JSON body for POST /internal/call/enqueue
type enqueueRequest struct {
	VQ     string `json:"vq"`
	CallID string `json:"callId,omitempty"`
}

// enqueueResponse is the JSON response for a successful enqueue
type enqueueResponse struct {
	CallID string `json:"callId"`
	VQ     string `json:"vq"`
	Status string `json:"status"`
}

// HandleEnqueue handles POST /internal/call/enqueue
func (h *CallHandler) HandleEnqueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req enqueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.VQ == "" {
		http.Error(w, "missing vq field", http.StatusBadRequest)
		return
	}

	vqName := types.VQName(req.VQ)

	// Validate that the VQ name exists
	if _, ok := types.VQDepartmentMapping[vqName]; !ok {
		http.Error(w, "invalid vq name", http.StatusBadRequest)
		return
	}

	call := h.mgr.EnqueueCall(vqName, req.CallID)
	if call == nil {
		http.Error(w, "failed to enqueue call", http.StatusInternalServerError)
		return
	}

	resp := enqueueResponse{
		CallID: call.CallID,
		VQ:     string(call.VQ),
		Status: "waiting",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// HandleWipeAll handles DELETE /internal/calls/all
func (h *CallHandler) HandleWipeAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	count := h.mgr.WipeAllCalls()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "all calls wiped",
		"cleared": count,
	})
}

// HandleStats returns call queue statistics
// GET /internal/calls/stats
func (h *CallHandler) HandleStats(w http.ResponseWriter, r *http.Request) {
	snapshots := h.mgr.GetAllSnapshots()

	stats := map[string]interface{}{
		"totalQueues": len(snapshots),
		"queues":      snapshots,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
