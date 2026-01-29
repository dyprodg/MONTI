# MONTI v2 - Realistic Call Center Simulation Roadmap

## Overview

Transform the alpha simulation into a realistic call center with actual call routing, virtual queues, DynamoDB persistence, and a restructured frontend. Implemented in 10 incremental steps, shipped as one release from the `dev` branch.

---

## Step 1: Data Model Foundation

**New files:**
- `Backend/internal/types/call.go` - Call, VirtualQueue, ServiceLevel, CallRecord, AgentDailyStats types
- `Backend/internal/types/vq.go` - VQ constants (4 per department, 16 total)
- `AgentSim/internal/types/call.go` - Mirror call types for AgentSim

**Modify:** `Backend/internal/types/types.go` - Add `CurrentCallID`, `CurrentVQ`, `ACWStartTime`, `BreakStartTime` to AgentInfo

**Key types:**
- `Call` - ID, VQ, department, status (queued/ringing/active/completed/abandoned), agent, timestamps, durations
- `VirtualQueue` - ID, name, department, waitingCalls, activeCalls, availableAgents, serviceLevel, longestWait
- `ServiceLevel` - target %, target seconds, current %, calls offered, calls answered in SL, abandoned, avg wait/handle
- `CallRecord` - DynamoDB record for completed calls
- `AgentDailyStats` - DynamoDB record for agent daily history

**VQ names (4 per department):**
- Sales: General, Enterprise, Upsell, Renewal
- Support: Tier 1, Tier 2, Billing, Returns
- Technical: Network, Software, Hardware, Security
- Retention: Cancel, Winback, Loyalty, Escalation

**Call ID format:** `CALL-{timestamp}-{counter}` (e.g., `CALL-20260129-000042`)

---

## Step 2: Call Queue Engine

**New files:**
- `Backend/internal/callqueue/manager.go` - CallQueueManager with per-VQ FIFO queues
- `Backend/internal/callqueue/routing.go` - Agent selection (longest-idle-first per department)

