import { useState, useMemo } from 'react'
import { AgentState, AgentInfo } from '../types'
import { useTheme } from '../contexts/ThemeContext'

interface AgentGridProps {
  agents: AgentInfo[]
  onAgentClick: (agent: AgentInfo) => void
  compact?: boolean
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
  available: 'Avail',
  on_call: 'Call',
  after_call_work: 'ACW',
  break: 'Break',
  lunch: 'Lunch',
  meeting: 'Meet',
  training: 'Train',
  offline: 'Off',
  busy: 'Busy',
  on_hold: 'Hold',
  transferring: 'Xfer',
  conference: 'Conf',
}

const ALL_STATES: AgentState[] = [
  'available', 'on_call', 'after_call_work', 'break', 'lunch',
  'meeting', 'training', 'offline', 'busy', 'on_hold', 'transferring', 'conference',
]

const formatDuration = (stateStart: string): string => {
  const start = new Date(stateStart)
  const now = new Date()
  const durationSeconds = Math.floor((now.getTime() - start.getTime()) / 1000)
  if (durationSeconds < 60) return `${durationSeconds}s`
  const mins = Math.floor(durationSeconds / 60)
  const secs = durationSeconds % 60
  if (durationSeconds < 3600) return `${mins}m ${secs}s`
  const hours = Math.floor(durationSeconds / 3600)
  const minutes = Math.floor((durationSeconds % 3600) / 60)
  return `${hours}h ${minutes}m`
}

const formatSeconds = (seconds: number): string => {
  if (seconds <= 0) return '-'
  if (seconds < 60) return `${Math.round(seconds)}s`
  const mins = Math.floor(seconds / 60)
  const secs = Math.round(seconds % 60)
  return `${mins}m ${secs}s`
}

type SortField =
  | 'agentId'
  | 'state'
  | 'duration'
  | 'location'
  | 'totalCalls'
  | 'avgCallDuration'
  | 'avgHandleTime'
  | 'acwCount'
  | 'acwTime'
  | 'breakTime'
  | 'holdCount'
  | 'holdTime'
  | 'transferCount'
  | 'conferenceCount'
  | 'loginTime'
  | 'occupancy'
  | 'adherence'
  | 'firstCallResolution'
  | 'customerSatisfaction'

type SortDir = 'asc' | 'desc'
type SortLevel = { field: SortField; dir: SortDir }

const ALL_LOCATIONS = ['berlin', 'munich', 'hamburg', 'frankfurt', 'remote'] as const

const getSortValue = (agent: AgentInfo, field: SortField): string | number => {
  switch (field) {
    case 'agentId': return agent.agentId
    case 'state': return agent.state
    case 'duration': return Math.floor((Date.now() - new Date(agent.stateStart).getTime()) / 1000)
    case 'location': return agent.location
    case 'totalCalls': return agent.kpis?.totalCalls ?? 0
    case 'avgCallDuration': return agent.kpis?.avgCallDuration ?? 0
    case 'avgHandleTime': return agent.kpis?.avgHandleTime ?? 0
    case 'acwCount': return agent.kpis?.acwCount ?? 0
    case 'acwTime': return agent.kpis?.acwTime ?? 0
    case 'breakTime': return agent.kpis?.breakTime ?? 0
    case 'holdCount': return agent.kpis?.holdCount ?? 0
    case 'holdTime': return agent.kpis?.holdTime ?? 0
    case 'transferCount': return agent.kpis?.transferCount ?? 0
    case 'conferenceCount': return agent.kpis?.conferenceCount ?? 0
    case 'loginTime': return agent.kpis?.loginTime ?? 0
    case 'occupancy': return agent.kpis?.occupancy ?? 0
    case 'adherence': return agent.kpis?.adherence ?? 0
    case 'firstCallResolution': return agent.kpis?.firstCallResolution ?? 0
    case 'customerSatisfaction': return agent.kpis?.customerSatisfaction ?? 0
    default: return ''
  }
}

type ColumnFormat = 'text' | 'status' | 'duration' | 'time' | 'number' | 'percent' | 'score'

interface ColumnDef {
  key: SortField
  label: string
  format: ColumnFormat
  width: number      // px — minimum width for this column
  frozen?: boolean    // sticky left
  left?: number       // sticky left offset in px
}

