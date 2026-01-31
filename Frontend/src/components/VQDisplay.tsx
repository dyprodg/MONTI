import { VQSnapshot, Department } from '../types'
import { useTheme } from '../contexts/ThemeContext'

interface VQDisplayProps {
  department: Department
  queues: VQSnapshot[]
}

const getSLColor = (sl: number): string => {
  if (sl >= 80) return '#22c55e' // green
  if (sl >= 60) return '#eab308' // yellow
  return '#ef4444' // red
}

const formatVQName = (vq: string): string => {
  // Convert "sales_inbound" -> "Inbound", "tech_l1" -> "L1", etc.
  const parts = vq.split('_')
  if (parts.length < 2) return vq
  const suffix = parts.slice(1).join(' ')
  return suffix.charAt(0).toUpperCase() + suffix.slice(1)
}

export const VQDisplay = ({ department, queues }: VQDisplayProps) => {
  const { colors } = useTheme()

  return (
    <div
      style={{
        backgroundColor: colors.surface,
        borderRadius: '8px',
        padding: '8px',
        boxShadow: '0 2px 4px rgb(0 0 0 / 0.1)',
        minWidth: 0,
        overflow: 'hidden',
      }}
    >
      <h3
        style={{
          fontSize: '13px',
          fontWeight: '600',
          color: colors.text,
          margin: '0 0 8px 0',
          textTransform: 'capitalize',
        }}
      >
        {department} Queues
      </h3>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '1fr 1fr',
          gap: '6px',
        }}
      >
        {queues.map((q) => {
          const slColor = getSLColor(q.serviceLevel.currentSL)
          return (
            <div
              key={q.vq}
              style={{
                backgroundColor: colors.background,
                borderRadius: '6px',
                padding: '8px',
                border: `1px solid ${colors.border}`,
              }}
            >
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  marginBottom: '4px',
                }}
              >
                <span
                  style={{
                    fontSize: '11px',
                    fontWeight: '600',
                    color: colors.text,
                  }}
                >
                  {formatVQName(q.vq)}
                </span>
                <span
                  style={{
                    fontSize: '12px',
                    fontWeight: '700',
                    color: slColor,
                  }}
                >
                  {q.serviceLevel.currentSL.toFixed(0)}%
                </span>
              </div>
              <div
                style={{
                  display: 'flex',
                  gap: '8px',
                  fontSize: '10px',
                  color: colors.textSecondary,
                }}
              >
                <span>W:{q.waitingCount}</span>
                <span>A:{q.activeCount}</span>
                <span>
                  {q.longestWaitSecs > 0
                    ? `${Math.round(q.longestWaitSecs)}s`
                    : '-'}
                </span>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
