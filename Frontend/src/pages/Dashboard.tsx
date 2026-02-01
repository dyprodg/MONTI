import { useWebSocket } from '../hooks/useWebSocket'
import { useSnapshotBuffer } from '../hooks/useSnapshotBuffer'
import { ConnectionStatus } from '../components/ConnectionStatus'
import { TimelineControls } from '../components/TimelineControls'
import { VQDisplay } from '../components/VQDisplay'
import { AgentGrid } from '../components/AgentGrid'
import { AgentModal } from '../components/AgentModal'
import { AdminControlPanel } from '../components/AdminControlPanel'
import { ThemeToggle } from '../components/ThemeToggle'
import { useAuth } from '../contexts/AuthContext'
import { useTheme } from '../contexts/ThemeContext'
import { Snapshot, SnapshotHistory, Location, AgentInfo, AgentState, Department, VQSnapshot } from '../types'
import { useState, useEffect, useMemo, useCallback } from 'react'

const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws'

const VIEWS = ['all', 'sales', 'support', 'technical', 'retention'] as const

// Helper to format seconds
const formatSeconds = (seconds: number): string => {
  if (seconds < 60) return `${Math.round(seconds)}s`
  const mins = Math.floor(seconds / 60)
  const secs = Math.round(seconds % 60)
  return `${mins}m ${secs}s`
}

// KPI Card component
interface KPICardProps {
  label: string
  value: string
  highlight?: boolean
  colors: {
    background: string
    border: string
    text: string
    textSecondary: string
    highlightBg: string
    highlightBorder: string
  }
}

const KPICard = ({ label, value, highlight = false, colors }: KPICardProps) => (
  <div
    style={{
      backgroundColor: highlight ? colors.highlightBg : colors.background,
      borderRadius: '6px',
      padding: '10px',
      border: highlight ? `1px solid ${colors.highlightBorder}` : `1px solid ${colors.border}`,
    }}
  >
    <div style={{ fontSize: '9px', color: colors.textSecondary, marginBottom: '2px', textTransform: 'uppercase' }}>
      {label}
    </div>
    <div style={{ fontSize: '14px', fontWeight: '700', color: colors.text }}>{value}</div>
  </div>
)

// Agent list panel for a single department (used in 2x2 grid)
const DepartmentAgents = ({
  department,
  agents,
  onAgentClick,
  showOffline = true,
}: {
  department: Department
  agents: AgentInfo[]
  onAgentClick: (agent: AgentInfo) => void
  showOffline?: boolean
}) => {
  const { colors } = useTheme()
  return (
    <div
      style={{
        backgroundColor: colors.surface,
        borderRadius: '6px',
        padding: '8px',
        boxShadow: '0 1px 3px rgb(0 0 0 / 0.1)',
        display: 'flex',
        flexDirection: 'column',
        flex: 1,
        minHeight: 0,
        minWidth: 0,
        overflow: 'hidden',
      }}
    >
      <div
        style={{
          fontSize: '11px',
          fontWeight: '600',
          color: colors.text,
          marginBottom: '4px',
          textTransform: 'capitalize',
        }}
      >
        {department} ({showOffline ? agents.length : agents.filter((a) => a.state !== 'offline').length}{!showOffline && agents.some((a) => a.state === 'offline') ? ` / ${agents.length}` : ''})
      </div>
      <AgentGrid agents={agents} onAgentClick={onAgentClick} compact showOffline={showOffline} />
    </div>
  )
}

