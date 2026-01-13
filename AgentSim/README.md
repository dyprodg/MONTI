# AgentSim

Agent simulation service for generating and managing fake call center agent data.

## Purpose

Simulates 200 agents with realistic data patterns including:
- Agent states (available, busy, on call, break, offline, etc.)
- Department assignments (Sales, Support, Technical, Retention)
- Location assignments (Berlin, Munich, Hamburg, Frankfurt, Remote)
- Team assignments
- Real-time state transitions
- Pre-aggregated widget data for efficient WebSocket broadcasting

## Architecture

```
┌──────────────────┐
│   Agent          │  - 200 fake agents
│   Generator      │  - Assigned to departments, locations, teams
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│   Agent          │  - Simulates realistic state transitions
│   Simulator      │  - Sends state updates via channel
└────────┬─────────┘
         │ State updates
         ▼
┌──────────────────┐
│   Aggregator     │  - Collects all agent states
│                  │  - Groups by department
│                  │  - Creates complete widget snapshots every 1s
└────────┬─────────┘
         │ Widget JSON (1 per department per second)
         ▼
┌──────────────────┐
│   Backend        │  - Receives widgets via channel
│   WebSocket Hub  │  - Broadcasts to connected clients
└──────────────────┘
```

## Key Features

### Efficient Data Broadcasting
Instead of sending 100+ individual agent updates per second:
- Aggregates all agent states in memory
- Pre-filters by department
- Sends ONE complete widget JSON per department per second
- Dramatically reduces WebSocket traffic

### Agent States
- **Basic**: Available, Busy, On Call, Break, Offline
- **Extended**: After Call Work, Training, Meeting, Lunch
- **Call-specific**: On Hold, Transferring, Conference

### Realistic State Machine
- Probabilistic state transitions
- Realistic duration for each state
- Smooth flow: Available → On Call → After Call Work → Available
- Break patterns, meetings, training sessions

## Getting Started

### Build and Run

```bash
# Build
make build

# Run with auto-start (100 active agents)
make run

# Run with 150 active agents
make run-many

# Run in development mode with debug logging
make run-dev

# Run with custom settings
./bin/agentsim --control-port 8081 --agents 200 --auto-start --active 100
```

### CLI Flags

```
--control-port    Control API port (default: 8081)
--backend-url     Backend WebSocket URL (default: http://localhost:8080)
--agents          Total number of agents (default: 200)
--auto-start      Automatically start simulation on launch
--active          Number of active agents if auto-start is true (default: 100)
--log-level       Log level: debug, info, warn, error (default: info)
```

## Control API

AgentSim exposes a REST API for controlling the simulation:

### Endpoints

```bash
# Health check
curl http://localhost:8081/health

# Get simulation status
curl http://localhost:8081/status

# Start simulation with 100 active agents
curl -X POST http://localhost:8081/start \
  -H "Content-Type: application/json" \
  -d '{"activeAgents": 100}'

# Stop simulation
curl -X POST http://localhost:8081/stop

# Get configuration
curl http://localhost:8081/config

# Get statistics
curl http://localhost:8081/stats
```

### Response Examples

**Status Response:**
```json
{
  "running": true,
  "totalAgents": 200,
  "activeAgents": 100,
  "updatesPerSecond": 45.2,
  "widgetsSent": 1234,
  "startedAt": "2026-01-13T10:30:00Z"
}
```

**Widget Data Structure:**
```json
{
  "widgetType": "department_overview",
  "department": "sales",
  "timestamp": "2026-01-13T10:30:15Z",
  "summary": {
    "totalAgents": 60,
    "onlineAgents": 45,
    "onCall": 15,
    "available": 20,
    "stateBreakdown": {
      "available": 20,
      "on_call": 15,
      "break": 5,
      "offline": 15,
      "after_call_work": 5
    },
    "locationBreakdown": {
      "berlin": 15,
      "munich": 12,
      "remote": 18
    }
  },
  "agents": [...]
}
```

## Integration with Backend

The AgentSim service produces widget data that should be consumed by the Backend WebSocket hub:

1. AgentSim runs independently on port 8081
2. Aggregator produces widget JSONs every second
3. Backend service connects to AgentSim widget stream
4. Backend broadcasts widgets to connected frontend clients

## Development

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage
```

### Code Formatting

```bash
# Format code
make fmt

# Run linter
make lint
```

## Rules

### Code Organization
- All simulation logic must be in this directory
- No business logic should leak into other services
- Clear separation between data generation and data streaming

### Data Generation
- Agents must have realistic state transitions
- Support configurable number of agents (default: 200)
- Generate agents with varied teams, locations, and departments
- Ensure reproducible randomness with seed support

### Performance
- Must handle 200+ agents without performance degradation
- State updates are processed via channels
- Memory footprint remains constant
- Widget generation is non-blocking

### Configuration
- All simulation parameters configurable via CLI flags
- Default values work out of the box
- Can be controlled via REST API while running

### Testing
- Unit tests required for all data generation logic
- Performance tests for agent state transitions
- Integration tests with Backend service planned

## Dependencies

- **zerolog**: Structured logging
- **gorilla/mux**: HTTP routing for control API
- **google/uuid**: Unique ID generation (if needed in future)
- No dependency on Backend or Frontend code

## CI/CD

This service will only be built and tested when:
- Files in `AgentSim/` directory are changed
- Root configuration files affecting all services are changed
- Explicitly triggered via workflow dispatch
