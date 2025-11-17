/**
 * Session Detail Panel
 *
 * Displays detailed information about selected nodes or edges
 * Inspired by NeuVector's conversation detail view
 */

import { Card, Descriptions, Tag, Empty, Divider, Statistic, Row, Col } from 'antd';
import {
  NodeIndexOutlined,
  SwapOutlined,
  CloudServerOutlined,
  DatabaseOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
} from '@ant-design/icons';
import type { TopologyNode, TopologyEdge } from '@/types/topology';
import type { Flow } from '@/types/flow';
import { formatBytes, formatNumber } from '@/utils/format';
import { useMemo } from 'react';

interface SessionDetailProps {
  /** Selected node (if any) */
  selectedNode?: TopologyNode | null;
  /** Selected edge (if any) */
  selectedEdge?: TopologyEdge | null;
  /** Related flows */
  relatedFlows?: Flow[];
  /** Node detail statistics */
  nodeStats?: {
    inboundConnections: number;
    outboundConnections: number;
    totalTraffic: number;
  } | null;
  /** Connected neighbors */
  neighbors?: string[];
  /** Panel width */
  width?: number;
  /** Panel style */
  style?: React.CSSProperties;
}

/**
 * Session Detail Panel Component
 */
export default function SessionDetail({
  selectedNode,
  selectedEdge,
  relatedFlows = [],
  nodeStats,
  neighbors = [],
  width = 400,
  style,
}: SessionDetailProps) {
  // No selection
  if (!selectedNode && !selectedEdge) {
    return (
      <Card
        style={{ width, ...style }}
        title="Session Details"
        bordered={false}
      >
        <Empty
          description="Select a node or connection to view details"
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      </Card>
    );
  }

  // Node selected
  if (selectedNode) {
    return <NodeDetail node={selectedNode} stats={nodeStats} neighbors={neighbors} flows={relatedFlows} width={width} style={style} />;
  }

  // Edge selected
  if (selectedEdge) {
    return <EdgeDetail edge={selectedEdge} flows={relatedFlows} width={width} style={style} />;
  }

  return null;
}

/**
 * Node Detail View
 */