export const Dashboard = () => {
  const { user, logout, isAuthenticated } = useAuth()
  const { colors } = useTheme()
  const { data, connectionState, error } = useWebSocket(WS_URL, { enabled: isAuthenticated })

  const incomingSnapshot = useMemo(() => {
    if (!data) return null
    const message = data as any
    return message.type === 'snapshot' ? (message as Snapshot) : null
  }, [data])

  const {
    displaySnapshot,
    mode: playbackMode,
    bufferLength,
    cursorIndex,
    displayTimestamp,
    latestTimestamp,
    pause,
    goLive,
    scrubTo,
    seedHistory,
  } = useSnapshotBuffer(incomingSnapshot)

  // Handle snapshot_history messages from the server
  const handleSnapshotHistory = useCallback(() => {
    if (!data) return
    const message = data as any
    if (message.type === 'snapshot_history') {
      const historyMsg = message as SnapshotHistory
      if (historyMsg.snapshots && historyMsg.snapshots.length > 0) {
        seedHistory(historyMsg.snapshots)
      }
    }
  }, [data, seedHistory])

  useEffect(() => {
    handleSnapshotHistory()
  }, [handleSnapshotHistory])

  const [selectedCity, setSelectedCity] = useState<Location | 'all'>('all')
  const [visibleError, setVisibleError] = useState<string | null>(null)
  const [selectedAgent, setSelectedAgent] = useState<AgentInfo | null>(null)
  // State filter (can be set programmatically, no UI yet)
  const [selectedState, _setSelectedState] = useState<AgentState | null>(null)
  const [selectedView, setSelectedView] = useState<'all' | Department>('all')
  const [searchQuery, setSearchQuery] = useState('')
  const [showOffline, setShowOffline] = useState(false)
  const [showAdminPanel, setShowAdminPanel] = useState(false)

  const handleAgentClick = (agent: AgentInfo) => {
    setSelectedAgent(agent)
  }

  const handleCloseModal = () => {
    setSelectedAgent(null)
  }

  // Auto-dismiss errors after 5 seconds
  useEffect(() => {
    if (error) {
      setVisibleError(error.message)
      const timer = setTimeout(() => {
        setVisibleError(null)
      }, 5000)
      return () => clearTimeout(timer)
    }
  }, [error])

  const handleLogout = async () => {
    await logout()
  }

  // All 4 departments, always
  const allDepartments: Department[] = ['sales', 'support', 'technical', 'retention']

  // Filter agents per department by city, state, and search query
  const filteredDepts = useMemo(() => {
    if (!displaySnapshot) return {} as Record<Department, { agents: AgentInfo[]; queues: VQSnapshot[] }>

    const result: Record<string, { agents: AgentInfo[]; queues: VQSnapshot[] }> = {}
    for (const dept of allDepartments) {
      const data = displaySnapshot.departments[dept]
      if (!data) {
        result[dept] = { agents: [], queues: [] }
        continue
      }

      let agents = data.agents || []

      if (selectedCity !== 'all') {
        agents = agents.filter((a) => a.location === selectedCity)
      }
      if (selectedState) {
        agents = agents.filter((a) => a.state === selectedState)
      }
      if (searchQuery.trim()) {
        const query = searchQuery.toLowerCase().trim()
        agents = agents.filter((a) => a.agentId.toLowerCase().includes(query))
      }

      result[dept] = { agents, queues: data.queues || [] }
    }
    return result as Record<Department, { agents: AgentInfo[]; queues: VQSnapshot[] }>
  }, [displaySnapshot, selectedCity, selectedState, searchQuery])

  const hasData = displaySnapshot !== null

  return (
    <div
      style={{
        height: '100vh',
        backgroundColor: colors.background,
        padding: '8px 12px',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
      }}
    >
      <div
        style={{
          width: '100%',
          display: 'flex',
          flexDirection: 'column',
          flex: 1,
          minHeight: 0,
        }}
      >
        {/* Header with User Info */}
        <div
          style={{
            marginBottom: '6px',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <h1
              style={{
                fontSize: '20px',
                fontWeight: '700',
                color: colors.text,
                margin: 0,
              }}
            >
              MONTI
            </h1>
            <ConnectionStatus state={connectionState} />
          </div>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '12px',
            }}
          >
            {user && user.role === 'admin' && (
              <button
                onClick={() => setShowAdminPanel(true)}
                style={{
                  padding: '6px 12px',
                  backgroundColor: colors.primary,
                  color: 'white',
                  border: 'none',
                  borderRadius: '4px',
                  fontSize: '12px',
                  cursor: 'pointer',
                  fontWeight: '600',
                }}
              >
                Admin
              </button>
            )}
            <ThemeToggle />
            {user && (
              <>
                <div
                  style={{
                    fontSize: '12px',
                    color: colors.textSecondary,
                    textAlign: 'right',
                  }}
                >
                  <div style={{ fontWeight: '600', color: colors.text }}>
                    {user.name}
                  </div>
                  <div style={{ fontSize: '11px' }}>
                    {user.role}
                    {user.businessUnits && user.businessUnits.length > 0 && (
                      <span style={{ marginLeft: '8px', color: colors.primary }}>
                        [{user.businessUnits.join(', ')}]
                      </span>
                    )}
                  </div>
                </div>
                <button
                  onClick={handleLogout}
                  style={{
                    padding: '6px 12px',
                    backgroundColor: colors.surfaceHover,
                    color: colors.text,
                    border: 'none',
                    borderRadius: '4px',
                    fontSize: '12px',
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

        {/* Error Display */}
        {visibleError && (
          <div
            style={{
              padding: '12px',
              backgroundColor: colors.errorBg,
              border: `1px solid ${colors.errorBorder}`,
              borderRadius: '6px',
              marginBottom: '12px',
              color: colors.error,
              fontSize: '12px',
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
            }}
          >
            <span>
              <strong>Error:</strong> {visibleError}
            </span>
            <button
              onClick={() => setVisibleError(null)}
              style={{
                background: 'none',
                border: 'none',
                color: colors.error,
                cursor: 'pointer',
                fontSize: '18px',
                padding: '0 4px',
              }}
            >
              ×
            </button>
          </div>
        )}

        {/* Department View Tabs */}
        <div
          style={{
            display: 'flex',
            gap: '4px',
            marginBottom: '6px',
          }}
        >
          {VIEWS.map((view) => (
            <button
              key={view}
              onClick={() => setSelectedView(view)}
              style={{
                padding: '8px 16px',
                borderRadius: '6px',
                border: 'none',
                fontSize: '13px',
                fontWeight: '600',
                cursor: 'pointer',
                backgroundColor: selectedView === view ? colors.primary : colors.surfaceHover,
                color: selectedView === view ? 'white' : colors.text,
                transition: 'all 0.2s',
              }}
            >
              {view === 'all' ? 'All Departments' : view.charAt(0).toUpperCase() + view.slice(1)}
            </button>
          ))}
        </div>

        {/* Filters Bar */}
        <div
          style={{
            marginBottom: '6px',
            display: 'flex',
            alignItems: 'center',
            gap: '16px',
            backgroundColor: colors.surface,
            padding: '6px 12px',
            borderRadius: '6px',
            boxShadow: '0 2px 4px rgb(0 0 0 / 0.1)',
          }}
        >
          {/* Search */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <span
              style={{
                fontSize: '12px',
                fontWeight: '600',
                color: colors.text,
              }}
            >
              Search:
            </span>
            <input
              type="text"
              placeholder="Agent ID..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              style={{
                padding: '4px 10px',
                borderRadius: '4px',
                border: `1px solid ${colors.border}`,
                fontSize: '11px',
                width: '120px',
                outline: 'none',
                backgroundColor: colors.surface,
                color: colors.text,
              }}
            />
            {searchQuery && (
              <button
                onClick={() => setSearchQuery('')}
                style={{
                  padding: '2px 6px',
                  borderRadius: '4px',
                  border: 'none',
                  fontSize: '11px',
                  cursor: 'pointer',
                  backgroundColor: colors.surfaceHover,
                  color: colors.text,
                }}
              >
                Clear
              </button>
            )}
          </div>

          {/* Divider */}
          <div style={{ width: '1px', height: '24px', backgroundColor: colors.border }} />

          {/* Show Offline Toggle */}
          <button
            onClick={() => setShowOffline((v) => !v)}
            style={{
              padding: '4px 10px',
              borderRadius: '4px',
              border: 'none',
              fontSize: '11px',
              fontWeight: '500',
              cursor: 'pointer',
              backgroundColor: showOffline ? colors.primary : colors.surfaceHover,
              color: showOffline ? 'white' : colors.text,
              transition: 'all 0.2s',
            }}
          >
            Show Offline
          </button>

          {/* Divider */}
          <div style={{ width: '1px', height: '24px', backgroundColor: colors.border }} />

          {/* City Filter - only show cities the user has access to */}
          <span
            style={{
              fontSize: '12px',
              fontWeight: '600',
              color: colors.text,
            }}
          >
            City:
          </span>
          <div style={{ display: 'flex', gap: '4px', flexWrap: 'wrap' }}>
            <button
              onClick={() => setSelectedCity('all')}
              style={{
                padding: '4px 10px',
                borderRadius: '4px',
                border: 'none',
                fontSize: '11px',
                fontWeight: '500',
                cursor: 'pointer',
                backgroundColor:
                  selectedCity === 'all' ? colors.primary : colors.surfaceHover,
                color: selectedCity === 'all' ? 'white' : colors.text,
                transition: 'all 0.2s',
              }}
            >
              All
            </button>
            {/* Filter cities based on user's allowed locations */}
            {(['berlin', 'munich', 'hamburg', 'frankfurt', 'remote'] as Location[])
              .filter((city) => !user?.allowedLocations || user.allowedLocations.length === 0 || user.allowedLocations.includes(city))
              .map((city) => (
                <button
                  key={city}
                  onClick={() => setSelectedCity(city)}
                  style={{
                    padding: '4px 10px',
                    borderRadius: '4px',
                    border: 'none',
                    fontSize: '11px',
                    fontWeight: '500',
                    cursor: 'pointer',
                    backgroundColor:
                      selectedCity === city ? colors.primary : colors.surfaceHover,
                    color: selectedCity === city ? 'white' : colors.text,
                    transition: 'all 0.2s',
                  }}
                >
                  {city.charAt(0).toUpperCase() + city.slice(1)}
                </button>
              ))}
          </div>
        </div>

        {/* Widgets Display */}
        {!hasData ? (
          <div
            style={{
              textAlign: 'center',
              padding: '48px',
              backgroundColor: colors.surface,
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
                color: colors.text,
                marginBottom: '8px',
              }}
            >
              Waiting for data...
            </h3>
            <p
              style={{
                color: colors.textSecondary,
                fontSize: '14px',
              }}
            >
              Department data will appear here when agent events are received
            </p>
          </div>
        ) : (
          <>
            {selectedView === 'all' ? (
              /* 4 rows x 3 cols grid — VQ panels align with their agent panels */
              /* Row 1: Sales agents  | Sales VQ    | Support agents  */
              /* Row 2: (cont.)       | Support VQ  | (cont.)         */
              /* Row 3: Retention agt | Retention VQ| Technical agents*/
              /* Row 4: (cont.)       | Technical VQ| (cont.)         */
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: '1fr 240px 1fr',
                  gridTemplateRows: '1fr 1fr 1fr 1fr',
                  gap: '8px',
                  flex: 1,
                  minHeight: 0,
                  overflow: 'hidden',
                }}
              >
                {/* Sales agents — rows 1-2, col 1 */}
                <div style={{ gridRow: '1 / 3', gridColumn: '1', display: 'flex', flexDirection: 'column', minHeight: 0, minWidth: 0 }}>
                  <DepartmentAgents
                    department="sales"
                    agents={filteredDepts.sales?.agents || []}
                    onAgentClick={handleAgentClick}
                    showOffline={showOffline}
                  />
                </div>
                {/* Sales VQ — row 1, col 2 */}
                <div style={{ gridRow: '1', gridColumn: '2', minHeight: 0, minWidth: 0, overflow: 'hidden' }}>
                  <VQDisplay department="sales" queues={filteredDepts.sales?.queues || []} />
                </div>
                {/* Support VQ — row 2, col 2 */}
                <div style={{ gridRow: '2', gridColumn: '2', minHeight: 0, minWidth: 0, overflow: 'hidden' }}>
                  <VQDisplay department="support" queues={filteredDepts.support?.queues || []} />
                </div>
                {/* Support agents — rows 1-2, col 3 */}
                <div style={{ gridRow: '1 / 3', gridColumn: '3', display: 'flex', flexDirection: 'column', minHeight: 0, minWidth: 0 }}>
                  <DepartmentAgents
                    department="support"
                    agents={filteredDepts.support?.agents || []}
                    onAgentClick={handleAgentClick}
                    showOffline={showOffline}
                  />
                </div>

                {/* Retention agents — rows 3-4, col 1 */}
                <div style={{ gridRow: '3 / 5', gridColumn: '1', display: 'flex', flexDirection: 'column', minHeight: 0, minWidth: 0 }}>
                  <DepartmentAgents
                    department="retention"
                    agents={filteredDepts.retention?.agents || []}
                    onAgentClick={handleAgentClick}
                    showOffline={showOffline}
                  />
                </div>
                {/* Retention VQ — row 3, col 2 */}
                <div style={{ gridRow: '3', gridColumn: '2', minHeight: 0, minWidth: 0, overflow: 'hidden' }}>
                  <VQDisplay department="retention" queues={filteredDepts.retention?.queues || []} />
                </div>
                {/* Technical VQ — row 4, col 2 */}
                <div style={{ gridRow: '4', gridColumn: '2', minHeight: 0, minWidth: 0, overflow: 'hidden' }}>
                  <VQDisplay department="technical" queues={filteredDepts.technical?.queues || []} />
                </div>
                {/* Technical agents — rows 3-4, col 3 */}
                <div style={{ gridRow: '3 / 5', gridColumn: '3', display: 'flex', flexDirection: 'column', minHeight: 0, minWidth: 0 }}>
                  <DepartmentAgents
                    department="technical"
                    agents={filteredDepts.technical?.agents || []}
                    onAgentClick={handleAgentClick}
                    showOffline={showOffline}
                  />
                </div>
              </div>
            ) : (
              /* Single department full-width view with KPIs */
              <div style={{ flex: 1, minHeight: 0, display: 'flex', gap: '12px' }}>
                {(() => {
                  const deptData = filteredDepts[selectedView]
                  const agents = deptData?.agents || []
                  const queues = deptData?.queues || []

                  // Calculate aggregate KPIs
                  const totalCalls = agents.reduce((sum, a) => sum + (a.kpis?.totalCalls || 0), 0)
                  const avgOccupancy = agents.length > 0
                    ? agents.reduce((sum, a) => sum + (a.kpis?.occupancy || 0), 0) / agents.length
                    : 0
                  const avgAdherence = agents.length > 0
                    ? agents.reduce((sum, a) => sum + (a.kpis?.adherence || 0), 0) / agents.length
                    : 0
                  const avgCSAT = agents.length > 0
                    ? agents.reduce((sum, a) => sum + (a.kpis?.customerSatisfaction || 0), 0) / agents.length
                    : 0
                  const avgFCR = agents.length > 0
                    ? agents.reduce((sum, a) => sum + (a.kpis?.firstCallResolution || 0), 0) / agents.length
                    : 0
                  const totalHolds = agents.reduce((sum, a) => sum + (a.kpis?.holdCount || 0), 0)
                  const totalTransfers = agents.reduce((sum, a) => sum + (a.kpis?.transferCount || 0), 0)
                  const avgHandleTime = agents.length > 0
                    ? agents.reduce((sum, a) => sum + (a.kpis?.avgHandleTime || 0), 0) / agents.length
                    : 0

                  const kpiColors = {
                    background: colors.background,
                    border: colors.border,
                    text: colors.text,
                    textSecondary: colors.textSecondary,
                    highlightBg: colors.highlightBg,
                    highlightBorder: colors.highlightBorder,
                  }

                  return (
                    <>
                      {/* KPIs Panel */}
                      <div
                        style={{
                          width: '280px',
                          backgroundColor: colors.surface,
                          borderRadius: '8px',
                          padding: '16px',
                          boxShadow: '0 2px 4px rgb(0 0 0 / 0.1)',
                          display: 'flex',
                          flexDirection: 'column',
                          gap: '12px',
                        }}
                      >
                        <h3 style={{ fontSize: '14px', fontWeight: '600', color: colors.text, margin: 0 }}>
                          Department KPIs
                        </h3>
                        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '8px' }}>
                          <KPICard label="Total Calls" value={totalCalls.toString()} colors={kpiColors} />
                          <KPICard label="Avg Handle Time" value={formatSeconds(avgHandleTime)} colors={kpiColors} />
                          <KPICard label="Occupancy" value={`${avgOccupancy.toFixed(1)}%`} highlight colors={kpiColors} />
                          <KPICard label="Adherence" value={`${avgAdherence.toFixed(1)}%`} highlight colors={kpiColors} />
                          <KPICard label="CSAT" value={`${avgCSAT.toFixed(2)}/5`} highlight colors={kpiColors} />
                          <KPICard label="FCR" value={`${avgFCR.toFixed(1)}%`} highlight colors={kpiColors} />
                          <KPICard label="Total Holds" value={totalHolds.toString()} colors={kpiColors} />
                          <KPICard label="Transfers" value={totalTransfers.toString()} colors={kpiColors} />
                        </div>
                      </div>
                      {/* Agent List */}
                      <div
                        style={{
                          flex: 1,
                          minWidth: 0,
                          backgroundColor: colors.surface,
                          borderRadius: '8px',
                          padding: '12px',
                          boxShadow: '0 2px 4px rgb(0 0 0 / 0.1)',
                          display: 'flex',
                          flexDirection: 'column',
                          minHeight: 0,
                          overflow: 'hidden',
                        }}
                      >
                        <div style={{ marginBottom: '8px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                          <h2 style={{ fontSize: '16px', fontWeight: '600', color: colors.text, margin: 0 }}>
                            {selectedView.charAt(0).toUpperCase() + selectedView.slice(1)} Department
                          </h2>
                          <div style={{ fontSize: '11px', color: colors.textSecondary }}>
                            {agents.length} agents
                          </div>
                        </div>
                        <AgentGrid agents={agents} onAgentClick={handleAgentClick} showOffline={showOffline} />
                      </div>
                      {/* VQ Panel */}
                      <div style={{ width: '280px' }}>
                        <VQDisplay department={selectedView} queues={queues} />
                      </div>
                    </>
                  )
                })()}
              </div>
            )}

            {/* Timeline Controls */}
            <TimelineControls
              mode={playbackMode}
              bufferLength={bufferLength}
              cursorIndex={cursorIndex}
              displayTimestamp={displayTimestamp}
              latestTimestamp={latestTimestamp}
              onPause={pause}
              onGoLive={goLive}
              onScrub={scrubTo}
            />

            {/* Stats Footer */}
            <div
              style={{
                marginTop: '4px',
                textAlign: 'center',
                fontSize: '10px',
                color: colors.textSecondary,
              }}
            >
              <p>
                {(() => {
                  const allAgents = allDepartments.flatMap((d) => filteredDepts[d]?.agents || [])
                  const activeCount = allAgents.filter((a) => a.state !== 'offline').length
                  const offlineCount = allAgents.filter((a) => a.state === 'offline').length
                  const totalCount = allAgents.length
                  return `${activeCount} active${offlineCount > 0 ? ` • Offline: ${offlineCount}` : ''} • ${totalCount} total`
                })()}
                {playbackMode === 'live' ? ' • Real-time' : ' • Historical'}
                {selectedCity !== 'all' && ` • ${selectedCity.charAt(0).toUpperCase() + selectedCity.slice(1)}`}
                {selectedState && ` • ${selectedState.replace('_', ' ')}`}
              </p>
            </div>
          </>
        )}
      </div>

      {/* Agent Modal */}
      {selectedAgent && (
        <AgentModal agent={selectedAgent} onClose={handleCloseModal} />
      )}

      {/* Admin Control Panel */}
      {showAdminPanel && (
        <AdminControlPanel onClose={() => setShowAdminPanel(false)} />
      )}
    </div>
  )
}
