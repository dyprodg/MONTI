# Claude Code Instructions for MONTI

## CRITICAL - DO NOT DO THESE THINGS

### Keycloak - NEVER TOUCH
- **NEVER** delete, recreate, or restart the Keycloak container
- **NEVER** remove the `keycloak-data` volume
- **NEVER** run `docker compose down -v` (destroys volumes)
- **NEVER** modify Keycloak configuration without explicit user permission

The Keycloak instance contains:
- Custom realm configuration
- User accounts with specific roles
- **Group restrictions** that the frontend depends on
- Business unit federation settings

If Keycloak needs changes, ASK THE USER FIRST.

### Docker Volumes - PRESERVE DATA
- `keycloak-data` - Authentication config (CRITICAL)
- `grafana-data` - Dashboard customizations
- `prometheus-data` - Metrics history

Use `docker compose down` (without -v) to stop services safely.

## Safe Operations
- `docker compose up -d` - Start/restart services (safe)
- `docker compose restart <service>` - Restart single service (safe, except keycloak)
- `docker compose logs` - View logs (safe)
- `docker compose ps` - Check status (safe)

## Project Structure
- Backend: Go 1.23, WebSocket server
- Frontend: React 18, TypeScript, Vite
- AgentSim: Go agent simulator
- Auth: Keycloak with OIDC
- Monitoring: Prometheus + Grafana

## Component Rules
Detailed coding rules and guidelines for each component:
- [Backend rules](.claude/BACKEND.md)
- [Frontend rules](.claude/FRONTEND.md)
- [AgentSim rules](.claude/AGENTSIM.md)
- [AgentSim API reference](.claude/AGENTSIM-API.md)
