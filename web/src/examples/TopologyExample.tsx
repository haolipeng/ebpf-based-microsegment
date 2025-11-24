/**
 * Topology Example
 *
 * Demonstrates integration of graph database with topology visualization
 * This example shows how to use the new topology features
 */

import { useState, useMemo } from 'react';
import { Layout, Card, Space, Button, Select, InputNumber, Row, Col, Statistic } from 'antd';
import {
  ReloadOutlined,
  BarChartOutlined,
  NodeIndexOutlined,
} from '@ant-design/icons';
import TopologyGraph from '@/components/topology/TopologyGraph';
import SessionDetail from '@/components/topology/SessionDetail';
import { useTopologyGraph, useTopologyStats, useNodeFocus } from '@/hooks/useTopologyGraph';
import type { TopologyNode, TopologyEdge, TopologyViewMode } from '@/types/topology';
import type { Flow } from '@/types/flow';

const { Content, Sider } = Layout;

// Mock flow data generator
function generateMockFlows(count: number): Flow[] {
  const ips = ['192.168.1.10', '192.168.1.20', '192.168.1.30', '192.168.1.40', '10.0.0.5'];
  const protocols: Array<'TCP' | 'UDP' | 'ICMP'> = ['TCP', 'UDP', 'ICMP'];
  const states: Array<'ACTIVE' | 'CLOSED' | 'TIMEOUT'> = ['ACTIVE', 'CLOSED', 'TIMEOUT'];
  const actions: Array<'ALLOW' | 'DENY' | 'LOG'> = ['ALLOW', 'DENY', 'LOG'];

  const flows: Flow[] = [];
  for (let i = 0; i < count; i++) {
    const sourceIp = ips[Math.floor(Math.random() * ips.length)];
    const destIp = ips[Math.floor(Math.random() * ips.length)];

    if (sourceIp === destIp) continue;

    flows.push({
      id: `flow-${i}`,
      sourceIp,
      sourcePort: 1024 + Math.floor(Math.random() * 64000),
      destIp,
      destPort: [80, 443, 3306, 5432, 6379][Math.floor(Math.random() * 5)],
      protocol: protocols[Math.floor(Math.random() * protocols.length)],
      packetCount: Math.floor(Math.random() * 1000) + 10,
      byteCount: Math.floor(Math.random() * 1000000) + 1000,
      durationMs: Math.floor(Math.random() * 60000),
      startTime: new Date(Date.now() - Math.random() * 3600000).toISOString(),
      lastSeen: new Date().toISOString(),
      policyAction: actions[Math.floor(Math.random() * actions.length)],
      state: states[Math.floor(Math.random() * states.length)],
      direction: Math.random() > 0.5 ? 'INGRESS' : 'EGRESS',
      eventType: 'UPDATE',
    });
  }
  return flows;
}

/**
 * Topology Example Component
 */