function NodeDetail({
  node,
  stats,
  neighbors,
  flows,
  width,
  style,
}: {
  node: TopologyNode;
  stats?: {
    inboundConnections: number;
    outboundConnections: number;
    totalTraffic: number;
  } | null;
  neighbors: string[];
  flows: Flow[];
  width: number;
  style?: React.CSSProperties;
}) {
  // Calculate protocol distribution
  const protocolStats = useMemo(() => {
    const protocols = new Map<string, number>();
    for (const flow of flows) {
      protocols.set(flow.protocol, (protocols.get(flow.protocol) || 0) + 1);
    }
    return Array.from(protocols.entries())
      .sort((a, b) => b[1] - a[1])
      .slice(0, 5);
  }, [flows]);

  // Calculate state distribution
  const stateStats = useMemo(() => {
    const states = new Map<string, number>();
    for (const flow of flows) {
      states.set(flow.state, (states.get(flow.state) || 0) + 1);
    }
    return states;
  }, [flows]);

  return (
    <Card
      style={{ width, ...style }}
      title={
        <span>
          {node.type === 'IP' ? <CloudServerOutlined /> : <DatabaseOutlined />}{' '}
          Node Details
        </span>
      }
      bordered={false}
    >
      {/* Basic Information */}
      <Descriptions column={1} size="small" bordered>
        <Descriptions.Item label="ID">{node.id}</Descriptions.Item>
        <Descriptions.Item label="Label">{node.label}</Descriptions.Item>
        <Descriptions.Item label="Type">
          <Tag color={node.type === 'IP' ? 'blue' : 'green'}>{node.type}</Tag>
        </Descriptions.Item>
      </Descriptions>

      {/* Labels (if in service view) */}
      {node.labels && Object.keys(node.labels).length > 0 && (
        <>
          <Divider orientation="left" plain>
            Labels
          </Divider>
          <div style={{ marginBottom: 16 }}>
            {Object.entries(node.labels).map(([key, value]) => (
              <Tag key={key} style={{ marginBottom: 4 }}>
                {key}={value}
              </Tag>
            ))}
          </div>
        </>
      )}

      {/* Traffic Metrics */}
      <Divider orientation="left" plain>
        Traffic Metrics
      </Divider>
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={12}>
          <Statistic
            title="Total Flows"
            value={node.metrics.flowCount}
            valueStyle={{ fontSize: 18 }}
          />
        </Col>
        <Col span={12}>
          <Statistic
            title="Active Flows"
            value={node.metrics.activeFlows}
            valueStyle={{ fontSize: 18, color: '#52c41a' }}
          />
        </Col>
        <Col span={12}>
          <Statistic
            title="Packets"
            value={formatNumber(node.metrics.packetCount)}
            valueStyle={{ fontSize: 18 }}
          />
        </Col>
        <Col span={12}>
          <Statistic
            title="Traffic"
            value={formatBytes(node.metrics.byteCount)}
            valueStyle={{ fontSize: 18 }}
          />
        </Col>
      </Row>

      {/* Connection Statistics */}
      {stats && (
        <>
          <Divider orientation="left" plain>
            Connections
          </Divider>
          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col span={12}>
              <Statistic
                title="Inbound"
                value={stats.inboundConnections}
                prefix={<ArrowDownOutlined />}
                valueStyle={{ fontSize: 18, color: '#1890ff' }}
              />
            </Col>
            <Col span={12}>
              <Statistic
                title="Outbound"
                value={stats.outboundConnections}
                prefix={<ArrowUpOutlined />}
                valueStyle={{ fontSize: 18, color: '#faad14' }}
              />
            </Col>
          </Row>
        </>
      )}

      {/* Protocol Distribution */}
      {protocolStats.length > 0 && (
        <>
          <Divider orientation="left" plain>
            Top Protocols
          </Divider>
          <div style={{ marginBottom: 16 }}>
            {protocolStats.map(([protocol, count]) => (
              <Tag key={protocol} color="blue" style={{ marginBottom: 4 }}>
                {protocol}: {count}
              </Tag>
            ))}
          </div>
        </>
      )}

      {/* State Distribution */}
      {stateStats.size > 0 && (
        <>
          <Divider orientation="left" plain>
            Flow States
          </Divider>
          <div style={{ marginBottom: 16 }}>
            {Array.from(stateStats.entries()).map(([state, count]) => (
              <Tag
                key={state}
                color={
                  state === 'ACTIVE' ? 'green'
                  : state === 'CLOSED' ? 'default'
                  : 'orange'
                }
                style={{ marginBottom: 4 }}
              >
                {state}: {count}
              </Tag>
            ))}
          </div>
        </>
      )}

      {/* Connected Neighbors */}
      {neighbors.length > 0 && (
        <>
          <Divider orientation="left" plain>
            Connected Nodes ({neighbors.length})
          </Divider>
          <div style={{ maxHeight: 200, overflow: 'auto' }}>
            {neighbors.slice(0, 10).map(neighbor => (
              <Tag key={neighbor} icon={<NodeIndexOutlined />} style={{ marginBottom: 4 }}>
                {neighbor}
              </Tag>
            ))}
            {neighbors.length > 10 && (
              <div style={{ marginTop: 8, color: '#8c8c8c', fontSize: 12 }}>
                ... and {neighbors.length - 10} more
              </div>
            )}
          </div>
        </>
      )}
    </Card>
  );
}

/**
 * Edge Detail View
 */
