import { ConnectionState } from '../types'
import { useTheme } from '../contexts/ThemeContext'

interface ConnectionStatusProps {
  state: ConnectionState
}

export const ConnectionStatus = ({ state }: ConnectionStatusProps) => {
  const { colors } = useTheme()

  const getStatusColor = () => {
    switch (state) {
      case ConnectionState.OPEN:
        return '#22c55e' // green
      case ConnectionState.CONNECTING:
        return '#eab308' // yellow
      case ConnectionState.ERROR:
        return '#ef4444' // red
      case ConnectionState.CLOSED:
        return '#6b7280' // gray
      default:
        return '#6b7280'
    }
  }

  const getStatusText = () => {
    switch (state) {
      case ConnectionState.OPEN:
        return 'Connected'
      case ConnectionState.CONNECTING:
        return 'Connecting...'
      case ConnectionState.ERROR:
        return 'Error'
      case ConnectionState.CLOSED:
        return 'Disconnected'
      default:
        return 'Unknown'
    }
  }

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '8px',
        padding: '8px 16px',
        backgroundColor: colors.surfaceHover,
        borderRadius: '8px',
        fontSize: '14px',
        fontWeight: '500',
        color: colors.text,
      }}
    >
      <div
        style={{
          width: '8px',
          height: '8px',
          borderRadius: '50%',
          backgroundColor: getStatusColor(),
        }}
      />
      <span>{getStatusText()}</span>
    </div>
  )
}
