package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dennisdiepolder/monti/backend/internal/aggregator"
	"github.com/dennisdiepolder/monti/backend/internal/api"
	"github.com/dennisdiepolder/monti/backend/internal/auth"
	"github.com/dennisdiepolder/monti/backend/internal/cache"
	"github.com/dennisdiepolder/monti/backend/internal/callqueue"
	"github.com/dennisdiepolder/monti/backend/internal/config"
	"github.com/dennisdiepolder/monti/backend/internal/event"
	"github.com/dennisdiepolder/monti/backend/internal/ingestion"
	"github.com/dennisdiepolder/monti/backend/internal/metrics"
	"github.com/dennisdiepolder/monti/backend/internal/storage"
	"github.com/dennisdiepolder/monti/backend/internal/websocket"
	"github.com/dennisdiepolder/monti/backend/pkg/middleware"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configure logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	// Set log level
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Warn().Str("level", cfg.LogLevel).Msg("invalid log level, using info")
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	log.Info().
		Str("port", cfg.Port).
		Strs("allowed_origins", cfg.AllowedOrigins).
		Str("log_level", cfg.LogLevel).
		Msg("starting MONTI backend server")

	// Create WebSocket hub for frontend clients
	hub := websocket.NewHub(log.Logger)
	go hub.Run()

	// Create context for services
	ctx, cancel := context.WithCancel(context.Background())

	// Create agent state tracker
	stateTracker := cache.NewAgentStateTracker()

	// Create event processor
	processor := ingestion.NewDefaultProcessor(stateTracker, log.Logger)

	// Initialize storage
	store, err := storage.NewStore(ctx, log.Logger)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize storage")
	}

	// Create call queue manager
	callQueueMgr := callqueue.NewCallQueueManager(stateTracker, log.Logger)
	callQueueMgr.SetStore(store)
	processor.SetCallCompleter(callQueueMgr)

	// Create agent WebSocket hub
	agentHub := websocket.NewAgentHub(stateTracker, processor, log.Logger)
	go agentHub.Run()

	// Start stale agent checker (every 2 seconds)
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stateTracker.CheckStaleAgents()
				stateTracker.RemoveDisconnected(30 * time.Second) // Remove after 30s disconnected
			}
		}
	}()
	defer cancel()

	// Ticker disabled - now using widget aggregator for all broadcasts
	// tickerService := ticker.NewTicker(hub, 1*time.Second, log.Logger)
	// go tickerService.Start(ctx)

	// Create WebSocket handler for frontend clients
	wsHandler := websocket.NewHandler(hub, cfg, log.Logger)

	// Create agent WebSocket handler
	agentWsHandler := websocket.NewAgentHandler(agentHub, log.Logger)

	// Create call handler and routing loop
	callHandler := callqueue.NewCallHandler(callQueueMgr, log.Logger)
	routingLoop := callqueue.NewRoutingLoop(callQueueMgr, agentHub, log.Logger)
	go routingLoop.Start(ctx)

	// Create event cache
	eventCache := cache.NewEventCache()

	// Create event receiver (uses the already created stateTracker)
	eventReceiver := event.NewReceiver(eventCache, stateTracker, log.Logger)

	// Create aggregator
	aggregatorService := aggregator.NewAggregator(eventCache, stateTracker, hub, log.Logger)
	aggregatorService.SetCallQueue(callQueueMgr)
	go aggregatorService.Start(ctx)

	// Initialize JWKS for production token verification
	skipAuth := os.Getenv("SKIP_AUTH")
	if skipAuth != "true" {
		issuer := os.Getenv("OIDC_ISSUER")
		if issuer != "" {
			if err := auth.InitJWKS(issuer, 20); err != nil {
				log.Fatal().Err(err).Msg("failed to initialize JWKS (Keycloak not reachable)")
			}
		}
	}

	// Create router
	r := chi.NewRouter()

	// Add middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.Logger(log.Logger))
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	// Register public routes (no auth required)
	r.Get("/health", healthHandler)
	r.Get("/metrics", metrics.Get().Handler())

	// Create roster handler
	rosterHandler := api.NewRosterHandler(stateTracker, log.Logger)

	// Internal routes (no auth - for internal services like AgentSim)
	r.Route("/internal", func(r chi.Router) {
		r.Post("/event", eventReceiver.HandleEvent)
		r.Get("/event/stats", eventReceiver.GetStats)
		r.Post("/call/enqueue", callHandler.HandleEnqueue)
		r.Post("/calls/inject", callHandler.HandleEnqueue) // alias for inject
		r.Get("/calls/stats", callHandler.HandleStats)
		r.Delete("/calls/all", callHandler.HandleWipeAll)
		r.Post("/agents/roster", rosterHandler.HandleRoster)
	})

	// Agent WebSocket endpoints (no auth - for internal AgentSim connections)
	r.Get("/ws/agent", agentWsHandler.ServeHTTP)
	r.Get("/ws/agent/multiplexed", agentWsHandler.ServeMultiplexedHTTP)

	// Create agent history handler
	agentHistoryHandler := api.NewAgentHistoryHandler(store, log.Logger)

	// Create agent actions handler
	agentActionsHandler := api.NewAgentActionsHandler(agentHub, callQueueMgr, log.Logger)

	// Create admin handler for simulation control
	agentSimURL := os.Getenv("AGENTSIM_URL")
	if agentSimURL == "" {
		agentSimURL = "http://localhost:8081"
	}
	adminHandler := api.NewAdminHandler(agentSimURL, stateTracker, callQueueMgr, store, log.Logger)

	// Add auth middleware for protected routes
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware)

		// Public authenticated routes (any role)
		r.Get("/ws", wsHandler.ServeHTTP)
		r.Get("/api/agents/{agentId}/history", agentHistoryHandler.GetHistory)
		r.Get("/api/agents/{agentId}/calls", agentHistoryHandler.GetCalls)

		// Supervisor routes (manager/supervisor + admin only)
		r.Group(func(r chi.Router) {
			r.Use(api.RequireManagerOrAdmin)
			r.Post("/api/agents/{agentId}/calls/{callId}/end", agentActionsHandler.ForceEndCall)
			r.Post("/api/agents/{agentId}/logout", agentActionsHandler.Logout)
		})

		// Admin routes (admin only)
		r.Route("/api/admin", func(r chi.Router) {
			r.Use(api.RequireAdmin)
			r.Get("/sim/status", adminHandler.GetSimStatus)
			r.Post("/sim/start", adminHandler.StartSim)
			r.Post("/sim/stop", adminHandler.StopSim)
			r.Post("/sim/scale", adminHandler.ScaleSim)
			r.Get("/calls/config", adminHandler.GetCallConfig)
			r.Put("/calls/config", adminHandler.UpdateCallConfig)
			r.Post("/calls/inject", adminHandler.InjectCalls)
			r.Delete("/calls/all", adminHandler.WipeAllCalls)
			r.Post("/reset/memory", adminHandler.ResetMemory)
			r.Delete("/reset/dynamo", adminHandler.WipeDynamo)
			r.Post("/agents/logoff-all", adminHandler.LogoffAll)
		})
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Info().Msgf("server listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down server...")

	// Cancel ticker context
	cancel()

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("server forced to shutdown")
	}

	log.Info().Msg("server stopped")
}

// healthHandler handles health check requests
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","service":"monti-backend"}`)
}
