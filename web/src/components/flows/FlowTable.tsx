import { Table, Tag } from 'antd'
import type { Flow } from '../../types/flow'
import { formatBytes, formatNumber, formatRelativeTime } from '../../utils/format'
import type { ColumnsType } from 'antd/es/table'
import '../../styles/flows.css'

interface FlowTableProps {
  flows: Flow[]
  loading?: boolean
  highlightedIds?: Set<string>
}

export default function FlowTable({
  flows,
  loading = false,
  highlightedIds,
}: FlowTableProps) {
  const getStateColor = (state: Flow['state']) => {
    switch (state) {
      case 'ACTIVE':
        return 'success'
      case 'CLOSED':
        return 'default'
      case 'TIMEOUT':
        return 'warning'
      default:
        return 'default'
    }
  }

  const getStateText = (state: Flow['state']) => {
    switch (state) {
      case 'ACTIVE':
        return 'Active'
      case 'CLOSED':
        return 'Closed'
      case 'TIMEOUT':
        return 'Timeout'
      default:
        return state
    }
  }

  const getActionColor = (action: Flow['policyAction']) => {
    switch (action) {
      case 'ALLOW':
        return 'success'
      case 'DENY':
        return 'error'
      case 'LOG':
        return 'processing'
      default:
        return 'default'
    }
  }

  const columns: ColumnsType<Flow> = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 120,
      ellipsis: true,
      render: (id: string) => (
        <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{id.slice(0, 8)}</span>
      ),
    },
    {
      title: 'Source',
      key: 'source',
      width: 180,
      render: (_, record) => (
        <span style={{ fontFamily: 'monospace' }}>
          {record.sourceIp}:{record.sourcePort}
        </span>
      ),
    },
    {
      title: 'Destination',
      key: 'destination',
      width: 180,
      render: (_, record) => (
        <span style={{ fontFamily: 'monospace' }}>
          {record.destIp}:{record.destPort}
        </span>
      ),
    },
    {
      title: 'Protocol',
      dataIndex: 'protocol',
      key: 'protocol',
      width: 100,
    },
    {
      title: 'State',
      dataIndex: 'state',
      key: 'state',
      width: 100,
      filters: [
        { text: 'Active', value: 'ACTIVE' },
        { text: 'Closed', value: 'CLOSED' },
        { text: 'Timeout', value: 'TIMEOUT' },
      ],
      onFilter: (value, record) => record.state === value,
      render: (state: Flow['state']) => (
        <Tag color={getStateColor(state)}>{getStateText(state)}</Tag>
      ),
    },
    {
      title: 'Action',
      dataIndex: 'policyAction',
      key: 'policyAction',
      width: 100,
      filters: [
        { text: 'Allow', value: 'ALLOW' },
        { text: 'Deny', value: 'DENY' },
        { text: 'Log', value: 'LOG' },
      ],
      onFilter: (value, record) => record.policyAction === value,
      render: (action: Flow['policyAction']) => (
        <Tag color={getActionColor(action)}>{action}</Tag>
      ),
    },
    {
      title: 'Packets',
      dataIndex: 'packetCount',
      key: 'packetCount',
      width: 100,
      align: 'right',
      sorter: (a, b) => a.packetCount - b.packetCount,
      render: (count: number) => formatNumber(count),
    },
    {
      title: 'Bytes',
      dataIndex: 'byteCount',
      key: 'byteCount',
      width: 120,
      align: 'right',
      sorter: (a, b) => a.byteCount - b.byteCount,
      render: (bytes: number) => formatBytes(bytes),
    },
    {
      title: 'Duration',
      dataIndex: 'durationMs',
      key: 'durationMs',
      width: 100,
      align: 'right',
      render: (ms: number) => {
        if (ms < 1000) return `${ms}ms`
        if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
        return `${(ms / 60000).toFixed(1)}m`
      },
    },
    {
      title: 'Start Time',
      dataIndex: 'startTime',
      key: 'startTime',
      width: 150,
      sorter: (a, b) => new Date(a.startTime).getTime() - new Date(b.startTime).getTime(),
      render: (time: string) => formatRelativeTime(time),
    },
  ]

  return (
    <Table
      columns={columns}
      dataSource={flows}
      rowKey="id"
      loading={loading}
      rowClassName={record =>
        highlightedIds?.has(record.id) ? 'flow-row-highlighted' : ''
      }
      pagination={{
        pageSize: 50,
        showSizeChanger: true,
        pageSizeOptions: ['20', '50', '100', '200'],
        showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} flows`,
      }}
      scroll={{ x: 'max-content' }}
      expandable={{
        expandedRowRender: record => (
          <div style={{ margin: 0 }}>
            <p>
              <strong>Direction:</strong> {record.direction}
            </p>
            <p>
              <strong>Duration:</strong> {record.durationMs}ms
            </p>
            {record.sourceLabels && Object.keys(record.sourceLabels).length > 0 && (
              <p>
                <strong>Source Labels:</strong>{' '}
                {Object.entries(record.sourceLabels).map(([key, value]) => (
                  <Tag key={key} style={{ marginRight: 4 }}>
                    {key}={value}
                  </Tag>
                ))}
              </p>
            )}
            {record.destLabels && Object.keys(record.destLabels).length > 0 && (
              <p>
                <strong>Destination Labels:</strong>{' '}
                {Object.entries(record.destLabels).map(([key, value]) => (
                  <Tag key={key} style={{ marginRight: 4 }}>
                    {key}={value}
                  </Tag>
                ))}
              </p>
            )}
            {record.policyId && (
              <p>
                <strong>Policy ID:</strong> {record.policyId}
              </p>
            )}
            <p>
              <strong>Last Seen:</strong> {formatRelativeTime(record.lastSeen)}
            </p>
          </div>
        ),
        rowExpandable: () => true,
      }}
    />
  )
}
