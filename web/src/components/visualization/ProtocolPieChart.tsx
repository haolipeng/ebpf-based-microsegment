/* eslint-disable @typescript-eslint/no-explicit-any */
import { useMemo } from 'react'
import ReactECharts from 'echarts-for-react'
import { Card, Spin, Alert, Button, Space } from 'antd'
import { ReloadOutlined, DownloadOutlined } from '@ant-design/icons'
import { useProtocolDistribution } from '../../hooks/useVisualization'
import { getBaseChartOption, formatNumber, getProtocolColor } from '../../utils/chartHelpers'

interface ProtocolPieChartProps {
  startTime?: string
  endTime?: string
  height?: number
  showToolbar?: boolean
}

/**
 * 协议分布饼图组件
 *
 * 显示 TCP/UDP/ICMP 等协议的流量占比
 * 使用百分比标签和颜色编码
 */
export default function ProtocolPieChart({
  startTime,
  endTime,
  height = 400,
  showToolbar = true,
}: ProtocolPieChartProps) {
  const { data, isLoading, error, refetch } = useProtocolDistribution(startTime, endTime)

  const option = useMemo(() => {
    if (!data || data.protocols.length === 0) {
      return null
    }

    const baseOption = getBaseChartOption()

    // 计算总流量
    const totalFlows = data.protocols.reduce((sum, p) => sum + p.flowCount, 0)

    // 构建饼图数据
    const pieData = data.protocols.map(protocol => ({
      name: protocol.protocol.toUpperCase(),
      value: protocol.flowCount,
      itemStyle: {
        color: getProtocolColor(protocol.protocol),
      },
    }))

    return {
      ...baseOption,
      title: {
        text: '协议分布',
        subtext: `总流量: ${formatNumber(totalFlows)}`,
        left: 'center',
        textStyle: {
          fontSize: 16,
          fontWeight: 'bold',
        },
        subtextStyle: {
          fontSize: 12,
          color: '#666',
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
          const percent = ((params.value / totalFlows) * 100).toFixed(2)
          return `
            <div style="padding: 8px;">
              <div style="margin-bottom: 4px; font-weight: bold;">${params.name}</div>
              <div style="margin: 4px 0;">
                <strong>流数量:</strong> ${formatNumber(params.value)}
              </div>
              <div style="margin: 4px 0;">
                <strong>占比:</strong> ${percent}%
              </div>
            </div>
          `
        },
      },
      legend: {
        orient: 'vertical',
        left: 'left',
        top: 'middle',
        data: pieData.map(d => d.name),
      },
      series: [
        {
          name: '协议分布',
          type: 'pie',
          radius: '60%',
          center: ['60%', '50%'],
          data: pieData,
          label: {
            formatter: (params: any) => {
              const percent = ((params.value / totalFlows) * 100).toFixed(1)
              return `{name|${params.name}}\n{percent|${percent}%}`
            },
            rich: {
              name: {
                fontSize: 12,
                fontWeight: 'bold',
                lineHeight: 20,
              },
              percent: {
                fontSize: 11,
                color: '#999',
                lineHeight: 18,
              },
            },
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
    const echartInstance = (window as any).__PROTOCOL_PIE_CHART__
    if (echartInstance) {
      const url = echartInstance.getDataURL({
        type: 'png',
        pixelRatio: 2,
        backgroundColor: '#fff',
      })
      const link = document.createElement('a')
      link.href = url
      link.download = `protocol-pie-${Date.now()}.png`
      link.click()
    }
  }

  return (
    <Card
      title="协议分布饼图"
      extra={
        showToolbar && (
          <Space>
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

      {!isLoading && !error && data && data.protocols.length === 0 && (
        <Alert
          message="暂无数据"
          description="当前时间范围内没有流量数据"
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
            ;(window as any).__PROTOCOL_PIE_CHART__ = chart
          }}
        />
      )}
    </Card>
  )
}
