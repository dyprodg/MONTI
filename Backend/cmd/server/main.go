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
	"github.com/dennisdiepolder/monti/backend/internal/auth"
	"github.com/dennisdiepolder/monti/backend/internal/cache"
	"github.com/dennisdiepolder/monti/backend/internal/config"
	"github.com/dennisdiepolder/monti/backend/internal/event"
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

	// Create WebSocket hub
	hub := websocket.NewHub(log.Logger)
	go hub.Run()

	// Create context for services
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Ticker disabled - now using widget aggregator for all broadcasts
	// tickerService := ticker.NewTicker(hub, 1*time.Second, log.Logger)
	// go tickerService.Start(ctx)

	// Create WebSocket handler
	wsHandler := websocket.NewHandler(hub, cfg, log.Logger)

	// Create event cache
	eventCache := cache.NewEventCache()

	// Create agent state tracker
	stateTracker := cache.NewAgentStateTracker()

	// Create event receiver
	eventReceiver := event.NewReceiver(eventCache, stateTracker, log.Logger)

	// Create aggregator
	aggregatorService := aggregator.NewAggregator(eventCache, stateTracker, hub, log.Logger)
	go aggregatorService.Start(ctx)

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

	// Internal routes (no auth - for internal services like AgentSim)
	r.Route("/internal", func(r chi.Router) {
		r.Post("/event", eventReceiver.HandleEvent)
		r.Get("/event/stats", eventReceiver.GetStats)
	})

	// Add auth middleware for protected routes
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware)
		r.Get("/ws", wsHandler.ServeHTTP)
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
