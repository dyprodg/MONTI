import { AgentInfo, AgentState } from '../types'

interface AgentModalProps {
  agent: AgentInfo
  onClose: () => void
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

// Format seconds to human readable
const formatTime = (seconds: number): string => {
  if (seconds < 60) return `${Math.round(seconds)}s`
  if (seconds < 3600) {
    const mins = Math.floor(seconds / 60)
    const secs = Math.round(seconds % 60)
    return `${mins}m ${secs}s`
  }
  const hrs = Math.floor(seconds / 3600)
  const mins = Math.floor((seconds % 3600) / 60)
  return `${hrs}h ${mins}m`
}

// Format percentage
const formatPercent = (value: number): string => `${value.toFixed(1)}%`

// Format CSAT score
const formatCSAT = (value: number): string => `${value.toFixed(2)} / 5`

// Helper component for KPI cards
const KPICard = ({
  label,
  value,
  highlight = false,
}: {
  label: string
  value: string
  highlight?: boolean
}) => (
  <div
    style={{
      backgroundColor: highlight ? '#f0f9ff' : '#f9fafb',
      borderRadius: '8px',
      padding: '12px',
      border: highlight ? '1px solid #bae6fd' : '1px solid #e5e7eb',
    }}
  >
    <div
      style={{
        fontSize: '10px',
        color: '#6b7280',
        marginBottom: '4px',
        textTransform: 'uppercase',
        letterSpacing: '0.5px',
      }}
    >
      {label}
    </div>
    <div
      style={{
        fontSize: '16px',
        fontWeight: '700',
        color: '#111827',
      }}
    >
      {value}
    </div>
  </div>
)

export const AgentModal = ({ agent, onClose }: AgentModalProps) => {
  const handleLogoutAgent = () => {
    const confirmed = window.confirm(
      `This would log out agent ${agent.agentId}.\n\n` +
        `Note: This is a demonstration. In a production environment, ` +
        `this action would POST to a logout endpoint with admin role privileges.`
    )
    if (confirmed) {
      // Placeholder: Would POST to /api/agents/{agentId}/logout
      console.log(`Would POST to /api/agents/${agent.agentId}/logout`)
      alert('Logout request would be sent (placeholder functionality)')
    }
  }

  const kpis = agent.kpis

  return (
    <>
      {/* Backdrop */}
      <div
        onClick={onClose}
        style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          backgroundColor: 'rgba(0, 0, 0, 0.6)',
          zIndex: 1000,
        }}
      />

      {/* Modal */}
      <div
        style={{
          position: 'fixed',
          top: '50%',
          left: '50%',
          transform: 'translate(-50%, -50%)',
          backgroundColor: 'white',
          borderRadius: '12px',
          padding: '24px',
          width: '90%',
          maxWidth: '600px',
          maxHeight: '80vh',
          overflowY: 'auto',
          zIndex: 1001,
          boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.25)',
        }}
      >
        {/* Header */}
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'flex-start',
            marginBottom: '20px',
          }}
        >
          <div>
            <h2
              style={{
                margin: 0,
                fontSize: '20px',
                fontWeight: '700',
                color: '#111827',
              }}
            >
              {agent.agentId}
            </h2>
            <div
              style={{
                marginTop: '4px',
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
              }}
            >
              <div
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: '6px',
                  padding: '4px 10px',
                  borderRadius: '6px',
                  backgroundColor: STATE_COLORS[agent.state] + '20',
                }}
              >
                <div
                  style={{
                    width: '8px',
                    height: '8px',
                    borderRadius: '50%',
                    backgroundColor: STATE_COLORS[agent.state],
                  }}
                />
                <span style={{ fontSize: '12px', fontWeight: '600', color: '#111827' }}>
                  {STATE_LABELS[agent.state]}
                </span>
              </div>
              <span style={{ fontSize: '12px', color: '#6b7280' }}>
                {agent.department.charAt(0).toUpperCase() + agent.department.slice(1)} |{' '}
                {agent.team}
              </span>
            </div>
          </div>

          {/* Close button */}
          <button
            onClick={onClose}
            style={{
              background: 'none',
              border: 'none',
              fontSize: '24px',
              cursor: 'pointer',
              color: '#6b7280',
              padding: '0',
              lineHeight: '1',
            }}
          >
            &times;
          </button>
        </div>

        {/* KPIs Grid */}
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(3, 1fr)',
            gap: '12px',
            marginBottom: '20px',
          }}
        >
          {/* Call Metrics */}
          <KPICard label="Total Calls" value={kpis.totalCalls.toString()} />
          <KPICard label="Avg Call Duration" value={formatTime(kpis.avgCallDuration)} />
          <KPICard label="Avg Handle Time" value={formatTime(kpis.avgHandleTime)} />

          {/* ACW Metrics */}
          <KPICard label="ACW Sessions" value={kpis.acwCount.toString()} />
          <KPICard label="Total ACW Time" value={formatTime(kpis.acwTime)} />
          <KPICard label="Break Time" value={formatTime(kpis.breakTime)} />

          {/* Hold/Transfer */}
          <KPICard label="Hold Count" value={kpis.holdCount.toString()} />
          <KPICard label="Total Hold Time" value={formatTime(kpis.holdTime)} />
          <KPICard label="Transfers" value={kpis.transferCount.toString()} />

          {/* Conference/Login */}
          <KPICard label="Conferences" value={kpis.conferenceCount.toString()} />
          <KPICard label="Login Duration" value={formatTime(kpis.loginTime)} />
          <KPICard label="Occupancy" value={formatPercent(kpis.occupancy)} highlight />

          {/* Performance */}
          <KPICard label="Adherence" value={formatPercent(kpis.adherence)} highlight />
          <KPICard label="FCR" value={formatPercent(kpis.firstCallResolution)} highlight />
          <KPICard label="CSAT" value={formatCSAT(kpis.customerSatisfaction)} highlight />
        </div>

        {/* Log Out Button */}
        <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
          <button
            onClick={handleLogoutAgent}
            style={{
              padding: '10px 20px',
              backgroundColor: '#ef4444',
              color: 'white',
              border: 'none',
              borderRadius: '6px',
              fontSize: '13px',
              fontWeight: '600',
              cursor: 'pointer',
            }}
          >
            Log Out Agent
          </button>
        </div>
      </div>
    </>
  )
}
