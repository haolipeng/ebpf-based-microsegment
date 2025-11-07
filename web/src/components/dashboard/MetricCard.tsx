import { Card, Statistic, Skeleton } from 'antd'
import type { ReactNode } from 'react'

interface MetricCardProps {
  title: string
  value?: number | string
  prefix?: ReactNode
  suffix?: string
  loading?: boolean
  valueStyle?: React.CSSProperties
  precision?: number
}

export default function MetricCard({
  title,
  value,
  prefix,
  suffix,
  loading = false,
  valueStyle,
  precision = 0,
}: MetricCardProps) {
  return (
    <Card bordered={false} style={{ height: '100%' }}>
      {loading ? (
        <Skeleton active paragraph={{ rows: 1 }} />
      ) : (
        <Statistic
          title={title}
          value={value}
          prefix={prefix}
          suffix={suffix}
          valueStyle={valueStyle}
          precision={precision}
        />
      )}
    </Card>
  )
}
