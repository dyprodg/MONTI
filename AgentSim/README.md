# AgentSim

Go-based simulator that generates realistic call center agent data and streams it to the backend via WebSocket.

## Key Dependencies

- Go 1.23
- gorilla/mux (control API)
- zerolog (structured logging)

## Local Development

```bash
# Run via Docker Compose (from project root)
docker compose up -d agentsim

# Or build and run directly
cd AgentSim
make build
make run          # 100 active agents
make run-many     # 150 active agents
make run-dev      # Debug logging

# Run with custom settings
./bin/agentsim --control-port 8081 --agents 200 --auto-start --active 100
```

## Control API

```bash
curl http://localhost:8081/status                                                    # Status
curl -X POST http://localhost:8081/start -H 'Content-Type: application/json' \
  -d '{"activeAgents": 100}'                                                        # Start
curl -X POST http://localhost:8081/stop                                             # Stop
```
