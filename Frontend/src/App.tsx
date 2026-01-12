import { useWebSocket } from './hooks/useWebSocket'
import { ConnectionStatus } from './components/ConnectionStatus'
import { TimeDisplay } from './components/TimeDisplay'

const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws'

function App() {
  const { data, connectionState, error } = useWebSocket(WS_URL)

  return (
    <div
      style={{
        minHeight: '100vh',
        backgroundColor: '#f9fafb',
        padding: '32px',
      }}
    >
      <div
        style={{
          maxWidth: '800px',
          margin: '0 auto',
        }}
      >
        {/* Header */}
        <div
          style={{
            marginBottom: '32px',
            textAlign: 'center',
          }}
        >
          <h1
            style={{
              fontSize: '36px',
              fontWeight: '700',
              color: '#111827',
              marginBottom: '8px',
            }}
          >
            MONTI
          </h1>
          <p
            style={{
              fontSize: '16px',
              color: '#6b7280',
            }}
          >
            Live Call Center Monitoring
          </p>
        </div>

        {/* Connection Status */}
        <div
          style={{
            display: 'flex',
            justifyContent: 'center',
            marginBottom: '32px',
          }}
        >
          <ConnectionStatus state={connectionState} />
        </div>

        {/* Error Display */}
        {error && (
          <div
            style={{
              padding: '16px',
              backgroundColor: '#fef2f2',
              border: '1px solid #fecaca',
              borderRadius: '8px',
              marginBottom: '24px',
              color: '#991b1b',
            }}
          >
            <strong>Error:</strong> {error.message}
          </div>
        )}

        {/* Time Display */}
        <TimeDisplay data={data} />

        {/* Footer */}
        <div
          style={{
            marginTop: '48px',
            textAlign: 'center',
            fontSize: '14px',
            color: '#9ca3af',
          }}
        >
          <p>
            WebSocket Demo - Time updates every second
          </p>
        </div>
      </div>
    </div>
  )
}

export default App