function EdgeDetail({
  edge,
  flows,
  width,
  style,
}: {
  edge: TopologyEdge;
  flows: Flow[];
  width: number;
  style?: React.CSSProperties;
}) {
  // Calculate average duration
  const avgDuration = useMemo(() => {
    if (flows.length === 0) return 0;
    const totalDuration = flows.reduce((sum, flow) => sum + flow.durationMs, 0);
    return totalDuration / flows.length;
  }, [flows]);

  // Get policy actions
  const policyActions = useMemo(() => {
    const actions = new Map<string, number>();
    for (const flow of flows) {
      actions.set(flow.policyAction, (actions.get(flow.policyAction) || 0) + 1);
    }
    return actions;
  }, [flows]);

  return (
    <Card
      style={{ width, ...style }}
      title={
        <span>
          <SwapOutlined /> Connection Details
        </span>
      }
      bordered={false}
    >
      {/* Basic Information */}
      <Descriptions column={1} size="small" bordered>
        <Descriptions.Item label="Source">{edge.source}</Descriptions.Item>
        <Descriptions.Item label="Target">{edge.target}</Descriptions.Item>
        <Descriptions.Item label="Direction">
          <Tag
            color={
              edge.direction === 'INGRESS' ? 'blue'
              : edge.direction === 'EGRESS' ? 'orange'
              : 'green'
            }
          >
            {edge.direction === 'INGRESS' ? 'Inbound'
            : edge.direction === 'EGRESS' ? 'Outbound'
            : 'Bidirectional'}
          </Tag>
        </Descriptions.Item>
      </Descriptions>

      {/* Traffic Metrics */}
      <Divider orientation="left" plain>
        Traffic Metrics
      </Divider>
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={12}>
          <Statistic
            title="Total Flows"
            value={edge.metrics.flowCount}
            valueStyle={{ fontSize: 18 }}
          />
        </Col>
        <Col span={12}>
          <Statistic
            title="Avg Duration"
            value={avgDuration.toFixed(0)}
            suffix="ms"
            valueStyle={{ fontSize: 18 }}
          />
        </Col>
        <Col span={12}>
          <Statistic
            title="Packets"
            value={formatNumber(edge.metrics.packetCount)}
            valueStyle={{ fontSize: 18 }}
          />
        </Col>
        <Col span={12}>
          <Statistic
            title="Traffic"
            value={formatBytes(edge.metrics.byteCount)}
            valueStyle={{ fontSize: 18 }}
          />
        </Col>
      </Row>

      {/* Protocols */}
      <Divider orientation="left" plain>
        Protocols
      </Divider>
      <div style={{ marginBottom: 16 }}>
        {edge.metrics.protocols.map(protocol => (
          <Tag key={protocol} color="blue" style={{ marginBottom: 4 }}>
            {protocol}
          </Tag>
        ))}
      </div>

      {/* Policy Actions */}
      {policyActions.size > 0 && (
        <>
          <Divider orientation="left" plain>
            Policy Actions
          </Divider>
          <div style={{ marginBottom: 16 }}>
            {Array.from(policyActions.entries()).map(([action, count]) => (
              <Tag
                key={action}
                color={
                  action === 'ALLOW' ? 'green'
                  : action === 'DENY' ? 'red'
                  : 'blue'
                }
                style={{ marginBottom: 4 }}
              >
                {action}: {count}
              </Tag>
            ))}
          </div>
        </>
      )}

      {/* Recent Flows */}
      {flows.length > 0 && (
        <>
          <Divider orientation="left" plain>
            Recent Flows ({flows.length})
          </Divider>
          <div style={{ maxHeight: 300, overflow: 'auto' }}>
            {flows.slice(0, 5).map(flow => (
              <Card
                key={flow.id}
                size="small"
                style={{ marginBottom: 8 }}
                bodyStyle={{ padding: 8 }}
              >
                <div style={{ fontSize: 12 }}>
                  <div>
                    <strong>Protocol:</strong> {flow.protocol}
                  </div>
                  <div>
                    <strong>Ports:</strong> {flow.sourcePort} → {flow.destPort}
                  </div>
                  <div>
                    <strong>State:</strong>{' '}
                    <Tag
                      size="small"
                      color={flow.state === 'ACTIVE' ? 'green' : 'default'}
                    >
                      {flow.state}
                    </Tag>
                  </div>
                  <div>
                    <strong>Traffic:</strong> {formatBytes(flow.byteCount)}
                  </div>
                </div>
              </Card>
            ))}
            {flows.length > 5 && (
              <div style={{ marginTop: 8, color: '#8c8c8c', fontSize: 12 }}>
                ... and {flows.length - 5} more flows
              </div>
            )}
          </div>
        </>
      )}
    </Card>
  );
}
