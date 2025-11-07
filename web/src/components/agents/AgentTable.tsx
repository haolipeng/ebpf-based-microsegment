import { Table, Tag, Button, Space } from 'antd'
import { EyeOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import type { Agent } from '../../types/agent'
import { formatRelativeTime } from '../../utils/format'
import type { ColumnsType } from 'antd/es/table'

interface AgentTableProps {
  agents: Agent[]
  loading?: boolean
}

export default function AgentTable({ agents, loading = false }: AgentTableProps) {
  const navigate = useNavigate()

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

  const columns: ColumnsType<Agent> = [
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
      title: 'Hostname',
      dataIndex: 'hostname',
      key: 'hostname',
      sorter: (a, b) => a.hostname.localeCompare(b.hostname),
    },
    {
      title: 'IP Address',
      dataIndex: 'ipAddress',
      key: 'ipAddress',
      width: 140,
      render: (ip: string) => <span style={{ fontFamily: 'monospace' }}>{ip}</span>,
    },
    {
      title: 'Version',
      dataIndex: 'version',
      key: 'version',
      width: 100,
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      filters: [
        { text: 'Online', value: 'online' },
        { text: 'Offline', value: 'offline' },
        { text: 'Error', value: 'error' },
      ],
      onFilter: (value, record) => record.status === value,
      render: (status: Agent['status']) => (
        <Tag color={getStatusColor(status)}>{getStatusText(status)}</Tag>
      ),
    },
    {
      title: 'Last Heartbeat',
      dataIndex: 'lastHeartbeat',
      key: 'lastHeartbeat',
      width: 150,
      sorter: (a, b) =>
        new Date(a.lastHeartbeat).getTime() - new Date(b.lastHeartbeat).getTime(),
      render: (time: string) => formatRelativeTime(time),
    },
    {
      title: 'Action',
      key: 'action',
      width: 100,
      fixed: 'right',
      render: (_, record) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => navigate(`/agents/${record.id}`)}
          >
            Details
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <Table
      columns={columns}
      dataSource={agents}
      rowKey="id"
      loading={loading}
      pagination={{
        pageSize: 10,
        showSizeChanger: true,
        showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} agents`,
      }}
      scroll={{ x: 'max-content' }}
    />
  )
}
