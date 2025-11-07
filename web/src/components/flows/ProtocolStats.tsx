import { Card, List, Tag, Progress } from 'antd'
import type { ProtocolStats } from '../../types/flow'
import { formatBytes, formatNumber } from '../../utils/format'

interface ProtocolStatsProps {
  protocols: ProtocolStats[]
  loading?: boolean
}

export default function ProtocolStatsComponent({ protocols, loading = false }: ProtocolStatsProps) {
  // Calculate total for percentage
  const totalBytes = protocols.reduce((sum, p) => sum + p.byteCount, 0)

  const getProtocolColor = (protocol: string) => {
    switch (protocol.toUpperCase()) {
      case 'TCP':
        return 'blue'
      case 'UDP':
        return 'green'
      case 'ICMP':
        return 'orange'
      default:
        return 'default'
    }
  }

  return (
    <Card title="Top Protocols" bordered={false} loading={loading}>
      <List
        itemLayout="horizontal"
        dataSource={protocols}
        renderItem={item => {
          const percentage = totalBytes > 0 ? (item.byteCount / totalBytes) * 100 : 0
          return (
            <List.Item>
              <List.Item.Meta
                avatar={<Tag color={getProtocolColor(item.protocol)}>{item.protocol}</Tag>}
                title={
                  <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                    <span>{formatNumber(item.flowCount)} flows</span>
                    <span>{formatBytes(item.byteCount)}</span>
                  </div>
                }
                description={
                  <Progress
                    percent={Math.round(percentage)}
                    strokeColor={
                      item.protocol.toUpperCase() === 'TCP'
                        ? '#1890ff'
                        : item.protocol.toUpperCase() === 'UDP'
                          ? '#52c41a'
                          : '#faad14'
                    }
                    showInfo={false}
                  />
                }
              />
            </List.Item>
          )
        }}
      />
    </Card>
  )
}
