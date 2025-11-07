import { useQuery } from '@tanstack/react-query'
import { agentsApi } from '../api/agents'

export function useAgents() {
  return useQuery({
    queryKey: ['agents'],
    queryFn: agentsApi.list,
    refetchInterval: 30000, // Refetch every 30 seconds
  })
}

export function useAgent(id: string) {
  return useQuery({
    queryKey: ['agent', id],
    queryFn: () => agentsApi.get(id),
    enabled: !!id,
    refetchInterval: 10000, // Refetch every 10 seconds for details
  })
}

export function useHealthCheck() {
  return useQuery({
    queryKey: ['health'],
    queryFn: agentsApi.health,
    refetchInterval: 30000, // Refetch every 30 seconds
  })
}
