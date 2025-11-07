import { Row, Col, Card, Statistic } from 'antd'
import {
  FileProtectOutlined,
  CheckCircleOutlined,
  StopOutlined,
  SafetyOutlined,
  CloseCircleOutlined,
  FileSearchOutlined,
} from '@ant-design/icons'
import type { Policy } from '../../types/policy'

interface PolicyStatsCardsProps {
  policies: Policy[]
  loading?: boolean
}

export default function PolicyStatsCards({ policies, loading = false }: PolicyStatsCardsProps) {
  // Calculate statistics
  const totalPolicies = policies.length
  const enabledPolicies = policies.filter(p => p.enabled !== false).length
  const disabledPolicies = totalPolicies - enabledPolicies

  const allowPolicies = policies.filter(p => p.action === 'allow').length
  const denyPolicies = policies.filter(p => p.action === 'deny').length
  const logPolicies = policies.filter(p => p.action === 'log').length

  return (
    <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
      <Col xs={24} sm={12} lg={8}>
        <Card>
          <Statistic
            title="Total Policies"
            value={totalPolicies}
            prefix={<FileProtectOutlined />}
            loading={loading}
          />
        </Card>
      </Col>

      <Col xs={24} sm={12} lg={8}>
        <Card>
          <Statistic
            title="Enabled"
            value={enabledPolicies}
            prefix={<CheckCircleOutlined />}
            valueStyle={{ color: '#52c41a' }}
            loading={loading}
          />
        </Card>
      </Col>

      <Col xs={24} sm={12} lg={8}>
        <Card>
          <Statistic
            title="Disabled"
            value={disabledPolicies}
            prefix={<StopOutlined />}
            valueStyle={{ color: '#8c8c8c' }}
            loading={loading}
          />
        </Card>
      </Col>

      <Col xs={24} sm={12} lg={8}>
        <Card>
          <Statistic
            title="Allow Policies"
            value={allowPolicies}
            prefix={<SafetyOutlined />}
            valueStyle={{ color: '#52c41a' }}
            loading={loading}
          />
        </Card>
      </Col>

      <Col xs={24} sm={12} lg={8}>
        <Card>
          <Statistic
            title="Deny Policies"
            value={denyPolicies}
            prefix={<CloseCircleOutlined />}
            valueStyle={{ color: '#ff4d4f' }}
            loading={loading}
          />
        </Card>
      </Col>

      <Col xs={24} sm={12} lg={8}>
        <Card>
          <Statistic
            title="Log Policies"
            value={logPolicies}
            prefix={<FileSearchOutlined />}
            valueStyle={{ color: '#1890ff' }}
            loading={loading}
          />
        </Card>
      </Col>
    </Row>
  )
}
