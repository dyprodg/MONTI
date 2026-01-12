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