// First 3 columns are frozen (always visible), rest scroll horizontally
const COLUMNS: ColumnDef[] = [
  { key: 'agentId',              label: 'ID',       format: 'text',     width: 76,  frozen: true, left: 0 },
  { key: 'state',                label: 'Status',   format: 'status',   width: 56,  frozen: true, left: 76 },
  { key: 'duration',             label: 'Dur',      format: 'duration', width: 58,  frozen: true, left: 132 },
  // --- scrollable from here ---
  { key: 'location',             label: 'City',     format: 'text',     width: 64 },
  { key: 'totalCalls',           label: 'Calls',    format: 'number',   width: 42 },
  { key: 'avgCallDuration',      label: 'Avg Call', format: 'time',     width: 58 },
  { key: 'avgHandleTime',        label: 'AHT',      format: 'time',     width: 58 },
  { key: 'acwCount',             label: 'ACW#',     format: 'number',   width: 42 },
  { key: 'acwTime',              label: 'ACW T',    format: 'time',     width: 56 },
  { key: 'breakTime',            label: 'Break',    format: 'time',     width: 56 },
  { key: 'holdCount',            label: 'Hold#',    format: 'number',   width: 42 },
  { key: 'holdTime',             label: 'Hold T',   format: 'time',     width: 56 },
  { key: 'transferCount',        label: 'Xfer',     format: 'number',   width: 40 },
  { key: 'conferenceCount',      label: 'Conf',     format: 'number',   width: 40 },
  { key: 'loginTime',            label: 'Login',    format: 'time',     width: 56 },
  { key: 'occupancy',            label: 'Occ%',     format: 'percent',  width: 46 },
  { key: 'adherence',            label: 'Adh%',     format: 'percent',  width: 46 },
  { key: 'firstCallResolution',  label: 'FCR%',     format: 'percent',  width: 46 },
  { key: 'customerSatisfaction', label: 'CSAT',     format: 'score',    width: 42 },
]

// total min-width of the table so the browser never squishes it
const TABLE_MIN_WIDTH = COLUMNS.reduce((sum, c) => sum + c.width, 0)


const formatCellValue = (agent: AgentInfo, col: ColumnDef): string => {
  const val = getSortValue(agent, col.key)
  switch (col.format) {
    case 'text':
      if (col.key === 'location') return String(val).charAt(0).toUpperCase() + String(val).slice(1)
      return String(val)
    case 'duration':
      return formatDuration(agent.stateStart)
    case 'time':
      return formatSeconds(val as number)
    case 'number':
      return String(val)
    case 'percent':
      return `${(val as number).toFixed(1)}%`
    case 'score':
      return (val as number).toFixed(1)
    default:
      return String(val)
  }
}


