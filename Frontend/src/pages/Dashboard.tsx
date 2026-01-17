import { useWebSocket } from '../hooks/useWebSocket'
import { ConnectionStatus } from '../components/ConnectionStatus'
import { WidgetDisplay } from '../components/WidgetDisplay'
import { AgentModal } from '../components/AgentModal'
import { ThemeToggle } from '../components/ThemeToggle'
import { useAuth } from '../contexts/AuthContext'
import { useTheme } from '../contexts/ThemeContext'
import { Widget, Location, AgentInfo, AgentState, Department } from '../types'
import { useState, useEffect } from 'react'

const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws'

const DEPARTMENTS = ['sales', 'support', 'technical', 'retention'] as const
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

export const Dashboard = () => {
  const { user, logout, isAuthenticated } = useAuth()
  const { colors } = useTheme()
  const { data, connectionState, error } = useWebSocket(WS_URL, { enabled: isAuthenticated })
  const [widgets, setWidgets] = useState<Map<string, Widget>>(new Map())
  const [selectedCity, setSelectedCity] = useState<Location | 'all'>('all')
  const [visibleError, setVisibleError] = useState<string | null>(null)
  const [selectedAgent, setSelectedAgent] = useState<AgentInfo | null>(null)
  const [selectedState, setSelectedState] = useState<AgentState | null>(null)
  const [selectedView, setSelectedView] = useState<'all' | Department>('all')
  const [searchQuery, setSearchQuery] = useState('')

  const handleAgentClick = (agent: AgentInfo) => {
    setSelectedAgent(agent)
  }

  const handleCloseModal = () => {
    setSelectedAgent(null)
  }

  const handleStateFilter = (state: AgentState) => {
    setSelectedState(state === selectedState ? null : state)
  }

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

  // Get only department widgets (not global overview)
  const departmentWidgets = Array.from(widgets.values())
    .filter((w) => w.type === 'department_overview')
    .sort((a, b) => (a.department || '').localeCompare(b.department || ''))

  // Filter widgets by selected city, state, and search query
  const filteredWidgets = departmentWidgets.map((widget) => {
    if (!widget.agents) {
      return widget
    }

    // Filter agents by city, state, and search query
    let filteredAgents = widget.agents

    if (selectedCity !== 'all') {
      filteredAgents = filteredAgents.filter(
        (agent) => agent.location === selectedCity
      )
    }

    if (selectedState) {
      filteredAgents = filteredAgents.filter(
        (agent) => agent.state === selectedState
      )
    }

    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase().trim()
      filteredAgents = filteredAgents.filter(
        (agent) => agent.agentId.toLowerCase().includes(query)
      )
    }

    // Recalculate summary for filtered agents
    const stateBreakdown: Record<string, number> = {}
    const locationBreakdown: Record<string, number> = {}

    filteredAgents.forEach((agent) => {
      stateBreakdown[agent.state] = (stateBreakdown[agent.state] || 0) + 1
      locationBreakdown[agent.location] = (locationBreakdown[agent.location] || 0) + 1
    })

    return {
      ...widget,
      agents: filteredAgents,
      summary: {
        ...widget.summary,
        totalAgents: filteredAgents.length,
        stateBreakdown,
        locationBreakdown,
      },
    }
  })

  return (
    <div
      style={{
        minHeight: '100vh',
        backgroundColor: colors.background,
        padding: '16px',
      }}
    >
      <div
        style={{
          maxWidth: '100%',
          margin: '0 auto',
        }}
      >
        {/* Header with User Info */}
        <div
          style={{
            marginBottom: '12px',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <h1
              style={{
                fontSize: '24px',
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
            marginBottom: '12px',
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
            marginBottom: '12px',
            display: 'flex',
            alignItems: 'center',
            gap: '16px',
            backgroundColor: colors.surface,
            padding: '8px 12px',
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
        {departmentWidgets.length === 0 ? (
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
              /* Grid layout for all department widgets (2x2) - each cell 50% width x 50% height */
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: '1fr 1fr',
                  gridTemplateRows: '1fr 1fr',
                  gap: '12px',
                  height: 'calc(100vh - 200px)',
                  minHeight: 0,
                }}
              >
                {DEPARTMENTS.map((dept) => {
                  const widget = filteredWidgets.find((w) => w.department === dept)
                  return (
                    <div
                      key={dept}
                      style={{
                        minHeight: 0,
                        minWidth: 0,
                        overflow: 'hidden',
                      }}
                    >
                      {widget ? (
                        <WidgetDisplay
                          widget={widget}
                          onAgentClick={handleAgentClick}
                          selectedState={selectedState}
                          onStateFilter={handleStateFilter}
                        />
                      ) : (
                        <div
                          style={{
                            backgroundColor: colors.surface,
                            borderRadius: '8px',
                            padding: '12px',
                            boxShadow: '0 2px 4px rgb(0 0 0 / 0.1)',
                            textAlign: 'center',
                            color: colors.textSecondary,
                            height: '100%',
                          }}
                        >
                          <h2
                            style={{
                              fontSize: '16px',
                              fontWeight: '600',
                              color: colors.text,
                              marginBottom: '8px',
                            }}
                          >
                            {dept.charAt(0).toUpperCase() + dept.slice(1)} Department
                          </h2>
                          <p style={{ fontSize: '11px' }}>No data available</p>
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>
            ) : (
              /* Single department full-width view with KPIs */
              <div style={{ height: 'calc(100vh - 200px)', display: 'flex', gap: '12px' }}>
                {(() => {
                  const widget = filteredWidgets.find((w) => w.department === selectedView)
                  if (!widget) {
                    return (
                      <div
                        style={{
                          backgroundColor: colors.surface,
                          borderRadius: '8px',
                          padding: '12px',
                          boxShadow: '0 2px 4px rgb(0 0 0 / 0.1)',
                          textAlign: 'center',
                          color: colors.textSecondary,
                          flex: 1,
                        }}
                      >
                        <h2
                          style={{
                            fontSize: '16px',
                            fontWeight: '600',
                            color: colors.text,
                            marginBottom: '8px',
                          }}
                        >
                          {selectedView.charAt(0).toUpperCase() + selectedView.slice(1)} Department
                        </h2>
                        <p style={{ fontSize: '11px' }}>No data available</p>
                      </div>
                    )
                  }

                  // Calculate aggregate KPIs
                  const agents = widget.agents || []
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
                      <div style={{ flex: 1 }}>
                        <WidgetDisplay
                          widget={widget}
                          onAgentClick={handleAgentClick}
                          selectedState={selectedState}
                          onStateFilter={handleStateFilter}
                        />
                      </div>
                    </>
                  )
                })()}
              </div>
            )}

            {/* Stats Footer */}
            <div
              style={{
                marginTop: '8px',
                textAlign: 'center',
                fontSize: '11px',
                color: colors.textSecondary,
              }}
            >
              <p>
                {filteredWidgets.reduce((sum, w) => sum + (w.summary.totalAgents || 0), 0)} agents • Real-time
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
    </div>
  )
}
