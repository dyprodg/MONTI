import { useRef, useState, useCallback, useEffect } from 'react'
import { Snapshot, PlaybackMode } from '../types'

const DEFAULT_MAX_SIZE = 300

export function useSnapshotBuffer(incoming: Snapshot | null, maxSize = DEFAULT_MAX_SIZE) {
  const bufferRef = useRef<Snapshot[]>([])
  const [mode, setMode] = useState<PlaybackMode>('live')
  const [cursorIndex, setCursorIndex] = useState(0)
  const [renderTick, setRenderTick] = useState(0)

  // Append incoming snapshot to buffer
  useEffect(() => {
    if (!incoming) return
    const buf = bufferRef.current

    buf.push(incoming)

    // Evict oldest if over max
    if (buf.length > maxSize) {
      buf.shift()
      // If paused/scrubbing, adjust cursor to keep pointing at same snapshot
      if (mode !== 'live') {
        setCursorIndex((prev) => Math.max(0, prev - 1))
      }
    }

    if (mode === 'live') {
      setCursorIndex(buf.length - 1)
      setRenderTick((t) => t + 1)
    }
  }, [incoming, maxSize, mode])

  // Seed buffer with history (bulk insert)
  const seedHistory = useCallback((snapshots: Snapshot[]) => {
    const buf = bufferRef.current
    // Only seed if buffer is empty or very small (avoid re-seeding)
    if (buf.length > 5) return

    // Prepend history before any existing snapshots
    const combined = [...snapshots, ...buf]
    // Trim to maxSize keeping the most recent
    const trimmed = combined.length > maxSize
      ? combined.slice(combined.length - maxSize)
      : combined

    bufferRef.current = trimmed

    if (mode === 'live') {
      setCursorIndex(trimmed.length > 0 ? trimmed.length - 1 : 0)
      setRenderTick((t) => t + 1)
    }
  }, [maxSize, mode])

  const pause = useCallback(() => {
    setMode('paused')
  }, [])

  const goLive = useCallback(() => {
    setMode('live')
    const buf = bufferRef.current
    setCursorIndex(buf.length > 0 ? buf.length - 1 : 0)
  }, [])

  const scrubTo = useCallback((index: number) => {
    const buf = bufferRef.current
    if (buf.length === 0) return

    const clamped = Math.max(0, Math.min(index, buf.length - 1))

    // If scrubbed to the end, go live
    if (clamped === buf.length - 1) {
      setMode('live')
    } else {
      setMode('scrubbing')
    }
    setCursorIndex(clamped)
  }, [])

  const buf = bufferRef.current
  const displaySnapshot = buf.length > 0 ? buf[Math.min(cursorIndex, buf.length - 1)] : null
  const displayTimestamp = displaySnapshot?.timestamp ?? null
  const latestTimestamp = buf.length > 0 ? buf[buf.length - 1].timestamp : null

  return {
    displaySnapshot,
    mode,
    bufferLength: buf.length,
    cursorIndex: Math.min(cursorIndex, Math.max(0, buf.length - 1)),
    displayTimestamp,
    latestTimestamp,
    pause,
    goLive,
    scrubTo,
    seedHistory,
    _renderTick: renderTick,
  }
}
