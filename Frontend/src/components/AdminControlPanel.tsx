import { useState, useEffect, useCallback } from 'react'
import { useTheme } from '../contexts/ThemeContext'
import { useAuth } from '../contexts/AuthContext'
import { SimStatus, CallConfig, VQName } from '../types'
import {
  getSimStatus,
  startSim,
  stopSim,
  scaleSim,
  getCallConfig,
  updateCallConfig,
  injectCalls,
  wipeAllCalls,
  resetMemory,
  wipeDynamo,
  logoffAllAgents,
} from '../services/api'

interface AdminControlPanelProps {
  onClose: () => void
}

const ALL_VQS: VQName[] = [
  'sales_inbound', 'sales_outbound', 'sales_callback', 'sales_chat',
  'support_general', 'support_billing', 'support_callback', 'support_chat',
  'tech_l1', 'tech_l2', 'tech_callback', 'tech_chat',
  'retention_save', 'retention_cancel', 'retention_callback', 'retention_chat',
]

const DEPARTMENTS = ['sales', 'support', 'technical', 'retention'] as const

export const AdminControlPanel = ({ onClose }: AdminControlPanelProps) => {
  const { colors } = useTheme()
  const { getToken } = useAuth()

  // State
  const [simStatus, setSimStatus] = useState<SimStatus | null>(null)
  const [, setCallConfig] = useState<CallConfig | null>(null)
  const [actionLoading, setActionLoading] = useState<string | null>(null)
  const [agentCount, setAgentCount] = useState(100)
  const [deptRates, setDeptRates] = useState<Record<string, number>>({
    sales: 5,
    support: 5,
    technical: 3,
    retention: 2,
  })
  const [peakFactor, setPeakFactor] = useState(1.0)
  const [injectCount, setInjectCount] = useState(5)
  const [injectVQ, setInjectVQ] = useState<string>('')
  const [confirmReset, setConfirmReset] = useState(false)
  const [confirmDynamo, setConfirmDynamo] = useState(false)
  const [statusMsg, setStatusMsg] = useState<string | null>(null)
  const [configLoaded, setConfigLoaded] = useState(false)

  // Sync slider state from a CallConfig response
  const syncSlidersFromConfig = useCallback((config: CallConfig) => {
    setPeakFactor(config.peakHourFactor || 1.0)
    const rates: Record<string, number> = {}
    for (const dept of DEPARTMENTS) {
      rates[dept] = config.departments?.[dept]?.callsPerMin ?? 0
    }
    setDeptRates(rates)
  }, [])

  // Poll status only (doesn't touch sliders)
  const pollStatus = useCallback(async () => {
    try {
      const token = await getToken()
      const status = await getSimStatus(token)
      setSimStatus(status)
    } catch {
      // silently ignore poll errors
    }
  }, [getToken])

  // Initial load: fetch both status + config and populate sliders once
  useEffect(() => {
    const init = async () => {
      try {
        const token = await getToken()
        const [status, config] = await Promise.all([
          getSimStatus(token),
          getCallConfig(token).catch(() => null),
        ])
        setSimStatus(status)
        if (config) {
          setCallConfig(config)
          syncSlidersFromConfig(config)
        }
        setConfigLoaded(true)
      } catch {
        setConfigLoaded(true)
      }
    }
    init()
  }, [getToken, syncSlidersFromConfig])

  // Poll only status every 2.5s (sliders are user-controlled)
  useEffect(() => {
    if (!configLoaded) return
    const interval = setInterval(pollStatus, 2500)
    return () => clearInterval(interval)
  }, [configLoaded, pollStatus])

  // Auto-revert confirm states after 3s
  useEffect(() => {
    if (confirmReset) {
      const t = setTimeout(() => setConfirmReset(false), 3000)
      return () => clearTimeout(t)
    }
  }, [confirmReset])

  useEffect(() => {
    if (confirmDynamo) {
      const t = setTimeout(() => setConfirmDynamo(false), 3000)
      return () => clearTimeout(t)
    }
  }, [confirmDynamo])

  const showStatus = (msg: string) => {
    setStatusMsg(msg)
    setTimeout(() => setStatusMsg(null), 3000)
  }

  // Re-fetch config from AgentSim and sync sliders (called after Apply)
  const refreshConfig = useCallback(async () => {
    try {
      const token = await getToken()
      const config = await getCallConfig(token)
      setCallConfig(config)
      syncSlidersFromConfig(config)
    } catch {
      // ignore
    }
  }, [getToken, syncSlidersFromConfig])

  const withAction = async (name: string, fn: () => Promise<void>, syncConfig = false) => {
    setActionLoading(name)
    try {
      await fn()
      await pollStatus()
      if (syncConfig) await refreshConfig()
    } catch (err) {
      showStatus(`Error: ${err instanceof Error ? err.message : 'Unknown'}`)
    } finally {
      setActionLoading(null)
    }
  }

  const handleStart = () => {
    withAction('start', async () => {
      const token = await getToken()
      await startSim(agentCount, token)
      showStatus(`Simulation started with ${agentCount} agents`)
    })
  }

  const handleStop = () => {
    withAction('stop', async () => {
      const token = await getToken()
      await stopSim(token)
      showStatus('Simulation stopped')
    })
  }

  const handleScale = () => {
    withAction('scale', async () => {
      const token = await getToken()
      await scaleSim(agentCount, token)
      showStatus(`Scaled to ${agentCount} agents`)
    })
  }

  const handleApplyCallConfig = () => {
    withAction('callConfig', async () => {
      const token = await getToken()
      const config: CallConfig = {
        peakHourFactor: peakFactor,
        departments: {},
      }
      for (const dept of DEPARTMENTS) {
        config.departments[dept] = { callsPerMin: deptRates[dept] || 0 }
      }
      await updateCallConfig(config, token)
      showStatus('Call config updated')
    }, true)
  }

  const handleInject = () => {
    withAction('inject', async () => {
      const token = await getToken()
      const result = await injectCalls(injectCount, injectVQ || null, token)
      showStatus(`Injected ${result.injected} calls${result.errors ? ` (${result.errors} errors)` : ''}`)
    })
  }

  const handleWipeCalls = () => {
    withAction('wipeCalls', async () => {
      const token = await getToken()
      await wipeAllCalls(token)
      showStatus('All calls wiped')
    })
  }

  const handleLogoffAll = () => {
    withAction('logoff', async () => {
      const token = await getToken()
      await logoffAllAgents(token)
      showStatus('All agents logged off')
    })
  }

  const handleResetMemory = () => {
    if (!confirmReset) {
      setConfirmReset(true)
      return
    }
    setConfirmReset(false)
    withAction('resetMemory', async () => {
      const token = await getToken()
      const result = await resetMemory(token)
      showStatus(`Memory reset: ${result.agentsCleared} agents, ${result.callsCleared} calls cleared`)
    })
  }

  const handleWipeDynamo = () => {
    if (!confirmDynamo) {
      setConfirmDynamo(true)
      return
    }
    setConfirmDynamo(false)
    withAction('wipeDynamo', async () => {
      const token = await getToken()
      await wipeDynamo(token)
      showStatus('DynamoDB tables wiped')
    })
  }

  const btnStyle = (bg: string, disabled = false): React.CSSProperties => ({
    padding: '8px 16px',
    borderRadius: '6px',
    border: 'none',
    fontSize: '12px',
    fontWeight: '600',
    cursor: disabled ? 'not-allowed' : 'pointer',
    backgroundColor: disabled ? colors.surfaceHover : bg,
    color: 'white',
    opacity: disabled ? 0.5 : 1,
  })

  const sectionStyle: React.CSSProperties = {
    backgroundColor: colors.background,
    borderRadius: '8px',
    padding: '16px',
    border: `1px solid ${colors.border}`,
  }

  const labelStyle: React.CSSProperties = {
    fontSize: '11px',
    fontWeight: '600',
    color: colors.textSecondary,
    textTransform: 'uppercase',
    letterSpacing: '0.5px',
    marginBottom: '8px',
  }

  const sliderRow = (label: string, value: number, min: number, max: number, step: number, onChange: (v: number) => void) => (
    <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '8px' }}>
      <span style={{ fontSize: '11px', color: colors.text, width: '100px', flexShrink: 0 }}>{label}</span>
      <input
        type="range"
        min={min}
        max={max}
        step={step}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        style={{ flex: 1 }}
      />
      <span style={{ fontSize: '12px', fontWeight: '700', color: colors.text, width: '50px', textAlign: 'right' }}>
        {step < 1 ? value.toFixed(1) : value}
      </span>
    </div>
  )

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
          backgroundColor: 'rgba(0, 0, 0, 0.7)',
          zIndex: 2000,
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
          width: '90vw',
          maxWidth: '900px',
          height: '90vh',
          maxHeight: '90vh',
          overflowY: 'auto',
          zIndex: 2001,
          boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.4)',
        }}
      >
        {/* Header */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
          <div>
            <h2 style={{ margin: 0, fontSize: '20px', fontWeight: '700', color: colors.text }}>
              Admin Control Panel
            </h2>
            {statusMsg && (
              <div style={{ fontSize: '11px', color: colors.primary, marginTop: '4px' }}>{statusMsg}</div>
            )}
          </div>
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

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
          {/* A. Simulation Control */}
          <div style={sectionStyle}>
            <div style={labelStyle}>Simulation Control</div>

            {/* Status Badge */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '12px' }}>
              <div
                style={{
                  width: '10px',
                  height: '10px',
                  borderRadius: '50%',
                  backgroundColor: simStatus?.running ? '#22c55e' : '#6b7280',
                }}
              />
              <span style={{ fontSize: '13px', fontWeight: '600', color: colors.text }}>
                {simStatus?.running ? 'Running' : 'Stopped'}
              </span>
              {simStatus && (
                <span style={{ fontSize: '11px', color: colors.textSecondary, marginLeft: 'auto' }}>
                  {simStatus.activeAgents} / {simStatus.totalAgents} agents
                </span>
              )}
            </div>

            {/* Agent Slider + Scale */}
            {sliderRow('Agents', agentCount, 0, 2000, 10, setAgentCount)}
            <button
              onClick={handleScale}
              disabled={actionLoading !== null || !simStatus?.running}
              style={btnStyle(colors.primary, actionLoading !== null || !simStatus?.running)}
            >
              {actionLoading === 'scale' ? 'Scaling...' : 'Scale'}
            </button>

            {simStatus?.startedAt && (
              <div style={{ fontSize: '10px', color: colors.textSecondary, marginTop: '8px' }}>
                Started: {new Date(simStatus.startedAt).toLocaleTimeString()}
              </div>
            )}
          </div>

          {/* B. Call Generation */}
          <div style={sectionStyle}>
            <div style={labelStyle}>Call Generation</div>

            {DEPARTMENTS.map((dept) => (
              <div key={dept}>
                {sliderRow(
                  dept.charAt(0).toUpperCase() + dept.slice(1),
                  deptRates[dept] || 0,
                  0,
                  200,
                  1,
                  (v) => setDeptRates((prev) => ({ ...prev, [dept]: v }))
                )}
              </div>
            ))}
            {sliderRow('Peak Factor', peakFactor, 0.1, 5.0, 0.1, setPeakFactor)}

            <div style={{ display: 'flex', gap: '8px', marginBottom: '12px' }}>
              <button
                onClick={handleApplyCallConfig}
                disabled={actionLoading !== null}
                style={btnStyle(colors.primary, actionLoading !== null)}
              >
                {actionLoading === 'callConfig' ? 'Applying...' : 'Apply Config'}
              </button>
            </div>

            {/* Inject Calls */}
            <div style={{ borderTop: `1px solid ${colors.border}`, paddingTop: '12px', marginTop: '4px' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
                <span style={{ fontSize: '11px', color: colors.text }}>Push</span>
                <input
                  type="number"
                  min={1}
                  max={1000}
                  value={injectCount}
                  onChange={(e) => setInjectCount(Math.max(1, Math.min(1000, Number(e.target.value))))}
                  style={{
                    width: '60px',
                    padding: '4px 8px',
                    borderRadius: '4px',
                    border: `1px solid ${colors.border}`,
                    fontSize: '11px',
                    backgroundColor: colors.surface,
                    color: colors.text,
                  }}
                />
                <span style={{ fontSize: '11px', color: colors.text }}>calls to</span>
                <select
                  value={injectVQ}
                  onChange={(e) => setInjectVQ(e.target.value)}
                  style={{
                    padding: '4px 8px',
                    borderRadius: '4px',
                    border: `1px solid ${colors.border}`,
                    fontSize: '11px',
                    backgroundColor: colors.surface,
                    color: colors.text,
                    flex: 1,
                  }}
                >
                  <option value="">All VQs (round-robin)</option>
                  {ALL_VQS.map((vq) => (
                    <option key={vq} value={vq}>
                      {vq}
                    </option>
                  ))}
                </select>
                <button
                  onClick={handleInject}
                  disabled={actionLoading !== null}
                  style={btnStyle(colors.primary, actionLoading !== null)}
                >
                  {actionLoading === 'inject' ? '...' : 'Push'}
                </button>
              </div>
            </div>

            {/* Clear All Calls */}
            <button
              onClick={handleWipeCalls}
              disabled={actionLoading !== null}
              style={btnStyle('#f59e0b', actionLoading !== null)}
            >
              {actionLoading === 'wipeCalls' ? 'Clearing...' : 'Clear All Calls'}
            </button>
          </div>

          {/* C. Agent Management */}
          <div style={sectionStyle}>
            <div style={labelStyle}>Agent Management</div>
            <button
              onClick={handleLogoffAll}
              disabled={actionLoading !== null}
              style={btnStyle('#f59e0b', actionLoading !== null)}
            >
              {actionLoading === 'logoff' ? 'Logging off...' : 'Log Off All Agents'}
            </button>
            <div style={{ fontSize: '10px', color: colors.textSecondary, marginTop: '6px' }}>
              Disconnects all agents. The simulation keeps running.
            </div>
          </div>

          {/* D. Danger Zone */}
          <div
            style={{
              ...sectionStyle,
              border: `2px solid #ef4444`,
            }}
          >
            <div style={{ ...labelStyle, color: '#ef4444' }}>Danger Zone</div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
              <div style={{ display: 'flex', gap: '8px' }}>
                <button
                  onClick={handleStart}
                  disabled={actionLoading !== null || simStatus?.running === true}
                  style={btnStyle('#22c55e', actionLoading !== null || simStatus?.running === true)}
                >
                  {actionLoading === 'start' ? 'Starting...' : 'Start Simulation'}
                </button>
                <button
                  onClick={handleStop}
                  disabled={actionLoading !== null || simStatus?.running === false}
                  style={btnStyle('#ef4444', actionLoading !== null || simStatus?.running === false)}
                >
                  {actionLoading === 'stop' ? 'Stopping...' : 'Stop Simulation'}
                </button>
              </div>
              <div style={{ fontSize: '10px', color: colors.textSecondary }}>
                Start or stop the entire simulation engine.
              </div>

              <div style={{ borderTop: `1px solid ${colors.border}`, paddingTop: '10px' }}>
                <button
                  onClick={handleResetMemory}
                  disabled={actionLoading !== null}
                  style={btnStyle(confirmReset ? '#dc2626' : '#ef4444', actionLoading !== null)}
                >
                  {actionLoading === 'resetMemory'
                    ? 'Resetting...'
                    : confirmReset
                      ? 'Confirm Reset?'
                      : 'Reset Backend Memory'}
                </button>
                <div style={{ fontSize: '10px', color: colors.textSecondary, marginTop: '4px' }}>
                  Clears all agent state and call queues from memory.
                </div>
              </div>

              <div>
                <button
                  onClick={handleWipeDynamo}
                  disabled={actionLoading !== null}
                  style={btnStyle(confirmDynamo ? '#dc2626' : '#ef4444', actionLoading !== null)}
                >
                  {actionLoading === 'wipeDynamo'
                    ? 'Wiping...'
                    : confirmDynamo
                      ? 'Confirm Wipe?'
                      : 'Wipe DynamoDB Tables'}
                </button>
                <div style={{ fontSize: '10px', color: colors.textSecondary, marginTop: '4px' }}>
                  Permanently deletes all call records and agent daily stats.
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </>
  )
}