**Core logic:**
- `EnqueueCall(vqID, callerNumber, priority)` - Creates call, adds to VQ waiting queue
- `TickRouting()` - Called every 1s by Aggregator. For each VQ: match waiting calls to available agents (longest-idle first in the VQ's department)
- `CompleteCall(callID)` - Marks call done, updates service level stats
- `GetVQSnapshot()` - Returns current state of all 16 VQs for broadcasting
- Service Level: `SL = (CallsAnsweredInSL / CallsOffered) * 100`, target 80/20 (80% answered within 20s)

**Modify:**
- `Backend/cmd/server/main.go` - Instantiate CallQueueManager
- `Backend/internal/aggregator/aggregator.go` - Call `TickRouting()` in 1s loop, include VQ data in broadcast
- `Backend/internal/cache/agent_state.go` - Add `SetAgentState()`, `GetAvailableByDepartment()`

---

## Step 3: DynamoDB Integration

**New files:**
- `Backend/internal/dynamo/client.go` - DynamoDB client (WriteCallRecord, UpdateAgentDailyStats, GetAgentHistory, GetCallsByDate)
- `Backend/internal/dynamo/config.go` - DynamoDB config (region, endpoint, tables, local mode)
- `scripts/create-dynamo-tables.sh` - Table creation script

**DynamoDB tables:**
- `monti-call-records` - PK: `Date` (YYYY-MM-DD), SK: `CallID`
- `monti-agent-daily-stats` - PK: `AgentID`, SK: `Date` (YYYY-MM-DD)

**Both local and AWS supported** via config flag (`DYNAMO_USE_LOCAL=true` for dev, `false` for AWS).

**Modify:**
- `Backend/go.mod` - Add `aws-sdk-go-v2`
- `Backend/cmd/server/main.go` - Create DynamoClient, pass to CallQueueManager
- `Backend/internal/config/config.go` - Add DynamoDB config fields (region, endpoint, useLocal)
- `docker-compose.yml` - Add `dynamodb-local` service (amazon/dynamodb-local)

---

## Step 4: AgentSim Upgrade - Call-Driven Simulation

**Architecture change:** Agents no longer self-transition to `on_call`. Instead:
1. CallGenerator creates calls at configurable rate, sends `incoming_call` to Backend
2. Backend CallQueueManager queues calls in VQs, routes to available agents
3. Backend sends `call_assign` message to agent via WebSocket
4. Agent receives call, simulates talk (3-30 min), ACW (30s-4min), then available

**New files:**
- `AgentSim/internal/callgen/generator.go` - CallGenerator (configurable calls/min, VQ distribution)

**New message types:**
- `incoming_call` (AgentSim -> Backend) - New call for a VQ
- `call_assign` (Backend -> AgentSim) - Route call to specific agent
- `call_complete` (AgentSim -> Backend) - Agent finished handling call

**Modify:**
- `AgentSim/internal/agent/simulator.go` - Rewrite state machine: available waits for `call_assign`, on_call runs 3-30 min, ACW runs 30s-4min, break limited to ~5% of agents, max 10 min
- `AgentSim/internal/agent/agent_connection.go` - Parse incoming `call_assign` messages, deliver to agent goroutine via channel
- `AgentSim/internal/types/messages.go` - Add CallAssign, CallComplete, IncomingCall
- `AgentSim/internal/types/types.go` - Add CallChannel to Agent
- `AgentSim/cmd/agentsim/main.go` - Instantiate CallGenerator
- `Backend/internal/websocket/agent_hub.go` - Handle `incoming_call` and `call_complete` messages, send `call_assign`

---

## Step 5: Backend WebSocket - Broadcast VQ Data

**Modify:** `Backend/internal/types/types.go` - Add `VQWidget` type (type, department, timestamp, queues[], totalWaiting, totalActive)

**Modify:** `Backend/internal/aggregator/aggregator.go` - Create VQ widgets (one per department) in the 1s tick loop, broadcast alongside agent widgets

**Modify:** `Backend/internal/websocket/hub.go` - Handle VQWidget broadcast type

---

## Step 6: Frontend - VQ Display Component

**New files:**
- `Frontend/src/components/VQDisplay.tsx` - Shows 4 VQ cards per department (waiting calls, SL %, longest wait, active calls, available agents). Color coding: green/yellow/red.

**Modify:**
- `Frontend/src/types/index.ts` - Add VirtualQueue, ServiceLevel, VQWidget, AgentDailyStats types; extend AgentInfo
- `Frontend/src/pages/Dashboard.tsx` - Handle `vq_overview` messages, store in `vqWidgets` state

---

## Step 7: Frontend Layout - 2x4x2 Grid

**"All Departments" view** restructured to 4 rows x 3 columns (agents left, VQs center, agents right):

```
+---------------------+---------------+---------------------+
| Sales Agents (L)    | Sales VQs     | Sales Agents (R)    |
+---------------------+---------------+---------------------+
| Support Agents (L)  | Support VQs   | Support Agents (R)  |
+---------------------+---------------+---------------------+
| Technical Agents(L) | Tech VQs      | Technical Agents(R) |
+---------------------+---------------+---------------------+
| Retention Ag. (L)   | Retention VQs | Retention Ag. (R)   |
+---------------------+---------------+---------------------+
```

- Left/right: agent tables (split 50/50 alphabetically)
- Center: VQDisplay showing 4 VQ cards stacked per department

**Single Department view:** Full-width layout with KPI sidebar (left), agent table (center), and VQ panel (right side or below agents). VQ panel shows the 4 VQs for that department with expanded detail (SL chart, wait times, call counts).

**Modify:**
- `Frontend/src/pages/Dashboard.tsx` - CSS grid: `grid-template-columns: 1fr auto 1fr`, 4 rows
- `Frontend/src/components/WidgetDisplay.tsx` - Adapt for half-width panels

---

## Step 8: Agent History in Popup

**New files:**
- `Backend/internal/api/agent_history.go` - REST endpoint `GET /api/agents/{agentId}/history?days=7`
- `Frontend/src/services/api.ts` - `fetchAgentHistory()` function

**Modify:**
- `Backend/cmd/server/main.go` - Add `/api/agents/{agentId}/history` route
- `Frontend/src/components/AgentModal.tsx` - Add "Daily History" section showing table: Date, Calls, Talk Time, ACW Time, Hold Time, Break Time, Occupancy, ACW Breaches, Break Breaches

---

## Step 9: Simulation Control API

**New AgentSim endpoints (port 8081):**
- `GET /calls/config` - Current call generation config
- `PUT /calls/config` - Update calls/min, VQ distribution, priority weights
- `POST /calls/inject` - Inject a single call into a specific VQ
- `GET /calls/stats` - Call generation statistics

**New Backend endpoints (internal):**
- `POST /internal/calls/inject` - Direct call injection
- `GET /internal/calls/stats` - Queue stats

**Modify:**
- `AgentSim/internal/control/api.go` - Add call control routes
- `Backend/cmd/server/main.go` - Add internal call endpoints

---

## Step 10: Alerting and Polish

**Red alerts:**
- ACW > 5 minutes: agent row turns red in WidgetDisplay
- Break > 10 minutes: agent row turns red in WidgetDisplay
- Both tracked via `acwStartTime` / `breakStartTime` on AgentInfo

**Break limits:**
- Cap simultaneous breaks to ~5% of agents per department in AgentSim

**Modify:**
- `Frontend/src/components/WidgetDisplay.tsx` - Red background/text for breach agents
- `Frontend/src/components/VQDisplay.tsx` - SL color coding (green >= 80%, yellow 60-80%, red < 60%)
- `Backend/internal/cache/agent_state.go` - Track ACW/break start times
- `AgentSim/internal/agent/simulator.go` - Cap break agents per department

---

## Dependency Graph

```
Step 1 (types)
  |
  +---> Step 2 (queue engine)
  |       |
  |       +---> Step 3 (DynamoDB) ---------> Step 8 (agent history popup)
  |       |
  |       +---> Step 4 (AgentSim upgrade) -> Step 9 (control API)
  |       |
  |       +---> Step 5 (broadcast VQ) -----> Step 6 (VQ component) -> Step 7 (layout)
  |
  +---> Step 10 (alerting) -- after Steps 4, 6, 7
```

---

## Verification Plan

1. **Unit tests:** Call queue routing, service level calculation, DynamoDB read/write
2. **Integration test:** Start Backend + AgentSim + DynamoDB local, verify calls flow through VQs to agents
3. **Frontend test:** Verify 2x4x2 layout renders, VQ cards update, agent click shows history, red alerts fire at correct thresholds
4. **Control API test:** Inject calls via API, verify they appear in VQs, adjust calls/min and verify rate changes
5. **End-to-end:** `docker compose up -d`, start simulation via `/start`, observe realistic call flow in browser, check DynamoDB for call records
