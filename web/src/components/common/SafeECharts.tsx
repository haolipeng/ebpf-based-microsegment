/* eslint-disable @typescript-eslint/no-explicit-any */
import { useEffect, useRef } from 'react'
import ReactECharts from 'echarts-for-react'
import type { EChartsOption } from 'echarts'

interface SafeEChartsProps {
  option: EChartsOption | null
  style?: React.CSSProperties
  opts?: any
  onChartReady?: (chart: any) => void
  onEvents?: Record<string, (params: any) => void>
  notMerge?: boolean
  lazyUpdate?: boolean
}

/**
 * SafeECharts - 安全的 ECharts 包装器
 *
 * 解决 echarts-for-react 在组件卸载时可能出现的 "Cannot read properties of undefined (reading 'disconnect')" 错误
 */
export default function SafeECharts({
  option,
  style,
  opts = { renderer: 'canvas' },
  onChartReady,
  onEvents,
  notMerge = true,
  lazyUpdate = true,
}: SafeEChartsProps) {
  const chartRef = useRef<any>(null)
  const instanceRef = useRef<any>(null)

  // 安全的清理函数
  useEffect(() => {
    return () => {
      try {
        if (instanceRef.current) {
          instanceRef.current.dispose()
          instanceRef.current = null
        }
      } catch (error) {
        // 忽略清理错误
        console.debug('ECharts cleanup error (ignored):', error)
      }
    }
  }, [])

  const handleChartReady = (chart: any) => {
    if (chart) {
      instanceRef.current = chart
      onChartReady?.(chart)
    }
  }

  // 如果没有 option，不渲染图表
  if (!option) {
    return null
  }

  return (
    <ReactECharts
      ref={chartRef}
      option={option}
      style={style || { height: '100%', width: '100%' }}
      opts={opts}
      notMerge={notMerge}
      lazyUpdate={lazyUpdate}
      onChartReady={handleChartReady}
      onEvents={onEvents}
    />
  )
}
