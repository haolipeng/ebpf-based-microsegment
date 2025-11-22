import { useState, useMemo } from 'react'
import { Typography, Alert, Space, Row, Col, Statistic, Card } from 'antd'
import {
  ClusterOutlined,
  AppstoreOutlined,
  ApartmentOutlined,
  ContainerOutlined,
  GlobalOutlined,
  ApiOutlined,
} from '@ant-design/icons'
import dayjs from 'dayjs'
import TopologyGraph from '../../components/topology/TopologyGraph'
import TopologyLegend from '../../components/topology/TopologyLegend'
import TopologyControls from '../../components/topology/TopologyControls'
import NodeDetailPanel from '../../components/topology/NodeDetailPanel'
import { useTopology } from '../../hooks/useTopology'
import type { TopologyViewMode, TopologyFilters, TopologyNode, TopologyData } from '../../types/topology'
import type { Flow } from '../../types/flow'
import { formatBytes } from '../../utils/topologyUtils'

const { Title, Text } = Typography

/**
 * Network Topology Page with K8s/Docker support
 *
 * Features:
 * - Multiple view modes: Namespace, Service, Pod, Container, Process, IP
 * - Time range and filter controls
 * - Real-time WebSocket updates
 * - Node click for details
 * - Interactive force-directed graph
 */
