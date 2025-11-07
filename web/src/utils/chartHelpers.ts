/**
 * 图表工具函数
 * 提供通用的图表配置、颜色主题、格式化函数等
 */

// 颜色主题
export const CHART_COLORS = {
  primary: '#1890ff',
  success: '#52c41a',
  warning: '#faad14',
  error: '#f5222d',
  info: '#13c2c2',

  // 协议颜色
  tcp: '#1890ff',
  udp: '#52c41a',
  icmp: '#faad14',
  any: '#8c8c8c',

  // 动作颜色
  allow: '#52c41a',
  deny: '#f5222d',
  log: '#1890ff',
}

// 通用图表配置
export const getBaseChartOption = () => ({
  backgroundColor: 'transparent',
  textStyle: {
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial',
  },
  grid: {
    left: '3%',
    right: '4%',
    bottom: '3%',
    top: '10%',
    containLabel: true,
  },
  tooltip: {
    trigger: 'axis',
    backgroundColor: 'rgba(50, 50, 50, 0.9)',
    borderColor: '#333',
    borderWidth: 1,
    textStyle: {
      color: '#fff',
    },
  },
})

/**
 * 格式化字节数为人类可读格式
 */
export function formatBytes(bytes: number, decimals = 2): string {
  if (bytes === 0) return '0 Bytes'

  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB']

  const i = Math.floor(Math.log(bytes) / Math.log(k))

  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i]
}

/**
 * 格式化数字为千分位格式
 */
export function formatNumber(num: number): string {
  return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}

/**
 * 格式化时间戳为可读格式
 */
export function formatTimestamp(timestamp: string | number): string {
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  // 小于 1 分钟
  if (diff < 60 * 1000) {
    return 'Just now'
  }

  // 小于 1 小时
  if (diff < 60 * 60 * 1000) {
    const minutes = Math.floor(diff / (60 * 1000))
    return `${minutes}m ago`
  }

  // 小于 1 天
  if (diff < 24 * 60 * 60 * 1000) {
    const hours = Math.floor(diff / (60 * 60 * 1000))
    return `${hours}h ago`
  }

  // 大于 1 天
  return date.toLocaleDateString() + ' ' + date.toLocaleTimeString()
}

/**
 * 格式化时间轴标签
 */
export function formatTimeAxis(timestamp: string | number, granularity: 'minute' | 'hour' | 'day' = 'minute'): string {
  const date = new Date(timestamp)

  switch (granularity) {
    case 'minute':
      return date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' })
    case 'hour':
      return date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' })
    case 'day':
      return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
    default:
      return date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' })
  }
}

/**
 * 数据聚合 - 按时间粒度聚合数据点
 */
export function aggregateDataPoints<T extends { timestamp: string | number; value: number }>(
  data: T[],
  granularity: 'minute' | 'hour' | 'day'
): T[] {
  if (data.length === 0) return []

  const granularityMs = {
    minute: 60 * 1000,
    hour: 60 * 60 * 1000,
    day: 24 * 60 * 60 * 1000,
  }[granularity]

  const buckets = new Map<number, T[]>()

  // 将数据点分组到时间桶中
  data.forEach(point => {
    const timestamp = new Date(point.timestamp).getTime()
    const bucketKey = Math.floor(timestamp / granularityMs) * granularityMs

    if (!buckets.has(bucketKey)) {
      buckets.set(bucketKey, [])
    }
    buckets.get(bucketKey)!.push(point)
  })

  // 聚合每个桶的数据
  const aggregated: T[] = []
  buckets.forEach((points, bucketKey) => {
    const sum = points.reduce((acc, p) => acc + p.value, 0)
    aggregated.push({
      ...points[0],
      timestamp: new Date(bucketKey).toISOString(),
      value: sum,
    })
  })

  return aggregated.sort((a, b) =>
    new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
  )
}

/**
 * 数据采样 - 当数据点过多时进行采样
 */
export function sampleDataPoints<T>(data: T[], maxPoints: number): T[] {
  if (data.length <= maxPoints) return data

  const step = Math.ceil(data.length / maxPoints)
  return data.filter((_, index) => index % step === 0)
}

/**
 * 获取协议颜色
 */
export function getProtocolColor(protocol: string): string {
  const p = protocol.toLowerCase()
  return CHART_COLORS[p as keyof typeof CHART_COLORS] || CHART_COLORS.any
}

/**
 * 获取动作颜色
 */
export function getActionColor(action: string): string {
  const a = action.toLowerCase()
  return CHART_COLORS[a as keyof typeof CHART_COLORS] || CHART_COLORS.info
}

/**
 * 生成渐变色配置
 */
export function getGradientColor(color: string) {
  return {
    type: 'linear' as const,
    x: 0,
    y: 0,
    x2: 0,
    y2: 1,
    colorStops: [
      {
        offset: 0,
        color: color,
      },
      {
        offset: 1,
        color: color + '20', // 添加透明度
      },
    ],
  }
}
