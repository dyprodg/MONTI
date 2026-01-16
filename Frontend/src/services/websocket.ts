import { ConnectionState, WebSocketError } from '../types'

type MessageHandler = (data: unknown) => void
type StateChangeHandler = (state: ConnectionState) => void
type ErrorHandler = (error: WebSocketError) => void

const INITIAL_RETRY_DELAY = 1000 // 1 second
const MAX_RETRY_DELAY = 30000 // 30 seconds
const BACKOFF_MULTIPLIER = 1.5

export class WebSocketService {
  private ws: WebSocket | null = null
  private url: string
  private messageHandlers: Set<MessageHandler> = new Set()
  private stateChangeHandlers: Set<StateChangeHandler> = new Set()
  private errorHandlers: Set<ErrorHandler> = new Set()
  private currentState: ConnectionState = ConnectionState.CLOSED
  private reconnectAttempts = 0
  private reconnectTimeout: number | null = null
  private shouldReconnect = true

  constructor(url: string) {
    this.url = url
  }

  connect(): void {
    if (this.ws && this.ws.readyState !== WebSocket.CLOSED) {
      return
    }

    // Reset reconnect flag when explicitly connecting
    this.shouldReconnect = true
    this.updateState(ConnectionState.CONNECTING)

    try {
      this.ws = new WebSocket(this.url)

      this.ws.onopen = () => {
        this.reconnectAttempts = 0
        this.updateState(ConnectionState.OPEN)
      }

      this.ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          this.messageHandlers.forEach((handler) => handler(data))
        } catch (error) {
          // Log parse errors to console instead of showing to user
          console.debug('[WebSocket] Failed to parse message:', event.data)
        }
      }

      this.ws.onerror = () => {
        this.updateState(ConnectionState.ERROR)
        this.notifyError({
          message: 'WebSocket error occurred',
        })
      }

      this.ws.onclose = (event) => {
        this.updateState(ConnectionState.CLOSED)

        if (this.shouldReconnect && !event.wasClean) {
          this.scheduleReconnect()
        }
      }
    } catch (error) {
      this.updateState(ConnectionState.ERROR)
      this.notifyError({
        message: error instanceof Error ? error.message : 'Failed to create WebSocket',
      })
    }
  }

  disconnect(): void {
    this.shouldReconnect = false
    this.clearReconnectTimeout()

    if (this.ws) {
      this.ws.close()
      this.ws = null
    }

    this.updateState(ConnectionState.CLOSED)
  }

  send(data: unknown): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data))
    } else {
      this.notifyError({
        message: 'Cannot send message: WebSocket is not open',
      })
    }
  }

  onMessage(handler: MessageHandler): () => void {
    this.messageHandlers.add(handler)
    return () => this.messageHandlers.delete(handler)
  }

  onStateChange(handler: StateChangeHandler): () => void {
    this.stateChangeHandlers.add(handler)
    // Immediately call with current state
    handler(this.currentState)
    return () => this.stateChangeHandlers.delete(handler)
  }

  onError(handler: ErrorHandler): () => void {
    this.errorHandlers.add(handler)
    return () => this.errorHandlers.delete(handler)
  }

  getState(): ConnectionState {
    return this.currentState
  }

  private updateState(state: ConnectionState): void {
    this.currentState = state
    this.stateChangeHandlers.forEach((handler) => handler(state))
  }

  private notifyError(error: WebSocketError): void {
    this.errorHandlers.forEach((handler) => handler(error))
  }

  private scheduleReconnect(): void {
    this.clearReconnectTimeout()

    const delay = Math.min(
      INITIAL_RETRY_DELAY * Math.pow(BACKOFF_MULTIPLIER, this.reconnectAttempts),
      MAX_RETRY_DELAY
    )

    this.reconnectTimeout = window.setTimeout(() => {
      this.reconnectAttempts++
      this.connect()
    }, delay)
  }

  private clearReconnectTimeout(): void {
    if (this.reconnectTimeout !== null) {
      clearTimeout(this.reconnectTimeout)
      this.reconnectTimeout = null
    }
  }
}

// Singleton instance
let instance: WebSocketService | null = null

export const getWebSocketService = (url?: string): WebSocketService => {
  if (!instance && url) {
    instance = new WebSocketService(url)
  }
  if (!instance) {
    throw new Error('WebSocketService not initialized. Provide URL on first call.')
  }
  return instance
}
