import { Card, List, Tag, Segmented } from 'antd'
import { useState } from 'react'
import type { IpStats } from '../../types/flow'
import { formatBytes, formatNumber } from '../../utils/format'

interface TopTalkersListProps {
  sourceIps?: IpStats[]
  destIps?: IpStats[]
  loading?: boolean
}

type ViewMode = 'source' | 'destination'

export default function TopTalkersList({
  sourceIps = [],
  destIps = [],
  loading = false,
}: TopTalkersListProps) {
  const [viewMode, setViewMode] = useState<ViewMode>('source')

  const currentData = viewMode === 'source' ? sourceIps : destIps

  return (
    <Card
      title="Top Talkers"
      loading={loading}
      extra={
        <Segmented
          options={[
            { label: 'Source IPs', value: 'source' },
            { label: 'Destination IPs', value: 'destination' },
          ]}
          value={viewMode}
          onChange={value => setViewMode(value as ViewMode)}
        />
      }
    >
      {currentData.length > 0 ? (
        <List
          dataSource={currentData}
          renderItem={(item, index) => (
            <List.Item>
              <List.Item.Meta
                avatar={
                  <Tag color={index < 3 ? 'red' : 'default'} style={{ fontSize: 16, padding: '4px 8px' }}>
                    #{index + 1}
                  </Tag>
                }
                title={<span style={{ fontFamily: 'monospace', fontSize: 14 }}>{item.ip}</span>}
                description={
                  <div>
                    <span style={{ marginRight: 16 }}>
                      <strong>Bytes:</strong> {formatBytes(item.byteCount)}
                    </span>
                    <span style={{ marginRight: 16 }}>
                      <strong>Packets:</strong> {formatNumber(item.packetCount)}
                    </span>
                    <span>
                      <strong>Flows:</strong> {formatNumber(item.flowCount)}
                    </span>
                  </div>
                }
              />
            </List.Item>
          )}
        />
      ) : (
        <div style={{ textAlign: 'center', padding: '40px 0', color: '#999' }}>
          No top talkers data available
        </div>
      )}
    </Card>
  )
}
