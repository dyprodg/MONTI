import { useRef, useState, useCallback, useEffect } from 'react'
import { Snapshot, PlaybackMode } from '../types'

const DEFAULT_MAX_SIZE = 120

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
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    _renderTick: renderTick,
  }
}
