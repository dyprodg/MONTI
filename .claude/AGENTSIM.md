# AgentSim

Go 1.23 service that simulates call center agents. Generates realistic agent data and connects each simulated agent to the backend via WebSocket, sending periodic heartbeats and state transitions.

## Project Structure

```
AgentSim/
├── cmd/agentsim/main.go    # Entry point
├── internal/
│   ├── agent/              # Generator, Simulator, AgentConnection
│   └── control/            # Control API (HTTP)
├── go.mod
└── Dockerfile
```

## How It Works

1. **Generator** creates agents with realistic attributes (name, location, business unit, skill group)
2. **Simulator** manages the lifecycle of all agents
3. Each active agent opens a WebSocket connection to the backend at `/ws/agent`
4. Agents send a heartbeat every 2 seconds
5. Agents cycle through states: `Available` -> `On Call` -> `After Call Work` -> `Available`
6. State transitions happen on randomized timers to simulate realistic call center activity

## Control API

All endpoints are on the control port (default: 8081).

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/status` | Simulation status (running, agent count) |
| `POST` | `/start` | Start simulation |
| `POST` | `/stop` | Stop simulation |
| `POST` | `/scale` | Scale active agent count |
| `GET` | `/config` | Current configuration |
| `GET` | `/stats` | Runtime statistics |
| `GET` | `/metrics` | Prometheus metrics |

## Commands

### Check Status

```bash
curl localhost:8081/status
```

Response:
```json
{"running":true,"totalAgents":2000,"activeAgents":500,"eventsSent":0,"startedAt":"2026-01-28T15:19:54Z"}
```

### Start Simulation

```bash
curl -X POST localhost:8081/start \
  -H 'Content-Type: application/json' \
  -d '{"activeAgents": 500}'
```

### Stop Simulation

```bash
curl -X POST localhost:8081/stop
```

### Scale Agents (up or down)

**Important:** Use `activeAgents`, not `targetAgents`

```bash
# Scale up to 500 agents
curl -X POST localhost:8081/scale \
  -H 'Content-Type: application/json' \
  -d '{"activeAgents": 500}'

# Scale down to 100 agents
curl -X POST localhost:8081/scale \
  -H 'Content-Type: application/json' \
  -d '{"activeAgents": 100}'
```

### View Config

```bash
curl localhost:8081/config
```

### View Stats

```bash
curl localhost:8081/stats
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `AGENTSIM_CONTROL_PORT` | Control API port | `8081` |
| `AGENTSIM_BACKEND_URL` | Backend URL to connect agents to | `http://localhost:8080` |
| `AGENTSIM_AGENTS` | Total number of agents to generate | `2000` |
| `AGENTSIM_ACTIVE_AGENTS` | Number of agents to activate on start | `100` |
| `AGENTSIM_AUTO_START` | Auto-start simulation on boot | `false` |
| `AGENTSIM_LOG_LEVEL` | Log level | `info` |

## Local Development

```bash
# With Docker
docker compose up -d agentsim

# Logs
docker compose logs -f agentsim

# Rebuild
docker compose up -d --build agentsim
```

## Production (EC2)

The AgentSim runs on EC2 alongside the backend. The control API is not exposed externally - use SSH to access it.

```bash
# SSH to EC2
ssh -i ~/.ssh/monti-key.pem ec2-user@3.69.80.81

# Check status
curl localhost:8081/status

# Start with 300 agents
curl -X POST localhost:8081/start \
  -H 'Content-Type: application/json' \
  -d '{"activeAgents": 300}'

# Scale to 500 agents
curl -X POST localhost:8081/scale \
  -H 'Content-Type: application/json' \
  -d '{"activeAgents": 1000}'

# Stop simulation
curl -X POST localhost:8081/stop
```

## Resource Guidelines

Recommended agent counts by EC2 instance type:

| Instance | vCPU | RAM | Max Agents | Notes |
|----------|------|-----|------------|-------|
| t3.small | 2 | 2 GB | ~500 | Current production |
| t3.medium | 2 | 4 GB | ~1000 | Recommended upgrade |
| t3.large | 2 | 8 GB | ~2000 | For load testing |

**Warning:** Running too many agents will overload the system - aggregation time increases, state changes freeze, and the dashboard becomes unresponsive.

Signs of overload:
- Aggregation time > 100ms (should be < 50ms)
- State changes/min drops to 0
- High CPU on backend and agentsim containers
