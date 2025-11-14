import { useState, useMemo } from 'react'
import { Typography, Alert, Space } from 'antd'
import dayjs from 'dayjs'
import TopologyGraph from '../../components/topology/TopologyGraph'
import TopologyLegend from '../../components/topology/TopologyLegend'
import TopologyControls from '../../components/topology/TopologyControls'
import NodeDetailPanel from '../../components/topology/NodeDetailPanel'
import { useTopology } from '../../hooks/useTopology'
import type { TopologyViewMode, TopologyFilters, TopologyNode } from '../../types/topology'
import type { Flow } from '../../types/flow'

const { Title } = Typography

/**
 * 网络拓扑图页面
 * 
 * 功能：
 * - IP视图和服务视图切换
 * - 时间范围、协议、状态等筛选
 * - 实时WebSocket更新
 * - 节点点击查看详情
 * - 交互式力导向图
 */
export default function Topology() {
  // 状态管理
  const [viewMode, setViewMode] = useState<TopologyViewMode>('IP')
  const [realtimeEnabled, setRealtimeEnabled] = useState(false)
  const [selectedNode, setSelectedNode] = useState<TopologyNode | null>(null)
  
  // 筛选条件
  const [filters, setFilters] = useState<TopologyFilters>({
    viewMode: 'IP',
    maxNodes: 100,
    startTime: dayjs().subtract(7, 'days').toISOString(),
    endTime: dayjs().toISOString(),
  })

  // 获取拓扑数据
  const { data, isLoading, error, isConnected, refetch } = useTopology(
    { ...filters, viewMode },
    realtimeEnabled
  )

  // 处理视图模式切换
  const handleViewModeChange = (mode: TopologyViewMode) => {
    setViewMode(mode)
    setSelectedNode(null) // 清除选中节点
  }

  // 处理筛选条件变更
  const handleFiltersChange = (newFilters: Partial<TopologyFilters>) => {
    setFilters((prev) => ({
      ...prev,
      ...newFilters,
    }))
  }

  // 处理实时更新切换
  const handleRealtimeToggle = (enabled: boolean) => {
    setRealtimeEnabled(enabled)
  }

  // 处理节点点击
  const handleNodeClick = (node: TopologyNode) => {
    setSelectedNode(node)
  }

  // 处理详情面板关闭
  const handleDetailClose = () => {
    setSelectedNode(null)
  }

  // 获取与选中节点相关的流（用于详情面板）
  const relatedFlows = useMemo<Flow[]>(() => {
    if (!selectedNode || !data) {
      return []
    }

    // 这里需要从实际的flows数据中筛选
    // 由于useTopology只返回聚合后的拓扑数据，
    // 我们需要另外获取flows或在useTopology中一并返回
    // 暂时返回空数组，后续可以优化
    return []
  }, [selectedNode, data])

  return (
    <div className="topology-page" style={{ padding: '24px', height: 'calc(100vh - 64px)' }}>
      <Space direction="vertical" size="large" style={{ width: '100%', height: '100%' }}>
        {/* 页面标题 */}
        <div>
          <Title level={2}>Network Topology</Title>
          <Typography.Text type="secondary">
            Visualize network flow topology relationships with IP and Service views
          </Typography.Text>
        </div>

        {/* 控制栏 */}
        <TopologyControls
          viewMode={viewMode}
          onViewModeChange={handleViewModeChange}
          filters={filters}
          onFiltersChange={handleFiltersChange}
          realtimeEnabled={realtimeEnabled}
          onRealtimeToggle={handleRealtimeToggle}
          onRefresh={refetch}
        />

        {/* 实时连接状态提示 */}
        {realtimeEnabled && (
          <Alert
            message={
              isConnected
                ? '✅ Real-time updates enabled, receiving latest flow data'
                : '⚠️ Real-time connection lost, attempting to reconnect...'
            }
            type={isConnected ? 'success' : 'warning'}
            showIcon
            closable
          />
        )}

        {/* 错误提示 */}
        {error && (
          <Alert
            message="Failed to load data"
            description={error.message}
            type="error"
            showIcon
            closable
          />
        )}

        {/* 拓扑图主体 */}
        <div style={{ position: 'relative', flex: 1, minHeight: 600 }}>
          <TopologyGraph
            data={data || { nodes: [], edges: [], stats: { totalNodes: 0, totalEdges: 0, totalFlows: 0 } }}
            viewMode={viewMode}
            onNodeClick={handleNodeClick}
            loading={isLoading}
            height="100%"
          />

          {/* 图例 */}
          <TopologyLegend viewMode={viewMode} />
        </div>

        {/* 节点详情面板 */}
        <NodeDetailPanel
          node={selectedNode}
          flows={relatedFlows}
          onClose={handleDetailClose}
        />
      </Space>
    </div>
  )
}

