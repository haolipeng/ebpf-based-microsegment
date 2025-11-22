import { useState, useEffect, useMemo, useCallback, useRef } from 'react'
import { useFlows } from './useFlows'
import { useFlowStream } from './useFlowStream'
import type { TopologyData, TopologyFilters } from '../types/topology'
import type { Flow } from '../types/flow'
import { aggregateFlowsToTopology, mergeTopologyUpdate } from '../utils/topologyUtils'

/**
 * Topology data hook with K8s/Docker support
 *
 * Features:
 * - Fetches flow data and transforms to topology
 * - Supports multiple aggregation levels (namespace, service, pod, container, process)
 * - Integrates WebSocket for real-time updates
 * - Debounced updates to avoid frequent re-computation
 * - Extracts unique namespaces for filtering
 *
 * @param filters - Topology filter settings
 * @param enableRealtime - Enable real-time updates via WebSocket
 * @returns Topology data and state
 */
export function useTopology(filters: TopologyFilters, enableRealtime: boolean = false) {
  const [realtimeFlows, setRealtimeFlows] = useState<Flow[]>([])
  const [topologyData, setTopologyData] = useState<TopologyData | undefined>(undefined)
  const [namespaces, setNamespaces] = useState<string[]>([])
  const updateTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Fetch base flow data
  const { data: flows, isLoading, error, refetch } = useFlows(filters)

  // Handle new real-time flow
  const handleNewFlow = useCallback(
    (flow: Flow) => {
      // Add to real-time flows list (limit to 200)
      setRealtimeFlows(prev => {
        const updated = [flow, ...prev].slice(0, 200)
        return updated
      })

      // Debounced topology update (500ms aggregation)
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

  // WebSocket real-time stream
  const { isConnected, error: wsError } = useFlowStream({
    enabled: enableRealtime,
    onFlow: handleNewFlow,
  })

  // Extract unique namespaces from flows
  useEffect(() => {
    if (!flows || flows.length === 0) return

    const nsSet = new Set<string>()

    flows.forEach(flow => {
      // Extract from source labels
      if (flow.sourceLabels) {
        const ns = flow.sourceLabels['kubernetes.io/namespace'] ||
                   flow.sourceLabels['namespace'] ||
                   flow.sourceLabels['k8s.namespace']
        if (ns) nsSet.add(ns)
      }

      // Extract from dest labels
      if (flow.destLabels) {
        const ns = flow.destLabels['kubernetes.io/namespace'] ||
                   flow.destLabels['namespace'] ||
                   flow.destLabels['k8s.namespace']
        if (ns) nsSet.add(ns)
      }
    })

    setNamespaces(Array.from(nsSet).sort())
  }, [flows])

  // Aggregate flows to topology data
  const aggregatedData = useMemo(() => {
    if (!flows || flows.length === 0) {
      return {
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
        viewMode: filters.viewMode,
        timestamp: new Date().toISOString(),
      } as TopologyData
    }

    // Merge real-time flows with base flows (if real-time enabled)
    const allFlows = enableRealtime && realtimeFlows.length > 0
      ? [...realtimeFlows, ...flows]
      : flows

    // Use the new aggregation function with full filters support
    return aggregateFlowsToTopology(
      allFlows,
      filters.viewMode,
      filters.maxNodes,
      {
        namespace: filters.namespace,
        service: filters.service,
        showExternal: filters.showExternal,
        onlySuspicious: filters.onlySuspicious,
        minFlowCount: filters.minFlowCount,
      }
    )
  }, [flows, filters.viewMode, filters.maxNodes, filters.namespace, filters.service,
      filters.showExternal, filters.onlySuspicious, filters.minFlowCount,
      enableRealtime, realtimeFlows])

  // Update topology data
  useEffect(() => {
    setTopologyData(aggregatedData)
  }, [aggregatedData])

  // Cleanup timer on unmount
  useEffect(() => {
    return () => {
      if (updateTimeoutRef.current) {
        clearTimeout(updateTimeoutRef.current)
      }
    }
  }, [])

  // Clear real-time flows when view mode changes
  useEffect(() => {
    setRealtimeFlows([])
  }, [filters.viewMode])

  return {
    /** Topology data */
    data: topologyData,
    /** Loading state */
    isLoading,
    /** Error (HTTP or WebSocket) */
    error: error || wsError,
    /** Real-time connection status */
    isConnected,
    /** Available namespaces for filtering */
    namespaces,
    /** Refetch data */
    refetch,
  }
}
