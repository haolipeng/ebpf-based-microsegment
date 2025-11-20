import { Card, Row, Col, Table, Tag } from 'antd'
import {
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts'
import type { ProcessStats, ContainerStats } from '../../types/flow'
import { formatBytes, formatNumber } from '../../utils/format'
import type { ColumnsType } from 'antd/es/table'

interface ProcessStatsProps {
  processStats: ProcessStats[]
  containerStats?: ContainerStats[]
  loading?: boolean
}

const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042', '#8884D8', '#82ca9d']

export default function ProcessStatsComponent({
  processStats,
  containerStats,
  loading = false,
}: ProcessStatsProps) {
  // Top processes by bandwidth for bar chart
  const topProcessesByBandwidth = [...processStats]
    .sort((a, b) => b.byteCount - a.byteCount)
    .slice(0, 10)
    .map(p => ({
      name: p.processName,
      bandwidth: p.byteCount,
      connections: p.connectionCount,
    }))

  // Process distribution by connection count for pie chart
  const processByConnections = [...processStats]
    .sort((a, b) => b.connectionCount - a.connectionCount)
    .slice(0, 8)
    .map(p => ({
      name: p.processName,
      value: p.connectionCount,
    }))

  // Process table columns
  const processColumns: ColumnsType<ProcessStats> = [
    {
      title: 'Process Name',
      dataIndex: 'processName',
      key: 'processName',
      width: 200,
      render: (name: string, record) => (
        <div>
          <span style={{ fontWeight: 500 }}>{name}</span>
          {record.isSuspicious && (
            <Tag color="error" style={{ marginLeft: 8 }}>
              Suspicious
            </Tag>
          )}
        </div>
      ),
    },
    {
      title: 'Executable Path',
      dataIndex: 'processPath',
      key: 'processPath',
      ellipsis: true,
      render: (path?: string) => (
        <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{path || '-'}</span>
      ),
    },
    {
      title: 'Flows',
      dataIndex: 'flowCount',
      key: 'flowCount',
      width: 100,
      align: 'right',
      sorter: (a, b) => a.flowCount - b.flowCount,
      render: (count: number) => formatNumber(count),
    },
    {
      title: 'Connections',
      dataIndex: 'connectionCount',
      key: 'connectionCount',
      width: 120,
      align: 'right',
      sorter: (a, b) => a.connectionCount - b.connectionCount,
      render: (count: number) => formatNumber(count),
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
      title: 'Bandwidth',
      dataIndex: 'byteCount',
      key: 'byteCount',
      width: 120,
      align: 'right',
      sorter: (a, b) => a.byteCount - b.byteCount,
      render: (bytes: number) => formatBytes(bytes),
    },
  ]

  // Container table columns
  const containerColumns: ColumnsType<ContainerStats> = [
    {
      title: 'Container ID',
      dataIndex: 'containerId',
      key: 'containerId',
      ellipsis: true,
      render: (id: string) => (
        <span style={{ fontFamily: 'monospace', fontSize: 11 }} title={id}>
          {id.slice(0, 12)}
        </span>
      ),
    },
    {
      title: 'Processes',
      dataIndex: 'processCount',
      key: 'processCount',
      width: 100,
      align: 'right',
      sorter: (a, b) => a.processCount - b.processCount,
      render: (count: number) => formatNumber(count),
    },
    {
      title: 'Flows',
      dataIndex: 'flowCount',
      key: 'flowCount',
      width: 100,
      align: 'right',
      sorter: (a, b) => a.flowCount - b.flowCount,
      render: (count: number) => formatNumber(count),
    },
    {
      title: 'Bandwidth',
      dataIndex: 'byteCount',
      key: 'byteCount',
      width: 120,
      align: 'right',
      sorter: (a, b) => a.byteCount - b.byteCount,
      render: (bytes: number) => formatBytes(bytes),
    },
  ]

  return (
    <div>
      <Row gutter={[16, 16]}>
        {/* Top Processes by Bandwidth Chart */}
        <Col xs={24} lg={12}>
          <Card title="Top Processes by Bandwidth" loading={loading}>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={topProcessesByBandwidth}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis
                  dataKey="name"
                  angle={-45}
                  textAnchor="end"
                  height={100}
                  interval={0}
                />
                <YAxis tickFormatter={value => formatBytes(value)} />
                <Tooltip formatter={(value: number) => formatBytes(value)} />
                <Legend />
                <Bar dataKey="bandwidth" fill="#8884d8" name="Bandwidth" />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>

        {/* Process Distribution by Connections Chart */}
        <Col xs={24} lg={12}>
          <Card title="Connection Distribution by Process" loading={loading}>
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie
                  data={processByConnections}
                  cx="50%"
                  cy="50%"
                  labelLine={false}
                  label={entry => `${entry.name}: ${entry.value}`}
                  outerRadius={100}
                  fill="#8884d8"
                  dataKey="value"
                >
                  {processByConnections.map((_, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </Card>
        </Col>
      </Row>

      {/* Process Statistics Table */}
      <Card title="Process Statistics" style={{ marginTop: 16 }}>
        <Table
          columns={processColumns}
          dataSource={processStats}
          rowKey="processName"
          loading={loading}
          pagination={{
            pageSize: 20,
            showSizeChanger: true,
            pageSizeOptions: ['10', '20', '50', '100'],
            showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} processes`,
          }}
        />
      </Card>

      {/* Container Statistics Table */}
      {containerStats && containerStats.length > 0 && (
        <Card title="Container Statistics" style={{ marginTop: 16 }}>
          <Table
            columns={containerColumns}
            dataSource={containerStats}
            rowKey="containerId"
            loading={loading}
            pagination={{
              pageSize: 20,
              showSizeChanger: true,
              pageSizeOptions: ['10', '20', '50'],
              showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} containers`,
            }}
          />
        </Card>
      )}
    </div>
  )
}
