# Backend

Go 1.23 server providing a WebSocket hub for real-time agent monitoring. Receives agent heartbeats from AgentSim, aggregates data, and broadcasts updates to browser clients.

## Project Structure

```
Backend/
├── cmd/server/main.go      # Entry point, routes
├── internal/
│   ├── auth/               # JWT validation, OIDC middleware
│   ├── websocket/          # Hub, AgentHub, Handler, Client
│   ├── cache/              # AgentStateTracker, event cache
│   ├── aggregator/         # Widget aggregation, broadcast loop
│   ├── config/             # Configuration
│   ├── metrics/            # Prometheus metrics
│   └── types/              # Shared types
├── .env.example
├── go.mod
├── Dockerfile              # Production
└── Dockerfile.dev          # Dev with hot reload
```

## API Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/health` | No | Health check |
| `GET` | `/metrics` | No | Prometheus metrics |
| `POST` | `/internal/event` | No | Receive events from AgentSim |
| `GET` | `/internal/event/stats` | No | Event statistics |
| `GET` | `/ws/agent` | No | Agent WebSocket (AgentSim connects here) |
| `GET` | `/ws` | Yes | Frontend WebSocket (browser clients) |

## WebSocket Protocol

### Frontend (`/ws`)

Browser clients connect with a JWT token:

```
ws://localhost:8080/ws?token=<access_token>
```

The backend:
1. Validates the JWT (signature via JWKS from Keycloak)
2. Extracts user roles and business unit groups from claims
3. Sends aggregated widget data every 1 second
4. Filters data based on the user's group memberships

### Agent (`/ws/agent`)

AgentSim connects one WebSocket per simulated agent:
- Agents send heartbeats every 2 seconds
- State change messages sent on demand
- Backend marks agents as stale after 6 seconds without a heartbeat (checked every 2 seconds)

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `ALLOWED_ORIGINS` | CORS origins (comma-separated) | `http://localhost:5173,http://localhost:3000` |
| `WS_READ_TIMEOUT` | WebSocket read timeout (seconds) | `60` |
| `WS_WRITE_TIMEOUT` | WebSocket write timeout (seconds) | `10` |
| `LOG_LEVEL` | Log level | `debug` |
| `ENV` | Environment (`development` / `production`) | - |
| `SKIP_AUTH` | Skip JWT validation (dev only) | `false` |
| `VERIFY_JWT_SIGNATURE` | Force JWT signature verification | auto (`true` in prod) |
| `OIDC_ISSUER` | Keycloak realm URL | `http://localhost:8180/realms/monti` |

## Local Development

```bash
# With Docker (hot reload via volume mounts)
docker compose up -d backend

# Logs
docker compose logs -f backend

# Rebuild
docker compose up -d --build backend
```

## Key Components

### Hub (`internal/websocket/`)
Manages frontend WebSocket connections. Each connected browser client is registered with the hub. The aggregator broadcasts to all connected clients through the hub.

### AgentHub (`internal/websocket/`)
Manages agent WebSocket connections from AgentSim. Each simulated agent maintains its own WebSocket connection.

### AgentStateTracker (`internal/cache/`)
In-memory store of current agent states. Tracks last heartbeat time and marks agents as stale when heartbeats stop.

### Aggregator (`internal/aggregator/`)
Runs a 1-second broadcast loop. Reads current agent states from the cache, groups them into widgets (by location, status, business unit), and sends the aggregated data to each frontend client (filtered by their groups).

### Auth Middleware (`internal/auth/`)
- Fetches and caches JWKS from Keycloak
- Validates JWT signatures
- Extracts `realm_access.roles` and `groups` claims
- In development with `SKIP_AUTH=true`, skips validation entirely

## Production

In production the backend runs behind Caddy (reverse proxy with automatic TLS). Caddy routes `/realms/*`, `/admin/*`, `/resources/*`, `/js/*` to Keycloak, and everything else to the backend.

The backend Docker image is pushed to ECR and pulled on the EC2 instance.

```bash
# Build and push
docker build -t monti-backend Backend/
docker tag monti-backend:latest <ECR_URL>:latest
docker push <ECR_URL>:latest

# On EC2
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d
```
