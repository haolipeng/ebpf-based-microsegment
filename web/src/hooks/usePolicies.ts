import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { policiesApi } from '../api/policies'
import type { Policy } from '../types/policy'

export function usePolicies() {
  return useQuery({
    queryKey: ['policies'],
    queryFn: policiesApi.list,
    refetchInterval: 30000, // Refetch every 30 seconds
  })
}

export function usePolicy(ruleId: number) {
  return useQuery({
    queryKey: ['policy', ruleId],
    queryFn: () => policiesApi.get(ruleId),
    enabled: !!ruleId,
  })
}

export function usePolicyStats(ruleId: number) {
  return useQuery({
    queryKey: ['policyStats', ruleId],
    queryFn: () => policiesApi.stats(ruleId),
    enabled: !!ruleId,
    refetchInterval: 30000,
  })
}

export function useAllPolicyStats(ruleIds: number[]) {
  return useQuery({
    queryKey: ['allPolicyStats', ruleIds],
    queryFn: async () => {
      const statsPromises = ruleIds.map(ruleId => policiesApi.stats(ruleId))
      const statsArray = await Promise.all(statsPromises)
      const statsMap = new Map(statsArray.map(stats => [stats.ruleId, stats]))
      return statsMap
    },
    enabled: ruleIds.length > 0,
    refetchInterval: 30000,
  })
}

export function useCreatePolicy() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (policy: Omit<Policy, 'ruleId'>) => policiesApi.create(policy),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] })
    },
  })
}

export function useUpdatePolicy() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ ruleId, policy }: { ruleId: number; policy: Partial<Policy> }) =>
      policiesApi.update(ruleId, policy),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['policies'] })
      queryClient.invalidateQueries({ queryKey: ['policy', variables.ruleId] })
    },
  })
}

export function useDeletePolicy() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (ruleId: number) => policiesApi.delete(ruleId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] })
    },
  })
}

export function useBatchDeletePolicies() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (ruleIds: number[]) => {
      await Promise.all(ruleIds.map(ruleId => policiesApi.delete(ruleId)))
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] })
    },
  })
}

export function useBatchUpdatePolicies() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ ruleIds, policy }: { ruleIds: number[]; policy: Partial<Policy> }) => {
      await Promise.all(ruleIds.map(ruleId => policiesApi.update(ruleId, policy)))
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] })
    },
  })
}
