import { Widget, AgentState, AgentInfo } from '../types'
import { AgentGrid } from './AgentGrid'
import { useTheme } from '../contexts/ThemeContext'

interface WidgetDisplayProps {
  widget: Widget
  onAgentClick: (agent: AgentInfo) => void
  selectedState: AgentState | null
  onStateFilter: (state: AgentState) => void
  showOffline?: boolean
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

export const WidgetDisplay = ({ widget, onAgentClick, selectedState, onStateFilter, showOffline = true }: WidgetDisplayProps) => {
  const { colors } = useTheme()
  const title =
    widget.type === 'global_overview'
      ? 'Global Overview'
      : `${widget.department?.charAt(0).toUpperCase()}${widget.department?.slice(1)} Department`

  const agents = widget.agents || []
  const activeCount = agents.filter((a) => a.state !== 'offline').length
  const totalCount = agents.length

  const sortedStates = Object.entries(widget.summary.stateBreakdown)
    .filter(([, count]) => count > 0)
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
        minWidth: 0,
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
          {!showOffline ? `${activeCount} active / ${totalCount} total` : `${totalCount} agents`}
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

      {/* Agent List — scrollable table with KPI columns */}
      <AgentGrid agents={agents} onAgentClick={onAgentClick} showOffline={showOffline} />
    </div>
  )
}
