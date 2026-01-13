import { useWebSocket } from '../hooks/useWebSocket'
import { ConnectionStatus } from '../components/ConnectionStatus'
import { WidgetDisplay } from '../components/WidgetDisplay'
import { useAuth } from '../contexts/AuthContext'
import { Widget } from '../types'
import { useState, useEffect } from 'react'

const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws'

export const Dashboard = () => {
  const { data, connectionState, error } = useWebSocket(WS_URL)
  const { user, logout } = useAuth()
  const [widgets, setWidgets] = useState<Map<string, Widget>>(new Map())

  // Handle incoming WebSocket messages
  useEffect(() => {
    if (!data) return

    // Check if the message is a widget
    const message = data as any
    if (message.type && (message.type === 'global_overview' || message.type === 'department_overview')) {
      const widget = message as Widget
      setWidgets((prev) => {
        const newWidgets = new Map(prev)
        // Use type + department as key to ensure we only keep the latest of each
        const key = widget.type === 'global_overview' ? 'global' : widget.department!
        newWidgets.set(key, widget)
        return newWidgets
      })
    }
  }, [data])

  const handleLogout = async () => {
    await logout()
  }

  // Sort widgets: global first, then departments alphabetically
  const sortedWidgets = Array.from(widgets.values()).sort((a, b) => {
    if (a.type === 'global_overview') return -1
    if (b.type === 'global_overview') return 1
    return (a.department || '').localeCompare(b.department || '')
  })

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
          maxWidth: '1400px',
          margin: '0 auto',
        }}
      >
        {/* Header with User Info */}
        <div
          style={{
            marginBottom: '32px',
            textAlign: 'center',
          }}
        >
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginBottom: '16px',
            }}
          >
            <div style={{ flex: 1 }} />
            <h1
              style={{
                fontSize: '36px',
                fontWeight: '700',
                color: '#111827',
                margin: 0,
              }}
            >
              MONTI
            </h1>
            <div
              style={{
                flex: 1,
                display: 'flex',
                justifyContent: 'flex-end',
                alignItems: 'center',
                gap: '12px',
              }}
            >
              {user && (
                <>
                  <div
                    style={{
                      fontSize: '14px',
                      color: '#6b7280',
                      textAlign: 'right',
                    }}
                  >
                    <div style={{ fontWeight: '600', color: '#111827' }}>
                      {user.name}
                    </div>
                    <div style={{ fontSize: '12px' }}>
                      {user.role}
                    </div>
                  </div>
                  <button
                    onClick={handleLogout}
                    style={{
                      padding: '8px 16px',
                      backgroundColor: '#f3f4f6',
                      color: '#374151',
                      border: 'none',
                      borderRadius: '6px',
                      fontSize: '14px',
                      cursor: 'pointer',
                      fontWeight: '500',
                    }}
                  >
                    Logout
                  </button>
                </>
              )}
            </div>
          </div>
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

        {/* Widgets Display */}
        {sortedWidgets.length === 0 ? (
          <div
            style={{
              textAlign: 'center',
              padding: '48px',
              backgroundColor: 'white',
              borderRadius: '12px',
              boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)',
            }}
          >
            <div
              style={{
                fontSize: '48px',
                marginBottom: '16px',
              }}
            >
              📊
            </div>
            <h3
              style={{
                fontSize: '18px',
                fontWeight: '600',
                color: '#111827',
                marginBottom: '8px',
              }}
            >
              Waiting for data...
            </h3>
            <p
              style={{
                color: '#6b7280',
                fontSize: '14px',
              }}
            >
              Widgets will appear here when agent events are received
            </p>
          </div>
        ) : (
          <>
            {/* Grid layout for widgets */}
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(500px, 1fr))',
                gap: '24px',
              }}
            >
              {sortedWidgets.map((widget) => (
                <WidgetDisplay
                  key={widget.type === 'global_overview' ? 'global' : widget.department}
                  widget={widget}
                />
              ))}
            </div>

            {/* Stats Footer */}
            <div
              style={{
                marginTop: '32px',
                textAlign: 'center',
                fontSize: '14px',
                color: '#9ca3af',
              }}
            >
              <p>
                Displaying {sortedWidgets.length} widget{sortedWidgets.length !== 1 ? 's' : ''} •
                Updated in real-time
              </p>
            </div>
          </>
        )}
      </div>
    </div>
  )
}
