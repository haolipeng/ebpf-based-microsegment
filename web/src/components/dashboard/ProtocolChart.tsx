import { Card } from 'antd'
import ReactECharts from 'echarts-for-react'
import type { ProtocolStats } from '../../types/flow'

interface ProtocolChartProps {
  data?: ProtocolStats[]
  loading?: boolean
}

export default function ProtocolChart({ data, loading = false }: ProtocolChartProps) {
  const chartData = data?.map(item => ({
    name: item.protocol,
    value: item.byteCount,
  })) || []

  const option = {
    title: {
      text: 'Protocol Distribution',
      left: 'center',
      textStyle: {
        fontSize: 14,
        fontWeight: 'normal',
      },
    },
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} bytes ({d}%)',
    },
    legend: {
      orient: 'vertical',
      right: 10,
      top: 'center',
    },
    series: [
      {
        name: 'Protocol',
        type: 'pie',
        radius: ['40%', '70%'],
        center: ['40%', '55%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 10,
          borderColor: '#fff',
          borderWidth: 2,
        },
        label: {
          show: false,
          position: 'center',
        },
        emphasis: {
          label: {
            show: true,
            fontSize: 20,
            fontWeight: 'bold',
          },
        },
        labelLine: {
          show: false,
        },
        data: chartData,
      },
    ],
    color: ['#5470c6', '#91cc75', '#fac858', '#ee6666', '#73c0de'],
  }

  return (
    <Card title="Protocol Distribution" loading={loading}>
      {chartData.length > 0 ? (
        <ReactECharts option={option} style={{ height: '300px' }} />
      ) : (
        <div style={{ height: '300px', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#999' }}>
          No protocol data available
        </div>
      )}
    </Card>
  )
}
