package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dennisdiepolder/monti/backend/internal/auth"
	"github.com/dennisdiepolder/monti/backend/internal/cache"
	"github.com/dennisdiepolder/monti/backend/internal/callqueue"
	"github.com/dennisdiepolder/monti/backend/internal/storage"
	"github.com/dennisdiepolder/monti/backend/internal/types"
	"github.com/rs/zerolog"
)

// AdminHandler proxies admin requests to AgentSim and handles local resets
type AdminHandler struct {
	simURL       string
	stateTracker *cache.AgentStateTracker
	callQueue    *callqueue.CallQueueManager
	store        storage.Store
	logger       zerolog.Logger
	client       *http.Client
}

// NewAdminHandler creates a new AdminHandler
func NewAdminHandler(simURL string, stateTracker *cache.AgentStateTracker, callQueue *callqueue.CallQueueManager, store storage.Store, logger zerolog.Logger) *AdminHandler {
	return &AdminHandler{
		simURL:       simURL,
		stateTracker: stateTracker,
		callQueue:    callQueue,
		store:        store,
		logger:       logger,
		client:       &http.Client{Timeout: 10 * time.Second},
	}
}

// RequireAdmin middleware — only admin role allowed
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetUserFromContext(r.Context())
		if !ok || !auth.HasRole(claims, "admin") {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"admin role required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireManagerOrAdmin middleware — manager or admin role allowed
func RequireManagerOrAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetUserFromContext(r.Context())
		if !ok || (claims.Role != "admin" && claims.Role != "manager" && claims.Role != "supervisor") {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"manager or admin role required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// proxyToSim forwards a request to AgentSim and copies the response back
func (h *AdminHandler) proxyToSim(w http.ResponseWriter, r *http.Request, method, path string) {
	url := h.simURL + path

	var body io.Reader
	if r.Body != nil && (method == http.MethodPost || method == http.MethodPut || method == http.MethodDelete) {
		body = r.Body
	}

	req, err := http.NewRequestWithContext(r.Context(), method, url, body)
	if err != nil {
		h.logger.Error().Err(err).Str("path", path).Msg("failed to create proxy request")
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		h.logger.Error().Err(err).Str("url", url).Msg("failed to reach AgentSim")
		http.Error(w, `{"error":"AgentSim unavailable"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// GetSimStatus proxies GET /status to AgentSim
func (h *AdminHandler) GetSimStatus(w http.ResponseWriter, r *http.Request) {
	h.proxyToSim(w, r, http.MethodGet, "/status")
}

// StartSim proxies POST /start to AgentSim
func (h *AdminHandler) StartSim(w http.ResponseWriter, r *http.Request) {
	h.proxyToSim(w, r, http.MethodPost, "/start")
}

// StopSim proxies POST /stop to AgentSim
func (h *AdminHandler) StopSim(w http.ResponseWriter, r *http.Request) {
	h.proxyToSim(w, r, http.MethodPost, "/stop")
}

// ScaleSim proxies POST /scale to AgentSim
func (h *AdminHandler) ScaleSim(w http.ResponseWriter, r *http.Request) {
	h.proxyToSim(w, r, http.MethodPost, "/scale")
}

// GetCallConfig proxies GET /calls/config to AgentSim
func (h *AdminHandler) GetCallConfig(w http.ResponseWriter, r *http.Request) {
	h.proxyToSim(w, r, http.MethodGet, "/calls/config")
}

// UpdateCallConfig proxies PUT /calls/config to AgentSim
func (h *AdminHandler) UpdateCallConfig(w http.ResponseWriter, r *http.Request) {
	h.proxyToSim(w, r, http.MethodPut, "/calls/config")
}

// InjectCalls enqueues calls directly into the local call queue
func (h *AdminHandler) InjectCalls(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Count int    `json:"count"`
		VQ    string `json:"vq,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.Count <= 0 {
		req.Count = 1
	}
	if req.Count > 1000 {
		req.Count = 1000
	}

	allVQs := []types.VQName{
		"sales_inbound", "sales_outbound", "sales_callback", "sales_chat",
		"support_general", "support_billing", "support_callback", "support_chat",
		"tech_l1", "tech_l2", "tech_callback", "tech_chat",
		"retention_save", "retention_cancel", "retention_callback", "retention_chat",
	}

	injected := 0
	for i := 0; i < req.Count; i++ {
		var vq types.VQName
		if req.VQ != "" {
			vq = types.VQName(req.VQ)
		} else {
			vq = allVQs[i%len(allVQs)]
		}
		if call := h.callQueue.EnqueueCall(vq, ""); call != nil {
			injected++
		}
	}

	h.logger.Info().Int("injected", injected).Int("requested", req.Count).Msg("calls injected via admin")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  fmt.Sprintf("injected %d calls", injected),
		"injected": injected,
		"errors":   req.Count - injected,
	})
}

// WipeAllCalls clears all local call queues
func (h *AdminHandler) WipeAllCalls(w http.ResponseWriter, r *http.Request) {
	cleared := h.callQueue.WipeAllCalls()

	h.logger.Info().Int("cleared", cleared).Msg("all calls wiped via admin")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "all calls wiped",
		"cleared": cleared,
	})
}

// ResetMemory clears backend in-memory state (agent tracker + call queues)
func (h *AdminHandler) ResetMemory(w http.ResponseWriter, r *http.Request) {
	agentsCleared := h.stateTracker.Clear()
	callsCleared := h.callQueue.WipeAllCalls()

	h.logger.Info().
		Int("agents", agentsCleared).
		Int("calls", callsCleared).
		Msg("backend memory reset")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":       "backend memory reset",
		"agentsCleared": agentsCleared,
		"callsCleared":  callsCleared,
	})
}

// WipeDynamo truncates all DynamoDB tables
func (h *AdminHandler) WipeDynamo(w http.ResponseWriter, r *http.Request) {
	if err := h.store.TruncateAll(); err != nil {
		h.logger.Error().Err(err).Msg("failed to truncate DynamoDB tables")
		http.Error(w, fmt.Sprintf(`{"error":"failed to truncate: %s"}`, err), http.StatusInternalServerError)
		return
	}

	h.logger.Info().Msg("DynamoDB tables truncated")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "DynamoDB tables truncated",
	})
}

// LogoffAll scales agents to 0 (keeps simulation running) and clears backend state.
func (h *AdminHandler) LogoffAll(w http.ResponseWriter, r *http.Request) {
	url := h.simURL + "/scale"
	body := strings.NewReader(`{"activeAgents":0}`)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, url, body)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to reach AgentSim for logoff-all")
		http.Error(w, `{"error":"AgentSim unavailable"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Clear local backend state
	agentsCleared := h.stateTracker.Clear()
	callsCleared := h.callQueue.WipeAllCalls()

	w.Header().Set("Content-Type", "application/json")
	if resp.StatusCode >= 400 {
		// Scale might fail if sim is not running — still clear local state
		h.logger.Warn().Int("status", resp.StatusCode).Msg("AgentSim scale to 0 returned error, local state still cleared")
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":       "all agents logged off",
		"agentsCleared": agentsCleared,
		"callsCleared":  callsCleared,
	})
}
