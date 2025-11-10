/* eslint-disable @typescript-eslint/no-explicit-any */
import { useMemo } from 'react'
import ReactECharts from 'echarts-for-react'
import { Card, Spin, Alert, Button, Space } from 'antd'
import { ReloadOutlined, DownloadOutlined } from '@ant-design/icons'
import { useProtocolDistribution } from '../../hooks/useVisualization'
import { getBaseChartOption, formatBytes, getProtocolColor } from '../../utils/chartHelpers'

interface ProtocolDonutChartProps {
  startTime?: string
  endTime?: string
  height?: number
  showToolbar?: boolean
}

/**
 * 协议流量环形图组件
 *
 * 显示不同协议的字节数分布
 * 中心显示总字节数，支持鼠标悬停高亮
 */
export default function ProtocolDonutChart({
  startTime,
  endTime,
  height = 400,
  showToolbar = true,
}: ProtocolDonutChartProps) {
  const { data, isLoading, error, refetch } = useProtocolDistribution(startTime, endTime)

  const option = useMemo(() => {
    if (!data || data.protocols.length === 0) {
      return null
    }

    const baseOption = getBaseChartOption()

    // 计算总字节数
    const totalBytes = data.protocols.reduce((sum, p) => sum + p.byteCount, 0)

    // 构建环形图数据
    const donutData = data.protocols.map(protocol => ({
      name: protocol.protocol.toUpperCase(),
      value: protocol.byteCount,
      itemStyle: {
        color: getProtocolColor(protocol.protocol),
      },
    }))

    return {
      ...baseOption,
      title: {
        text: formatBytes(totalBytes),
        subtext: 'Total Bytes',
        left: 'center',
        top: 'center',
        textStyle: {
          fontSize: 24,
          fontWeight: 'bold',
          color: '#333',
        },
        subtextStyle: {
          fontSize: 14,
          color: '#999',
        },
      },
      tooltip: {
        trigger: 'item',
        backgroundColor: 'rgba(50, 50, 50, 0.9)',
        borderColor: '#333',
        borderWidth: 1,
        textStyle: {
          color: '#fff',
        },
        formatter: (params: any) => {
          const percent = ((params.value / totalBytes) * 100).toFixed(2)
          return `
            <div style="padding: 8px;">
              <div style="margin-bottom: 4px; font-weight: bold;">${params.name}</div>
              <div style="margin: 4px 0;">
                <strong>Bytes:</strong> ${formatBytes(params.value)}
              </div>
              <div style="margin: 4px 0;">
                <strong>Percentage:</strong> ${percent}%
              </div>
            </div>
          `
        },
      },
      legend: {
        orient: 'horizontal',
        bottom: 'bottom',
        data: donutData.map(d => d.name),
      },
      series: [
        {
          name: 'Protocol Bytes Distribution',
          type: 'pie',
          radius: ['40%', '70%'],
          center: ['50%', '45%'],
          avoidLabelOverlap: true,
          data: donutData,
          label: {
            show: true,
            position: 'outside',
            formatter: (params: any) => {
              const percent = ((params.value / totalBytes) * 100).toFixed(1)
              return `${params.name}: ${percent}%`
            },
            fontSize: 12,
          },
          labelLine: {
            show: true,
            length: 15,
            length2: 10,
          },
          emphasis: {
            itemStyle: {
              shadowBlur: 10,
              shadowOffsetX: 0,
              shadowColor: 'rgba(0, 0, 0, 0.5)',
            },
            label: {
              show: true,
              fontSize: 14,
              fontWeight: 'bold',
              formatter: (params: any) => {
                return `${params.name}\n${formatBytes(params.value)}`
              },
            },
          },
          animationType: 'scale',
          animationEasing: 'elasticOut',
        },
      ],
    }
  }, [data])

  const handleRefresh = () => {
    refetch()
  }

  const handleExport = () => {
    const echartInstance = (window as any).__PROTOCOL_DONUT_CHART__
    if (echartInstance) {
      const url = echartInstance.getDataURL({
        type: 'png',
        pixelRatio: 2,
        backgroundColor: '#fff',
      })
      const link = document.createElement('a')
      link.href = url
      link.download = `protocol-donut-${Date.now()}.png`
      link.click()
    }
  }

  return (
    <Card
      title="Protocol Bytes Distribution Donut Chart"
      extra={
        showToolbar && (
          <Space>
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

      {!isLoading && !error && data && data.protocols.length === 0 && (
        <Alert
          message="No Data"
          description="No flow data in the current time range"
          type="info"
          showIcon
        />
      )}

      {!isLoading && !error && data && data.protocols.length > 0 && option && (
        <ReactECharts
          option={option}
          style={{ height: `${height}px` }}
          opts={{ renderer: 'canvas' }}
          onChartReady={(chart: any) => {
            ;(window as any).__PROTOCOL_DONUT_CHART__ = chart
          }}
        />
      )}
    </Card>
  )
}
