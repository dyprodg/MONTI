import { useTheme } from '../contexts/ThemeContext'
import { PlaybackMode } from '../types'

interface TimelineControlsProps {
  mode: PlaybackMode
  bufferLength: number
  cursorIndex: number
  displayTimestamp: string | null
  latestTimestamp: string | null
  onPause: () => void
  onGoLive: () => void
  onScrub: (index: number) => void
}

function formatTime(iso: string): string {
  try {
    const d = new Date(iso)
    return d.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  } catch {
    return '--:--:--'
  }
}

function secondsBehind(display: string | null, latest: string | null): number | null {
  if (!display || !latest) return null
  try {
    const diff = (new Date(latest).getTime() - new Date(display).getTime()) / 1000
    return Math.max(0, Math.round(diff))
  } catch {
    return null
  }
}

export const TimelineControls = ({
  mode,
  bufferLength,
  cursorIndex,
  displayTimestamp,
  latestTimestamp,
  onPause,
  onGoLive,
  onScrub,
}: TimelineControlsProps) => {
  const { colors } = useTheme()
  const isLive = mode === 'live'
  const disabled = bufferLength <= 1

  const behind = secondsBehind(displayTimestamp, latestTimestamp)
  const timeLabel = displayTimestamp
    ? `${formatTime(displayTimestamp)}${!isLive && behind != null && behind > 0 ? ` (-${behind}s)` : ''}`
    : '--:--:--'

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '10px',
        backgroundColor: colors.surface,
        padding: '4px 12px',
        borderRadius: '6px',
        boxShadow: '0 1px 3px rgb(0 0 0 / 0.1)',
        marginTop: '4px',
        borderTop: !isLive ? '2px solid #d97706' : '2px solid transparent',
      }}
    >
      {/* Pause / Play */}
      <button
        onClick={isLive ? onPause : onGoLive}
        disabled={disabled}
        title={isLive ? 'Pause' : 'Resume live'}
        style={{
          width: '28px',
          height: '28px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          border: `1px solid ${colors.border}`,
          borderRadius: '4px',
          backgroundColor: colors.surfaceHover,
          color: colors.text,
          fontSize: '14px',
          cursor: disabled ? 'default' : 'pointer',
          opacity: disabled ? 0.4 : 1,
        }}
      >
        {isLive ? '\u23F8' : '\u25B6'}
      </button>

      {/* Range slider */}
      <input
        type="range"
        min={0}
        max={Math.max(0, bufferLength - 1)}
        value={cursorIndex}
        disabled={disabled}
        onChange={(e) => onScrub(Number(e.target.value))}
        style={{
          flex: 1,
          height: '4px',
          cursor: disabled ? 'default' : 'pointer',
          accentColor: colors.primary,
        }}
      />

      {/* LIVE pill */}
      <button
        onClick={onGoLive}
        disabled={disabled}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: '5px',
          padding: '3px 10px',
          borderRadius: '12px',
          border: 'none',
          fontSize: '11px',
          fontWeight: '700',
          cursor: disabled ? 'default' : 'pointer',
          backgroundColor: isLive ? '#16a34a' : colors.surfaceHover,
          color: isLive ? '#fff' : colors.textSecondary,
          transition: 'all 0.2s',
          opacity: disabled ? 0.4 : 1,
        }}
      >
        <span
          style={{
            width: '6px',
            height: '6px',
            borderRadius: '50%',
            backgroundColor: isLive ? '#fff' : colors.textSecondary,
            display: 'inline-block',
            animation: isLive ? 'pulse-dot 1.5s ease-in-out infinite' : 'none',
          }}
        />
        LIVE
      </button>

      {/* Timestamp */}
      <span
        style={{
          fontSize: '11px',
          fontWeight: '500',
          color: isLive ? colors.textSecondary : '#d97706',
          fontVariantNumeric: 'tabular-nums',
          minWidth: '110px',
          textAlign: 'right',
        }}
      >
        {timeLabel}
      </span>

      {/* Pulse animation */}
      <style>{`
        @keyframes pulse-dot {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.3; }
        }
      `}</style>
    </div>
  )
}
