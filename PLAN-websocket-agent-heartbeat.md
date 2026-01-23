# Plan: WebSocket Agent Connections with Heartbeat

## Overview

Replace HTTP POST events with persistent WebSocket connections from each simulated agent to the backend. Each of the 2000 agents maintains its own WebSocket connection and sends heartbeats every 2 seconds.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        AgentSim                              │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐       ┌─────────┐     │
│  │ Agent 1 │ │ Agent 2 │ │ Agent 3 │  ...  │Agent 2k │     │
│  │   WS    │ │   WS    │ │   WS    │       │   WS    │     │
│  └────┬────┘ └────┬────┘ └────┬────┘       └────┬────┘     │
└───────┼──────────┼──────────┼────────────────┼─────────────┘
        │          │          │                │
        └──────────┴──────────┴────────────────┘
                          │
                    WebSocket connections (2000)
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                        Backend                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Agent WebSocket Hub                      │   │
│  │  - Registers agent connections                        │   │
│  │  - Receives heartbeats                                │   │
│  │  - Detects disconnects instantly                      │   │
│  └──────────────────────┬───────────────────────────────┘   │
│                         │                                    │
│                         ▼                                    │
│  ┌──────────────────────────────────────────────────────┐   │
│  │           AgentStateTracker (with TTL)                │   │
│  │  - Stores current state of all agents                 │   │
│  │  - Marks agents "stale" after 6s no heartbeat         │   │
│  │  - Marks agents "disconnected" on WS close            │   │
│  └──────────────────────┬───────────────────────────────┘   │
│                         │                                    │
│                         ▼ (every 1 second)                   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                   Aggregator                          │   │
│  │  - Creates widgets from current state                 │   │
│  │  - Includes stale/disconnected status                 │   │
│  └──────────────────────┬───────────────────────────────┘   │
│                         │                                    │
│                         ▼                                    │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Frontend WebSocket Hub                   │   │
│  │  - Broadcasts widgets to dashboard clients            │   │
│  │  - RBAC filtering per client                          │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Heartbeat Protocol

### Message Types

```go
// Agent → Backend
type AgentHeartbeat struct {
    Type      string          `json:"type"`      // "heartbeat"
    AgentID   string          `json:"agentId"`
    State     AgentState      `json:"state"`
    Timestamp time.Time       `json:"timestamp"`
    KPIs      AgentKPIs       `json:"kpis"`
}

// Agent → Backend (on state change)
type AgentStateChange struct {
    Type          string      `json:"type"`      // "state_change"
    AgentID       string      `json:"agentId"`
    PreviousState AgentState  `json:"previousState"`
    NewState      AgentState  `json:"newState"`
    Timestamp     time.Time   `json:"timestamp"`
    StateDuration float64     `json:"stateDuration"`
    KPIs          AgentKPIs   `json:"kpis"`
}

// Backend → Agent (optional acknowledgment)
type ServerAck struct {
    Type    string `json:"type"`    // "ack"
    AgentID string `json:"agentId"`
}
```

### Timing

| Event | Interval |
|-------|----------|
| Heartbeat | Every 2 seconds |
| State change | Immediate |
| Stale detection | 6 seconds (3 missed heartbeats) |
| Disconnect detection | Instant (WebSocket close) |

## Files to Modify/Create

### Backend

| File | Changes |
|------|---------|
| `internal/websocket/agent_hub.go` | **NEW** - Hub for agent WebSocket connections |
| `internal/websocket/agent_client.go` | **NEW** - Agent WebSocket client handler |
| `internal/websocket/agent_handler.go` | **NEW** - HTTP upgrade handler for agent connections |
| `internal/cache/agent_state.go` | Add TTL tracking, stale detection, connection status |
| `internal/types/types.go` | Add AgentConnectionStatus, heartbeat message types |
| `internal/aggregator/aggregator.go` | Include connection status in widgets |
| `cmd/server/main.go` | Add `/ws/agent` endpoint, initialize agent hub |

### AgentSim

