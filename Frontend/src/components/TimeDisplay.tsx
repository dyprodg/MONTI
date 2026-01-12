import { TimeMessage } from '../types'

interface TimeDisplayProps {
  data: TimeMessage | null
}

export const TimeDisplay = ({ data }: TimeDisplayProps) => {
  const formatTimestamp = (timestamp: string) => {
    try {
      const date = new Date(timestamp)
      return date.toLocaleString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false,
      })
    } catch {
      return timestamp
    }
  }

  return (
    <div
      style={{
        padding: '32px',
        backgroundColor: 'white',
        borderRadius: '12px',
        boxShadow: '0 1px 3px 0 rgb(0 0 0 / 0.1)',
        textAlign: 'center',
      }}
    >
      <h2
        style={{
          fontSize: '24px',
          fontWeight: '600',
          marginBottom: '16px',
          color: '#111827',
        }}
      >
        Server Time
      </h2>
      {data ? (
        <div>
          <div
            style={{
              fontSize: '48px',
              fontWeight: '700',
              color: '#3b82f6',
              marginBottom: '8px',
              fontFamily: 'monospace',
            }}
          >
            {formatTimestamp(data.timestamp).split(',')[1]}
          </div>
          <div
            style={{
              fontSize: '18px',
              color: '#6b7280',
              marginBottom: '16px',
            }}
          >
            {formatTimestamp(data.timestamp).split(',')[0]}
          </div>
          <div
            style={{
              fontSize: '14px',
              color: '#9ca3af',
              fontFamily: 'monospace',
            }}
          >
            Unix: {data.serverTime}
          </div>
        </div>
      ) : (
        <div style={{ color: '#9ca3af', fontSize: '16px' }}>
          Waiting for data...
        </div>
      )}
    </div>
  )
}
