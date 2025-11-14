import { useState, useEffect, useMemo, useCallback, useRef } from 'react'
import { useFlows } from './useFlows'
import { useFlowStream } from './useFlowStream'
import type { TopologyData, TopologyFilters } from '../types/topology'
import type { Flow } from '../types/flow'
import { aggregateFlowsToTopology, mergeTopologyUpdate } from '../utils/topologyUtils'

/**
 * 拓扑图数据获取Hook
 * 
 * 功能：
 * - 获取flows数据并转换为拓扑数据
 * - 支持时间范围筛选
 * - 集成WebSocket实时更新
 * - 防抖更新避免频繁重新计算
 * 
 * @param filters - 拓扑图筛选条件
 * @param enableRealtime - 是否启用实时更新
 * @returns 拓扑数据和状态
 */
export function useTopology(filters: TopologyFilters, enableRealtime: boolean = false) {
  const [realtimeFlows, setRealtimeFlows] = useState<Flow[]>([])
  const [topologyData, setTopologyData] = useState<TopologyData | undefined>(undefined)
  const updateTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // 获取基础流数据
  const { data: flows, isLoading, error, refetch } = useFlows(filters)

  // 实时流更新处理
  const handleNewFlow = useCallback(
    (flow: Flow) => {
      // 添加到实时流列表（限制最多100条）
      setRealtimeFlows(prev => {
        const updated = [flow, ...prev].slice(0, 100)
        return updated
      })

      // 防抖更新拓扑数据（500ms聚合更新）
      if (updateTimeoutRef.current) {
        clearTimeout(updateTimeoutRef.current)
      }

      updateTimeoutRef.current = setTimeout(() => {
        setTopologyData(prevData => {
          if (!prevData) return prevData
          return mergeTopologyUpdate(prevData, flow, filters.viewMode)
        })
      }, 500)
    },
    [filters.viewMode]
  )

  // WebSocket实时流
  const { isConnected, error: wsError } = useFlowStream({
    enabled: enableRealtime,
    onFlow: handleNewFlow,
  })

  // 聚合flows为拓扑数据
  const aggregatedData = useMemo(() => {
    if (!flows || flows.length === 0) {
      return {
        nodes: [],
        edges: [],
        stats: {
          totalNodes: 0,
          totalEdges: 0,
          totalFlows: 0,
        },
      }
    }

    // 合并实时流到基础flows（如果启用实时模式）
    const allFlows = enableRealtime && realtimeFlows.length > 0
      ? [...realtimeFlows, ...flows]
      : flows

    return aggregateFlowsToTopology(allFlows, filters.viewMode, filters.maxNodes)
  }, [flows, filters.viewMode, filters.maxNodes, enableRealtime, realtimeFlows])

  // 更新拓扑数据
  useEffect(() => {
    setTopologyData(aggregatedData)
  }, [aggregatedData])

  // 清理定时器
  useEffect(() => {
    return () => {
      if (updateTimeoutRef.current) {
        clearTimeout(updateTimeoutRef.current)
      }
    }
  }, [])

  return {
    /** 拓扑数据 */
    data: topologyData,
    /** 是否加载中 */
    isLoading,
    /** 错误信息 */
    error: error || wsError,
    /** 实时连接状态 */
    isConnected,
    /** 重新获取数据 */
    refetch,
  }
}