| File | Changes |
|------|---------|
| `internal/agent/simulator.go` | Replace HTTP client with WebSocket connections |
| `internal/agent/agent_connection.go` | **NEW** - WebSocket connection per agent |
| `internal/types/messages.go` | **NEW** - Message types for WebSocket protocol |
| `cmd/agentsim/main.go` | Update initialization |

### Configuration

| File | Changes |
|------|---------|
| `docker-compose.yml` | Update environment variables if needed |

## Implementation Details

### 1. Backend: Agent WebSocket Hub (`internal/websocket/agent_hub.go`)

```go
type AgentHub struct {
    agents     map[string]*AgentClient  // agentID → client
    register   chan *AgentClient
    unregister chan *AgentClient
    heartbeat  chan *AgentHeartbeat
    mu         sync.RWMutex
    logger     zerolog.Logger
    tracker    *cache.AgentStateTracker
}

func (h *AgentHub) Run() {
    for {
        select {
        case client := <-h.register:
            h.agents[client.agentID] = client
            h.tracker.SetConnected(client.agentID, true)

        case client := <-h.unregister:
            delete(h.agents, client.agentID)
            h.tracker.SetConnected(client.agentID, false)

        case hb := <-h.heartbeat:
            h.tracker.UpdateFromHeartbeat(hb)
        }
    }
}
```

### 2. Backend: Agent State Tracker Updates (`internal/cache/agent_state.go`)

```go
type AgentInfo struct {
    // ... existing fields ...

    // New fields
    LastHeartbeat    time.Time              `json:"lastHeartbeat"`
    ConnectionStatus AgentConnectionStatus  `json:"connectionStatus"`
}

type AgentConnectionStatus string

const (
    StatusConnected    AgentConnectionStatus = "connected"
    StatusDisconnected AgentConnectionStatus = "disconnected"
    StatusStale        AgentConnectionStatus = "stale"      // no heartbeat > 6s
)

func (t *AgentStateTracker) UpdateFromHeartbeat(hb *AgentHeartbeat) {
    t.mu.Lock()
    defer t.mu.Unlock()

    agent := t.agents[hb.AgentID]
    agent.State = hb.State
    agent.KPIs = hb.KPIs
    agent.LastHeartbeat = time.Now()
    agent.LastUpdate = time.Now()
    agent.ConnectionStatus = StatusConnected
}

func (t *AgentStateTracker) CheckStaleAgents() {
    t.mu.Lock()
    defer t.mu.Unlock()

    threshold := time.Now().Add(-6 * time.Second)
    for id, agent := range t.agents {
        if agent.ConnectionStatus == StatusConnected &&
           agent.LastHeartbeat.Before(threshold) {
            t.agents[id].ConnectionStatus = StatusStale
        }
    }
}
```

### 3. AgentSim: WebSocket Connection per Agent (`internal/agent/agent_connection.go`)

```go
type AgentConnection struct {
    agent      *types.Agent
    conn       *websocket.Conn
    send       chan []byte
    done       chan struct{}
    logger     zerolog.Logger
    backendURL string
}

func (ac *AgentConnection) Run(ctx context.Context) {
    // Connect to backend
    conn, _, err := websocket.DefaultDialer.Dial(ac.backendURL+"/ws/agent", nil)
    if err != nil {
        ac.logger.Error().Err(err).Msg("failed to connect")
        return
    }
    ac.conn = conn
    defer conn.Close()

    // Start heartbeat ticker
    heartbeatTicker := time.NewTicker(2 * time.Second)
    defer heartbeatTicker.Stop()

    for {
        select {
        case <-ctx.Done():
            return

        case <-heartbeatTicker.C:
            ac.sendHeartbeat()

        case msg := <-ac.send:
            ac.conn.WriteMessage(websocket.TextMessage, msg)
        }
    }
}

func (ac *AgentConnection) sendHeartbeat() {
    hb := AgentHeartbeat{
        Type:      "heartbeat",
        AgentID:   ac.agent.ID,
        State:     ac.agent.State,
        Timestamp: time.Now(),
        KPIs:      ac.agent.KPIs,
    }
    data, _ := json.Marshal(hb)
    ac.conn.WriteMessage(websocket.TextMessage, data)
}

func (ac *AgentConnection) sendStateChange(prev, new AgentState, duration float64) {
    msg := AgentStateChange{
        Type:          "state_change",
        AgentID:       ac.agent.ID,
        PreviousState: prev,
        NewState:      new,
        Timestamp:     time.Now(),
        StateDuration: duration,
        KPIs:          ac.agent.KPIs,
    }
    data, _ := json.Marshal(msg)
    ac.conn.WriteMessage(websocket.TextMessage, data)
}
```

