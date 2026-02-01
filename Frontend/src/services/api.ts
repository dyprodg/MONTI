import { AgentDailyStats, CallRecord, SimStatus, CallConfig } from '../types'

// VITE_API_URL already includes /api (e.g. http://localhost:8080/api)
// Fallback strips it to keep paths consistent
const API_BASE = (import.meta.env.VITE_API_URL || 'http://localhost:8080/api').replace(/\/api\/?$/, '')

export const fetchAgentHistory = async (
  agentId: string,
  token: string | null
): Promise<AgentDailyStats[]> => {
  const headers: HeadersInit = token ? { Authorization: `Bearer ${token}` } : {}
  const res = await fetch(`${API_BASE}/api/agents/${encodeURIComponent(agentId)}/history`, {
    headers,
  })
  if (!res.ok) {
    throw new Error(`Failed to fetch agent history: ${res.statusText}`)
  }
  return res.json()
}

export const killAgentCall = async (
  agentId: string,
  callId: string,
  token: string | null
): Promise<void> => {
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  }
  const res = await fetch(
    `${API_BASE}/api/agents/${encodeURIComponent(agentId)}/calls/${encodeURIComponent(callId)}/end`,
    { method: 'POST', headers }
  )
  if (!res.ok) {
    throw new Error(`Failed to end call: ${res.statusText}`)
  }
}

export const logoutAgent = async (
  agentId: string,
  token: string | null
): Promise<void> => {
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  }
  const res = await fetch(
    `${API_BASE}/api/agents/${encodeURIComponent(agentId)}/logout`,
    { method: 'POST', headers }
  )
  if (!res.ok) {
    throw new Error(`Failed to logout agent: ${res.statusText}`)
  }
}

export const fetchAgentCalls = async (
  agentId: string,
  date: string,
  token: string | null
): Promise<CallRecord[]> => {
  const headers: HeadersInit = token ? { Authorization: `Bearer ${token}` } : {}
  const res = await fetch(
    `${API_BASE}/api/agents/${encodeURIComponent(agentId)}/calls?date=${encodeURIComponent(date)}`,
    { headers }
  )
  if (!res.ok) {
    throw new Error(`Failed to fetch agent calls: ${res.statusText}`)
  }
  return res.json()
}

// --- Admin API functions ---

const adminHeaders = (token: string | null): HeadersInit => ({
  'Content-Type': 'application/json',
  ...(token ? { Authorization: `Bearer ${token}` } : {}),
})

export const getSimStatus = async (token: string | null): Promise<SimStatus> => {
  const res = await fetch(`${API_BASE}/api/admin/sim/status`, { headers: adminHeaders(token) })
  if (!res.ok) throw new Error(`Failed to get sim status: ${res.statusText}`)
  return res.json()
}

export const startSim = async (activeAgents: number, token: string | null): Promise<void> => {
  const res = await fetch(`${API_BASE}/api/admin/sim/start`, {
    method: 'POST',
    headers: adminHeaders(token),
    body: JSON.stringify({ activeAgents }),
  })
  if (res.status === 409) return // already running
  if (!res.ok) throw new Error(`Failed to start sim: ${res.statusText}`)
}

export const stopSim = async (token: string | null): Promise<void> => {
  const res = await fetch(`${API_BASE}/api/admin/sim/stop`, {
    method: 'POST',
    headers: adminHeaders(token),
    body: '{}',
  })
  if (res.status === 409) return // already stopped
  if (!res.ok) throw new Error(`Failed to stop sim: ${res.statusText}`)
}

export const scaleSim = async (activeAgents: number, token: string | null): Promise<void> => {
  const res = await fetch(`${API_BASE}/api/admin/sim/scale`, {
    method: 'POST',
    headers: adminHeaders(token),
    body: JSON.stringify({ activeAgents }),
  })
  if (!res.ok) throw new Error(`Failed to scale sim: ${res.statusText}`)
}

export const getCallConfig = async (token: string | null): Promise<CallConfig> => {
  const res = await fetch(`${API_BASE}/api/admin/calls/config`, { headers: adminHeaders(token) })
  if (!res.ok) throw new Error(`Failed to get call config: ${res.statusText}`)
  return res.json()
}

export const updateCallConfig = async (config: CallConfig, token: string | null): Promise<void> => {
  const res = await fetch(`${API_BASE}/api/admin/calls/config`, {
    method: 'PUT',
    headers: adminHeaders(token),
    body: JSON.stringify(config),
  })
  if (!res.ok) throw new Error(`Failed to update call config: ${res.statusText}`)
}

export const injectCalls = async (count: number, vq: string | null, token: string | null): Promise<{ injected: number; errors: number }> => {
  const body: Record<string, unknown> = { count }
  if (vq) body.vq = vq
  const res = await fetch(`${API_BASE}/api/admin/calls/inject`, {
    method: 'POST',
    headers: adminHeaders(token),
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`Failed to inject calls: ${res.statusText}`)
  return res.json()
}

export const wipeAllCalls = async (token: string | null): Promise<void> => {
  const res = await fetch(`${API_BASE}/api/admin/calls/all`, {
    method: 'DELETE',
    headers: adminHeaders(token),
  })
  if (!res.ok) throw new Error(`Failed to wipe calls: ${res.statusText}`)
}

export const resetMemory = async (token: string | null): Promise<{ agentsCleared: number; callsCleared: number }> => {
  const res = await fetch(`${API_BASE}/api/admin/reset/memory`, {
    method: 'POST',
    headers: adminHeaders(token),
    body: '{}',
  })
  if (!res.ok) throw new Error(`Failed to reset memory: ${res.statusText}`)
  return res.json()
}

export const wipeDynamo = async (token: string | null): Promise<void> => {
  const res = await fetch(`${API_BASE}/api/admin/reset/dynamo`, {
    method: 'DELETE',
    headers: adminHeaders(token),
  })
  if (!res.ok) throw new Error(`Failed to wipe DynamoDB: ${res.statusText}`)
}

export const logoffAllAgents = async (token: string | null): Promise<void> => {
  const res = await fetch(`${API_BASE}/api/admin/agents/logoff-all`, {
    method: 'POST',
    headers: adminHeaders(token),
    body: '{}',
  })
  if (!res.ok) throw new Error(`Failed to logoff all agents: ${res.statusText}`)
}
