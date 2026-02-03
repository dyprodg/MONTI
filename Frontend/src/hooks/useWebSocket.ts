import { useEffect, useState, useCallback, useRef } from 'react'
import { WebSocketService } from '../services/websocket'
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
  const wsServiceRef = useRef<WebSocketService | null>(null)

  // Initialize WebSocket service and connect when enabled
  useEffect(() => {
    if (!enabled) {
      return
    }

    const ws = new WebSocketService(url, () => authService.getToken())
    wsServiceRef.current = ws

    // Message handler
    const unsubscribeMessage = ws.onMessage((message) => {
      setData(message as TimeMessage)
    })

    // State change handler
    const unsubscribeState = ws.onStateChange((state) => {
      setConnectionState(state)
      if (state === ConnectionState.OPEN) {
        setError(null)
      }
    })

    // Error handler
    const unsubscribeError = ws.onError((err) => {
      setError(err)
    })

    // Connect
    ws.connect()

    // Cleanup
    return () => {
      unsubscribeMessage()
      unsubscribeState()
      unsubscribeError()
      ws.disconnect()
      wsServiceRef.current = null
    }
  }, [url, enabled])

  // Connect function
  const connect = useCallback(() => {
    wsServiceRef.current?.connect()
  }, [])

  // Disconnect function
  const disconnect = useCallback(() => {
    wsServiceRef.current?.disconnect()
  }, [])

  return {
    data,
    connectionState,
    error,
    connect,
    disconnect,
  }
}
