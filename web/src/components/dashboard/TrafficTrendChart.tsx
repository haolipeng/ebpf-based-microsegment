import { Card, Segmented } from 'antd'
import ReactECharts from 'echarts-for-react'
import { useState } from 'react'

interface TrafficTrendChartProps {
  data?: {
    timestamps: string[]
    bytes: number[]
    packets: number[]
  }
  loading?: boolean
}

type TimeRange = '1h' | '6h' | '24h'

export default function TrafficTrendChart({ data, loading = false }: TrafficTrendChartProps) {
  const [timeRange, setTimeRange] = useState<TimeRange>('1h')

  const option = {
    title: {
      text: 'Traffic Trend',
      left: 'center',
      textStyle: {
        fontSize: 14,
        fontWeight: 'normal',
      },
    },
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross',
      },
    },
    legend: {
      data: ['Bytes', 'Packets'],
      bottom: 10,
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '15%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: data?.timestamps || [],
    },
    yAxis: [
      {
        type: 'value',
        name: 'Bytes',
        axisLabel: {
          formatter: (value: number) => {
            if (value >= 1e9) return `${(value / 1e9).toFixed(1)}GB`
            if (value >= 1e6) return `${(value / 1e6).toFixed(1)}MB`
            if (value >= 1e3) return `${(value / 1e3).toFixed(1)}KB`
            return `${value}B`
          },
        },
      },
      {
        type: 'value',
        name: 'Packets',
        splitLine: { show: false },
      },
    ],
    series: [
      {
        name: 'Bytes',
        type: 'line',
        smooth: true,
        data: data?.bytes || [],
        areaStyle: {
          opacity: 0.3,
        },
        yAxisIndex: 0,
      },
      {
        name: 'Packets',
        type: 'line',
        smooth: true,
        data: data?.packets || [],
        areaStyle: {
          opacity: 0.3,
        },
        yAxisIndex: 1,
      },
    ],
  }

  return (
    <Card
      title="Traffic Trend"
      loading={loading}
      extra={
        <Segmented
          options={[
            { label: '1 Hour', value: '1h' },
            { label: '6 Hours', value: '6h' },
            { label: '24 Hours', value: '24h' },
          ]}
          value={timeRange}
          onChange={value => setTimeRange(value as TimeRange)}
        />
      }
    >
      <ReactECharts option={option} style={{ height: '300px' }} />
    </Card>
  )
}
