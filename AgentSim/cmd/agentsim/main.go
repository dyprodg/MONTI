package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/dennisdiepolder/monti/agentsim/internal/agent"
	"github.com/dennisdiepolder/monti/agentsim/internal/control"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type App struct {
	generator  *agent.Generator
	simulator  *agent.Simulator
	controlAPI *control.API
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.Mutex
	logger     zerolog.Logger
	backendURL string
}

func main() {
	// CLI flags
	var (
		controlPort = flag.String("control-port", "8081", "Control API port")
		backendURL  = flag.String("backend-url", "http://localhost:8080", "Backend URL")
		agentCount  = flag.Int("agents", 200, "Total number of agents to generate")
		autoStart   = flag.Bool("auto-start", false, "Automatically start simulation")
		activeAgents = flag.Int("active", 100, "Number of active agents (if auto-start is true)")
		logLevel    = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	)
	flag.Parse()

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

	// Generate agents
	logger.Info().Int("count", *agentCount).Msg("generating agents")
	app.generator = agent.NewGenerator(time.Now().UnixNano())
	agents := app.generator.GenerateAgents(*agentCount)
	logger.Info().Int("generated", len(agents)).Msg("agents generated")

	// Create simulator
	app.simulator = agent.NewSimulator(agents, *backendURL, logger)

	// Create control API
	app.controlAPI = control.NewAPI(logger)
	app.controlAPI.SetHandlers(
		app.startSimulation,
		app.stopSimulation,
		app.getStats,
	)

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

	// Start simulator
	go app.simulator.Start(app.ctx, activeAgents)

	return nil
}

func (app *App) stopSimulation() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	app.logger.Info().Msg("stopping simulation")

	// Cancel context to stop all goroutines
	app.cancel()

	// Recreate context for potential restart
	app.ctx, app.cancel = context.WithCancel(context.Background())

	return nil
}

func (app *App) getStats() map[string]interface{} {
	return map[string]interface{}{
		"active_agents": app.simulator.GetActiveCount(),
		"events_sent":   app.simulator.GetEventsSent(),
	}
}

func printUsage(port string) {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    AgentSim Control API                        ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Available endpoints:")
	fmt.Printf("  GET  http://localhost:%s/health  - Health check\n", port)
	fmt.Printf("  GET  http://localhost:%s/status  - Simulation status\n", port)
	fmt.Printf("  POST http://localhost:%s/start   - Start simulation\n", port)
	fmt.Printf("  POST http://localhost:%s/stop    - Stop simulation\n", port)
	fmt.Printf("  GET  http://localhost:%s/config  - Get configuration\n", port)
	fmt.Printf("  GET  http://localhost:%s/stats   - Get statistics\n", port)
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  curl http://localhost:%s/status\n", port)
	fmt.Printf("  curl -X POST http://localhost:%s/start -d '{\"activeAgents\":100}'\n", port)
	fmt.Printf("  curl -X POST http://localhost:%s/stop\n", port)
	fmt.Println()
}
