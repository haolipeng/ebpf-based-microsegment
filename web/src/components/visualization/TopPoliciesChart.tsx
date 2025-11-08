/* eslint-disable @typescript-eslint/no-explicit-any */
import { useMemo } from 'react'
import { Card, Spin, Alert, Button, Space, Tag } from 'antd'
import { ReloadOutlined, DownloadOutlined } from '@ant-design/icons'
import SafeECharts from '../common/SafeECharts'
import { useTopPolicies } from '../../hooks/useVisualization'
import { getBaseChartOption, formatNumber, CHART_COLORS } from '../../utils/chartHelpers'

interface TopPoliciesChartProps {
  height?: number
  showToolbar?: boolean
  /**
   * 显示的 Top N 数量
   * @default 10
   */
  topN?: number
  /**
   * 点击策略时的回调函数
   */
  onPolicyClick?: (ruleId: number) => void
}

/**
 * Top 策略横向柱状图组件
 *
 * 显示命中次数最多的策略列表
 * 支持点击跳转到策略详情
 */
export default function TopPoliciesChart({
  height = 400,
  showToolbar = true,
  topN = 10,
  onPolicyClick,
}: TopPoliciesChartProps) {
  const { data, isLoading, error, refetch } = useTopPolicies(topN)

  const option = useMemo(() => {
    if (!data || data.length === 0) {
      return null
    }

    // 按命中次数升序排序（横向柱状图从下到上）
    const sortedData = [...data].sort((a, b) => a.hitCount - b.hitCount)

    const baseOption = getBaseChartOption()

    return {
      ...baseOption,
      title: {
        text: 'Top 策略',
        subtext: `命中次数最多的 ${sortedData.length} 个策略`,
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
            const policyData = sortedData[dataIndex]

            return `
              <div style="padding: 8px;">
                <div style="margin-bottom: 8px;">
                  <strong>Rule ID: ${policyData.ruleId}</strong>
                </div>
                ${
                  policyData.description
                    ? `<div style="margin-bottom: 8px; color: #666; max-width: 300px;">
                        ${policyData.description}
                      </div>`
                    : ''
                }
                <div style="margin: 4px 0;">
                  <strong>命中次数:</strong> ${formatNumber(policyData.hitCount)}
                </div>
                <div style="margin-top: 8px; font-size: 12px; color: #999;">
                  点击查看详情
                </div>
              </div>
            `
          }
          return ''
        },
      },
      grid: {
        left: '12%',
        right: '10%',
        bottom: '3%',
        top: '20%',
        containLabel: true,
      },
      xAxis: {
        type: 'value',
        name: '命中次数',
        axisLabel: {
          formatter: (value: number) => formatNumber(value),
        },
        splitLine: {
          lineStyle: {
            type: 'dashed',
          },
        },
      },
      yAxis: {
        type: 'category',
        data: sortedData.map(d => `Rule ${d.ruleId}`),
        axisLabel: {
          formatter: (value: string) => {
            return value.length > 15 ? value.substring(0, 13) + '...' : value
          },
        },
      },
      series: [
        {
          name: '命中次数',
          type: 'bar',
          data: sortedData.map(d => d.hitCount),
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
                  color: CHART_COLORS.warning + '60',
                },
                {
                  offset: 1,
                  color: CHART_COLORS.warning,
                },
              ],
            },
            borderRadius: [0, 4, 4, 0],
          },
          emphasis: {
            itemStyle: {
              color: CHART_COLORS.warning,
              shadowBlur: 10,
              shadowColor: 'rgba(0, 0, 0, 0.3)',
            },
          },
          label: {
            show: true,
            position: 'right',
            formatter: (params: any) => formatNumber(params.value),
            fontSize: 11,
          },
        },
      ],
    }
  }, [data])

  const handleRefresh = () => {
    refetch()
  }

  const handleExport = () => {
    const echartInstance = (window as any).__TOP_POLICIES_CHART__
    if (echartInstance) {
      const url = echartInstance.getDataURL({
        type: 'png',
        pixelRatio: 2,
        backgroundColor: '#fff',
      })
      const link = document.createElement('a')
      link.href = url
      link.download = `top-policies-${Date.now()}.png`
      link.click()
    }
  }

  const handleChartClick = (params: any) => {
    if (params.componentType === 'series' && onPolicyClick && data) {
      const sortedData = [...data].sort((a, b) => a.hitCount - b.hitCount)
      const ruleId = sortedData[params.dataIndex]?.ruleId
      if (ruleId !== undefined) {
        onPolicyClick(ruleId)
      }
    }
  }

  return (
    <Card
      title="Top 命中策略"
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

      {!isLoading && !error && data && data.length === 0 && (
        <Alert
          message="暂无数据"
          description="没有策略命中记录"
          type="info"
          showIcon
        />
      )}

      {!isLoading && !error && data && data.length > 0 && (
        <>
          {/* 数据摘要 */}
          <div style={{ marginBottom: 16, padding: '8px 0', borderBottom: '1px solid #f0f0f0' }}>
            <Space>
              <Tag color="blue">总策略数: {data.length}</Tag>
              <Tag color="green">
                总命中数: {formatNumber(data.reduce((sum, p) => sum + p.hitCount, 0))}
              </Tag>
              {data[0] && (
                <Tag color="orange">
                  最高命中: Rule {data[0].ruleId} ({formatNumber(data[0].hitCount)} 次)
                </Tag>
              )}
            </Space>
          </div>

          <SafeECharts
            option={option}
            style={{ height: `${height}px` }}
            onChartReady={(chart: any) => {
              if (chart) {
                ;(window as any).__TOP_POLICIES_CHART__ = chart
              }
            }}
            onEvents={{
              click: handleChartClick,
            }}
          />
        </>
      )}
    </Card>
  )
}