export const AgentGrid = ({ agents, onAgentClick, compact = false, showOffline = true }: AgentGridProps) => {
  const { colors } = useTheme()
  const [sorts, setSorts] = useState<SortLevel[]>([{ field: 'agentId', dir: 'asc' }])
  const [filterState, setFilterState] = useState<string>('')
  const [filterCity, setFilterCity] = useState<string>('')

  const handleSort = (field: SortField) => {
    setSorts((prev) => {
      const idx = prev.findIndex((s) => s.field === field)
      if (idx === -1) {
        // Not in sort list — add as next level, or replace all if already 3
        if (prev.length >= 3) {
          return [{ field, dir: 'asc' }]
        }
        return [...prev, { field, dir: 'asc' }]
      }
      // Already sorting by this field
      const current = prev[idx]
      if (current.dir === 'asc') {
        // Toggle to desc
        const next = [...prev]
        next[idx] = { field, dir: 'desc' }
        return next
      }
      // Was desc — remove it
      const next = prev.filter((_, i) => i !== idx)
      return next.length === 0 ? [{ field: 'agentId', dir: 'asc' }] : next
    })
  }

  // Step 1: filter by offline toggle
  const filteredByOffline = useMemo(() => {
    if (showOffline) return agents
    return agents.filter((a) => a.state !== 'offline')
  }, [agents, showOffline])

  // Step 2: apply state + city filters (AND)
  const filteredByConditions = useMemo(() => {
    return filteredByOffline.filter((agent) => {
      if (filterState && agent.state !== filterState) return false
      if (filterCity && agent.location !== filterCity) return false
      return true
    })
  }, [filteredByOffline, filterState, filterCity])

  // Step 3: multi-level sort
  const sortedAgents = useMemo(() => {
    return [...filteredByConditions].sort((a, b) => {
      for (const { field, dir } of sorts) {
        const va = getSortValue(a, field)
        const vb = getSortValue(b, field)
        let cmp = 0
        if (typeof va === 'number' && typeof vb === 'number') {
          cmp = va - vb
        } else {
          cmp = String(va).localeCompare(String(vb))
        }
        if (cmp !== 0) return dir === 'asc' ? cmp : -cmp
      }
      return 0
    })
  }, [filteredByConditions, sorts])

  const fontSize = compact ? '9px' : '10px'
  const cellPad = compact ? '3px 5px' : '3px 6px'

  if (sortedAgents.length === 0 && !showOffline) {
    return (
      <div style={{ textAlign: 'center', padding: '8px', color: colors.textSecondary, fontSize: '11px' }}>
        No active agents
      </div>
    )
  }

  if (agents.length === 0) {
    return (
      <div style={{ textAlign: 'center', padding: '8px', color: colors.textSecondary, fontSize: '11px' }}>
        No agents
      </div>
    )
  }

  const sortIndicator = (field: SortField): string => {
    const idx = sorts.findIndex((s) => s.field === field)
    if (idx === -1) return ''
    const arrow = sorts[idx].dir === 'asc' ? '\u25B2' : '\u25BC'
    // Single sort: just arrow. Multi: number + arrow
    if (sorts.length === 1) return ` ${arrow}`
    return ` ${idx + 1}${arrow}`
  }

  // Shared sticky styles for frozen columns
  const stickyStyle = (col: ColumnDef, bg: string): React.CSSProperties =>
    col.frozen
      ? { position: 'sticky', left: col.left, zIndex: 1, backgroundColor: bg }
      : {}

  const hasActiveFilters = filterState !== '' || filterCity !== ''
  const dropdownStyle: React.CSSProperties = {
    fontSize,
    padding: '1px 2px',
    backgroundColor: colors.surface,
    color: colors.text,
    border: `1px solid ${colors.border}`,
    borderRadius: '2px',
    outline: 'none',
    height: '20px',
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0, minWidth: 0 }}>
      {/* Filter bar — always visible */}
      <div
        style={{
          padding: '3px 6px',
          backgroundColor: colors.surface,
          border: `1px solid ${colors.border}`,
          borderBottom: 'none',
          borderRadius: '4px 4px 0 0',
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
          flexWrap: 'wrap',
        }}
      >
        <span style={{ fontSize, color: colors.textSecondary, fontWeight: '500' }}>Filter:</span>

        {/* Status filter */}
        <label style={{ display: 'flex', alignItems: 'center', gap: '3px', fontSize, color: colors.textSecondary }}>
          Status
          <select
            value={filterState}
            onChange={(e) => setFilterState(e.target.value)}
            style={dropdownStyle}
          >
            <option value="">All</option>
            {ALL_STATES.map((s) => (
              <option key={s} value={s}>{STATE_LABELS[s]}</option>
            ))}
          </select>
        </label>

        {/* City filter */}
        <label style={{ display: 'flex', alignItems: 'center', gap: '3px', fontSize, color: colors.textSecondary }}>
          City
          <select
            value={filterCity}
            onChange={(e) => setFilterCity(e.target.value)}
            style={dropdownStyle}
          >
            <option value="">All</option>
            {ALL_LOCATIONS.map((loc) => (
              <option key={loc} value={loc}>{loc.charAt(0).toUpperCase() + loc.slice(1)}</option>
            ))}
          </select>
        </label>

        {/* Clear button — only show when filters active */}
        {hasActiveFilters && (
          <button
            onClick={() => { setFilterState(''); setFilterCity('') }}
            style={{
              fontSize,
              background: 'none',
              border: `1px solid ${colors.border}`,
              borderRadius: '2px',
              color: colors.textSecondary,
              cursor: 'pointer',
              padding: '1px 5px',
            }}
          >
            Clear
          </button>
        )}
      </div>

      {/* Table */}
      <div
        style={{
          flex: 1,
          minHeight: 0,
          minWidth: 0,
          overflow: 'auto',
          border: `1px solid ${colors.border}`,
          borderRadius: '0 0 4px 4px',
        }}
      >
        <table
          style={{
            borderCollapse: 'collapse',
            fontSize,
            minWidth: `${TABLE_MIN_WIDTH}px`,
          }}
        >
          <thead>
            <tr
              style={{
                backgroundColor: colors.surfaceHover,
                borderBottom: `1px solid ${colors.border}`,
                position: 'sticky',
                top: 0,
                zIndex: 3,
              }}
            >
              {COLUMNS.map((col) => (
                <th
                  key={col.key}
                  onClick={() => handleSort(col.key)}
                  style={{
                    padding: cellPad,
                    textAlign: 'left',
                    fontWeight: '600',
                    color: colors.text,
                    fontSize,
                    cursor: 'pointer',
                    userSelect: 'none',
                    whiteSpace: 'nowrap',
                    minWidth: `${col.width}px`,
                    // frozen headers need higher z-index (corner overlap with sticky row)
                    ...(col.frozen
                      ? { position: 'sticky', left: col.left, zIndex: 4, backgroundColor: colors.surfaceHover }
                      : {}),
                  }}
                >
                  {col.label}{sortIndicator(col.key)}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {sortedAgents.map((agent) => {
              const isOffline = agent.state === 'offline'
              const hasCritical = agent.alerts?.some((a) => a.severity === 'critical')
              const hasWarning = agent.alerts?.some((a) => a.severity === 'warning')
              const alertBg = hasCritical
                ? 'rgba(239, 68, 68, 0.15)'
                : hasWarning
                  ? 'rgba(245, 158, 11, 0.12)'
                  : 'transparent'
              const rowBg = isOffline ? colors.surface : (alertBg !== 'transparent' ? alertBg : colors.surface)

              return (
                <tr
                  key={agent.agentId}
                  onClick={() => onAgentClick(agent)}
                  style={{
                    borderBottom: `1px solid ${colors.border}`,
                    cursor: 'pointer',
                    transition: 'background-color 0.15s',
                    backgroundColor: alertBg,
                    opacity: isOffline ? 0.45 : 1,
                  }}
                  onMouseEnter={(e) => (e.currentTarget.style.backgroundColor = colors.surfaceHover)}
                  onMouseLeave={(e) => (e.currentTarget.style.backgroundColor = alertBg)}
                >
                  {COLUMNS.map((col) => {
                    if (col.format === 'status') {
                      return (
                        <td
                          key={col.key}
                          style={{
                            padding: cellPad,
                            minWidth: `${col.width}px`,
                            ...stickyStyle(col, rowBg),
                          }}
                        >
                          <div
                            style={{
                              display: 'inline-flex',
                              alignItems: 'center',
                              gap: '3px',
                              padding: '1px 4px',
                              borderRadius: '3px',
                              backgroundColor: STATE_COLORS[agent.state] + '20',
                            }}
                          >
                            <div
                              style={{
                                width: '5px',
                                height: '5px',
                                borderRadius: '50%',
                                backgroundColor: STATE_COLORS[agent.state],
                              }}
                            />
                            <span style={{ fontSize, fontWeight: '500', color: colors.text }}>
                              {STATE_LABELS[agent.state]}
                            </span>
                          </div>
                        </td>
                      )
                    }

                    const isIdCol = col.key === 'agentId'
                    const isMonospace = col.format === 'duration' || col.format === 'time'

                    return (
                      <td
                        key={col.key}
                        style={{
                          padding: cellPad,
                          color: isIdCol ? colors.text : colors.textSecondary,
                          fontWeight: isIdCol ? '500' : '400',
                          fontFamily: isMonospace ? 'monospace' : 'inherit',
                          fontSize,
                          whiteSpace: 'nowrap',
                          minWidth: `${col.width}px`,
                          ...stickyStyle(col, rowBg),
                        }}
                      >
                        {formatCellValue(agent, col)}
                      </td>
                    )
                  })}
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}
