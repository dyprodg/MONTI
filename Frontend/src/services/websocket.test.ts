import { describe, it, expect, beforeEach, vi } from 'vitest'
import { WebSocketService } from './websocket'
import { ConnectionState } from '../types'

describe('WebSocketService', () => {
  let ws: WebSocketService

  beforeEach(() => {
    ws = new WebSocketService('ws://localhost:8080/ws')
  })

  it('should initialize with CLOSED state', () => {
    expect(ws.getState()).toBe(ConnectionState.CLOSED)
  })

  it('should transition to CONNECTING state when connect is called', () => {
    const stateChanges: ConnectionState[] = []
    ws.onStateChange((state) => stateChanges.push(state))

    ws.connect()

    expect(stateChanges).toContain(ConnectionState.CONNECTING)
  })

  it('should transition to OPEN state after connection', async () => {
    return new Promise<void>((resolve) => {
      ws.onStateChange((state) => {
        if (state === ConnectionState.OPEN) {
          expect(ws.getState()).toBe(ConnectionState.OPEN)
          resolve()
        }
      })

      ws.connect()
    })
  })

  it('should handle incoming messages', async () => {
    return new Promise<void>((resolve) => {
      const testData = { test: 'data' }

      ws.onMessage((data) => {
        expect(data).toEqual(testData)
        resolve()
      })

      ws.connect()

      // Simulate incoming message after connection
      setTimeout(() => {
        const mockWs = // eslint-disable-next-line @typescript-eslint/no-explicit-any
        (ws as any).ws
        if (mockWs && mockWs.onmessage) {
          mockWs.onmessage(
            new MessageEvent('message', { data: JSON.stringify(testData) })
          )
        }
      }, 10)
    })
  })

  it('should clean up when disconnected', () => {
    ws.connect()

    const unsubscribe = ws.onMessage(() => {})
    unsubscribe()

    ws.disconnect()

    expect(ws.getState()).toBe(ConnectionState.CLOSED)
  })

  it('should notify error handlers on error', async () => {
    return new Promise<void>((resolve) => {
      ws.onError((error) => {
        expect(error.message).toBeTruthy()
        resolve()
      })

      ws.connect()

      // Simulate error after connection
      setTimeout(() => {
        const mockWs = // eslint-disable-next-line @typescript-eslint/no-explicit-any
        (ws as any).ws
        if (mockWs && mockWs.onerror) {
          mockWs.onerror(new Event('error'))
        }
      }, 10)
    })
  })

  it('should unsubscribe handlers correctly', () => {
    const handler = vi.fn()
    const unsubscribe = ws.onMessage(handler)

    unsubscribe()

    ws.connect()

    // Handler should not be called after unsubscribe
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    expect((ws as any).messageHandlers.has(handler)).toBe(false)
  })
})
