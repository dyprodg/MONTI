# MONTI - Testing Documentation

## Test Coverage Summary

### Backend (Go) - ✅ 19 Tests Passing

#### Config Tests (`internal/config/config_test.go`)
- ✅ Load with default values
- ✅ Load with custom values
- ✅ Handle invalid WS_READ_TIMEOUT
- ✅ Handle invalid WS_WRITE_TIMEOUT
- ✅ WebSocket constants calculation

#### WebSocket Hub Tests (`internal/websocket/hub_test.go`)
- ✅ Hub initialization
- ✅ Client count tracking
- ✅ Broadcast functionality
- ✅ Client registration
- ✅ Client unregistration
- ✅ Broadcast to multiple clients

#### Ticker Tests (`internal/ticker/ticker_test.go`)
- ✅ Ticker initialization
- ✅ Ticker starts and stops with context
- ✅ Broadcasts time messages
- ✅ TimeMessage JSON serialization
- ✅ Stops on context cancellation

#### Middleware Tests (`pkg/middleware/*_test.go`)
- ✅ CORS with allowed origins
- ✅ CORS with disallowed origins
- ✅ CORS preflight requests
- ✅ Logger captures request details
- ✅ Logger captures error status codes

#### Handler Tests (`cmd/server/main_test.go`)
- ✅ Health endpoint returns 200
- ✅ Health endpoint returns correct JSON

### Frontend (React + TypeScript) - ✅ 17 Tests Passing

#### WebSocket Service Tests (`services/websocket.test.ts`)
- ✅ Initialize with CLOSED state
- ✅ Transition to CONNECTING on connect
- ✅ Transition to OPEN after connection
- ✅ Handle incoming messages
- ✅ Clean up when disconnected
- ✅ Notify error handlers on error
- ✅ Unsubscribe handlers correctly

#### ConnectionStatus Component Tests (`components/ConnectionStatus.test.tsx`)
- ✅ Render "Connected" when OPEN
- ✅ Render "Connecting..." when CONNECTING
- ✅ Render "Error" when ERROR
- ✅ Render "Disconnected" when CLOSED
- ✅ Show green indicator for OPEN state
- ✅ Show red indicator for ERROR state

#### TimeDisplay Component Tests (`components/TimeDisplay.test.tsx`)
- ✅ Render "Waiting for data..." when no data
- ✅ Render time data when provided
- ✅ Format timestamp correctly
- ✅ Handle invalid timestamp gracefully

---

## Running Tests

### Backend Tests

```bash
# Run all tests
cd Backend
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/websocket
go test ./internal/ticker
go test ./internal/config
```

### Frontend Tests

```bash
# Run all tests
cd Frontend
npm test

# Run tests once (CI mode)
npm test -- --run

# Run tests with UI
npm run test:ui

# Run specific test file
npm test -- websocket.test.ts
```

---

## Test Structure

### Backend Test Organization

```
Backend/
├── cmd/server/
│   └── main_test.go           # HTTP handler tests
├── internal/
│   ├── config/
│   │   └── config_test.go     # Configuration tests
│   ├── websocket/
│   │   └── hub_test.go        # WebSocket hub tests
│   └── ticker/
│       └── ticker_test.go     # Ticker service tests
└── pkg/middleware/
    ├── cors_test.go           # CORS middleware tests
    └── logger_test.go         # Logger middleware tests
```

### Frontend Test Organization

```
Frontend/
├── src/
│   ├── services/
│   │   └── websocket.test.ts      # WebSocket service tests
│   ├── components/
│   │   ├── ConnectionStatus.test.tsx  # Component tests
│   │   └── TimeDisplay.test.tsx       # Component tests
│   └── test/
│       └── setup.ts                   # Test configuration & mocks
```

---

## Testing Philosophy

### Backend (Go)
- **Unit Tests**: Test individual functions and methods in isolation
- **Integration Tests**: Test component interactions (e.g., hub + clients)
- **Table-Driven Tests**: Use Go's table-driven test pattern for multiple scenarios
- **Coverage Target**: 80% minimum

### Frontend (React + TypeScript)
- **Unit Tests**: Test services and utilities in isolation
- **Component Tests**: Test React components with @testing-library/react
- **Mock WebSocket**: Use mock WebSocket for testing real-time features
- **User-Centric**: Test what users see and interact with
- **Coverage Target**: 70% minimum

---

## CI/CD Integration

### GitHub Actions

Both Backend and Frontend tests run automatically in CI:

**Backend CI** (`.github/workflows/backend.yml`):
```yaml
- name: Run tests
  run: go test -v -race -coverprofile=coverage.out ./...

- name: Check coverage
  run: |
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
    if (( $(echo "$COVERAGE < 80" | bc -l) )); then
      echo "Coverage below 80%"
      exit 1
    fi
```

