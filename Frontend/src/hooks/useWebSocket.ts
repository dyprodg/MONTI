import { useEffect, useState, useCallback, useRef } from 'react'
import { getWebSocketService } from '../services/websocket'
import { ConnectionState, TimeMessage, WebSocketError } from '../types'

interface UseWebSocketReturn {
  data: TimeMessage | null
  connectionState: ConnectionState
  error: WebSocketError | null
  connect: () => void
  disconnect: () => void
}

export const useWebSocket = (url: string): UseWebSocketReturn => {
  const [data, setData] = useState<TimeMessage | null>(null)
  const [connectionState, setConnectionState] = useState<ConnectionState>(ConnectionState.CLOSED)
  const [error, setError] = useState<WebSocketError | null>(null)
  const wsServiceRef = useRef<ReturnType<typeof getWebSocketService> | null>(null)

  // Initialize WebSocket service
  useEffect(() => {
    wsServiceRef.current = getWebSocketService(url)
  }, [url])

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

  // Set up event listeners
  useEffect(() => {
    const ws = wsServiceRef.current
    if (!ws) return

    // Message handler
    const unsubscribeMessage = ws.onMessage((message) => {
      setData(message as TimeMessage)
    })

    // State change handler
    const unsubscribeState = ws.onStateChange((state) => {
      setConnectionState(state)
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
  }, [connect, disconnect])

  return {
    data,
    connectionState,
    error,
    connect,
    disconnect,
  }
}
