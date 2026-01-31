package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/dennisdiepolder/monti/agentsim/internal/agent"
	"github.com/dennisdiepolder/monti/agentsim/internal/callgen"
	"github.com/dennisdiepolder/monti/agentsim/internal/control"
	agentTypes "github.com/dennisdiepolder/monti/agentsim/internal/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// getEnvString returns the environment variable value or fallback
func getEnvString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvInt returns the environment variable as int or fallback
func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

// getEnvBool returns the environment variable as bool or fallback
func getEnvBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return fallback
}

type App struct {
	generator     *agent.Generator
	simulator     *agent.Simulator
	callGenerator *callgen.CallGenerator
	controlAPI    *control.API
	ctx           context.Context    // root context — only cancelled on process shutdown
	cancel        context.CancelFunc // root cancel
	simCtx        context.Context    // simulation context — cancelled on stop
	simCancel     context.CancelFunc // simulation cancel
	mu            sync.Mutex
	logger        zerolog.Logger
	backendURL    string
}

func main() {
	// CLI flags (with env var fallbacks)
	var (
		controlPort  = flag.String("control-port", "8081", "Control API port")
		backendURL   = flag.String("backend-url", "http://localhost:8080", "Backend URL")
		agentCount   = flag.Int("agents", 200, "Total number of agents to generate")
		autoStart    = flag.Bool("auto-start", false, "Automatically start simulation")
		activeAgents = flag.Int("active", 100, "Number of active agents (if auto-start is true)")
		logLevel     = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	)
	flag.Parse()

	// Environment variables override CLI flags
	// AGENTSIM_CONTROL_PORT, AGENTSIM_BACKEND_URL, AGENTSIM_AGENTS,
	// AGENTSIM_AUTO_START, AGENTSIM_ACTIVE_AGENTS, AGENTSIM_LOG_LEVEL
	*controlPort = getEnvString("AGENTSIM_CONTROL_PORT", *controlPort)
	*backendURL = getEnvString("AGENTSIM_BACKEND_URL", *backendURL)
	*agentCount = getEnvInt("AGENTSIM_AGENTS", *agentCount)
	*autoStart = getEnvBool("AGENTSIM_AUTO_START", *autoStart)
	*activeAgents = getEnvInt("AGENTSIM_ACTIVE_AGENTS", *activeAgents)
	*logLevel = getEnvString("AGENTSIM_LOG_LEVEL", *logLevel)

	// Setup logger
	level, err := zerolog.ParseLevel(*logLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
		With().
		Str("service", "agentsim").
		Logger()

	logger.Info().Msg("starting AgentSim service")

	// Create application
	app := &App{
		logger:     logger,
		backendURL: *backendURL,
	}
	app.ctx, app.cancel = context.WithCancel(context.Background())

	// Generate agents (always 2000: 500 per department)
	logger.Info().Msg("generating agents (500 per department)")
	app.generator = agent.NewGenerator(time.Now().UnixNano())
	agents := app.generator.GenerateAgents(0) // count ignored, always 2000
	logger.Info().Int("generated", len(agents)).Msg("agents generated")

	// POST roster to backend so all agents are pre-registered (retry until backend is reachable)
	go postRoster(logger, *backendURL, agents)

	// Create simulator
	app.simulator = agent.NewSimulator(agents, *backendURL, logger)

	// Create call generator
	callAPIClient := callgen.NewCallAPIClient(*backendURL)
	app.callGenerator = callgen.NewCallGenerator(callAPIClient)

	// Create control API
	app.controlAPI = control.NewAPI(logger)
	app.controlAPI.SetTotalAgents(len(agents))
	app.controlAPI.SetHandlers(
		app.startSimulation,
		app.stopSimulation,
		app.scaleSimulation,
		app.getStats,
		app.getMetrics,
	)
	app.controlAPI.SetCallGenerator(app.callGenerator)
	app.controlAPI.SetCallAPIClient(callAPIClient, *backendURL)

	// Start control API
	go func() {
		addr := fmt.Sprintf(":%s", *controlPort)
		if err := app.controlAPI.Start(app.ctx, addr); err != nil {
			logger.Error().Err(err).Msg("control API stopped")
		}
	}()

	// Auto-start if requested
	if *autoStart {
		logger.Info().Int("active_agents", *activeAgents).Msg("auto-starting simulation")
		if err := app.startSimulation(*activeAgents); err != nil {
			logger.Error().Err(err).Msg("failed to auto-start simulation")
		}
	}

	// Print usage
	logger.Info().
		Str("control_api", fmt.Sprintf("http://localhost:%s", *controlPort)).
		Str("backend_url", *backendURL).
		Msg("AgentSim ready")

	printUsage(*controlPort)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info().Msg("shutting down AgentSim")
	app.cancel()
	time.Sleep(1 * time.Second)
}

func (app *App) startSimulation(activeAgents int) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	app.logger.Info().Int("active_agents", activeAgents).Msg("starting simulation")

	// Create a child context for this simulation run
	app.simCtx, app.simCancel = context.WithCancel(app.ctx)

	// Start simulator
	go app.simulator.Start(app.simCtx, activeAgents)

	// Start call generator (generates calls and posts to backend)
	go app.callGenerator.Run(app.simCtx)

	return nil
}

