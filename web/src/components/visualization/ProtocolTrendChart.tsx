/* eslint-disable @typescript-eslint/no-explicit-any */
import { useMemo, useState } from 'react'
import ReactECharts from 'echarts-for-react'
import { Card, Radio, Spin, Alert, Button, Space, Switch } from 'antd'
import { ReloadOutlined, DownloadOutlined } from '@ant-design/icons'
import { useProtocolTrend, type TimeGranularity } from '../../hooks/useVisualization'
import { getBaseChartOption, formatNumber, formatTimeAxis, getProtocolColor } from '../../utils/chartHelpers'

interface ProtocolTrendChartProps {
  startTime: string
  endTime: string
  height?: number
  showToolbar?: boolean
}

/**
 * 按协议分组的多折线图组件
 *
 * 显示不同协议（TCP/UDP/ICMP）随时间变化的流量趋势
 * 支持堆叠模式切换和图例交互
 */
export default function ProtocolTrendChart({
  startTime,
  endTime,
  height = 400,
  showToolbar = true,
}: ProtocolTrendChartProps) {
  const [granularity, setGranularity] = useState<TimeGranularity>('minute')
  const [stacked, setStacked] = useState(false)

  const { data, isLoading, error, refetch } = useProtocolTrend(startTime, endTime, granularity)

  const option = useMemo(() => {
    if (!data || data.length === 0) {
      return null
    }

    const baseOption = getBaseChartOption()

    // 提取所有唯一的协议
    const protocols = Array.from(new Set(data.map(d => d.protocol)))

    // 提取所有唯一的时间戳
    const timestamps = Array.from(new Set(data.map(d => d.timestamp))).sort(
      (a, b) => new Date(a).getTime() - new Date(b).getTime()
    )

    // 为每个协议构建时间序列数据
    const seriesData = protocols.map(protocol => {
      const protocolData = timestamps.map(timestamp => {
        const point = data.find(d => d.timestamp === timestamp && d.protocol === protocol)
        return point ? point.flowCount : 0
      })

      return {
        name: protocol,
        type: 'line' as const,
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        data: protocolData,
        stack: stacked ? 'total' : undefined,
        lineStyle: {
          width: 2,
          color: getProtocolColor(protocol),
        },
        itemStyle: {
          color: getProtocolColor(protocol),
        },
        areaStyle: stacked
          ? {
              color: {
                type: 'linear',
                x: 0,
                y: 0,
                x2: 0,
                y2: 1,
                colorStops: [
                  {
                    offset: 0,
                    color: getProtocolColor(protocol) + '60',
                  },
                  {
                    offset: 1,
                    color: getProtocolColor(protocol) + '20',
                  },
                ],
              },
            }
          : undefined,
        emphasis: {
          focus: 'series',
        },
      }
    })

    return {
      ...baseOption,
      title: {
        text: '协议流量趋势',
        left: 'center',
        textStyle: {
          fontSize: 16,
          fontWeight: 'bold',
        },
      },
      legend: {
        data: protocols,
        top: 30,
        selected: protocols.reduce((acc, protocol) => {
          acc[protocol] = true
          return acc
        }, {} as Record<string, boolean>),
      },
      tooltip: {
        ...baseOption.tooltip,
        trigger: 'axis',
        axisPointer: {
          type: 'cross',
          label: {
            backgroundColor: '#6a7985',
          },
        },
        formatter: (params: any) => {
          if (Array.isArray(params) && params.length > 0) {
            const timestamp = params[0].name
            let html = `<div style="padding: 8px;">
              <div style="margin-bottom: 8px; font-weight: bold;">${formatTimeAxis(timestamp, granularity)}</div>`

            params.forEach((param: any) => {
              html += `
                <div style="margin: 4px 0; display: flex; align-items: center; gap: 8px;">
                  <span style="display: inline-block; width: 10px; height: 10px; border-radius: 50%; background-color: ${param.color};"></span>
                  <span>${param.seriesName}:</span>
                  <strong>${formatNumber(param.value)}</strong>
                </div>`
            })

            html += '</div>'
            return html
          }
          return ''
        },
      },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: timestamps.map(t => formatTimeAxis(t, granularity)),
        axisLabel: {
          rotate: 45,
        },
      },
      yAxis: {
        type: 'value',
        name: '流数量',
        axisLabel: {
          formatter: (value: number) => formatNumber(value),
        },
        splitLine: {
          lineStyle: {
            type: 'dashed',
          },
        },
      },
      series: seriesData,
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
  }, [data, granularity, stacked])

  const handleRefresh = () => {
    refetch()
  }

  const handleExport = () => {
    const echartInstance = (window as any).__PROTOCOL_TREND_CHART__
    if (echartInstance) {
      const url = echartInstance.getDataURL({
        type: 'png',
        pixelRatio: 2,
        backgroundColor: '#fff',
      })
      const link = document.createElement('a')
      link.href = url
      link.download = `protocol-trend-${Date.now()}.png`
      link.click()
    }
  }

  return (
    <Card
      title="协议流量趋势图"
      extra={
        showToolbar && (
          <Space>
            <span style={{ fontSize: '12px', color: '#666' }}>堆叠模式:</span>
            <Switch checked={stacked} onChange={setStacked} size="small" />
            <Radio.Group value={granularity} onChange={e => setGranularity(e.target.value)} size="small">
              <Radio.Button value="minute">1分钟</Radio.Button>
              <Radio.Button value="hour">1小时</Radio.Button>
              <Radio.Button value="day">1天</Radio.Button>
            </Radio.Group>
            <Button icon={<ReloadOutlined />} onClick={handleRefresh} size="small">
              刷新
            </Button>
            <Button icon={<DownloadOutlined />} onClick={handleExport} size="small">
              导出
            </Button>
          </Space>
        )
      }
      styles={{ body: { padding: '16px' } }}
    >
      {isLoading && (
        <div style={{ textAlign: 'center', padding: '60px 0' }}>
          <Spin size="large" tip="加载中..." />
        </div>
      )}

      {error && (
        <Alert
          message="加载失败"
          description={error instanceof Error ? error.message : '未知错误'}
          type="error"
          showIcon
        />
      )}

      {!isLoading && !error && data && data.length === 0 && (
        <Alert
          message="暂无数据"
          description="当前时间范围内没有流量数据"
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
            ;(window as any).__PROTOCOL_TREND_CHART__ = chart
          }}
        />
      )}
    </Card>
  )
}