export default function TopologyExample() {
  // State
  const [viewMode, setViewMode] = useState<TopologyViewMode>('IP');
  const [maxNodes, setMaxNodes] = useState<number>(20);
  const [flowCount, setFlowCount] = useState<number>(50);

  // Generate mock flows
  const flows = useMemo(() => generateMockFlows(flowCount), [flowCount]);

  // Use topology graph hook
  const {
    data,
    findConnectedNodes: _findConnectedNodes,
    findTrafficHubs,
    findConnectedComponents,
    getNodeDetail,
    getNeighbors,
    refresh,
  } = useTopologyGraph({
    flows,
    filters: {
      viewMode,
      maxNodes,
      startTime: new Date(Date.now() - 3600000).toISOString(),
    },
  });

  // Use node focus hook
  const { selectedNode, selectNode, focusedNeighbors: _focusedNeighbors } = useNodeFocus();

  // Calculate stats
  const stats = useTopologyStats(data);

  // Handle node click
  const handleNodeClick = (node: TopologyNode) => {
    selectNode(node);
  };

  // Handle edge click
  const [selectedEdge, setSelectedEdge] = useState<TopologyEdge | null>(null);
  const handleEdgeClick = (edge: TopologyEdge) => {
    setSelectedEdge(edge);
    selectNode(null);
  };

  // Get related flows for selected node/edge
  const relatedFlows = useMemo(() => {
    if (selectedNode) {
      return flows.filter(
        f => f.sourceIp === selectedNode.id || f.destIp === selectedNode.id
      );
    }
    if (selectedEdge) {
      return flows.filter(
        f =>
          (f.sourceIp === selectedEdge.source && f.destIp === selectedEdge.target) ||
          (f.sourceIp === selectedEdge.target && f.destIp === selectedEdge.source)
      );
    }
    return [];
  }, [selectedNode, selectedEdge, flows]);

  // Get node stats
  const nodeStats = selectedNode ? getNodeDetail(selectedNode.id) : null;

  // Get neighbors
  const neighbors = selectedNode ? getNeighbors(selectedNode.id) : [];

  // Find hubs
  const hubs = useMemo(() => findTrafficHubs(5), [findTrafficHubs]);

  // Find components
  const components = useMemo(() => findConnectedComponents(), [findConnectedComponents]);

  return (
    <Layout style={{ minHeight: '100vh', background: '#f0f2f5' }}>
      <Content style={{ padding: '24px' }}>
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          {/* Controls */}
          <Card>
            <Space direction="vertical" size="middle" style={{ width: '100%' }}>
              <Row gutter={16}>
                <Col span={6}>
                  <div>
                    <div style={{ marginBottom: 8, fontWeight: 500 }}>View Mode:</div>
                    <Select
                      value={viewMode}
                      onChange={setViewMode}
                      style={{ width: '100%' }}
                      options={[
                        { label: 'IP View', value: 'IP' },
                        { label: 'Label View', value: 'LABEL' },
                      ]}
                    />
                  </div>
                </Col>
                <Col span={6}>
                  <div>
                    <div style={{ marginBottom: 8, fontWeight: 500 }}>Max Nodes:</div>
                    <InputNumber
                      value={maxNodes}
                      onChange={value => setMaxNodes(value || 20)}
                      min={5}
                      max={100}
                      style={{ width: '100%' }}
                    />
                  </div>
                </Col>
                <Col span={6}>
                  <div>
                    <div style={{ marginBottom: 8, fontWeight: 500 }}>Flow Count:</div>
                    <InputNumber
                      value={flowCount}
                      onChange={value => setFlowCount(value || 50)}
                      min={10}
                      max={200}
                      style={{ width: '100%' }}
                    />
                  </div>
                </Col>
                <Col span={6}>
                  <div>
                    <div style={{ marginBottom: 8, fontWeight: 500 }}>Actions:</div>
                    <Button
                      type="primary"
                      icon={<ReloadOutlined />}
                      onClick={refresh}
                      block
                    >
                      Refresh
                    </Button>
                  </div>
                </Col>
              </Row>

              {/* Statistics */}
              <Card size="small" title={<span><BarChartOutlined /> Graph Statistics</span>}>
                <Row gutter={16}>
                  <Col span={4}>
                    <Statistic title="Nodes" value={data.stats.totalNodes} />
                  </Col>
                  <Col span={4}>
                    <Statistic title="Edges" value={data.stats.totalEdges} />
                  </Col>
                  <Col span={4}>
                    <Statistic title="Avg Degree" value={stats.avgDegree.toFixed(2)} />
                  </Col>
                  <Col span={4}>
                    <Statistic title="Max Degree" value={stats.maxDegree} />
                  </Col>
                  <Col span={4}>
                    <Statistic title="Components" value={components.length} />
                  </Col>
                  <Col span={4}>
                    <Statistic title="Isolated" value={stats.isolatedCount} />
                  </Col>
                </Row>
              </Card>

              {/* Traffic Hubs */}
              {hubs.length > 0 && (
                <Card size="small" title={<span><NodeIndexOutlined /> Top Traffic Hubs</span>}>
                  <Space wrap>
                    {hubs.map(([nodeId, degree]) => (
                      <Button
                        key={nodeId}
                        size="small"
                        onClick={() => {
                          const node = data.nodes.find(n => n.id === nodeId);
                          if (node) handleNodeClick(node);
                        }}
                      >
                        {nodeId} (degree: {degree})
                      </Button>
                    ))}
                  </Space>
                </Card>
              )}
            </Space>
          </Card>

          {/* Topology Visualization */}
          <Layout style={{ background: '#fff' }}>
            <Content style={{ padding: '16px' }}>
              <TopologyGraph
                data={data}
                viewMode={viewMode}
                onNodeClick={handleNodeClick}
                onEdgeClick={handleEdgeClick}
                height={600}
              />
            </Content>

            {/* Session Detail Panel */}
            {(selectedNode || selectedEdge) && (
              <Sider
                width={400}
                style={{
                  background: '#fff',
                  borderLeft: '1px solid #f0f0f0',
                  overflow: 'auto',
                }}
              >
                <SessionDetail
                  selectedNode={selectedNode}
                  selectedEdge={selectedEdge}
                  relatedFlows={relatedFlows}
                  nodeStats={nodeStats}
                  neighbors={neighbors}
                  width={400}
                />
              </Sider>
            )}
          </Layout>
        </Space>
      </Content>
    </Layout>
  );
}
