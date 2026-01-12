# Backend

Go-based backend service providing API and WebSocket connectivity for the MONTI call center monitoring app.

## Purpose

Core backend service that:
- Manages agent data in database and cache
- Provides REST/GraphQL API endpoints
- Handles WebSocket connections for real-time updates
- Implements authentication and authorization
- Aggregates and groups agent data for efficient transmission

## Rules

### Code Organization
- Follow standard Go project layout
- Separate packages for: `api`, `websocket`, `cache`, `database`, `auth`, `middleware`
- Keep handlers thin, business logic in service layer
- Use dependency injection for testability

### API Design
- RESTful endpoints for CRUD operations
- GraphQL optional for complex queries
- Version all APIs (e.g., `/api/v1/`)
- Use standard HTTP status codes
- Return consistent error responses

### WebSocket
- Implement efficient message batching
- Send grouped/aggregated data, not individual agents
- Handle connection lifecycle (connect, disconnect, reconnect)
- Implement heartbeat/ping-pong for connection health
- Scale to support many concurrent connections

### Caching Strategy
- Use Redis or in-memory cache for hot data
- Cache aggregated/grouped data, not raw agent data
- Implement cache invalidation strategy
- TTL should be configurable

### Database
- Use migrations for schema changes
- Support PostgreSQL/MySQL
- Connection pooling configured appropriately
- Use prepared statements to prevent SQL injection
- Index strategy for performance

### Authentication & Authorization
- Integrate with AWS IAM Identity Center (OIDC)
- Validate JWT tokens on all protected endpoints
- Implement tenant-based and location-based access control
- Use middleware for auth checks
- No hardcoded credentials

### Security
- Input validation on all endpoints
- Rate limiting to prevent abuse
- CORS configured appropriately
- No sensitive data in logs
- Follow OWASP security best practices

### Performance
- Must handle 2000+ agents with minimal latency
- WebSocket updates should be < 100ms
- API response time should be < 200ms
- Memory usage should be predictable and monitored
- Load test before production deployment

### Testing
- Unit tests for all business logic (minimum 80% coverage)
- Integration tests for API endpoints
- WebSocket connection tests
- Performance/load tests
- Mock external dependencies

### Logging & Monitoring
- Structured logging (JSON format)
- Log levels: DEBUG, INFO, WARN, ERROR
- Include correlation IDs for tracing
- Metrics exported for Prometheus
- Health check endpoint (`/health`)

### Configuration
- All config via environment variables or config file
- Use `.env` for local development (not committed)
- Document all environment variables
- Sensible defaults for development

## Dependencies

- Can depend on AgentSim for simulation data
- Must not depend on Frontend
- Must not depend on Infra code

## CI/CD

This service will only be built and tested when:
- Files in `Backend/` directory are changed
- Root configuration files affecting all services are changed
- Explicitly triggered via workflow dispatch

### Build Steps
1. Run `go fmt` and `go vet`
2. Run linter (golangci-lint)
3. Run unit tests with coverage
4. Run integration tests
5. Build binary
6. Create Docker image (if applicable)
7. Run security scanning (gosec)

### Deployment
- Deploy to AWS Lambda, ECS, or EC2 based on Infra setup
- Use blue-green or rolling deployment strategy
- Run smoke tests post-deployment
