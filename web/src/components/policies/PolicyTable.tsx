import { Table, Tag, Switch, Button, Space, Popconfirm, Tooltip, Badge } from 'antd'
import { EditOutlined, DeleteOutlined, BarChartOutlined } from '@ant-design/icons'
import type { Policy, PolicyStats } from '../../types/policy'
import type { ColumnsType } from 'antd/es/table'

interface PolicyTableProps {
  policies: Policy[]
  policyStats?: Map<number, PolicyStats>
  loading?: boolean
  onEdit: (policy: Policy) => void
  onDelete: (ruleId: number) => void
  onToggleEnabled: (ruleId: number, enabled: boolean) => void
  onViewStats?: (ruleId: number) => void
  selectedRowKeys?: number[]
  onSelectionChange?: (selectedKeys: number[]) => void
}

export default function PolicyTable({
  policies,
  policyStats,
  loading = false,
  onEdit,
  onDelete,
  onToggleEnabled,
  onViewStats,
  selectedRowKeys = [],
  onSelectionChange,
}: PolicyTableProps) {
  const getProtocolText = (protocol: string) => {
    return protocol.toUpperCase()
  }

  const getActionColor = (action: Policy['action']) => {
    switch (action) {
      case 'allow':
        return 'success'
      case 'deny':
        return 'error'
      case 'log':
        return 'processing'
      default:
        return 'default'
    }
  }

  const getActionText = (action: Policy['action']) => {
    switch (action) {
      case 'allow':
        return 'Allow'
      case 'deny':
        return 'Deny'
      case 'log':
        return 'Log'
      default:
        return action
    }
  }

  const columns: ColumnsType<Policy> = [
    {
      title: 'Rule ID',
      dataIndex: 'ruleId',
      key: 'ruleId',
      width: 100,
      sorter: (a, b) => a.ruleId - b.ruleId,
    },
    {
      title: 'Source IP',
      dataIndex: 'srcIp',
      key: 'srcIp',
      width: 150,
      render: (ip: string) => <span style={{ fontFamily: 'monospace' }}>{ip}</span>,
    },
    {
      title: 'Dest IP',
      dataIndex: 'dstIp',
      key: 'dstIp',
      width: 150,
      render: (ip: string) => <span style={{ fontFamily: 'monospace' }}>{ip}</span>,
    },
    {
      title: 'Src Port',
      dataIndex: 'srcPort',
      key: 'srcPort',
      width: 100,
      render: (port: number) => (port === 0 ? 'Any' : port),
    },
    {
      title: 'Dest Port',
      dataIndex: 'dstPort',
      key: 'dstPort',
      width: 100,
      render: (port: number) => (port === 0 ? 'Any' : port),
    },
    {
      title: 'Protocol',
      dataIndex: 'protocol',
      key: 'protocol',
      width: 100,
      filters: [
        { text: 'TCP', value: 'tcp' },
        { text: 'UDP', value: 'udp' },
        { text: 'ICMP', value: 'icmp' },
        { text: 'Any', value: 'any' },
      ],
      onFilter: (value, record) => record.protocol === value,
      render: (protocol: string) => getProtocolText(protocol),
    },
    {
      title: 'Action',
      dataIndex: 'action',
      key: 'action',
      width: 100,
      filters: [
        { text: 'Allow', value: 'allow' },
        { text: 'Deny', value: 'deny' },
        { text: 'Log', value: 'log' },
      ],
      onFilter: (value, record) => record.action === value,
      render: (action: Policy['action']) => (
        <Tag color={getActionColor(action)}>{getActionText(action)}</Tag>
      ),
    },
    {
      title: 'Priority',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      sorter: (a, b) => a.priority - b.priority,
    },
    {
      title: 'Enabled',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 100,
      filters: [
        { text: 'Enabled', value: true },
        { text: 'Disabled', value: false },
      ],
      onFilter: (value, record) => record.enabled === value,
      render: (enabled: boolean, record) => (
        <Switch
          checked={enabled !== false}
          onChange={checked => onToggleEnabled(record.ruleId, checked)}
          size="small"
        />
      ),
    },
    {
      title: 'Hit Count',
      key: 'hitCount',
      width: 120,
      sorter: (a, b) => {
        const hitsA = policyStats?.get(a.ruleId)?.hitCount || 0
        const hitsB = policyStats?.get(b.ruleId)?.hitCount || 0
        return hitsA - hitsB
      },
      render: (_, record) => {
        const stats = policyStats?.get(record.ruleId)
        const hitCount = stats?.hitCount || 0
        const lastHit = stats?.lastHit

        return (
          <Tooltip
            title={
              lastHit
                ? `Last hit: ${new Date(lastHit).toLocaleString()}`
                : 'No hits recorded'
            }
          >
            <Badge
              count={hitCount}
              showZero
              overflowCount={999999}
              style={{
                backgroundColor: hitCount > 0 ? '#52c41a' : '#d9d9d9',
              }}
            />
          </Tooltip>
        )
      },
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 160,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          {onViewStats && (
            <Tooltip title="View Statistics">
              <Button
                type="link"
                icon={<BarChartOutlined />}
                onClick={() => onViewStats(record.ruleId)}
                size="small"
              />
            </Tooltip>
          )}
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => onEdit(record)}
            size="small"
          >
            Edit
          </Button>
          <Popconfirm
            title="Delete Policy"
            description={`Are you sure you want to delete policy ${record.ruleId}?`}
            onConfirm={() => onDelete(record.ruleId)}
            okText="Yes"
            cancelText="No"
          >
            <Button type="link" danger icon={<DeleteOutlined />} size="small">
              Delete
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  const rowSelection = onSelectionChange
    ? {
        selectedRowKeys,
        onChange: (selectedKeys: React.Key[]) => {
          onSelectionChange(selectedKeys as number[])
        },
        selections: [
          Table.SELECTION_ALL,
          Table.SELECTION_INVERT,
          Table.SELECTION_NONE,
        ],
      }
    : undefined

  return (
    <Table
      columns={columns}
      dataSource={policies}
      rowKey="ruleId"
      loading={loading}
      rowSelection={rowSelection}
      pagination={{
        pageSize: 50,
        showSizeChanger: true,
        pageSizeOptions: ['20', '50', '100', '200'],
        showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} policies`,
      }}
      scroll={{ x: 'max-content' }}
      expandable={{
        expandedRowRender: record => (
          <div style={{ margin: 0 }}>
            {record.description && (
              <p>
                <strong>Description:</strong> {record.description}
              </p>
            )}
            {record.createdAt && (
              <p>
                <strong>Created:</strong> {new Date(record.createdAt).toLocaleString()}
              </p>
            )}
            {record.updatedAt && (
              <p>
                <strong>Updated:</strong> {new Date(record.updatedAt).toLocaleString()}
              </p>
            )}
          </div>
        ),
        rowExpandable: () => true,
      }}
    />
  )
}
