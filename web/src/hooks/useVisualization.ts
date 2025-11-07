import { useQuery } from '@tanstack/react-query'
import { flowsApi } from '../api/flows'
import { policiesApi } from '../api/policies'
import { sampleDataPoints } from '../utils/chartHelpers'
import type { ProtocolStats, IpStats } from '../types/flow'

// 性能优化配置
const MAX_CHART_POINTS = 200 // 图表最大数据点数

/**
 * 流量趋势数据点
 */
export interface FlowTrendPoint {
  timestamp: string
  flowCount: number
  packetCount: number
  byteCount: number
}

/**
 * 协议趋势数据点
 */
export interface ProtocolTrendPoint {
  timestamp: string
  protocol: string
  flowCount: number
  byteCount: number
}

/**
 * Top Talkers 数据
 */
export interface TopTalkersData {
  topSourceIps: IpStats[]
  topDestIps: IpStats[]
}

/**
 * 协议分布数据
 */
export interface ProtocolDistribution {
  protocols: ProtocolStats[]
}

/**
 * 策略效果数据点
 */
export interface PolicyEffectivenessPoint {
  timestamp: string
  allow: number
  deny: number
  log: number
}

/**
 * Top 策略数据
 */
export interface TopPolicyData {
  ruleId: number
  hitCount: number
  description?: string
}

/**
 * 时间粒度
 */
export type TimeGranularity = 'minute' | 'hour' | 'day'

/**
 * 获取流量趋势数据
 *
 * 此 Hook 用于获取时间序列的流量趋势数据，包括流数量、包数量和字节数
 *
 * @param startTime 开始时间 (ISO 8601 格式)
 * @param endTime 结束时间 (ISO 8601 格式)
 * @param granularity 时间粒度 ('minute' | 'hour' | 'day')
 * @returns React Query 结果，包含流量趋势数据点数组
 */
