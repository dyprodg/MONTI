# AgentSim Quick Start Guide

This guide will help you get AgentSim up and running quickly.

## Prerequisites

- Go 1.23 or higher
- Backend service running on `http://localhost:8080`
- Make (optional, for easier commands)

## Quick Start

### Option 1: Using Make (Recommended)

```bash
# 1. Install dependencies
make install-deps

# 2. Build the service
make build

# 3. Run with auto-start (100 active agents)
make run
```

### Option 2: Manual Build and Run

```bash
# 1. Install dependencies
cd AgentSim
go mod download

# 2. Build
go build -o bin/agentsim ./cmd/agentsim

# 3. Run
./bin/agentsim --auto-start --active 100
```

### Option 3: Direct Run (No Build)

```bash
go run ./cmd/agentsim --auto-start --active 50
```

## Verifying It Works

### 1. Check AgentSim Health

```bash
curl http://localhost:8081/health
```

Expected response:
```json
{
  "status": "healthy",
  "time": "2026-01-13T10:30:00Z"
}
```

### 2. Check Simulation Status

```bash
curl http://localhost:8081/status
```

Expected response:
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

### 3. Check Backend is Receiving Widgets

```bash
curl http://localhost:8080/internal/widget/stats
```

Expected response:
```json
{
  "widgets_received": 450,
  "last_received": "2026-01-13T10:35:15Z",
  "connected_clients": 2
}
```

## Testing the Full Flow

### 1. Start Backend (Terminal 1)

```bash
cd Backend
go run ./cmd/server
```

### 2. Start AgentSim (Terminal 2)

```bash
cd AgentSim
make run
# or
./bin/agentsim --auto-start --active 100 --log-level debug
```

### 3. Connect Frontend (Terminal 3)

Open your browser and navigate to the frontend. Connect to the WebSocket at `ws://localhost:8080/ws` (with proper auth token).

You should see widget updates coming in every second with complete agent data for each department.

## Controlling the Simulation

### Start Simulation (if not auto-started)

```bash
curl -X POST http://localhost:8081/start \
  -H "Content-Type: application/json" \
  -d '{"activeAgents": 150}'
```

### Stop Simulation

```bash
curl -X POST http://localhost:8081/stop
```

### Get Statistics

```bash
curl http://localhost:8081/stats
```

Response:
```json
{
  "widgets_sent": 3456,
  "last_snapshot": "2026-01-13T10:35:00Z",
  "tracked_agents": 200,
  "active_agents": 100,
  "widgets_sent_to_backend": 3456
}
```

## Configuration Options

All settings can be configured via CLI flags:

```bash
./bin/agentsim \
  --control-port 8081 \
  --backend-url http://localhost:8080 \
  --agents 200 \
  --auto-start \
  --active 100 \
  --log-level info
```

## Understanding the Data Flow

```
┌─────────────┐
│  AgentSim   │  Port 8081
│  Simulator  │  - Generates 200 fake agents
└──────┬──────┘  - Simulates state changes
       │
       │ Every 1 second, sends 4 widget JSONs
       │ (one per department: Sales, Support, Technical, Retention)
       │
       ▼
┌─────────────────────┐
│  Backend            │  Port 8080
│  /internal/widget   │  - Receives aggregated widgets
└──────┬──────────────┘  - Broadcasts via WebSocket
       │
       ▼
┌─────────────────────┐
│  Frontend Clients   │
│  ws://localhost:8080/ws
└─────────────────────┘
```

## Widget Data Structure

Each widget sent to the backend looks like this:

```json
{
  "widgetType": "department_overview",
  "department": "sales",
  "timestamp": "2026-01-13T10:30:15.123Z",
  "summary": {
    "totalAgents": 60,
    "onlineAgents": 45,
    "onCall": 15,
    "available": 20,
    "stateBreakdown": {
      "available": 20,
      "on_call": 15,
      "after_call_work": 5,
      "break": 5,
      "offline": 15
    },
    "locationBreakdown": {
      "berlin": 15,
      "munich": 12,
      "hamburg": 8,
      "frankfurt": 7,
      "remote": 18
    }
  },
  "agents": [
    {
      "id": "AGT-00001",
      "department": "sales",
      "location": "berlin",
      "team": "Sales-Team-1",
      "state": "on_call",
      "stateStart": "2026-01-13T10:29:45Z",
      "lastUpdate": "2026-01-13T10:30:15Z"
    }
    // ... more agents
  ],
  "metadata": {
    "generatedBy": "agentsim",
    "version": "1.0"
  }
}
```

## Troubleshooting

### AgentSim can't connect to Backend

Check that:
1. Backend is running on port 8080
2. Backend has the `/internal/widget` endpoint
3. No firewall blocking the connection

Test manually:
```bash
curl -X POST http://localhost:8080/internal/widget \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}'
```

### No widgets being sent

Check AgentSim logs:
```bash
./bin/agentsim --log-level debug --auto-start --active 10
```

Look for messages like:
- "agent simulation started"
- "aggregator started"
- "widgets sent to backend"

### High CPU usage

If you're running too many active agents:
```bash
# Stop simulation
curl -X POST http://localhost:8081/stop

# Restart with fewer agents
curl -X POST http://localhost:8081/start \
  -d '{"activeAgents": 50}'
```

## Performance Tuning

### For Development (lower load)
```bash
./bin/agentsim --active 50
```

### For Testing (realistic load)
```bash
./bin/agentsim --active 150
```

### For Stress Testing (max load)
```bash
./bin/agentsim --active 200
```

## Next Steps

1. Connect your Frontend to the WebSocket
2. Create visualizations for the widget data
3. Filter widgets by department based on user role
4. Add more widget types (team overview, location overview, etc.)

## Support

For issues or questions, check:
- AgentSim logs (stdout)
- Backend logs
- GitHub issues: https://github.com/dennisdiepolder/monti/issues
