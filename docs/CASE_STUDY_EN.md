# MONTI — Real-Time Contact Center Monitoring

**Stack:** Go 1.23 / React 18 / WebSocket / Keycloak / AWS / Terraform
**Role:** Solo developer — architecture, backend, frontend, infrastructure, deployment

---

## The Problem

Enterprise contact centers running on Genesys Cloud rely on native monitoring tools that poll APIs on 15–30 second intervals. For a floor of 2,000 agents, that means:

- **Stale data.** Supervisors see agent states that are half a minute old. A "ghost agent" — someone who dropped off due to a WebRTC failure but still shows as available — sits undetected for minutes, tanking queue performance.
- **Blind spots.** Native dashboards show queue-level aggregates but can't surface per-agent anomalies: abnormal after-call-work times, occupancy spikes, agents stuck in limbo states.
- **No filtering.** Multi-site operations need business-unit-scoped views. The built-in tools don't support it without manual workarounds.

Supervisors were flying blind during peak hours. Calls were queuing because the system thought agents were available when they weren't.

---

## What I Built

A real-time monitoring platform that replaces polling with a persistent event-driven pipeline. Every agent state change and heartbeat flows through WebSocket into a Go backend that aggregates, enriches, and broadcasts a full operational snapshot to every connected supervisor — once per second.

### Data Flow

```
Agents (2,000 WebSocket connections)
  → Heartbeats every 2s + state changes on transition
    → Go Backend (AgentHub)
      → In-memory state tracker (RWMutex-protected map)
      → 1-second aggregation loop
        → Snapshot: all agents + 16 virtual queues
          → Per-client RBAC filtering
            → Broadcast to supervisor dashboards (WebSocket Hub)
```

### What Supervisors See

- Live agent grid — color-coded by state, filterable by department/location
- Virtual queue panels — waiting calls, average wait time, service level per skill group
- Alert indicators — occupancy > 80%, ACW > 5 min, idle > 16 min
- Ghost agent detection — agents marked stale after 6 seconds (3 missed heartbeats)

---

## Architectural Decisions and Tradeoffs

### 1. WebSocket everywhere, not SSE or polling

**Decision:** Both agent-to-backend and backend-to-frontend use persistent WebSocket connections.

**Why:** I needed bidirectional communication for the agent connections (heartbeats up, commands down) and sub-second latency for the frontend. SSE would work for the frontend half but would mean maintaining two transport layers. WebSocket everywhere kept the mental model simple.

**Tradeoff:** 2,000+ persistent connections on a single t3.small. Solved with goroutine-per-connection in Go (cheap — ~50KB per goroutine) and buffered channels to prevent backpressure from propagating.

### 2. Single aggregated snapshot vs. differential updates

**Decision:** Every second, the backend builds one complete snapshot (all agents + all queues) and broadcasts it to every client.

**Why:** Differential updates are more bandwidth-efficient but massively more complex — you need reliable ordering, client-side reconciliation, and recovery logic for missed deltas. A full snapshot at ~200KB compressed is well within WebSocket limits, and it makes the frontend trivially simple: replace state, re-render.

**Tradeoff:** More bandwidth per cycle. Acceptable for <50 concurrent supervisors. Would need to revisit at scale with delta compression.

### 3. In-memory state, not a database in the hot path

**Decision:** All agent state lives in a single `map[string]*AgentInfo` behind an `RWMutex`. No Redis, no message queue.

**Why:** The entire dataset fits in ~260MB. Adding a cache layer (Redis) or a message broker (NATS, Kafka) would add latency, operational complexity, and failure modes — all for data that's ephemeral by nature. If the process restarts, agents re-register within seconds.

**Tradeoff:** Single-process, single-node. No horizontal scaling. This is fine for the target deployment (one contact center, one backend instance). If I needed multi-region or >10k agents, I'd shard by department and add NATS for cross-shard aggregation.

### 4. Agent simulation as a first-class component

**Decision:** Built a full agent simulator (AgentSim) as a separate Go service that generates realistic call center traffic — state machines with configurable transition times, call generation with peak-hour modeling, and a control API for scaling.

**Why:** You can't develop or demo a real-time monitoring system without real-time data. AgentSim lets me spin up 1,000 agents with realistic behavior patterns in seconds. It also doubles as a load testing tool — I can verify the backend handles 1,000 heartbeats/second without the aggregation loop exceeding 100ms.

**Tradeoff:** Another service to maintain. Worth it — it made development 10x faster and the demo is always live.

### 5. Keycloak for auth, not a custom JWT solution

**Decision:** Full OIDC flow with Keycloak — Authorization Code + PKCE, JWT validation via JWKS, role-based access, business unit filtering via group claims.

**Why:** Enterprise auth is not something you want to hand-roll. Keycloak gives me user management, group-based access control, and token lifecycle out of the box. Business unit filtering (SGB/NGB/RGB) maps directly to Keycloak groups, so supervisors only see agents in their region.

**Tradeoff:** Keycloak is heavy (~500MB RAM). For a portfolio project, a simpler auth setup might suffice. But this mirrors real enterprise deployments, which was the point.

### 6. Terraform + scheduled EC2 instead of always-on

**Decision:** The production EC2 instance runs on a schedule — EventBridge starts it at 13:45 CET and stops it at 16:05 CET on weekdays. Total cost: ~$20/month.

**Why:** This is a portfolio project, not a revenue-generating SaaS. I wanted production-grade infrastructure (IaC, TLS, CDN, monitoring) without production-grade bills. The schedule window gives a reliable demo window every weekday afternoon.

**Tradeoff:** Not available 24/7. The frontend handles this gracefully — a `ScheduleGate` component checks backend health and shows an offline page outside the window.

---

## Key Results

| Metric | Native Tooling | MONTI |
|--------|---------------|-------|
| Data freshness | 15–30s polling | **1s broadcast cycle** |
| Ghost agent detection | Minutes (manual) | **6 seconds (automatic)** |
| Aggregation latency | N/A | **<50ms for 2,000 agents** |
| Per-agent anomaly alerts | Not available | **Real-time (occupancy, ACW, idle)** |
| Business unit filtering | Manual | **Automatic via OIDC groups** |
| WebSocket throughput | N/A | **1,000 heartbeats/sec sustained** |

---

## What I Learned

**Go's concurrency model is the right tool for this.** Goroutine-per-connection with channel-based coordination made the concurrent architecture straightforward. The same system in Node.js would have been an event-loop scheduling nightmare at this connection count.

**Full snapshots beat clever diffs — until they don't.** For this scale, sending the whole world every second is simpler and more reliable than managing differential state. But I can see exactly where that breaks down (~100 clients or ~10k agents), and I know what the migration path looks like.

**Simulate your data layer early.** Building AgentSim before the frontend paid off enormously. Every dashboard component was developed against realistic data from day one, which surfaced edge cases (agents in transitional states, empty queues, concurrent state transitions) that I never would have caught with mock data.

**Infrastructure as code isn't optional, even for side projects.** The Terraform setup took a day. It's saved me dozens of hours in reproducible deployments, and it demonstrates infrastructure competency that a README alone can't.

---

## Links

- **Live demo:** [monti.dennisdiepolder.com](https://monti.dennisdiepolder.com) *(weekdays 14:00–16:00 CET)*
- **Source:** Available on request

---

*Built by Dennis Diepolder. Questions welcome.*
