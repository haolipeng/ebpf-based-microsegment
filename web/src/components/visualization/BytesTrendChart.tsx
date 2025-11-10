/* eslint-disable @typescript-eslint/no-explicit-any */
import { useMemo, useState } from 'react'
import ReactECharts from 'echarts-for-react'
import { Card, Radio, Spin, Alert, Button, Space } from 'antd'
import { ReloadOutlined, DownloadOutlined } from '@ant-design/icons'
import { useFlowTrend, type TimeGranularity } from '../../hooks/useVisualization'
import { getBaseChartOption, formatBytes, formatTimeAxis, CHART_COLORS } from '../../utils/chartHelpers'

interface BytesTrendChartProps {
  startTime: string
  endTime: string
  height?: number
  showToolbar?: boolean
}

/**
 * 字节数趋势面积图组件
 *
 * 显示时间序列的字节数趋势，使用面积图和渐变填充
 * Y 轴自动格式化为 KB/MB/GB/TB
 */
export default function BytesTrendChart({
  startTime,
  endTime,
  height = 400,
  showToolbar = true,
}: BytesTrendChartProps) {
  const [granularity, setGranularity] = useState<TimeGranularity>('minute')

  const { data, isLoading, error, refetch } = useFlowTrend(startTime, endTime, granularity)

  const option = useMemo(() => {
    if (!data || data.length === 0) {
      return null
    }

    const baseOption = getBaseChartOption()

    return {
      ...baseOption,
      title: {
        text: 'Bytes Trend',
        left: 'center',
        textStyle: {
          fontSize: 16,
          fontWeight: 'bold',
        },
      },
      tooltip: {
        ...baseOption.tooltip,
        formatter: (params: any) => {
          if (Array.isArray(params)) {
            const param = params[0]
            return `
              <div style="padding: 8px;">
                <div style="margin-bottom: 4px;">${formatTimeAxis(param.name, granularity)}</div>
                <div style="color: ${CHART_COLORS.success};">
                  <strong>Bytes:</strong> ${formatBytes(param.value)}
                </div>
              </div>
            `
          }
          return ''
        },
      },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: data.map(d => formatTimeAxis(d.timestamp, granularity)),
        axisLabel: {
          rotate: 45,
        },
      },
      yAxis: {
        type: 'value',
        name: 'Bytes',
        axisLabel: {
          formatter: (value: number) => formatBytes(value),
        },
        splitLine: {
          lineStyle: {
            type: 'dashed',
          },
        },
      },
      series: [
        {
          name: 'Bytes',
          type: 'line',
          smooth: true,
          symbol: 'circle',
          symbolSize: 6,
          data: data.map(d => d.byteCount),
          lineStyle: {
            width: 3,
            color: CHART_COLORS.success,
          },
          itemStyle: {
            color: CHART_COLORS.success,
          },
          areaStyle: {
            color: {
              type: 'linear',
              x: 0,
              y: 0,
              x2: 0,
              y2: 1,
              colorStops: [
                {
                  offset: 0,
                  color: CHART_COLORS.success + '50',
                },
                {
                  offset: 0.5,
                  color: CHART_COLORS.success + '30',
                },
                {
                  offset: 1,
                  color: CHART_COLORS.success + '10',
                },
              ],
            },
          },
          emphasis: {
            focus: 'series',
          },
        },
      ],
      dataZoom: [
        {
          type: 'inside',
          start: 0,
          end: 100,
        },
        {
          type: 'slider',
          start: 0,
          end: 100,
          height: 20,
          bottom: 10,
        },
      ],
    }
  }, [data, granularity])

  const handleRefresh = () => {
    refetch()
  }

  const handleExport = () => {
    const echartInstance = (window as any).__BYTES_TREND_CHART__
    if (echartInstance) {
      const url = echartInstance.getDataURL({
        type: 'png',
        pixelRatio: 2,
        backgroundColor: '#fff',
      })
      const link = document.createElement('a')
      link.href = url
      link.download = `bytes-trend-${Date.now()}.png`
      link.click()
    }
  }

  return (
    <Card
      title="Bytes Trend Chart"
      extra={
        showToolbar && (
          <Space>
            <Radio.Group value={granularity} onChange={e => setGranularity(e.target.value)} size="small">
              <Radio.Button value="minute">1 Min</Radio.Button>
              <Radio.Button value="hour">1 Hour</Radio.Button>
              <Radio.Button value="day">1 Day</Radio.Button>
            </Radio.Group>
            <Button icon={<ReloadOutlined />} onClick={handleRefresh} size="small">
              Refresh
            </Button>
            <Button icon={<DownloadOutlined />} onClick={handleExport} size="small">
              Export
            </Button>
          </Space>
        )
      }
      styles={{ body: { padding: '16px' } }}
    >
      {isLoading && (
        <div style={{ textAlign: 'center', padding: '60px 0' }}>
          <Spin size="large" tip="Loading..." />
        </div>
      )}

      {error && (
        <Alert
          message="Load Failed"
          description={error instanceof Error ? error.message : 'Unknown error'}
          type="error"
          showIcon
        />
      )}

      {!isLoading && !error && data && data.length === 0 && (
        <Alert
          message="No Data"
          description="No flow data in the current time range"
          type="info"
          showIcon
        />
      )}

      {!isLoading && !error && data && data.length > 0 && option && (
        <ReactECharts
          option={option}
          style={{ height: `${height}px` }}
          opts={{ renderer: 'canvas' }}
          onChartReady={(chart: any) => {
            ;(window as any).__BYTES_TREND_CHART__ = chart
          }}
        />
      )}
    </Card>
  )
}
