import { AgentGrid } from './AgentGrid'
import { VQDisplay } from './VQDisplay'
import { AgentInfo, VQSnapshot, Department } from '../types'
import { useTheme } from '../contexts/ThemeContext'

interface DepartmentCellProps {
  department: Department
  agents: AgentInfo[]
  queues: VQSnapshot[]
  onAgentClick: (agent: AgentInfo) => void
  showOffline?: boolean
}

export const DepartmentCell = ({ department, agents, queues, onAgentClick, showOffline = true }: DepartmentCellProps) => {
  const { colors } = useTheme()

  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: '1fr 220px',
        gap: '8px',
        minHeight: 0,
        overflow: 'hidden',
      }}
    >
      {/* Agent Grid */}
      <div
        style={{
          backgroundColor: colors.surface,
          borderRadius: '6px',
          padding: '8px',
          boxShadow: '0 1px 3px rgb(0 0 0 / 0.1)',
          display: 'flex',
          flexDirection: 'column',
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
          {department} ({agents.length})
        </div>
        <AgentGrid agents={agents} onAgentClick={onAgentClick} compact showOffline={showOffline} />
      </div>

      {/* VQ Column */}
      <div style={{ minHeight: 0, overflow: 'auto' }}>
        <VQDisplay department={department} queues={queues} />
      </div>
    </div>
  )
}
