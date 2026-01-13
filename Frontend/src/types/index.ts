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

// Agent Event
export interface AgentEvent {
  agentId: string
  state: AgentState
  department: Department
  location: Location
  team: string
  timestamp: string
  stateDuration: number
}

// Widget Summary
export interface WidgetSummary {
  totalEvents: number
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
  events: AgentEvent[]
}