**Frontend CI** (`.github/workflows/frontend.yml`):
```yaml
- name: Run tests
  run: npm run test -- --run

- name: Type check
  run: npm run type-check
```

---

## Adding New Tests

### Backend (Go)

1. Create `*_test.go` file next to the code being tested
2. Import `testing` package
3. Write test functions starting with `Test`
4. Use table-driven tests for multiple scenarios

Example:
```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name string
        input string
        want string
    }{
        {"case 1", "input1", "output1"},
        {"case 2", "input2", "output2"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := MyFunction(tt.input)
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Frontend (TypeScript)

1. Create `*.test.ts` or `*.test.tsx` file next to the code
2. Import `describe`, `it`, `expect` from vitest
3. Use `@testing-library/react` for component tests

Example:
```typescript
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MyComponent } from './MyComponent'

describe('MyComponent', () => {
  it('should render text', () => {
    render(<MyComponent />)
    expect(screen.getByText('Hello')).toBeInTheDocument()
  })
})
```

---

## Test Quality Guidelines

### Do's ✅
- Test behavior, not implementation
- Use descriptive test names
- Keep tests simple and focused
- Mock external dependencies
- Test edge cases and error conditions
- Write tests before or alongside code (TDD)

### Don'ts ❌
- Don't test third-party libraries
- Don't test private implementation details
- Don't write flaky tests
- Don't duplicate tests
- Don't skip tests without good reason

---

## Test Results

### Current Status

| Component | Tests | Status |
|-----------|-------|--------|
| Backend Config | 5 | ✅ Passing |
| Backend WebSocket | 5 | ✅ Passing |
| Backend Ticker | 5 | ✅ Passing |
| Backend Middleware | 4 | ✅ Passing |
| Frontend WebSocket | 7 | ✅ Passing |
| Frontend Components | 10 | ✅ Passing |
| **TOTAL** | **36** | **✅ All Passing** |

---

## Next Testing Steps

When adding new features, ensure you add tests for:

1. **Redis Integration**
   - Cache read/write operations
   - Connection handling
   - Error scenarios

2. **PostgreSQL Integration**
   - Database queries
   - Migrations
   - Transaction handling

3. **Authentication**
   - JWT validation
   - Token refresh
   - Access control

4. **Agent Data**
   - Data aggregation
   - Grouping logic
   - Performance with 2000+ agents

5. **Frontend Features**
   - Filtering components
   - Search functionality
   - Data visualization

---

## Troubleshooting Tests

### Backend Tests Failing

```bash
# Clear test cache
go clean -testcache

# Run with race detector
go test -race ./...

# Run specific test
go test -run TestSpecificFunction ./internal/package
```

### Frontend Tests Failing

```bash
# Clear node modules and reinstall
rm -rf node_modules
npm install

# Update snapshots (if using)
npm test -- -u

# Run in watch mode for debugging
npm test
```

---

## Resources

- Go Testing: https://golang.org/pkg/testing/
- Vitest: https://vitest.dev/
- React Testing Library: https://testing-library.com/react
- Testing Best Practices: https://kentcdodds.com/blog/common-mistakes-with-react-testing-library







 Here are the commands to start the entire system:

  Quick Start Commands

  1. Start Backend & Frontend (Docker)

  cd /Users/dennisdiepolder/Documents/Dev/MONTI
  docker compose up -d backend frontend

  2. Start AgentSim (Standalone)

  cd /Users/dennisdiepolder/Documents/Dev/MONTI/AgentSim
  ./bin/agentsim --agents 50 --auto-start --active 30 --backend-url http://localhost:8080 --log-level info

  Or run AgentSim in the background:
  cd /Users/dennisdiepolder/Documents/Dev/MONTI/AgentSim
  nohup ./bin/agentsim --agents 50 --auto-start --active 30 --backend-url http://localhost:8080 --log-level info > agentsim.log 2>&1 &

  One-Liner to Start Everything

  cd /Users/dennisdiepolder/Documents/Dev/MONTI && \
  docker compose up -d backend frontend && \
  cd AgentSim && \
  nohup ./bin/agentsim --agents 50 --auto-start --active 30 --backend-url http://localhost:8080 --log-level info > agentsim.log 2>&1 &

  Access Points

  - Dashboard: http://localhost:5173/
  - Backend: http://localhost:8080/health
  - AgentSim Control: http://localhost:8081/stats

  Stop Everything

  # Stop Docker services
  cd /Users/dennisdiepolder/Documents/Dev/MONTI
  docker compose down

  # Stop AgentSim (if running in background)
  pkill -f agentsim

  Check Status

  # Check Docker services
  docker ps | grep monti

  # Check AgentSim
  curl http://localhost:8081/stats

  # Check Backend events
  curl http://localhost:8080/internal/event/stats

  That's it! Run the one-liner and everything will be up and running. 🚀