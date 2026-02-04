# Backend

Go-based backend service providing REST API and WebSocket connectivity for real-time agent monitoring.

## Key Dependencies

- Go 1.23
- Chi router
- gorilla/websocket
- Keycloak (OIDC authentication)

## Local Development

```bash
# Run via Docker Compose (from project root)
docker compose up -d backend

# Or run directly
cd Backend
go run ./cmd/server

# Run tests
go test ./...
```

## Environment Variables

Configured via `.env` or environment variables. See `docker-compose.yml` for defaults.