export default function Topology() {
  // State management
  const [viewMode, setViewMode] = useState<TopologyViewMode>('SERVICE')
  const [realtimeEnabled, setRealtimeEnabled] = useState(false)
  const [selectedNode, setSelectedNode] = useState<TopologyNode | null>(null)

  // Filter settings
  const [filters, setFilters] = useState<TopologyFilters>({
    viewMode: 'SERVICE',
    maxNodes: 100,
    showExternal: true,
    onlySuspicious: false,
    startTime: dayjs().subtract(1, 'hour').toISOString(),
    endTime: dayjs().toISOString(),
  })

  // Fetch topology data with namespaces
  const { data, isLoading, error, isConnected, namespaces, refetch } = useTopology(
    { ...filters, viewMode },
    realtimeEnabled
  )

  // Handle view mode change
  const handleViewModeChange = (mode: TopologyViewMode) => {
    setViewMode(mode)
    setSelectedNode(null)
    setFilters(prev => ({ ...prev, viewMode: mode }))
  }

  // Handle filter changes
  const handleFiltersChange = (newFilters: Partial<TopologyFilters>) => {
    setFilters(prev => ({ ...prev, ...newFilters }))
  }

  // Handle real-time toggle
  const handleRealtimeToggle = (enabled: boolean) => {
    setRealtimeEnabled(enabled)
  }

  // Handle node click
  const handleNodeClick = (node: TopologyNode) => {
    setSelectedNode(node)
  }

  // Handle detail panel close
  const handleDetailClose = () => {
    setSelectedNode(null)
  }

  // Related flows for selected node (placeholder)
  const relatedFlows = useMemo<Flow[]>(() => {
    if (!selectedNode || !data) {
      return []
    }
    return []
  }, [selectedNode, data])

  // Create default empty data
  const defaultData: TopologyData = {
    nodes: [],
    edges: [],
    groups: [],
    stats: {
      totalNodes: 0,
      totalEdges: 0,
      totalFlows: 0,
      activeFlows: 0,
      totalBytes: 0,
    },
    viewMode,
    timestamp: new Date().toISOString(),
  }

  const topologyData = data || defaultData

  return (
    <div className="topology-page" style={{ padding: '24px', height: 'calc(100vh - 64px)' }}>
      <Space direction="vertical" size="middle" style={{ width: '100%', height: '100%' }}>
        {/* Page Header */}
        <div>
          <Title level={2} style={{ marginBottom: 8 }}>Network Topology</Title>
          <Text type="secondary">
            Visualize traffic relationships between Kubernetes namespaces, services, pods, containers, and processes
          </Text>
        </div>

        {/* Statistics Summary */}
        <Card size="small">
          <Row gutter={16}>
            <Col span={3}>
              <Statistic
                title="Nodes"
                value={topologyData.stats.totalNodes}
                prefix={<ApiOutlined />}
                valueStyle={{ fontSize: 18 }}
              />
            </Col>
            <Col span={3}>
              <Statistic
                title="Connections"
                value={topologyData.stats.totalEdges}
                valueStyle={{ fontSize: 18 }}
              />
            </Col>
            <Col span={3}>
              <Statistic
                title="Flows"
                value={topologyData.stats.totalFlows}
                valueStyle={{ fontSize: 18 }}
              />
            </Col>
            <Col span={3}>
              <Statistic
                title="Active"
                value={topologyData.stats.activeFlows}
                valueStyle={{ fontSize: 18, color: '#52c41a' }}
              />
            </Col>
            <Col span={3}>
              <Statistic
                title="Traffic"
                value={formatBytes(topologyData.stats.totalBytes || 0)}
                valueStyle={{ fontSize: 18 }}
              />
            </Col>
            {topologyData.stats.namespaceCount !== undefined && topologyData.stats.namespaceCount > 0 && (
              <Col span={3}>
                <Statistic
                  title="Namespaces"
                  value={topologyData.stats.namespaceCount}
                  prefix={<ClusterOutlined />}
                  valueStyle={{ fontSize: 18 }}
                />
              </Col>
            )}
            {topologyData.stats.serviceCount !== undefined && topologyData.stats.serviceCount > 0 && (
              <Col span={3}>
                <Statistic
                  title="Services"
                  value={topologyData.stats.serviceCount}
                  prefix={<AppstoreOutlined />}
                  valueStyle={{ fontSize: 18 }}
                />
              </Col>
            )}
            {topologyData.stats.podCount !== undefined && topologyData.stats.podCount > 0 && (
              <Col span={3}>
                <Statistic
                  title="Pods"
                  value={topologyData.stats.podCount}
                  prefix={<ApartmentOutlined />}
                  valueStyle={{ fontSize: 18 }}
                />
              </Col>
            )}
            {topologyData.stats.containerCount !== undefined && topologyData.stats.containerCount > 0 && (
              <Col span={3}>
                <Statistic
                  title="Containers"
                  value={topologyData.stats.containerCount}
                  prefix={<ContainerOutlined />}
                  valueStyle={{ fontSize: 18 }}
                />
              </Col>
            )}
            {topologyData.stats.externalCount !== undefined && topologyData.stats.externalCount > 0 && (
              <Col span={3}>
                <Statistic
                  title="External"
                  value={topologyData.stats.externalCount}
                  prefix={<GlobalOutlined />}
                  valueStyle={{ fontSize: 18, color: '#f5222d' }}
                />
              </Col>
            )}
          </Row>
        </Card>

        {/* Controls */}
        <TopologyControls
          viewMode={viewMode}
          onViewModeChange={handleViewModeChange}
          filters={filters}
          onFiltersChange={handleFiltersChange}
          realtimeEnabled={realtimeEnabled}
          onRealtimeToggle={handleRealtimeToggle}
          onRefresh={refetch}
          namespaces={namespaces}
        />

        {/* Real-time connection status */}
        {realtimeEnabled && (
          <Alert
            message={
              isConnected
                ? '✅ Real-time updates enabled - receiving live flow data'
                : '⚠️ Real-time connection lost - attempting to reconnect...'
            }
            type={isConnected ? 'success' : 'warning'}
            showIcon
            closable
            banner
          />
        )}

        {/* Error message */}
        {error && (
          <Alert
            message="Failed to load topology data"
            description={error.message}
            type="error"
            showIcon
            closable
          />
        )}

        {/* Main topology graph */}
        <div style={{ position: 'relative', flex: 1, minHeight: 500 }}>
          <TopologyGraph
            data={topologyData}
            viewMode={viewMode}
            onNodeClick={handleNodeClick}
            loading={isLoading}
            height="100%"
          />

          {/* Legend */}
          <TopologyLegend viewMode={viewMode} />
        </div>

        {/* Node detail panel */}
        <NodeDetailPanel
          node={selectedNode}
          flows={relatedFlows}
          onClose={handleDetailClose}
        />
      </Space>
    </div>
  )
}
