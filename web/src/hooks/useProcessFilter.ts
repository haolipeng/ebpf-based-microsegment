import { useMemo } from 'react'
import type { Flow, ProcessStats, ContainerStats } from '../types/flow'

export interface ProcessFilterOptions {
  processName?: string
  processPath?: string
  containerId?: string
  pid?: number
}

/**
 * Hook for filtering flows by process information
 * Provides client-side filtering logic for process-based queries
 */
export function useProcessFilter(flows: Flow[], filters: ProcessFilterOptions) {
  const filteredFlows = useMemo(() => {
    if (!flows || flows.length === 0) return []

    let result = flows

    // Filter by process name (case-insensitive partial match)
    if (filters.processName) {
      const searchTerm = filters.processName.toLowerCase()
      result = result.filter(flow =>
        flow.processInfo?.comm?.toLowerCase().includes(searchTerm)
      )
    }

    // Filter by process path (case-insensitive partial match)
    if (filters.processPath) {
      const searchTerm = filters.processPath.toLowerCase()
      result = result.filter(flow =>
        flow.processInfo?.exePath?.toLowerCase().includes(searchTerm)
      )
    }

    // Filter by container ID (case-insensitive partial match)
    if (filters.containerId) {
      const searchTerm = filters.containerId.toLowerCase()
      result = result.filter(flow =>
        flow.processInfo?.containerId?.toLowerCase().includes(searchTerm)
      )
    }

    // Filter by exact PID
    if (filters.pid !== undefined) {
      result = result.filter(flow => flow.processInfo?.pid === filters.pid)
    }

    return result
  }, [flows, filters])

  return { filteredFlows }
}

/**
 * Hook for computing process statistics from flows
 * Aggregates flow data to generate process-level metrics
 */
export function useProcessStats(flows: Flow[]) {
  const processStats = useMemo<ProcessStats[]>(() => {
    const statsMap = new Map<
      string,
      {
        processName: string
        processPath?: string
        flows: Flow[]
        connections: Set<string>
        packetCount: number
        byteCount: number
        containers: Set<string>
        isSuspicious: boolean
      }
    >()

    flows.forEach(flow => {
      if (!flow.processInfo) return

      const key = flow.processInfo.comm
      const existing = statsMap.get(key)

      // Create connection key from source/dest IP:port
      const connectionKey = `${flow.sourceIp}:${flow.sourcePort}-${flow.destIp}:${flow.destPort}`

      if (existing) {
        existing.flows.push(flow)
        existing.connections.add(connectionKey)
        existing.packetCount += flow.packetCount || 0
        existing.byteCount += flow.byteCount || 0
        if (flow.processInfo.containerId) {
          existing.containers.add(flow.processInfo.containerId)
        }
        if (flow.processInfo.isSuspicious) {
          existing.isSuspicious = true
        }
      } else {
        const containers = new Set<string>()
        if (flow.processInfo.containerId) {
          containers.add(flow.processInfo.containerId)
        }

        statsMap.set(key, {
          processName: flow.processInfo.comm,
          processPath: flow.processInfo.exePath,
          flows: [flow],
          connections: new Set([connectionKey]),
          packetCount: flow.packetCount || 0,
          byteCount: flow.byteCount || 0,
          containers,
          isSuspicious: flow.processInfo.isSuspicious || false,
        })
      }
    })

    return Array.from(statsMap.values()).map(stat => ({
      processName: stat.processName,
      processPath: stat.processPath,
      flowCount: stat.flows.length,
      connectionCount: stat.connections.size,
      packetCount: stat.packetCount,
      byteCount: stat.byteCount,
      containerCount: stat.containers.size,
      isSuspicious: stat.isSuspicious,
    }))
  }, [flows])

  const containerStats = useMemo<ContainerStats[]>(() => {
    const statsMap = new Map<
      string,
      {
        containerId: string
        processes: Set<string>
        flows: Flow[]
        byteCount: number
      }
    >()

    flows.forEach(flow => {
      if (!flow.processInfo?.containerId) return

      const key = flow.processInfo.containerId
      const existing = statsMap.get(key)

      if (existing) {
        existing.processes.add(flow.processInfo.comm)
        existing.flows.push(flow)
        existing.byteCount += flow.byteCount || 0
      } else {
        statsMap.set(key, {
          containerId: key,
          processes: new Set([flow.processInfo.comm]),
          flows: [flow],
          byteCount: flow.byteCount || 0,
        })
      }
    })

    return Array.from(statsMap.values()).map(stat => ({
      containerId: stat.containerId,
      processCount: stat.processes.size,
      flowCount: stat.flows.length,
      byteCount: stat.byteCount,
    }))
  }, [flows])

  return { processStats, containerStats }
}
