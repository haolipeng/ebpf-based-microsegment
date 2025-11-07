import { Card, Descriptions, Tag } from 'antd'
import type { Agent } from '../../types/agent'
import { formatRelativeTime } from '../../utils/format'

interface AgentInfoCardProps {
  agent: Agent
}

export default function AgentInfoCard({ agent }: AgentInfoCardProps) {
  const getStatusColor = (status: Agent['status']) => {
    switch (status) {
      case 'online':
        return 'success'
      case 'offline':
        return 'default'
      case 'error':
        return 'error'
      default:
        return 'default'
    }
  }

  const getStatusText = (status: Agent['status']) => {
    switch (status) {
      case 'online':
        return 'Online'
      case 'offline':
        return 'Offline'
      case 'error':
        return 'Error'
      default:
        return status
    }
  }

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp)
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    })
  }

  return (
    <Card title="Basic Information" bordered={false}>
      <Descriptions column={1} size="small">
        <Descriptions.Item label="Agent ID">
          <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{agent.id}</span>
        </Descriptions.Item>
        <Descriptions.Item label="Hostname">{agent.hostname}</Descriptions.Item>
        <Descriptions.Item label="IP Address">
          <span style={{ fontFamily: 'monospace' }}>{agent.ipAddress}</span>
        </Descriptions.Item>
        <Descriptions.Item label="Version">{agent.version}</Descriptions.Item>
        <Descriptions.Item label="Status">
          <Tag color={getStatusColor(agent.status)}>{getStatusText(agent.status)}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label="Start Time">{formatTimestamp(agent.startTime)}</Descriptions.Item>
        <Descriptions.Item label="Last Heartbeat">
          <div>
            <div>{formatTimestamp(agent.lastHeartbeat)}</div>
            <div style={{ color: '#8c8c8c', fontSize: 12, marginTop: 4 }}>
              {formatRelativeTime(agent.lastHeartbeat)}
            </div>
          </div>
        </Descriptions.Item>
      </Descriptions>
    </Card>
  )
}