func (app *App) stopSimulation() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	app.logger.Info().Msg("stopping simulation")

	// Stop the simulator (cancels agent goroutines and closes connections)
	app.simulator.Stop()

	// Cancel the simulation context to stop call generator and any other sim goroutines
	if app.simCancel != nil {
		app.simCancel()
		app.simCancel = nil
	}

	return nil
}

func (app *App) scaleSimulation(targetAgents int) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	app.logger.Info().Int("target_agents", targetAgents).Msg("scaling simulation")

	if app.simCtx == nil {
		return fmt.Errorf("simulation not running")
	}

	// Scale the simulator to the target number of agents
	return app.simulator.Scale(app.simCtx, targetAgents)
}

func (app *App) getStats() map[string]interface{} {
	return map[string]interface{}{
		"active_agents": app.simulator.GetActiveCount(),
		"events_sent":   app.simulator.GetEventsSent(),
	}
}

func (app *App) getMetrics() map[string]interface{} {
	return app.simulator.GetMetrics()
}

// rosterEntry is the JSON payload for each agent in the roster POST
type rosterEntry struct {
	AgentID    string               `json:"agentId"`
	Department agentTypes.Department `json:"department"`
	Location   agentTypes.Location   `json:"location"`
	Team       string               `json:"team"`
}

// postRoster sends the full agent roster to the backend with retry
func postRoster(logger zerolog.Logger, backendURL string, agents []agentTypes.Agent) {
	roster := make([]rosterEntry, len(agents))
	for i, a := range agents {
		roster[i] = rosterEntry{
			AgentID:    a.ID,
			Department: a.Department,
			Location:   a.Location,
			Team:       a.Team,
		}
	}

	body, err := json.Marshal(roster)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to marshal roster")
	}

	url := backendURL + "/internal/agents/roster"
	for {
		resp, err := http.Post(url, "application/json", bytes.NewReader(body))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				logger.Info().Int("agents", len(roster)).Msg("roster posted to backend")
				return
			}
			logger.Warn().Int("status", resp.StatusCode).Msg("roster POST failed, retrying...")
		} else {
			logger.Warn().Err(err).Msg("backend not reachable for roster, retrying...")
		}
		time.Sleep(2 * time.Second)
	}
}

func printUsage(port string) {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    AgentSim Control API                        ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Available endpoints:")
	fmt.Printf("  GET  http://localhost:%s/health   - Health check\n", port)
	fmt.Printf("  GET  http://localhost:%s/status   - Simulation status\n", port)
	fmt.Printf("  POST http://localhost:%s/start    - Start simulation\n", port)
	fmt.Printf("  POST http://localhost:%s/stop     - Stop simulation\n", port)
	fmt.Printf("  POST http://localhost:%s/scale    - Scale active agents\n", port)
	fmt.Printf("  GET  http://localhost:%s/config   - Get configuration\n", port)
	fmt.Printf("  GET  http://localhost:%s/stats    - Get statistics\n", port)
	fmt.Printf("  GET  http://localhost:%s/metrics       - Prometheus metrics\n", port)
	fmt.Printf("  GET  http://localhost:%s/calls/config  - Call generation config\n", port)
	fmt.Printf("  PUT  http://localhost:%s/calls/config  - Update call gen config\n", port)
	fmt.Printf("  POST http://localhost:%s/calls/inject  - Inject single call\n", port)
	fmt.Printf("  GET  http://localhost:%s/calls/stats   - Call gen statistics\n", port)
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  curl http://localhost:%s/status\n", port)
	fmt.Printf("  curl -X POST http://localhost:%s/start -d '{\"activeAgents\":100}'\n", port)
	fmt.Printf("  curl -X POST http://localhost:%s/scale -d '{\"activeAgents\":500}'\n", port)
	fmt.Printf("  curl -X POST http://localhost:%s/stop\n", port)
	fmt.Printf("  curl http://localhost:%s/calls/config\n", port)
	fmt.Printf("  curl -X PUT http://localhost:%s/calls/config -d '{\"peakHourFactor\":1.5}'\n", port)
	fmt.Println()
}
