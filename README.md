# MONTI

Real-time call center monitoring platform. Displays 2000+ agents with live status updates via WebSocket.

## Architecture

```
                        HTTPS
  ┌──────────┐      ┌───────────┐      ┌────────────────────────────────────┐
  │ Browser  │─────►│CloudFront │      │  EC2 (Docker Compose + Caddy)     │
  │ (React)  │      │   + S3    │      │                                   │
  └──────────┘      └───────────┘      │  ┌──────────┐    ┌────────────┐  │
       │                               │  │ Backend  │◄───│  AgentSim  │  │
       │  WebSocket (wss://)           │  │  :8080   │    │   :8081    │  │
       └──────────────────────────────►│  └──────────┘    └────────────┘  │
                                       │  ┌──────────┐    ┌────────────┐  │
                                       │  │ Keycloak │    │ Prometheus │  │
                                       │  │  :8180   │    │   :9090    │  │
                                       │  └──────────┘    └────────────┘  │
                                       │  ┌──────────┐                    │
                                       │  │ Grafana  │                    │
                                       │  │  :3001   │                    │
                                       │  └──────────┘                    │
                                       └────────────────────────────────────┘
```

## Tech Stack

| Component | Technology |
|-----------|------------|
| Frontend | React 18, TypeScript, Vite |
| Backend | Go 1.23, Chi router, gorilla/websocket |
| AgentSim | Go 1.23, gorilla/websocket |
| Auth | Keycloak 23 (OIDC + PKCE) |
| Infra | Terraform, S3 + CloudFront, EC2, Caddy |
| Monitoring | Prometheus + Grafana |

## Quick Start (Local Dev)

```bash
# Start all services
docker compose up -d

# Wait for Keycloak, then run setup
./scripts/setup-keycloak.sh

# Start agent simulation
curl -X POST localhost:8081/start -H 'Content-Type: application/json' -d '{"activeAgents":200}'

# Open dashboard
open http://localhost:5173
```

## Demo Credentials

| Username | Password | Role | Sees |
|----------|----------|------|------|
| admin | admin | admin | All locations |
| supervisor | supervisor | supervisor | SGB + NGB |
| agent | agent | agent | SGB only |
| demo | demo | viewer | RGB only |

Keycloak Admin: http://localhost:8180/admin (admin / admin)
Grafana: http://localhost:3001 (admin / admin)

## Production URLs

| Service | URL |
|---------|-----|
| Frontend | https://monti.dennisdiepolder.com |
| Backend / API | https://montibackend.dennisdiepolder.com |
| Keycloak | https://montibackend.dennisdiepolder.com/realms/monti |
| Grafana | http://3.69.80.81:3001 |

## Documentation

- [Infrastructure](docs/INFRASTRUCTURE.md) -- Terraform, EC2, CloudFront, Caddy, deployment
- [Auth](docs/AUTH.md) -- Keycloak, OIDC, roles, groups
- [Monitoring](docs/monitoring.md) -- Prometheus, Grafana access

## Common Commands

```bash
# Logs
docker compose logs -f backend
docker compose logs -f agentsim

# Rebuild after code changes
docker compose up -d --build

# Stop services (preserves data)
docker compose down
```

> **WARNING:** Never use `docker compose down -v` -- this destroys all volumes including Keycloak configuration.
