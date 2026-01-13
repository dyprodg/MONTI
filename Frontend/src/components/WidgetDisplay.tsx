import { Widget, AgentState } from '../types'

interface WidgetDisplayProps {
  widget: Widget
}

const STATE_COLORS: Record<AgentState, string> = {
  available: '#10b981',
  on_call: '#3b82f6',
  after_call_work: '#8b5cf6',
  break: '#f59e0b',
  lunch: '#f59e0b',
  meeting: '#6366f1',
  training: '#6366f1',
  offline: '#6b7280',
  busy: '#ef4444',
  on_hold: '#f59e0b',
  transferring: '#8b5cf6',
  conference: '#3b82f6',
}

const STATE_LABELS: Record<AgentState, string> = {
  available: 'Available',
  on_call: 'On Call',
  after_call_work: 'After Call',
  break: 'Break',
  lunch: 'Lunch',
  meeting: 'Meeting',
  training: 'Training',
  offline: 'Offline',
  busy: 'Busy',
  on_hold: 'On Hold',
  transferring: 'Transferring',
  conference: 'Conference',
}

export const WidgetDisplay = ({ widget }: WidgetDisplayProps) => {
  const title =
    widget.type === 'global_overview'
      ? 'Global Overview'
      : `${widget.department?.charAt(0).toUpperCase()}${widget.department?.slice(1)} Department`

  const sortedStates = Object.entries(widget.summary.stateBreakdown)
    .filter(([_, count]) => count > 0)
    .sort(([, a], [, b]) => b - a)

  return (
    <div
      style={{
        backgroundColor: 'white',
        borderRadius: '12px',
        padding: '24px',
        boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)',
        marginBottom: '24px',
      }}
    >
      {/* Header */}
      <div style={{ marginBottom: '20px' }}>
        <h2
          style={{
            fontSize: '20px',
            fontWeight: '600',
            color: '#111827',
            margin: '0 0 8px 0',
          }}
        >
          {title}
        </h2>
        <div
          style={{
            fontSize: '14px',
            color: '#6b7280',
          }}
        >
          {widget.summary.totalEvents} events •{' '}
          {new Date(widget.timestamp).toLocaleTimeString()}
        </div>
      </div>

      {/* State Breakdown */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))',
          gap: '12px',
        }}
      >
        {sortedStates.map(([state, count]) => (
          <div
            key={state}
            style={{
              padding: '12px',
              borderRadius: '8px',
              backgroundColor: '#f9fafb',
              border: '1px solid #e5e7eb',
            }}
          >
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                marginBottom: '4px',
              }}
            >
              <div
                style={{
                  width: '8px',
                  height: '8px',
                  borderRadius: '50%',
                  backgroundColor: STATE_COLORS[state as AgentState],
                }}
              />
              <div
                style={{
                  fontSize: '12px',
                  color: '#6b7280',
                  fontWeight: '500',
                }}
              >
                {STATE_LABELS[state as AgentState]}
              </div>
            </div>
            <div
              style={{
                fontSize: '24px',
                fontWeight: '700',
                color: '#111827',
              }}
            >
              {count}
            </div>
          </div>
        ))}
      </div>

      {/* Location Breakdown (for department widgets) */}
      {widget.summary.locationBreakdown && (
        <div style={{ marginTop: '20px' }}>
          <h3
            style={{
              fontSize: '14px',
              fontWeight: '600',
              color: '#374151',
              marginBottom: '12px',
            }}
          >
            By Location
          </h3>
          <div
            style={{
              display: 'flex',
              gap: '8px',
              flexWrap: 'wrap',
            }}
          >
            {Object.entries(widget.summary.locationBreakdown)
              .filter(([_, count]) => count > 0)
              .map(([location, count]) => (
                <div
                  key={location}
                  style={{
                    padding: '6px 12px',
                    backgroundColor: '#f3f4f6',
                    borderRadius: '6px',
                    fontSize: '12px',
                    color: '#374151',
                  }}
                >
                  <span style={{ fontWeight: '600' }}>
                    {location.charAt(0).toUpperCase() + location.slice(1)}:
                  </span>{' '}
                  {count}
                </div>
              ))}
          </div>
        </div>
      )}

      {/* Department Breakdown (for global widget) */}
      {widget.summary.departmentBreakdown && (
        <div style={{ marginTop: '20px' }}>
          <h3
            style={{
              fontSize: '14px',
              fontWeight: '600',
              color: '#374151',
              marginBottom: '12px',
            }}
          >
            By Department
          </h3>
          <div
            style={{
              display: 'flex',
              gap: '8px',
              flexWrap: 'wrap',
            }}
          >
            {Object.entries(widget.summary.departmentBreakdown)
              .filter(([_, count]) => count > 0)
              .map(([dept, count]) => (
                <div
                  key={dept}
                  style={{
                    padding: '6px 12px',
                    backgroundColor: '#f3f4f6',
                    borderRadius: '6px',
                    fontSize: '12px',
                    color: '#374151',
                  }}
                >
                  <span style={{ fontWeight: '600' }}>
                    {dept.charAt(0).toUpperCase() + dept.slice(1)}:
                  </span>{' '}
                  {count}
                </div>
              ))}
          </div>
        </div>
      )}
    </div>
  )
}
