import { useEffect, useState, useCallback, useRef } from 'react'
import { getWebSocketService } from '../services/websocket'
import { ConnectionState, TimeMessage, WebSocketError } from '../types'
import { authService } from '../services/auth'

interface UseWebSocketOptions {
  enabled?: boolean
}

interface UseWebSocketReturn {
  data: TimeMessage | null
  connectionState: ConnectionState
  error: WebSocketError | null
  connect: () => void
  disconnect: () => void
}

export const useWebSocket = (url: string, options: UseWebSocketOptions = {}): UseWebSocketReturn => {
  const { enabled = true } = options
  const [data, setData] = useState<TimeMessage | null>(null)
  const [connectionState, setConnectionState] = useState<ConnectionState>(ConnectionState.CLOSED)
  const [error, setError] = useState<WebSocketError | null>(null)
  const wsServiceRef = useRef<ReturnType<typeof getWebSocketService> | null>(null)
  const [wsInitialized, setWsInitialized] = useState(false)

  // Initialize WebSocket service with auth token - only when enabled
  useEffect(() => {
    if (!enabled) {
      return
    }

    const initWebSocket = async () => {
      const token = await authService.getToken()
      if (!token) {
        // Don't initialize WebSocket without a valid token
        console.debug('[WebSocket] Skipping init - no token available')
        return
      }
      const wsUrl = `${url}?token=${encodeURIComponent(token)}`
      console.debug('[WebSocket] Initializing with token')
      wsServiceRef.current = getWebSocketService(wsUrl)
      setWsInitialized(true)
    }

    initWebSocket()
  }, [url, enabled])

  // Connect function
  const connect = useCallback(() => {
    if (wsServiceRef.current) {
      wsServiceRef.current.connect()
    }
  }, [])

  // Disconnect function
  const disconnect = useCallback(() => {
    if (wsServiceRef.current) {
      wsServiceRef.current.disconnect()
    }
  }, [])

  // Set up event listeners - only run after WebSocket is initialized
  useEffect(() => {
    if (!wsInitialized) return

    const ws = wsServiceRef.current
    if (!ws) return

    // Message handler
    const unsubscribeMessage = ws.onMessage((message) => {
      setData(message as TimeMessage)
    })

    // State change handler
    const unsubscribeState = ws.onStateChange((state) => {
      setConnectionState(state)
      // Clear error when connection is successful
      if (state === ConnectionState.OPEN) {
        setError(null)
      }
    })

    // Error handler
    const unsubscribeError = ws.onError((err) => {
      setError(err)
    })

    // Auto-connect
    connect()

    // Cleanup
    return () => {
      unsubscribeMessage()
      unsubscribeState()
      unsubscribeError()
      disconnect()
    }
  }, [wsInitialized, connect, disconnect])

  return {
    data,
    connectionState,
    error,
    connect,
    disconnect,
  }
}
