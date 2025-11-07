import { Card } from 'antd'
import ReactECharts from 'echarts-for-react'

interface PolicyActionChartProps {
  allowedFlows: number
  deniedFlows: number
  loading?: boolean
}

export default function PolicyActionChart({
  allowedFlows,
  deniedFlows,
  loading = false,
}: PolicyActionChartProps) {
  const total = allowedFlows + deniedFlows

  const option = {
    title: {
      text: 'Policy Actions',
      left: 'center',
      textStyle: {
        fontSize: 14,
        fontWeight: 'normal',
      },
    },
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} flows ({d}%)',
    },
    legend: {
      orient: 'vertical',
      right: 10,
      top: 'center',
    },
    series: [
      {
        name: 'Policy Action',
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
        data: [
          { value: allowedFlows, name: 'ALLOW' },
          { value: deniedFlows, name: 'DENY' },
        ],
      },
    ],
    color: ['#52c41a', '#ff4d4f'],
  }

  return (
    <Card title="Policy Actions" loading={loading}>
      {total > 0 ? (
        <ReactECharts option={option} style={{ height: '300px' }} />
      ) : (
        <div style={{ height: '300px', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#999' }}>
          No policy action data available
        </div>
      )}
    </Card>
  )
}
