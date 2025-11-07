/* eslint-disable @typescript-eslint/no-explicit-any */
import { useMemo, useState } from 'react'
import ReactECharts from 'echarts-for-react'
import { Card, Radio, Spin, Alert, Button, Space } from 'antd'
import { ReloadOutlined, DownloadOutlined } from '@ant-design/icons'
import { useFlowTrend, type TimeGranularity } from '../../hooks/useVisualization'
import { getBaseChartOption, formatNumber, formatTimeAxis, CHART_COLORS } from '../../utils/chartHelpers'

interface FlowTrendChartProps {
  /**
   * 开始时间 (ISO 8601 格式)
   */
  startTime: string
  /**
   * 结束时间 (ISO 8601 格式)
   */
  endTime: string
  /**
   * 图表高度 (像素)
   * @default 400
   */
  height?: number
  /**
   * 是否显示工具栏
   * @default true
   */
  showToolbar?: boolean
}

/**
 * 流量趋势折线图组件
 *
 * 显示时间序列的流数量趋势，支持不同时间粒度（分钟/小时/天）
 * 提供工具栏以刷新数据和导出图表
 */
export default function FlowTrendChart({
  startTime,
  endTime,
  height = 400,
  showToolbar = true,
}: FlowTrendChartProps) {
  const [granularity, setGranularity] = useState<TimeGranularity>('minute')

  const { data, isLoading, error, refetch } = useFlowTrend(startTime, endTime, granularity)

  // ECharts 配置
  const option = useMemo(() => {
    if (!data || data.length === 0) {
      return null
    }

    const baseOption = getBaseChartOption()

    return {
      ...baseOption,
      title: {
        text: '流量趋势',
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
                <div style="color: ${CHART_COLORS.primary};">
                  <strong>流数量:</strong> ${formatNumber(param.value)}
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
          formatter: (value: string) => value,
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
      series: [
        {
          name: '流数量',
          type: 'line',
          smooth: true,
          symbol: 'circle',
          symbolSize: 6,
          data: data.map(d => d.flowCount),
          lineStyle: {
            width: 2,
            color: CHART_COLORS.primary,
          },
          itemStyle: {
            color: CHART_COLORS.primary,
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
                  color: CHART_COLORS.primary + '40',
                },
                {
                  offset: 1,
                  color: CHART_COLORS.primary + '10',
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

  // 处理刷新
  const handleRefresh = () => {
    refetch()
  }

  // 处理导出
  const handleExport = () => {
    const echartInstance = (window as any).__FLOW_TREND_CHART__
    if (echartInstance) {
      const url = echartInstance.getDataURL({
        type: 'png',
        pixelRatio: 2,
        backgroundColor: '#fff',
      })
      const link = document.createElement('a')
      link.href = url
      link.download = `flow-trend-${Date.now()}.png`
      link.click()
    }
  }

  return (
    <Card
      title="流量趋势图"
      extra={
        showToolbar && (
          <Space>
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
            // 保存实例以便导出
            ;(window as any).__FLOW_TREND_CHART__ = chart
          }}
        />
      )}
    </Card>
  )
}
