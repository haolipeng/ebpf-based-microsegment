import { Card, Row, Col, Statistic } from 'antd'
import {
  FileTextOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  CloudServerOutlined,
  DatabaseOutlined,
} from '@ant-design/icons'
import type { FlowSummary } from '../../types/flow'
import { formatBytes, formatNumber } from '../../utils/format'

interface FlowSummaryCardsProps {
  summary: FlowSummary
  loading?: boolean
}

export default function FlowSummaryCards({ summary, loading = false }: FlowSummaryCardsProps) {
  return (
    <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
      <Col xs={24} sm={12} lg={8}>
        <Card bordered={false} loading={loading}>
          <Statistic
            title="Total Flows"
            value={formatNumber(summary.totalFlows)}
            prefix={<FileTextOutlined />}
            valueStyle={{ color: '#1890ff' }}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={8}>
        <Card bordered={false} loading={loading}>
          <Statistic
            title="Active Flows"
            value={formatNumber(summary.activeFlows)}
            prefix={<CheckCircleOutlined />}
            valueStyle={{ color: '#52c41a' }}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={8}>
        <Card bordered={false} loading={loading}>
          <Statistic
            title="Closed Flows"
            value={formatNumber(summary.closedFlows)}
            prefix={<CloseCircleOutlined />}
            valueStyle={{ color: '#8c8c8c' }}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={8}>
        <Card bordered={false} loading={loading}>
          <Statistic
            title="Total Packets"
            value={formatNumber(summary.totalPackets)}
            prefix={<CloudServerOutlined />}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={8}>
        <Card bordered={false} loading={loading}>
          <Statistic
            title="Total Bytes"
            value={formatBytes(summary.totalBytes)}
            prefix={<DatabaseOutlined />}
          />
        </Card>
      </Col>
      <Col xs={24} sm={12} lg={8}>
        <Card bordered={false} loading={loading}>
          <Row gutter={16}>
            <Col span={12}>
              <Statistic
                title="Allowed"
                value={formatNumber(summary.allowedFlows)}
                valueStyle={{ color: '#52c41a', fontSize: 20 }}
              />
            </Col>
            <Col span={12}>
              <Statistic
                title="Denied"
                value={formatNumber(summary.deniedFlows)}
                valueStyle={{ color: '#ff4d4f', fontSize: 20 }}
              />
            </Col>
          </Row>
        </Card>
      </Col>
    </Row>
  )
}
