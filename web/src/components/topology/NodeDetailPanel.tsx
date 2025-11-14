import { Drawer, Descriptions, Table, Tag, Space, Typography, Card } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import type { TopologyNode } from '../../types/topology'
import type { Flow } from '../../types/flow'
import { formatBytes, formatNumber } from '../../utils/format'
import { protocolName } from '../../api/flows'

const { Text } = Typography

interface NodeDetailPanelProps {
  /** 选中的节点 */
  node: TopologyNode | null
  /** 与该节点相关的流 */
  flows: Flow[]
  /** 关闭回调 */
  onClose: () => void
}

/**
 * 节点详情面板组件
 * 
 * 使用Drawer抽屉显示节点的详细信息和相关流
 */
export default function NodeDetailPanel({ node, flows, onClose }: NodeDetailPanelProps) {
  // 流表格列定义
  const flowColumns: ColumnsType<Flow> = [
    {
      title: 'Source IP',
      dataIndex: 'sourceIp',
      key: 'sourceIp',
      width: 130,
      ellipsis: true,
    },
    {
      title: 'Source Port',
      dataIndex: 'sourcePort',
      key: 'sourcePort',
      width: 80,
    },
    {
      title: 'Dest IP',
      dataIndex: 'destIp',
      key: 'destIp',
      width: 130,
      ellipsis: true,
    },
    {
      title: 'Dest Port',
      dataIndex: 'destPort',
      key: 'destPort',
      width: 80,
    },
    {
      title: 'Protocol',
      dataIndex: 'protocol',
      key: 'protocol',
      width: 80,
      render: (protocol: string) => protocolName(protocol),
    },
    {
      title: 'State',
      dataIndex: 'state',
      key: 'state',
      width: 80,
      render: (state: string) => {
        const colorMap: Record<string, string> = {
          ACTIVE: 'green',
          CLOSED: 'default',
          TIMEOUT: 'orange',
        }
        return <Tag color={colorMap[state] || 'default'}>{state}</Tag>
      },
    },
    {
      title: 'Action',
      dataIndex: 'policyAction',
      key: 'policyAction',
      width: 80,
      render: (action: string) => {
        const colorMap: Record<string, string> = {
          ALLOW: 'green',
          DENY: 'red',
          LOG: 'blue',
        }
        return <Tag color={colorMap[action] || 'default'}>{action}</Tag>
      },
    },
    {
      title: 'Bytes',
      dataIndex: 'byteCount',
      key: 'byteCount',
      width: 100,
      render: (bytes: number) => formatBytes(bytes),
      sorter: (a, b) => a.byteCount - b.byteCount,
    },
  ]

  return (
    <Drawer
      title={
        <Space>
          <span>{node?.type === 'IP' ? '🖥️' : '⚙️'}</span>
          <span>Node Details</span>
        </Space>
      }
      placement="right"
      width={800}
      open={node !== null}
      onClose={onClose}
      destroyOnClose
    >
      {node && (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          {/* 节点基本信息 */}
          <Card title="Basic Information" size="small">
            <Descriptions column={1} bordered size="small">
              <Descriptions.Item label="Node ID">{node.label}</Descriptions.Item>
              <Descriptions.Item label="Node Type">
                {node.type === 'IP' ? 'IP Address' : 'Service'}
              </Descriptions.Item>
              {node.labels && Object.keys(node.labels).length > 0 && (
                <Descriptions.Item label="Labels">
                  {Object.entries(node.labels).map(([key, value]) => (
                    <Tag key={key} color="blue">
                      {key}: {value}
                    </Tag>
                  ))}
                </Descriptions.Item>
              )}
            </Descriptions>
          </Card>

          {/* 流量统计 */}
          <Card title="Traffic Statistics" size="small">
            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label="Total Flows">
                <Text strong>{formatNumber(node.metrics.flowCount)}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="Active Flows">
                <Text strong style={{ color: '#52c41a' }}>
                  {formatNumber(node.metrics.activeFlows)}
                </Text>
              </Descriptions.Item>
              <Descriptions.Item label="Total Packets">
                {formatNumber(node.metrics.packetCount)}
              </Descriptions.Item>
              <Descriptions.Item label="Total Bytes">
                {formatBytes(node.metrics.byteCount)}
              </Descriptions.Item>
            </Descriptions>
          </Card>

          {/* 相关流列表 */}
          <Card
            title={
              <Space>
                <span>Related Flows</span>
                <Tag color="blue">{flows.length} items</Tag>
              </Space>
            }
            size="small"
          >
            <Table
              columns={flowColumns}
              dataSource={flows}
              rowKey="id"
              size="small"
              pagination={{
                pageSize: 10,
                showSizeChanger: true,
                showTotal: (total) => `Total ${total} items`,
              }}
              scroll={{ x: 'max-content' }}
            />
          </Card>

          {/* 入站/出站统计 */}
          <Card title="Connection Statistics" size="small">
            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label="Inbound">
                <Text strong>
                  {formatNumber(
                    flows.filter((f) => f.destIp === node.id || f.destLabels?.app === node.label)
                      .length
                  )}
                </Text>
              </Descriptions.Item>
              <Descriptions.Item label="Outbound">
                <Text strong>
                  {formatNumber(
                    flows.filter((f) => f.sourceIp === node.id || f.sourceLabels?.app === node.label)
                      .length
                  )}
                </Text>
              </Descriptions.Item>
            </Descriptions>
          </Card>
        </Space>
      )}
    </Drawer>
  )
}

