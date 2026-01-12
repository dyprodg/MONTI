# MONTI - Quick Start Guide

## Prerequisites

- Docker & Docker Compose installed
- Git (for cloning)

## Running the Application

### 1. Start all services

```bash
docker compose up
```

This will start:
- **Backend** (Go WebSocket server) on http://localhost:8080
- **Frontend** (React + Vite) on http://localhost:5173

### 2. Access the application

Open your browser to: **http://localhost:5173**

You should see:
- Connection status indicator (should show "Connected" in green)
- Server time updating every second
- Live WebSocket updates

### 3. Stop the services

```bash
docker compose down
```

## Development

### Hot Reload

Both services support hot reload:

- **Backend**: Air watches for Go file changes and rebuilds automatically
- **Frontend**: Vite HMR provides instant updates

Just edit files and see changes immediately!

### Run tests

**Backend:**
```bash
cd Backend
go test ./...
```

**Frontend:**
```bash
cd Frontend
npm test
```

### Build for production

**Backend:**
```bash
cd Backend
docker build -t monti-backend:prod -f Dockerfile .
```

**Frontend:**
```bash
cd Frontend
npm run build
```

## Project Structure

```
MONTI/
├── Backend/               # Go WebSocket server
│   ├── cmd/server/       # Application entry point
│   ├── internal/         # Private packages
│   │   ├── websocket/    # WebSocket hub & clients
│   │   ├── ticker/       # Time broadcaster
│   │   └── config/       # Configuration
│   ├── pkg/middleware/   # Reusable middleware
│   └── Dockerfile.dev    # Development Docker image
│
├── Frontend/             # React + TypeScript + Vite
│   ├── src/
│   │   ├── components/   # UI components
│   │   ├── hooks/        # Custom React hooks
│   │   ├── services/     # WebSocket service
│   │   └── types/        # TypeScript definitions
│   └── Dockerfile        # Development Docker image
│
└── docker-compose.yml    # Orchestration
```

## Architecture

```
┌─────────────────────────────────────┐
│  Frontend (React + Vite)            │
│  - WebSocket client                 │
│  - Real-time UI updates             │
│  - Connection management            │
└──────────────┬──────────────────────┘
               │ WebSocket
               ↓
┌─────────────────────────────────────┐
│  Backend (Go)                       │
│  - WebSocket Hub                    │
│  - Ticker Service (1s interval)    │
│  - Health endpoint                  │
└─────────────────────────────────────┘
```

## API Endpoints

### HTTP
- `GET /health` - Health check endpoint

### WebSocket
- `ws://localhost:8080/ws` - WebSocket connection endpoint

Message format:
```json
{
  "timestamp": "2026-01-12T10:00:00Z",
  "serverTime": 1736676000
}
```

## Next Steps

1. Replace time ticker with real agent data
2. Add Redis for caching
3. Add PostgreSQL for persistence
4. Implement AWS IAM Identity Center authentication
5. Add filtering and grouping UI

## Troubleshooting

### Backend not connecting
- Check if port 8080 is available
- View logs: `docker compose logs backend`

### Frontend not loading
- Check if port 5173 is available
- View logs: `docker compose logs frontend`

### WebSocket connection fails
- Ensure CORS is configured: check `ALLOWED_ORIGINS` in docker-compose.yml
- Check browser console for errors

## License

MIT
