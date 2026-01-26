# Project MONTI

**MONTI** is a high-performance live monitoring application for call centers, capable of displaying **2000+ agents in real-time**.

## Architecture

```
┌─────────────┐     WebSocket      ┌─────────────┐
│   Browser   │◄──────────────────►│   Backend   │
│  (React)    │                    │    (Go)     │
└─────────────┘                    └──────┬──────┘
                                          │ WebSocket
                                          ▼
                                   ┌─────────────┐
                                   │  AgentSim   │
                                   │    (Go)     │
                                   └─────────────┘
```

- **Real-time updates** via WebSocket (not HTTP polling)
- **Aggregated data** - Frontend receives grouped metrics, not individual agent updates
- **Scalable** - Tested with 2000 concurrent WebSocket connections

## Services

| Service | Description | Port |
|---------|-------------|------|
| **Frontend** | React dashboard (Vite) | 5173 |
| **Backend** | Go API + WebSocket server | 8080 |
| **AgentSim** | Agent simulation service | 8081 |
| **Keycloak** | Authentication (OIDC) | 8180 |
| **Prometheus** | Metrics collection | 9090 |
| **Grafana** | Monitoring dashboards | 3001 |

## Quick Start

```bash
# Start all services
docker compose up -d

# Setup Keycloak (realm, users, roles)
./scripts/setup-keycloak.sh

# Start agent simulation (2000 agents)
curl -X POST localhost:8081/start -d '{"activeAgents":2000}'

# View dashboard
open http://localhost:5173

# View Grafana metrics
open http://localhost:3001
```

## Demo Credentials

| Service | URL | Username | Password | Role |
|---------|-----|----------|----------|------|
| **Frontend** | localhost:5173 | admin | admin | Admin |
| | | supervisor | supervisor | Supervisor |
| | | agent | agent | Agent |
| | | demo | demo | Viewer |
| **Grafana** | localhost:3001 | admin | admin | - |
| **Keycloak Admin** | localhost:8180/admin | admin | admin | - |

### Stop simulation

```bash
curl -X POST localhost:8081/stop
```

## Tech Stack

| Component | Technology |
|-----------|------------|
| Frontend | React 18, TypeScript, Vite |
| Backend | Go 1.23, Chi, gorilla/websocket |
| AgentSim | Go 1.23, gorilla/websocket |
| Auth | Keycloak (OIDC) |
| Monitoring | Prometheus + Grafana |

## Development

```bash
# Run with hot-reload (volumes mounted)
docker compose up -d

# View logs
docker compose logs -f backend
docker compose logs -f agentsim

# Rebuild after changes
docker compose up -d --build

# Stop services (preserves data)
docker compose down
```

> **WARNING:** Never use `docker compose down -v` - this destroys all volumes including Keycloak configuration (users, groups, roles).

## Environment Variables

### Backend
- `PORT` - Server port (default: 8080)
- `ALLOWED_ORIGINS` - CORS origins
- `SKIP_AUTH` - Skip JWT validation (dev only)

### AgentSim
- `AGENTSIM_BACKEND_URL` - Backend WebSocket URL
- `AGENTSIM_AGENTS` - Total agents to simulate
- `AGENTSIM_AUTO_START` - Start simulation on boot

---

## Hosting Recommendation

### Recommended: EC2 + S3/CloudFront

```
┌─────────────────────┐     ┌───────────────────┐
│   S3 + CloudFront   │     │   EC2 t3.medium   │
│   (Static Frontend) │     │                   │
│                     │────►│  Backend :8080    │
│   ~$1/month         │     │  AgentSim :8081   │
│                     │     │  Prometheus       │
└─────────────────────┘     │  Grafana          │
                            │  Keycloak         │
                            │                   │
                            │  ~$30/month       │
                            └───────────────────┘
```

**Why this setup:**
- **Faster frontend**: CloudFront CDN globally distributed
- **Simpler EC2**: Only backend services
- **Security**: Frontend is static files, minimal attack surface

### Resource Requirements (2000 agents)

| Container | CPU | Memory |
|-----------|-----|--------|
| Backend | ~110% (1.1 cores) | ~260 MB |
| AgentSim | ~11% | ~192 MB |
| Keycloak | ~1% | ~382 MB |
| Grafana | ~1% | ~337 MB |
| Prometheus | ~0% | ~127 MB |
| **Total** | **~1.2 cores** | **~1.3 GB** |

**EC2 specs:**
- **t3.medium** (2 vCPU, 4GB RAM) - recommended for 2000 agents
- t3.small (2 vCPU, 2GB RAM) - minimum, runs at ~65% capacity
- Amazon Linux 2023
- Security Group: 80/443 inbound

### Cost Summary

| Resource | Cost/month |
|----------|-----------|
| EC2 t3.medium | ~$30 |
| EC2 t3.small (minimum) | ~$15 |
| S3 + CloudFront | ~$1 |
| **Total (recommended)** | **~$31** |
| **Total (minimum)** | **~$16** |

### CI/CD: GitHub Actions

```
Push to main
    │
    ├─► Frontend: Build → Deploy to S3
    │
    └─► Backend: SSH to EC2 → git pull → docker compose up -d
```
