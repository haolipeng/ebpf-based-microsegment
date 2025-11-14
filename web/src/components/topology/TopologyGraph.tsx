import { useRef, useEffect } from 'react'
import ReactECharts from 'echarts-for-react'
import type { ECharts } from 'echarts'
import { Spin, Empty } from 'antd'
import type { TopologyData, TopologyNode, TopologyEdge, TopologyViewMode } from '../../types/topology'
import { getTopologyChartOption } from './topologyConfig'

interface TopologyGraphProps {
  /** 拓扑数据 */
  data: TopologyData
  /** 视图模式 */
  viewMode: TopologyViewMode
  /** 节点点击回调 */
  onNodeClick?: (node: TopologyNode) => void
  /** 边点击回调 */
  onEdgeClick?: (edge: TopologyEdge) => void
  /** 图表高度 */
  height?: number | string
  /** 加载状态 */
  loading?: boolean
}

/**
 * 拓扑图核心可视化组件
 *
 * 使用ECharts Graph图表渲染网络拓扑
 *
 * 功能：
 * - 力导向布局
 * - 节点视觉映射（大小、颜色）
 * - 边视觉映射（粗细、颜色）
 * - 缩放、平移、拖拽交互
 * - 点击节点/边事件
 */
export default function TopologyGraph({
  data,
  viewMode,
  onNodeClick,
  onEdgeClick,
  height = 600,
  loading = false,
}: TopologyGraphProps) {
  const chartRef = useRef<ReactECharts>(null)

  // 处理图表事件
  useEffect(() => {
    const chart = chartRef.current?.getEchartsInstance() as ECharts | undefined

    if (!chart) return

    // 点击事件处理
    const handleClick = (params: {
      dataType?: string
      data?: TopologyNode | TopologyEdge
    }) => {
      if (params.dataType === 'node' && onNodeClick) {
        // 获取完整的节点数据
        const node = params.data as TopologyNode
        onNodeClick(node)
      } else if (params.dataType === 'edge' && onEdgeClick) {
        // 获取完整的边数据
        const edge = params.data as TopologyEdge
        onEdgeClick(edge)
      }
    }

    // 双击节点聚焦
    const handleDblClick = (params: { dataType?: string; dataIndex?: number }) => {
      if (params.dataType === 'node') {
        // 高亮相关节点和边
        chart.dispatchAction({
          type: 'highlight',
          seriesIndex: 0,
          dataIndex: params.dataIndex,
        })

        // 可以在这里添加更多聚焦逻辑
      }
    }

    chart.on('click', handleClick)
    chart.on('dblclick', handleDblClick)

    return () => {
      chart.off('click', handleClick)
      chart.off('dblclick', handleDblClick)
    }
  }, [onNodeClick, onEdgeClick])

  // 无数据时显示空状态
  if (!loading && (!data || data.nodes.length === 0)) {
    return (
      <div
        style={{
          height: typeof height === 'number' ? `${height}px` : height,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <Empty description="No topology data" />
      </div>
    )
  }

  // 生成图表配置
  const option = getTopologyChartOption(data, viewMode)

  return (
    <Spin spinning={loading} tip="Loading topology data...">
      <div className="topology-graph-container">
        <ReactECharts
          ref={chartRef}
          option={option}
          style={{
            height: typeof height === 'number' ? `${height}px` : height,
            width: '100%',
          }}
          notMerge={true}
          lazyUpdate={true}
          opts={{
            renderer: 'canvas', // 使用canvas渲染，性能更好
            locale: 'ZH',
          }}
        />
      </div>
    </Spin>
  )
}

