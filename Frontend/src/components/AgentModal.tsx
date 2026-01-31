import { useState, useEffect } from 'react'
import { AgentInfo, AgentState, AgentDailyStats, CallRecord } from '../types'
import { useTheme, ThemeColors } from '../contexts/ThemeContext'
import { useAuth } from '../contexts/AuthContext'
import { fetchAgentHistory, fetchAgentCalls, killAgentCall, logoutAgent } from '../services/api'

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
  colors,
}: {
  label: string
  value: string
  highlight?: boolean
  colors: ThemeColors
}) => (
  <div
    style={{
      backgroundColor: highlight ? colors.highlightBg : colors.background,
      borderRadius: '8px',
      padding: '12px',
      border: highlight ? `1px solid ${colors.highlightBorder}` : `1px solid ${colors.border}`,
    }}
  >
    <div
      style={{
        fontSize: '10px',
        color: colors.textSecondary,
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
        color: colors.text,
      }}
    >
      {value}
    </div>
  </div>
)

type ModalTab = 'kpis' | 'history'

export const AgentModal = ({ agent, onClose }: AgentModalProps) => {
  const { colors } = useTheme()
  const { user, getToken } = useAuth()
  const [activeTab, setActiveTab] = useState<ModalTab>('kpis')
  const [history, setHistory] = useState<AgentDailyStats[]>([])
  const [calls, setCalls] = useState<CallRecord[]>([])
  const today = new Date().toISOString().slice(0, 10)
  const [selectedDate, setSelectedDate] = useState<string | null>(today)
  const [loading, setLoading] = useState(false)

  // Load daily stats when history tab opens
  useEffect(() => {
    if (activeTab === 'history') {
      setLoading(true)
      getToken().then((token) => {
        fetchAgentHistory(agent.agentId, token)
          .then((data) => setHistory(data))
          .catch(() => setHistory([]))
          .finally(() => setLoading(false))
      })
    }
  }, [activeTab, agent.agentId, getToken])

  // Load calls for selected date (defaults to today)
  useEffect(() => {
    if (activeTab === 'history' && selectedDate) {
      getToken().then((token) => {
        fetchAgentCalls(agent.agentId, selectedDate, token)
          .then((data) => setCalls(data))
          .catch(() => setCalls([]))
      })
    }
  }, [activeTab, selectedDate, agent.agentId, getToken])

  const handleLogoutAgent = async () => {
    const confirmed = window.confirm(
      `Are you sure you want to log out agent ${agent.agentId}?\n\nThis will disconnect the agent immediately.`
    )
    if (confirmed) {
      try {
        const token = await getToken()
        await logoutAgent(agent.agentId, token)
        onClose()
      } catch (err) {
        alert(`Failed to log out agent: ${err instanceof Error ? err.message : 'Unknown error'}`)
      }
    }
  }

  const handleEndCall = async () => {
    if (!agent.currentCallId) return
    const confirmed = window.confirm(
      `Are you sure you want to end the active call for agent ${agent.agentId}?`
    )
    if (confirmed) {
      try {
        const token = await getToken()
        await killAgentCall(agent.agentId, agent.currentCallId, token)
        onClose()
      } catch (err) {
        alert(`Failed to end call: ${err instanceof Error ? err.message : 'Unknown error'}`)
      }
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
          backgroundColor: colors.surface,
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
                color: colors.text,
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
                <span style={{ fontSize: '12px', fontWeight: '600', color: colors.text }}>
                  {STATE_LABELS[agent.state]}
                </span>
              </div>
              <span style={{ fontSize: '12px', color: colors.textSecondary }}>
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
              color: colors.textSecondary,
              padding: '0',
              lineHeight: '1',
            }}
          >
            &times;
          </button>
        </div>

        {/* Active Alerts */}
        {agent.alerts && agent.alerts.length > 0 && (
          <div style={{ marginBottom: '12px', display: 'flex', flexDirection: 'column', gap: '6px' }}>
            {agent.alerts.map((alert, i) => (
              <div
                key={i}
                style={{
                  padding: '8px 12px',
                  borderRadius: '6px',
                  fontSize: '12px',
                  fontWeight: '500',
                  backgroundColor: alert.severity === 'critical' ? 'rgba(239, 68, 68, 0.15)' : 'rgba(245, 158, 11, 0.12)',
                  color: alert.severity === 'critical' ? '#ef4444' : '#f59e0b',
                  border: `1px solid ${alert.severity === 'critical' ? 'rgba(239, 68, 68, 0.3)' : 'rgba(245, 158, 11, 0.3)'}`,
                }}
              >
                {alert.severity === 'critical' ? 'CRITICAL' : 'WARNING'}: {alert.message}
              </div>
            ))}
          </div>
        )}

        {/* Tab Bar */}
        <div style={{ display: 'flex', gap: '4px', marginBottom: '16px' }}>
          {(['kpis', 'history'] as ModalTab[]).map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              style={{
                padding: '6px 14px',
                borderRadius: '6px',
                border: 'none',
                fontSize: '12px',
                fontWeight: '600',
                cursor: 'pointer',
                backgroundColor: activeTab === tab ? colors.primary : colors.surfaceHover,
                color: activeTab === tab ? 'white' : colors.text,
              }}
            >
              {tab === 'kpis' ? 'KPIs' : 'History'}
            </button>
          ))}
        </div>

        {activeTab === 'kpis' ? (
          <>
            {/* KPIs Grid */}
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(3, 1fr)',
                gap: '12px',
                marginBottom: '20px',
              }}
            >
              <KPICard label="Total Calls" value={kpis.totalCalls.toString()} colors={colors} />
              <KPICard label="Avg Call Duration" value={formatTime(kpis.avgCallDuration)} colors={colors} />
              <KPICard label="Avg Handle Time" value={formatTime(kpis.avgHandleTime)} colors={colors} />
              <KPICard label="ACW Sessions" value={kpis.acwCount.toString()} colors={colors} />
              <KPICard label="Total ACW Time" value={formatTime(kpis.acwTime)} colors={colors} />
              <KPICard label="Break Time" value={formatTime(kpis.breakTime)} colors={colors} />
              <KPICard label="Hold Count" value={kpis.holdCount.toString()} colors={colors} />
              <KPICard label="Total Hold Time" value={formatTime(kpis.holdTime)} colors={colors} />
              <KPICard label="Transfers" value={kpis.transferCount.toString()} colors={colors} />
              <KPICard label="Conferences" value={kpis.conferenceCount.toString()} colors={colors} />
              <KPICard label="Login Duration" value={formatTime(kpis.loginTime)} colors={colors} />
              <KPICard label="Occupancy" value={formatPercent(kpis.occupancy)} highlight colors={colors} />
              <KPICard label="Adherence" value={formatPercent(kpis.adherence)} highlight colors={colors} />
              <KPICard label="FCR" value={formatPercent(kpis.firstCallResolution)} highlight colors={colors} />
              <KPICard label="CSAT" value={formatCSAT(kpis.customerSatisfaction)} highlight colors={colors} />
            </div>
          </>
        ) : (
          <>
            {/* History Tab */}
            {loading ? (
              <div style={{ textAlign: 'center', padding: '24px', color: colors.textSecondary, fontSize: '12px' }}>
                Loading history...
              </div>
            ) : (
              <>
                {/* Daily Stats (if available) */}
                {history.length > 0 && (
                  <div style={{ marginBottom: '16px' }}>
                    <h4 style={{ fontSize: '13px', fontWeight: '600', color: colors.text, margin: '0 0 8px 0' }}>
                      Daily Stats
                    </h4>
                    <div style={{ border: `1px solid ${colors.border}`, borderRadius: '6px', overflow: 'auto', maxHeight: '200px' }}>
                      <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '11px' }}>
                        <thead>
                          <tr style={{ backgroundColor: colors.surfaceHover, position: 'sticky', top: 0 }}>
                            <th style={{ padding: '6px 8px', textAlign: 'left', fontWeight: '600', color: colors.text }}>Date</th>
                            <th style={{ padding: '6px 8px', textAlign: 'right', fontWeight: '600', color: colors.text }}>Calls</th>
                            <th style={{ padding: '6px 8px', textAlign: 'right', fontWeight: '600', color: colors.text }}>Talk</th>
                            <th style={{ padding: '6px 8px', textAlign: 'right', fontWeight: '600', color: colors.text }}>ACW</th>
                            <th style={{ padding: '6px 8px', textAlign: 'right', fontWeight: '600', color: colors.text }}>Break</th>
                            <th style={{ padding: '6px 8px', textAlign: 'right', fontWeight: '600', color: colors.text }}>Occ%</th>
                          </tr>
                        </thead>
                        <tbody>
                          {history.map((row) => (
                            <tr
                              key={row.date}
                              onClick={() => setSelectedDate(row.date)}
                              style={{
                                borderTop: `1px solid ${colors.border}`,
                                cursor: 'pointer',
                                backgroundColor: selectedDate === row.date ? colors.highlightBg : 'transparent',
                              }}
                              onMouseEnter={(e) => {
                                if (selectedDate !== row.date) e.currentTarget.style.backgroundColor = colors.surfaceHover
                              }}
                              onMouseLeave={(e) => {
                                if (selectedDate !== row.date) e.currentTarget.style.backgroundColor = 'transparent'
                              }}
                            >
                              <td style={{ padding: '5px 8px', color: colors.text, fontFamily: 'monospace' }}>{row.date}</td>
                              <td style={{ padding: '5px 8px', textAlign: 'right', color: colors.text }}>{row.totalCalls}</td>
                              <td style={{ padding: '5px 8px', textAlign: 'right', color: colors.textSecondary }}>{formatTime(row.totalTalkTime)}</td>
                              <td style={{ padding: '5px 8px', textAlign: 'right', color: colors.textSecondary }}>{formatTime(row.totalWrapTime)}</td>
                              <td style={{ padding: '5px 8px', textAlign: 'right', color: colors.textSecondary }}>{formatTime(row.totalBreakTime)}</td>
                              <td style={{ padding: '5px 8px', textAlign: 'right', color: colors.text, fontWeight: '600' }}>{row.occupancy.toFixed(1)}%</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </div>
                )}

                {/* Calls for selected date (defaults to today) */}
                <div>
                  <h4 style={{ fontSize: '13px', fontWeight: '600', color: colors.text, margin: '0 0 8px 0' }}>
                    Calls on {selectedDate}
                  </h4>
                  {calls.length === 0 ? (
                    <div style={{ textAlign: 'center', padding: '12px', color: colors.textSecondary, fontSize: '11px' }}>
                      No calls found
                    </div>
                  ) : (
                    <div style={{ border: `1px solid ${colors.border}`, borderRadius: '6px', overflow: 'auto', maxHeight: '250px' }}>
                      <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '10px' }}>
                        <thead>
                          <tr style={{ backgroundColor: colors.surfaceHover, position: 'sticky', top: 0 }}>
                            <th style={{ padding: '5px 6px', textAlign: 'left', fontWeight: '600', color: colors.text }}>VQ</th>
                            <th style={{ padding: '5px 6px', textAlign: 'right', fontWeight: '600', color: colors.text }}>Wait</th>
                            <th style={{ padding: '5px 6px', textAlign: 'right', fontWeight: '600', color: colors.text }}>Talk</th>
                            <th style={{ padding: '5px 6px', textAlign: 'right', fontWeight: '600', color: colors.text }}>Hold</th>
                            <th style={{ padding: '5px 6px', textAlign: 'right', fontWeight: '600', color: colors.text }}>Handle</th>
                            <th style={{ padding: '5px 6px', textAlign: 'center', fontWeight: '600', color: colors.text }}>SL</th>
                          </tr>
                        </thead>
                        <tbody>
                          {calls.map((call) => (
                            <tr key={call.callId} style={{ borderTop: `1px solid ${colors.border}` }}>
                              <td style={{ padding: '4px 6px', color: colors.text }}>{call.vq}</td>
                              <td style={{ padding: '4px 6px', textAlign: 'right', color: colors.textSecondary }}>{formatTime(call.waitTime)}</td>
                              <td style={{ padding: '4px 6px', textAlign: 'right', color: colors.textSecondary }}>{formatTime(call.talkTime)}</td>
                              <td style={{ padding: '4px 6px', textAlign: 'right', color: colors.textSecondary }}>{formatTime(call.holdTime)}</td>
                              <td style={{ padding: '4px 6px', textAlign: 'right', color: colors.text, fontWeight: '500' }}>{formatTime(call.handleTime)}</td>
                              <td style={{ padding: '4px 6px', textAlign: 'center', color: call.answeredInSL ? '#22c55e' : '#ef4444' }}>
                                {call.answeredInSL ? 'Y' : 'N'}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              </>
            )}
          </>
        )}

        {/* Action Buttons — hidden for viewers */}
        {user && user.role !== 'viewer' && (
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px', marginTop: '16px' }}>
            {agent.currentCallId && (
              <button
                onClick={handleEndCall}
                style={{
                  padding: '10px 20px',
                  backgroundColor: '#f59e0b',
                  color: 'white',
                  border: 'none',
                  borderRadius: '6px',
                  fontSize: '13px',
                  fontWeight: '600',
                  cursor: 'pointer',
                }}
              >
                End Call
              </button>
            )}
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
        )}
      </div>
    </>
  )
}
