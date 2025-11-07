import { Card, Statistic, Row, Col, Progress } from 'antd'
import {
  CloudServerOutlined,
  DatabaseOutlined,
  SafetyOutlined,
  FileTextOutlined,
} from '@ant-design/icons'
import type { Agent } from '../../types/agent'
import { formatBytes, formatNumber } from '../../utils/format'

interface AgentMetricsCardProps {
  agent: Agent
}

export default function AgentMetricsCard({ agent }: AgentMetricsCardProps) {
  const metrics = agent.metrics

  if (!metrics) {
    return (
      <Card title="Performance Metrics" bordered={false}>
        <div style={{ textAlign: 'center', padding: '40px 0', color: '#999' }}>
          <p>No metrics available</p>
        </div>
      </Card>
    )
  }

  const getCpuColor = (usage: number) => {
    if (usage >= 80) return '#ff4d4f'
    if (usage >= 60) return '#faad14'
    return '#52c41a'
  }

  return (
    <Card title="Performance Metrics" bordered={false}>
      {/* CPU Usage */}
      <div style={{ marginBottom: 24 }}>
        <div style={{ marginBottom: 8, display: 'flex', justifyContent: 'space-between' }}>
          <span>CPU Usage</span>
          <span style={{ fontWeight: 'bold' }}>{metrics.cpuUsage.toFixed(1)}%</span>
        </div>
        <Progress
          percent={metrics.cpuUsage}
          strokeColor={getCpuColor(metrics.cpuUsage)}
          showInfo={false}
        />
      </div>

      {/* Memory Usage */}
      <div style={{ marginBottom: 24 }}>
        <div style={{ marginBottom: 8, display: 'flex', justifyContent: 'space-between' }}>
          <span>Memory Usage</span>
          <span style={{ fontWeight: 'bold' }}>{formatBytes(metrics.memoryUsage)}</span>
        </div>
      </div>

      {/* Statistics Grid */}
      <Row gutter={[16, 16]}>
        <Col span={12}>
          <Statistic
            title="Flows Reported"
            value={formatNumber(metrics.flowsReported)}
            prefix={<FileTextOutlined />}
            valueStyle={{ fontSize: 20 }}
          />
        </Col>
        <Col span={12}>
          <Statistic
            title="Active Policies"
            value={formatNumber(metrics.activePolicies)}
            prefix={<SafetyOutlined />}
            valueStyle={{ fontSize: 20 }}
          />
        </Col>
        {metrics.packetsProcessed !== undefined && (
          <Col span={12}>
            <Statistic
              title="Packets Processed"
              value={formatNumber(metrics.packetsProcessed)}
              prefix={<CloudServerOutlined />}
              valueStyle={{ fontSize: 20 }}
            />
          </Col>
        )}
        {metrics.packetsDropped !== undefined && (
          <Col span={12}>
            <Statistic
              title="Packets Dropped"
              value={formatNumber(metrics.packetsDropped)}
              prefix={<DatabaseOutlined />}
              valueStyle={{ fontSize: 20, color: metrics.packetsDropped > 0 ? '#faad14' : undefined }}
            />
          </Col>
        )}
      </Row>
    </Card>
  )
}
