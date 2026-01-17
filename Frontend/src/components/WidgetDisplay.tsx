import { Widget, AgentState, AgentInfo } from '../types'
import { useTheme } from '../contexts/ThemeContext'

interface WidgetDisplayProps {
  widget: Widget
  onAgentClick: (agent: AgentInfo) => void
  selectedState: AgentState | null
  onStateFilter: (state: AgentState) => void
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

// Format duration in seconds to readable format
const formatDuration = (stateStart: string): string => {
  const start = new Date(stateStart)
  const now = new Date()
  const durationSeconds = Math.floor((now.getTime() - start.getTime()) / 1000)

  if (durationSeconds < 60) {
    return `${durationSeconds}s`
  } else if (durationSeconds < 3600) {
    const minutes = Math.floor(durationSeconds / 60)
    const seconds = durationSeconds % 60
    return `${minutes}m ${seconds}s`
  } else {
    const hours = Math.floor(durationSeconds / 3600)
    const minutes = Math.floor((durationSeconds % 3600) / 60)
    return `${hours}h ${minutes}m`
  }
}

export const WidgetDisplay = ({ widget, onAgentClick, selectedState, onStateFilter }: WidgetDisplayProps) => {
  const { colors } = useTheme()
  const title =
    widget.type === 'global_overview'
      ? 'Global Overview'
      : `${widget.department?.charAt(0).toUpperCase()}${widget.department?.slice(1)} Department`

  const agents = widget.agents || []

  // Sort agents by state, then by agent ID
  const sortedAgents = [...agents].sort((a, b) => {
    if (a.state !== b.state) {
      return a.state.localeCompare(b.state)
    }
    return a.agentId.localeCompare(b.agentId)
  })

  const sortedStates = Object.entries(widget.summary.stateBreakdown)
    .filter(([_, count]) => count > 0)
    .sort(([, a], [, b]) => b - a)

  return (
    <div
      style={{
        backgroundColor: colors.surface,
        borderRadius: '8px',
        padding: '12px',
        boxShadow: '0 2px 4px rgb(0 0 0 / 0.1)',
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
        minHeight: 0,
        overflow: 'hidden',
      }}
    >
      {/* Header */}
      <div style={{ marginBottom: '8px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2
          style={{
            fontSize: '16px',
            fontWeight: '600',
            color: colors.text,
            margin: 0,
          }}
        >
          {title}
        </h2>
        <div
          style={{
            fontSize: '11px',
            color: colors.textSecondary,
          }}
        >
          {widget.summary.totalAgents} agents
        </div>
      </div>

      {/* State Summary */}
      <div
        style={{
          display: 'flex',
          gap: '4px',
          flexWrap: 'wrap',
          marginBottom: '8px',
        }}
      >
        {sortedStates.map(([state, count]) => (
          <div
            key={state}
            onClick={() => onStateFilter(state as AgentState)}
            style={{
              padding: selectedState === state ? '1px 5px' : '2px 6px',
              borderRadius: '4px',
              backgroundColor: STATE_COLORS[state as AgentState] + '20',
              border: selectedState === state
                ? `3px solid ${STATE_COLORS[state as AgentState]}`
                : `1px solid ${STATE_COLORS[state as AgentState]}`,
              display: 'flex',
              alignItems: 'center',
              gap: '4px',
              cursor: 'pointer',
              transition: 'border-width 0.1s',
            }}
          >
            <div
              style={{
                width: '6px',
                height: '6px',
                borderRadius: '50%',
                backgroundColor: STATE_COLORS[state as AgentState],
              }}
            />
            <span
              style={{
                fontSize: '10px',
                fontWeight: '600',
                color: colors.text,
              }}
            >
              {STATE_LABELS[state as AgentState]}: {count}
            </span>
          </div>
        ))}
      </div>

      {/* Agent List */}
      {agents.length > 0 ? (
        <div
          style={{
            flex: 1,
            minHeight: 0,
            overflowY: 'auto',
            border: `1px solid ${colors.border}`,
            borderRadius: '4px',
          }}
        >
          <table
            style={{
              width: '100%',
              borderCollapse: 'collapse',
              fontSize: '9px',
            }}
          >
            <thead>
              <tr
                style={{
                  backgroundColor: colors.surfaceHover,
                  borderBottom: `1px solid ${colors.border}`,
                  position: 'sticky',
                  top: 0,
                }}
              >
                <th
                  style={{
                    padding: '4px 6px',
                    textAlign: 'left',
                    fontWeight: '600',
                    color: colors.text,
                    fontSize: '8px',
                  }}
                >
                  ID
                </th>
                <th
                  style={{
                    padding: '4px 6px',
                    textAlign: 'left',
                    fontWeight: '600',
                    color: colors.text,
                    fontSize: '8px',
                  }}
                >
                  Status
                </th>
                <th
                  style={{
                    padding: '4px 6px',
                    textAlign: 'left',
                    fontWeight: '600',
                    color: colors.text,
                    fontSize: '8px',
                  }}
                >
                  Duration
                </th>
                <th
                  style={{
                    padding: '4px 6px',
                    textAlign: 'left',
                    fontWeight: '600',
                    color: colors.text,
                    fontSize: '8px',
                  }}
                >
                  City
                </th>
              </tr>
            </thead>
            <tbody>
              {sortedAgents.map((agent) => (
                <tr
                  key={agent.agentId}
                  onClick={() => onAgentClick(agent)}
                  style={{
                    borderBottom: `1px solid ${colors.border}`,
                    cursor: 'pointer',
                    transition: 'background-color 0.15s',
                  }}
                  onMouseEnter={(e) => (e.currentTarget.style.backgroundColor = colors.surfaceHover)}
                  onMouseLeave={(e) => (e.currentTarget.style.backgroundColor = 'transparent')}
                >
                  <td
                    style={{
                      padding: '3px 6px',
                      color: colors.text,
                      fontWeight: '500',
                      fontSize: '9px',
                    }}
                  >
                    {agent.agentId}
                  </td>
                  <td style={{ padding: '3px 6px' }}>
                    <div
                      style={{
                        display: 'inline-flex',
                        alignItems: 'center',
                        gap: '3px',
                        padding: '1px 4px',
                        borderRadius: '3px',
                        backgroundColor:
                          STATE_COLORS[agent.state] + '20',
                      }}
                    >
                      <div
                        style={{
                          width: '4px',
                          height: '4px',
                          borderRadius: '50%',
                          backgroundColor: STATE_COLORS[agent.state],
                        }}
                      />
                      <span
                        style={{
                          fontSize: '8px',
                          fontWeight: '500',
                          color: colors.text,
                        }}
                      >
                        {STATE_LABELS[agent.state]}
                      </span>
                    </div>
                  </td>
                  <td
                    style={{
                      padding: '3px 6px',
                      color: colors.textSecondary,
                      fontFamily: 'monospace',
                      fontSize: '8px',
                    }}
                  >
                    {formatDuration(agent.stateStart)}
                  </td>
                  <td
                    style={{
                      padding: '3px 6px',
                      color: colors.textSecondary,
                      fontSize: '8px',
                    }}
                  >
                    {agent.location.charAt(0).toUpperCase() +
                      agent.location.slice(1)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div
          style={{
            textAlign: 'center',
            padding: '16px',
            color: colors.textSecondary,
            fontSize: '11px',
          }}
        >
          No agents in this department
        </div>
      )}
    </div>
  )
}
