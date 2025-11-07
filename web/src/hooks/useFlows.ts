import { useQuery } from '@tanstack/react-query'
import { flowsApi } from '../api/flows'
import type { FlowQuery } from '../types/flow'

export function useFlows(query?: FlowQuery) {
  return useQuery({
    queryKey: ['flows', query],
    queryFn: () => flowsApi.query(query || {}),
    refetchInterval: 30000, // Refetch every 30 seconds
  })
}

export function useFlowSummary(startTime?: string, endTime?: string) {
  return useQuery({
    queryKey: ['flowSummary', startTime, endTime],
    queryFn: () => flowsApi.summary(startTime, endTime),
    refetchInterval: 30000, // Refetch every 30 seconds
  })
}

export function useActiveFlows() {
  return useQuery({
    queryKey: ['flows', 'active'],
    queryFn: flowsApi.active,
    refetchInterval: 30000,
  })
}
