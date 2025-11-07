/* eslint-disable @typescript-eslint/no-explicit-any */
import { useMemo, useState } from 'react'
import ReactECharts from 'echarts-for-react'
import { Card, Radio, Spin, Alert, Button, Space } from 'antd'
import { ReloadOutlined, DownloadOutlined } from '@ant-design/icons'
import { usePolicyEffectiveness, type TimeGranularity } from '../../hooks/useVisualization'
import { getBaseChartOption, formatNumber, formatTimeAxis, getActionColor } from '../../utils/chartHelpers'

interface PolicyEffectivenessChartProps {
  startTime: string
  endTime: string
  height?: number
  showToolbar?: boolean
}

/**
 * 策略效果堆叠柱状图组件
 *
 * 显示 Allow/Deny/Log 策略动作随时间的变化趋势
 * 使用堆叠柱状图，颜色编码: Allow=绿色、Deny=红色、Log=蓝色
 */
export default function PolicyEffectivenessChart({
  startTime,
  endTime,
  height = 400,
  showToolbar = true,
}: PolicyEffectivenessChartProps) {
  const [granularity, setGranularity] = useState<TimeGranularity>('hour')

  const { data, isLoading, error, refetch } = usePolicyEffectiveness(startTime, endTime, granularity)

  const option = useMemo(() => {
    if (!data || data.length === 0) {
      return null
    }

    const baseOption = getBaseChartOption()

    // 提取时间戳
    const timestamps = data.map(d => formatTimeAxis(d.timestamp, granularity))

    return {
      ...baseOption,
      title: {
        text: '策略效果趋势',
        subtext: '按策略动作分组的流量统计',
        left: 'center',
        textStyle: {
          fontSize: 16,
          fontWeight: 'bold',
        },
        subtextStyle: {
          fontSize: 12,
          color: '#999',
        },
      },
      legend: {
        data: ['Allow', 'Deny', 'Log'],
        top: 40,
        selected: {
          Allow: true,
          Deny: true,
          Log: true,
        },
      },
      tooltip: {
        ...baseOption.tooltip,
        trigger: 'axis',
        axisPointer: {
          type: 'shadow',
        },
        formatter: (params: any) => {
          if (Array.isArray(params) && params.length > 0) {
            const timestamp = params[0].name
            let html = `<div style="padding: 8px;">
              <div style="margin-bottom: 8px; font-weight: bold;">${timestamp}</div>`

            let total = 0
            params.forEach((param: any) => {
              total += param.value
              html += `
                <div style="margin: 4px 0; display: flex; align-items: center; gap: 8px;">
                  <span style="display: inline-block; width: 10px; height: 10px; border-radius: 2px; background-color: ${param.color};"></span>
                  <span>${param.seriesName}:</span>
                  <strong>${formatNumber(param.value)}</strong>
                </div>`
            })

            html += `
              <div style="margin-top: 8px; padding-top: 8px; border-top: 1px solid #ddd;">
                <strong>总计: ${formatNumber(total)}</strong>
              </div>
            </div>`
            return html
          }
          return ''
        },
      },
      xAxis: {
        type: 'category',
        data: timestamps,
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
      series: [
        {
          name: 'Allow',
          type: 'bar',
          stack: 'total',
          data: data.map(d => d.allow),
          itemStyle: {
            color: getActionColor('allow'),
          },
          emphasis: {
            focus: 'series',
          },
          label: {
            show: false,
          },
        },
        {
          name: 'Deny',
          type: 'bar',
          stack: 'total',
          data: data.map(d => d.deny),
          itemStyle: {
            color: getActionColor('deny'),
          },
          emphasis: {
            focus: 'series',
          },
          label: {
            show: false,
          },
        },
        {
          name: 'Log',
          type: 'bar',
          stack: 'total',
          data: data.map(d => d.log),
          itemStyle: {
            color: getActionColor('log'),
          },
          emphasis: {
            focus: 'series',
          },
          label: {
            show: false,
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
    const echartInstance = (window as any).__POLICY_EFFECTIVENESS_CHART__
    if (echartInstance) {
      const url = echartInstance.getDataURL({
        type: 'png',
        pixelRatio: 2,
        backgroundColor: '#fff',
      })
      const link = document.createElement('a')
      link.href = url
      link.download = `policy-effectiveness-${Date.now()}.png`
      link.click()
    }
  }

  return (
    <Card
      title="策略效果趋势图"
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
          description="当前时间范围内没有策略数据"
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
            ;(window as any).__POLICY_EFFECTIVENESS_CHART__ = chart
          }}
        />
      )}
    </Card>
  )
}