export function useFlowTrend(
  startTime: string,
  endTime: string,
  granularity: TimeGranularity = 'minute'
) {
  return useQuery({
    queryKey: ['flowTrend', startTime, endTime, granularity],
    queryFn: async (): Promise<FlowTrendPoint[]> => {
      // 查询时间范围内的所有流
      const flows = await flowsApi.query({
        startTime,
        endTime,
        sortBy: 'startTime',
        sortOrder: 'asc',
      })

      // 按时间粒度聚合数据
      const granularityMs = {
        minute: 60 * 1000,
        hour: 60 * 60 * 1000,
        day: 24 * 60 * 60 * 1000,
      }[granularity]

      const buckets = new Map<number, FlowTrendPoint>()

      flows.forEach(flow => {
        const timestamp = new Date(flow.startTime).getTime()
        const bucketKey = Math.floor(timestamp / granularityMs) * granularityMs

        if (!buckets.has(bucketKey)) {
          buckets.set(bucketKey, {
            timestamp: new Date(bucketKey).toISOString(),
            flowCount: 0,
            packetCount: 0,
            byteCount: 0,
          })
        }

        const bucket = buckets.get(bucketKey)!
        bucket.flowCount += 1
        bucket.packetCount += flow.packetCount
        bucket.byteCount += flow.byteCount
      })

      // 转换为数组并排序
      const sortedData = Array.from(buckets.values()).sort(
        (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
      )

      // 性能优化: 数据点过多时进行采样
      return sampleDataPoints(sortedData, MAX_CHART_POINTS)
    },
    enabled: !!startTime && !!endTime,
    refetchInterval: 30000, // 每 30 秒自动刷新
    staleTime: 20000, // 20 秒内认为数据是新鲜的
  })
}

/**
 * 获取按协议分组的流量趋势
 *
 * 此 Hook 用于获取不同协议（TCP/UDP/ICMP）随时间变化的流量趋势
 *
 * @param startTime 开始时间
 * @param endTime 结束时间
 * @param granularity 时间粒度
 * @returns React Query 结果，包含按协议分组的趋势数据
 */
export function useProtocolTrend(
  startTime: string,
  endTime: string,
  granularity: TimeGranularity = 'minute'
) {
  return useQuery({
    queryKey: ['protocolTrend', startTime, endTime, granularity],
    queryFn: async (): Promise<ProtocolTrendPoint[]> => {
      const flows = await flowsApi.query({
        startTime,
        endTime,
        sortBy: 'startTime',
        sortOrder: 'asc',
      })

      const granularityMs = {
        minute: 60 * 1000,
        hour: 60 * 60 * 1000,
        day: 24 * 60 * 60 * 1000,
      }[granularity]

      // 创建嵌套的 Map: timestamp -> protocol -> data
      const buckets = new Map<number, Map<string, ProtocolTrendPoint>>()

      flows.forEach(flow => {
        const timestamp = new Date(flow.startTime).getTime()
        const bucketKey = Math.floor(timestamp / granularityMs) * granularityMs
        const protocol = flow.protocol.toUpperCase()

        if (!buckets.has(bucketKey)) {
          buckets.set(bucketKey, new Map())
        }

        const protocolMap = buckets.get(bucketKey)!

        if (!protocolMap.has(protocol)) {
          protocolMap.set(protocol, {
            timestamp: new Date(bucketKey).toISOString(),
            protocol,
            flowCount: 0,
            byteCount: 0,
          })
        }

        const point = protocolMap.get(protocol)!
        point.flowCount += 1
        point.byteCount += flow.byteCount
      })

      // 扁平化结果
      const result: ProtocolTrendPoint[] = []
      buckets.forEach(protocolMap => {
        protocolMap.forEach(point => {
          result.push(point)
        })
      })

      const sortedResult = result.sort(
        (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
      )

      // 性能优化: 数据点过多时进行采样
      return sampleDataPoints(sortedResult, MAX_CHART_POINTS)
    },
    enabled: !!startTime && !!endTime,
    refetchInterval: 30000,
    staleTime: 20000,
  })
}

/**
 * 获取 Top Talkers (最活跃的 IP 地址)
 *
 * 此 Hook 用于获取流量最大的源 IP 和目标 IP 地址
 *
 * @param startTime 开始时间
 * @param endTime 结束时间
 * @param topN 返回前 N 个结果 (默认 10)
 * @returns React Query 结果，包含 Top 源 IP 和 Top 目标 IP
 */
export function useTopTalkers(startTime?: string, endTime?: string, topN: number = 10) {
  return useQuery({
    queryKey: ['topTalkers', startTime, endTime, topN],
    queryFn: async (): Promise<TopTalkersData> => {
      const summary = await flowsApi.summary(startTime, endTime)

      return {
        topSourceIps: summary.topSourceIps.slice(0, topN),
        topDestIps: summary.topDestIps.slice(0, topN),
      }
    },
    refetchInterval: 30000,
    staleTime: 20000,
  })
}

/**
 * 获取协议分布数据
 *
 * 此 Hook 用于获取不同网络协议的流量分布统计
 *
 * @param startTime 开始时间
 * @param endTime 结束时间
 * @returns React Query 结果，包含协议分布数据
 */
export function useProtocolDistribution(startTime?: string, endTime?: string) {
  return useQuery({
    queryKey: ['protocolDistribution', startTime, endTime],
    queryFn: async (): Promise<ProtocolDistribution> => {
      const summary = await flowsApi.summary(startTime, endTime)

      return {
        protocols: summary.topProtocols,
      }
    },
    refetchInterval: 30000,
    staleTime: 20000,
  })
}

/**
 * 获取策略效果趋势
 *
 * 此 Hook 用于获取策略动作（Allow/Deny/Log）随时间的变化趋势
 *
 * @param startTime 开始时间
 * @param endTime 结束时间
 * @param granularity 时间粒度
 * @returns React Query 结果，包含策略效果数据点
 */
export function usePolicyEffectiveness(
  startTime: string,
  endTime: string,
  granularity: TimeGranularity = 'hour'
) {
  return useQuery({
    queryKey: ['policyEffectiveness', startTime, endTime, granularity],
    queryFn: async (): Promise<PolicyEffectivenessPoint[]> => {
      const flows = await flowsApi.query({
        startTime,
        endTime,
        sortBy: 'startTime',
        sortOrder: 'asc',
      })

      const granularityMs = {
        minute: 60 * 1000,
        hour: 60 * 60 * 1000,
        day: 24 * 60 * 60 * 1000,
      }[granularity]

      const buckets = new Map<number, PolicyEffectivenessPoint>()

      flows.forEach(flow => {
        const timestamp = new Date(flow.startTime).getTime()
        const bucketKey = Math.floor(timestamp / granularityMs) * granularityMs

        if (!buckets.has(bucketKey)) {
          buckets.set(bucketKey, {
            timestamp: new Date(bucketKey).toISOString(),
            allow: 0,
            deny: 0,
            log: 0,
          })
        }

        const bucket = buckets.get(bucketKey)!
        const action = flow.policyAction.toLowerCase() as 'allow' | 'deny' | 'log'

        if (action in bucket) {
          bucket[action] += 1
        }
      })

      const sortedData = Array.from(buckets.values()).sort(
        (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
      )

      // 性能优化: 数据点过多时进行采样
      return sampleDataPoints(sortedData, MAX_CHART_POINTS)
    },
    enabled: !!startTime && !!endTime,
    refetchInterval: 30000,
    staleTime: 20000,
  })
}

/**
 * 获取 Top 策略（最常命中的策略）
 *
 * 此 Hook 用于获取命中次数最多的策略列表
 *
 * @param topN 返回前 N 个策略 (默认 10)
 * @returns React Query 结果，包含 Top 策略数据
 */
export function useTopPolicies(topN: number = 10) {
  return useQuery({
    queryKey: ['topPolicies', topN],
    queryFn: async (): Promise<TopPolicyData[]> => {
      // 获取所有策略
      const policies = await policiesApi.list()

      // 获取所有策略的统计数据
      const statsPromises = policies.map(async policy => {
        try {
          const stats = await policiesApi.stats(policy.ruleId)
          return {
            ruleId: policy.ruleId,
            hitCount: stats.hitCount,
            description: policy.description,
          }
        } catch {
          // 如果某个策略的统计数据获取失败，返回默认值
          return {
            ruleId: policy.ruleId,
            hitCount: 0,
            description: policy.description,
          }
        }
      })

      const allStats = await Promise.all(statsPromises)

      // 按命中次数排序并返回前 N 个
      return allStats
        .sort((a, b) => b.hitCount - a.hitCount)
        .slice(0, topN)
    },
    refetchInterval: 30000,
    staleTime: 20000,
  })
}

/**
 * 获取实时流量统计
 *
 * 此 Hook 用于获取当前实时流量的摘要统计信息
 * 提供更频繁的刷新间隔以支持实时仪表盘
 *
 * @returns React Query 结果，包含实时流量统计
 */
export function useRealtimeStats() {
  return useQuery({
    queryKey: ['realtimeStats'],
    queryFn: () => flowsApi.summary(),
    refetchInterval: 5000, // 每 5 秒刷新一次
    staleTime: 3000, // 3 秒内认为数据是新鲜的
  })
}
