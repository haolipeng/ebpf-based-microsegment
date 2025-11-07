/* eslint-disable @typescript-eslint/no-explicit-any */
import { useMemo, useState } from 'react'
import ReactECharts from 'echarts-for-react'
import { Card, Radio, Spin, Alert, Button, Space, Tabs } from 'antd'
import { ReloadOutlined, DownloadOutlined } from '@ant-design/icons'
import { useTopTalkers } from '../../hooks/useVisualization'
import { getBaseChartOption, formatBytes, formatNumber, CHART_COLORS } from '../../utils/chartHelpers'
import type { IpStats } from '../../types/flow'

interface TopTalkersChartProps {
  startTime?: string
  endTime?: string
  height?: number
  showToolbar?: boolean
  /**
   * 显示的 Top N 数量
   * @default 10
   */
  topN?: number
  /**
   * 点击 IP 时的回调函数
   */
  onIpClick?: (ip: string) => void
}

type SortBy = 'flows' | 'bytes'
type TabType = 'source' | 'dest'

/**
 * Top Talkers 横向柱状图组件
 *
 * 显示 Top 10 源 IP 和目标 IP，支持按流量或字节数排序
 * 支持点击跳转到流量列表
 */
export default function TopTalkersChart({
  startTime,
  endTime,
  height = 400,
  showToolbar = true,
  topN = 10,
  onIpClick,
}: TopTalkersChartProps) {
  const [sortBy, setSortBy] = useState<SortBy>('flows')
  const [activeTab, setActiveTab] = useState<TabType>('source')

  const { data, isLoading, error, refetch } = useTopTalkers(startTime, endTime, topN)

  const option = useMemo(() => {
    if (!data) {
      return null
    }

    const statsData: IpStats[] = activeTab === 'source' ? data.topSourceIps : data.topDestIps

    if (statsData.length === 0) {
      return null
    }

    // 根据排序方式排序
    const sortedData = [...statsData].sort((a, b) => {
      if (sortBy === 'flows') {
        return a.flowCount - b.flowCount // 升序，因为横向柱状图从下到上
      } else {
        return a.byteCount - b.byteCount
      }
    })

    const baseOption = getBaseChartOption()

    return {
      ...baseOption,
      title: {
        text: activeTab === 'source' ? 'Top 源 IP' : 'Top 目标 IP',
        left: 'center',
        textStyle: {
          fontSize: 16,
          fontWeight: 'bold',
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
            const param = params[0]
            const dataIndex = param.dataIndex
            const ipData = sortedData[dataIndex]

            return `
              <div style="padding: 8px;">
                <div style="margin-bottom: 8px; font-weight: bold;">${ipData.ip}</div>
                <div style="margin: 4px 0;">
                  <strong>流数量:</strong> ${formatNumber(ipData.flowCount)}
                </div>
                <div style="margin: 4px 0;">
                  <strong>包数量:</strong> ${formatNumber(ipData.packetCount)}
                </div>
                <div style="margin: 4px 0;">
                  <strong>字节数:</strong> ${formatBytes(ipData.byteCount)}
                </div>
              </div>
            `
          }
          return ''
        },
      },
      grid: {
        left: '15%',
        right: '10%',
        bottom: '3%',
        top: '15%',
        containLabel: true,
      },
      xAxis: {
        type: 'value',
        name: sortBy === 'flows' ? '流数量' : '字节数',
        axisLabel: {
          formatter: (value: number) => {
            return sortBy === 'flows' ? formatNumber(value) : formatBytes(value)
          },
        },
        splitLine: {
          lineStyle: {
            type: 'dashed',
          },
        },
      },
      yAxis: {
        type: 'category',
        data: sortedData.map(d => d.ip),
        axisLabel: {
          formatter: (value: string) => {
            // 截断过长的 IP 地址
            return value.length > 20 ? value.substring(0, 18) + '...' : value
          },
        },
      },
      series: [
        {
          name: sortBy === 'flows' ? '流数量' : '字节数',
          type: 'bar',
          data: sortedData.map(d => (sortBy === 'flows' ? d.flowCount : d.byteCount)),
          itemStyle: {
            color: {
              type: 'linear',
              x: 0,
              y: 0,
              x2: 1,
              y2: 0,
              colorStops: [
                {
                  offset: 0,
                  color: CHART_COLORS.primary + '60',
                },
                {
                  offset: 1,
                  color: CHART_COLORS.primary,
                },
              ],
            },
            borderRadius: [0, 4, 4, 0],
          },
          emphasis: {
            itemStyle: {
              color: CHART_COLORS.primary,
              shadowBlur: 10,
              shadowColor: 'rgba(0, 0, 0, 0.3)',
            },
          },
          label: {
            show: true,
            position: 'right',
            formatter: (params: any) => {
              return sortBy === 'flows'
                ? formatNumber(params.value)
                : formatBytes(params.value)
            },
            fontSize: 11,
          },
        },
      ],
    }
  }, [data, sortBy, activeTab])

  const handleRefresh = () => {
    refetch()
  }

  const handleExport = () => {
    const echartInstance = (window as any).__TOP_TALKERS_CHART__
    if (echartInstance) {
      const url = echartInstance.getDataURL({
        type: 'png',
        pixelRatio: 2,
        backgroundColor: '#fff',
      })
      const link = document.createElement('a')
      link.href = url
      link.download = `top-talkers-${activeTab}-${Date.now()}.png`
      link.click()
    }
  }

  const handleChartClick = (params: any) => {
    if (params.componentType === 'series' && onIpClick) {
      const statsData: IpStats[] = activeTab === 'source' ? data!.topSourceIps : data!.topDestIps
      const sortedData = [...statsData].sort((a, b) => {
        if (sortBy === 'flows') {
          return a.flowCount - b.flowCount
        } else {
          return a.byteCount - b.byteCount
        }
      })
      const ip = sortedData[params.dataIndex]?.ip
      if (ip) {
        onIpClick(ip)
      }
    }
  }

  return (
    <Card
      title="Top Talkers"
      extra={
        showToolbar && (
          <Space>
            <Radio.Group value={sortBy} onChange={e => setSortBy(e.target.value)} size="small">
              <Radio.Button value="flows">按流量</Radio.Button>
              <Radio.Button value="bytes">按字节</Radio.Button>
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
      <Tabs
        activeKey={activeTab}
        onChange={key => setActiveTab(key as TabType)}
        items={[
          { key: 'source', label: '源 IP' },
          { key: 'dest', label: '目标 IP' },
        ]}
        style={{ marginBottom: 16 }}
      />

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

      {!isLoading && !error && data && (
        <>
          {(!data.topSourceIps.length && activeTab === 'source') ||
          (!data.topDestIps.length && activeTab === 'dest') ? (
            <Alert
              message="暂无数据"
              description="当前时间范围内没有流量数据"
              type="info"
              showIcon
            />
          ) : (
            option && (
              <ReactECharts
                option={option}
                style={{ height: `${height}px` }}
                opts={{ renderer: 'canvas' }}
                onChartReady={(chart: any) => {
                  ;(window as any).__TOP_TALKERS_CHART__ = chart
                }}
                onEvents={{
                  click: handleChartClick,
                }}
              />
            )
          )}
        </>
      )}
    </Card>
  )
}
