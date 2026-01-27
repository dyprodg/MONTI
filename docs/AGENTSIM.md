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
5. Agents cycle through states: `Ready` -> `In Call` -> `After Call` -> `Ready`
6. State transitions happen on randomized timers to simulate realistic call center activity

## Control API

All endpoints are on the control port (default: 8081).

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/status` | Simulation status (running, agent count) |
| `POST` | `/start` | Start simulation |
| `POST` | `/stop` | Stop simulation |
| `POST` | `/scale` | Scale to target agent count |
| `GET` | `/config` | Current configuration |
| `GET` | `/stats` | Runtime statistics |
| `GET` | `/metrics` | Prometheus metrics |

### Start Simulation

```bash
curl -X POST localhost:8081/start \
  -H 'Content-Type: application/json' \
  -d '{"activeAgents": 1000}'
```

### Stop Simulation

```bash
curl -X POST localhost:8081/stop
```

### Scale Agents

```bash
curl -X POST localhost:8081/scale \
  -H 'Content-Type: application/json' \
  -d '{"targetAgents": 500}'
```

### Check Status

```bash
curl localhost:8081/status
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `AGENTSIM_CONTROL_PORT` | Control API port | `8081` |
| `AGENTSIM_BACKEND_URL` | Backend URL to connect agents to | `http://localhost:8080` |
| `AGENTSIM_AGENTS` | Total number of agents to generate | `200` |
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

## Production

In production, the AgentSim runs on EC2 alongside the backend. The control API is not exposed externally -- use SSH to access it.

```bash
# SSH to EC2
ssh -i ~/.ssh/monti-key.pem ec2-user@3.69.80.81

# Start simulation with 1000 agents
curl -X POST http://localhost:8081/start \
  -H 'Content-Type: application/json' \
  -d '{"activeAgents": 1000}'

# Check status
curl http://localhost:8081/status

# Stop
curl -X POST http://localhost:8081/stop
```

## Resource Usage (2000 agents)

| Metric | Value |
|--------|-------|
| CPU | ~11% |
| Memory | ~192 MB |
| WebSocket connections | 2000 (one per agent) |
| Heartbeat interval | 2 seconds |