### 4. AgentSim: Updated Simulator (`internal/agent/simulator.go`)

```go
type Simulator struct {
    agents       []types.Agent
    connections  map[string]*AgentConnection  // agentID → connection
    // ... rest of fields
}

func (s *Simulator) activateAgent(agent *types.Agent) {
    // Create WebSocket connection for this agent
    conn := NewAgentConnection(agent, s.backendURL, s.logger)
    s.connections[agent.ID] = conn

    // Start connection in goroutine
    go conn.Run(s.ctx)

    // Start state machine in goroutine
    go s.simulateAgent(s.ctx, agent.ID)
}

func (s *Simulator) simulateAgent(ctx context.Context, agentID string) {
    conn := s.connections[agentID]

    for {
        select {
        case <-ctx.Done():
            return
        default:
            agent := s.getAgent(agentID)
            duration := s.getStateDuration(agent.State)
            time.Sleep(duration)

            prevState := agent.State
            nextState := s.getNextState(agent.State)
            s.updateAgentState(agentID, nextState)

            // Send state change over WebSocket
            conn.sendStateChange(prevState, nextState, duration.Seconds())
        }
    }
}
```

## Resource Estimates

| Resource | Estimate |
|----------|----------|
| WebSocket connections | 2000 |
| Memory per connection | ~10-20 KB |
| Total connection memory | ~40 MB |
| Heartbeat messages/sec | 1000 (2000 agents / 2 sec) |
| State change messages/sec | ~100-150 (same as before) |
| Total messages/sec | ~1100-1150 |

## Metrics to Add

```
# Backend
monti_agent_connections_active      - Current WebSocket connections
monti_agent_connections_total       - Total connections since start
monti_agent_disconnections_total    - Total disconnections
monti_agent_heartbeats_total        - Heartbeats received
monti_agent_stale_total             - Agents marked stale
monti_agent_state_changes_total     - State change messages received

# AgentSim
agentsim_websocket_connections      - Active connections
agentsim_websocket_reconnects       - Reconnection attempts
agentsim_heartbeats_sent_total      - Heartbeats sent
agentsim_state_changes_sent_total   - State changes sent
```

## Migration Steps

1. **Backend first**: Add `/ws/agent` endpoint alongside existing `/internal/event`
2. **Test with few agents**: Start AgentSim with 10 agents using WebSocket
3. **Verify metrics**: Check heartbeats received, state tracking works
4. **Scale up**: Increase to 100, 500, 1000, 2000 agents
5. **Remove old endpoint**: Deprecate `/internal/event` once stable

## Verification Checklist

- [ ] 2000 WebSocket connections established
- [ ] Heartbeats received every 2 seconds per agent
- [ ] State changes reflected immediately
- [ ] Disconnected agents detected instantly
- [ ] Stale agents detected within 6 seconds
- [ ] Reconnection works after network issues
- [ ] Memory usage stays within limits (~100MB for connections)
- [ ] Grafana dashboard shows connection metrics
- [ ] Frontend receives accurate agent states

## Rollback Plan

Keep the existing HTTP POST `/internal/event` endpoint functional during migration. If WebSocket approach has issues:

1. AgentSim falls back to HTTP POST
2. Backend continues accepting both
3. Investigate and fix WebSocket issues
4. Retry migration
