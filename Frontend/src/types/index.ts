// WebSocket connection states
export enum ConnectionState {
  CONNECTING = 'connecting',
  OPEN = 'open',
  CLOSING = 'closing',
  CLOSED = 'closed',
  ERROR = 'error',
}

// Time message from backend
export interface TimeMessage {
  timestamp: string
  serverTime: number
}

// WebSocket error
export interface WebSocketError {
  message: string
  code?: number
}

// Agent states
export type AgentState =
  | 'available'
  | 'busy'
  | 'on_call'
  | 'break'
  | 'offline'
  | 'after_call_work'
  | 'training'
  | 'meeting'
  | 'lunch'
  | 'on_hold'
  | 'transferring'
  | 'conference'

// Department
export type Department = 'sales' | 'support' | 'technical' | 'retention'

// Location
export type Location = 'berlin' | 'munich' | 'hamburg' | 'frankfurt' | 'remote'

// Agent KPIs - performance metrics
export interface AgentKPIs {
  totalCalls: number
  avgCallDuration: number      // seconds
  acwTime: number              // seconds
  acwCount: number
  holdCount: number
  holdTime: number             // seconds
  transferCount: number
  conferenceCount: number
  breakTime: number            // seconds
  loginTime: number            // seconds since login
  occupancy: number            // 0-100%
  adherence: number            // 0-100%
  avgHandleTime: number        // seconds
  firstCallResolution: number  // 0-100%
  customerSatisfaction: number // 1-5
}

// Agent Event
export interface AgentEvent {
  agentId: string
  state: AgentState
  department: Department
  location: Location
  team: string
  timestamp: string
  stateDuration: number
  kpis: AgentKPIs
}

// Agent Info - current state of an agent
export interface AgentInfo {
  agentId: string
  state: AgentState
  department: Department
  location: Location
  team: string
  stateStart: string    // when current state started
  lastUpdate: string    // last event received
  kpis: AgentKPIs
}

// Widget Summary
export interface WidgetSummary {
  totalAgents: number   // Total number of agents
  totalEvents?: number  // deprecated
  stateBreakdown: Record<AgentState, number>
  departmentBreakdown?: Record<Department, number>
  locationBreakdown?: Record<Location, number>
}

// Widget
export interface Widget {
  type: 'global_overview' | 'department_overview'
  department?: Department
  timestamp: string
  summary: WidgetSummary
  events?: AgentEvent[]  // deprecated
  agents?: AgentInfo[]   // All agents in this widget
}
