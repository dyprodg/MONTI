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

// Alert severity
export type AlertSeverity = 'warning' | 'critical'

// Agent Alert
export interface AgentAlert {
  rule: string
  severity: AlertSeverity
  message: string
}

// Agent connection status
export type ConnectionStatus = 'connected' | 'disconnected' | 'stale'

// Agent Info - current state of an agent
export interface AgentInfo {
  agentId: string
  state: AgentState
  department: Department
  location: Location
  team: string
  stateStart: string    // when current state started
  lastUpdate: string    // last event received
  connectionStatus?: ConnectionStatus
  kpis: AgentKPIs
  currentCallId?: string   // active call ID
  currentVq?: VQName       // VQ of active call
  callStartTime?: string   // when current call started
  acwStartTime?: string    // when ACW started
  breakStartTime?: string  // when break started
  alerts?: AgentAlert[]    // active alerts
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
  queues?: VQSnapshot[]  // VQ snapshots for this department
}

// Virtual Queue name
export type VQName =
  | 'sales_inbound' | 'sales_outbound' | 'sales_callback' | 'sales_chat'
  | 'support_general' | 'support_billing' | 'support_callback' | 'support_chat'
  | 'tech_l1' | 'tech_l2' | 'tech_callback' | 'tech_chat'
  | 'retention_save' | 'retention_cancel' | 'retention_callback' | 'retention_chat'

// Service Level metrics for a VQ
export interface ServiceLevel {
  target: number          // target percentage (e.g., 80)
  thresholdSecs: number   // threshold in seconds (e.g., 20)
  answeredInSL: number    // calls answered within threshold
  totalAnswered: number   // total calls answered
  currentSL: number       // calculated SL percentage
}

// VQ snapshot - current state of a virtual queue
export interface VQSnapshot {
  vq: VQName
  department: Department
  waitingCount: number
  activeCount: number
  completedCount: number
  abandonedCount: number
  longestWaitSecs: number
  availableAgents: number
  serviceLevel: ServiceLevel
}

// VQ Widget - all VQ snapshots for a department
export interface VQWidget {
  type: 'vq_overview'
  department: Department
  timestamp: string
  queues: VQSnapshot[]
}

// Department data within a snapshot
export interface DepartmentData {
  agents: AgentInfo[]
  queues: VQSnapshot[]
}

// Snapshot - single payload from backend every tick
// Contains all agents and all queues grouped by department
export interface Snapshot {
  type: 'snapshot'
  timestamp: string
  departments: Record<Department, DepartmentData>
}

// Call Record - completed call for history
export interface CallRecord {
  dateKey: string
  callId: string
  vq: VQName
  department: string
  agentId: string
  enqueueTime: string
  assignTime: string
  completeTime: string
  waitTime: number     // seconds
  talkTime: number     // seconds
  holdTime: number     // seconds
  wrapTime: number     // seconds
  handleTime: number   // seconds
  abandoned: boolean
  answeredInSL: boolean
}

// Simulation status from AgentSim
export interface SimStatus {
  running: boolean
  totalAgents: number
  activeAgents: number
  startedAt?: string
  eventsSent?: number
}

// Call generation config from AgentSim
export interface CallConfig {
  peakHourFactor: number
  departments: Record<string, { callsPerMin: number }>
}

// Snapshot history message sent on WebSocket connect
export interface SnapshotHistory {
  type: 'snapshot_history'
  snapshots: Snapshot[]
}

// Playback mode for snapshot time machine
export type PlaybackMode = 'live' | 'paused' | 'scrubbing'

// Agent Daily Stats - aggregated daily stats for history
export interface AgentDailyStats {
  agentId: string
  date: string
  department: string
  totalCalls: number
  totalTalkTime: number   // seconds
  totalHoldTime: number   // seconds
  totalWrapTime: number   // seconds
  totalBreakTime: number  // seconds
  avgHandleTime: number   // seconds
  occupancy: number       // 0-100%
  loginDuration: number   // seconds
}
