package control

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/dennisdiepolder/monti/agentsim/internal/types"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

// API provides HTTP control interface for the simulation
type API struct {
	config      *types.SimulationConfig
	status      *types.SimulationStatus
	mu          sync.RWMutex
	logger      zerolog.Logger
	startFunc   func(int) error
	stopFunc    func() error
	scaleFunc   func(int) error
	statsFunc   func() map[string]interface{}
	metricsFunc func() map[string]interface{}
}

// NewAPI creates a new control API
func NewAPI(logger zerolog.Logger) *API {
	return &API{
		config: &types.SimulationConfig{
			TotalAgents:  2000,
			ActiveAgents: 0,
		},
		status: &types.SimulationStatus{
			Running:      false,
			TotalAgents:  2000,
			ActiveAgents: 0,
		},
		logger: logger,
	}
}

// SetTotalAgents updates the total agent count (called after agents are generated)
func (api *API) SetTotalAgents(total int) {
	api.mu.Lock()
	defer api.mu.Unlock()
	api.config.TotalAgents = total
	api.status.TotalAgents = total
}

// SetHandlers sets the control functions
func (api *API) SetHandlers(start func(int) error, stop func() error, scale func(int) error, stats func() map[string]interface{}, metrics func() map[string]interface{}) {
	api.startFunc = start
	api.stopFunc = stop
	api.scaleFunc = scale
	api.statsFunc = stats
	api.metricsFunc = metrics
}

// SetupRoutes configures HTTP routes
func (api *API) SetupRoutes(router *mux.Router) {
	router.HandleFunc("/health", api.healthHandler).Methods("GET")
	router.HandleFunc("/status", api.statusHandler).Methods("GET")
	router.HandleFunc("/start", api.startHandler).Methods("POST")
	router.HandleFunc("/stop", api.stopHandler).Methods("POST")
	router.HandleFunc("/scale", api.scaleHandler).Methods("POST")
	router.HandleFunc("/config", api.configHandler).Methods("GET", "PUT")
	router.HandleFunc("/stats", api.statsHandler).Methods("GET")
	router.HandleFunc("/metrics", api.metricsHandler).Methods("GET")
}

// healthHandler returns service health
func (api *API) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// statusHandler returns current simulation status
func (api *API) statusHandler(w http.ResponseWriter, r *http.Request) {
	api.mu.RLock()
	status := *api.status
	api.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// startHandler starts the simulation
func (api *API) startHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ActiveAgents int `json:"activeAgents"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ActiveAgents <= 0 || req.ActiveAgents > api.config.TotalAgents {
		req.ActiveAgents = 100 // default to 100 active agents
	}

	api.mu.Lock()
	if api.status.Running {
		api.mu.Unlock()
		http.Error(w, "simulation already running", http.StatusConflict)
		return
	}
	api.mu.Unlock()

	if err := api.startFunc(req.ActiveAgents); err != nil {
		api.logger.Error().Err(err).Msg("failed to start simulation")
		http.Error(w, "failed to start simulation", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	api.mu.Lock()
	api.status.Running = true
	api.status.ActiveAgents = req.ActiveAgents
	api.status.StartedAt = &now
	api.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":       "simulation started",
		"active_agents": req.ActiveAgents,
	})
}

// stopHandler stops the simulation
func (api *API) stopHandler(w http.ResponseWriter, r *http.Request) {
	api.mu.Lock()
	if !api.status.Running {
		api.mu.Unlock()
		http.Error(w, "simulation not running", http.StatusConflict)
		return
	}
	api.mu.Unlock()

	if err := api.stopFunc(); err != nil {
		api.logger.Error().Err(err).Msg("failed to stop simulation")
		http.Error(w, "failed to stop simulation", http.StatusInternalServerError)
		return
	}

	api.mu.Lock()
	api.status.Running = false
	api.status.StartedAt = nil
	api.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "simulation stopped",
	})
}

// configHandler gets or updates configuration
func (api *API) configHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		api.mu.RLock()
		config := *api.config
		api.mu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)
		return
	}

	// PUT - update config
	var newConfig types.SimulationConfig
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	api.mu.Lock()
	if api.status.Running {
		api.mu.Unlock()
		http.Error(w, "cannot change config while simulation is running", http.StatusConflict)
		return
	}

	api.config = &newConfig
	api.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "configuration updated",
	})
}

// statsHandler returns aggregator statistics
func (api *API) statsHandler(w http.ResponseWriter, r *http.Request) {
	stats := api.statsFunc()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// scaleHandler dynamically scales the number of active agents
func (api *API) scaleHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ActiveAgents int `json:"activeAgents"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ActiveAgents < 0 || req.ActiveAgents > api.config.TotalAgents {
		http.Error(w, "activeAgents must be between 0 and total agents", http.StatusBadRequest)
		return
	}

	if err := api.scaleFunc(req.ActiveAgents); err != nil {
		api.logger.Error().Err(err).Msg("failed to scale simulation")
		http.Error(w, "failed to scale simulation", http.StatusInternalServerError)
		return
	}

	api.mu.Lock()
	api.status.ActiveAgents = req.ActiveAgents
	api.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":       "simulation scaled",
		"active_agents": req.ActiveAgents,
	})
}

// metricsHandler returns Prometheus-compatible metrics
func (api *API) metricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := api.metricsFunc()

	// Output in Prometheus text format
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	for name, value := range metrics {
		switch v := value.(type) {
		case int:
			fmt.Fprintf(w, "%s %d\n", name, v)
		case int64:
			fmt.Fprintf(w, "%s %d\n", name, v)
		case float64:
			fmt.Fprintf(w, "%s %f\n", name, v)
		case bool:
			if v {
				fmt.Fprintf(w, "%s 1\n", name)
			} else {
				fmt.Fprintf(w, "%s 0\n", name)
			}
		default:
			fmt.Fprintf(w, "%s %v\n", name, v)
		}
	}
}

// Start starts the HTTP server
func (api *API) Start(ctx context.Context, addr string) error {
	router := mux.NewRouter()
	api.SetupRoutes(router)

	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		<-ctx.Done()
		api.logger.Info().Msg("shutting down control API")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	api.logger.Info().Str("addr", addr).Msg("control API started")
	return server.ListenAndServe()
}

// GetConfig returns current config
func (api *API) GetConfig() types.SimulationConfig {
	api.mu.RLock()
	defer api.mu.RUnlock()
	return *api.config
}
