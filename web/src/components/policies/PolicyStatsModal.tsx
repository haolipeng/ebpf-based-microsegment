import { Modal, Descriptions, Spin, Alert, Statistic, Row, Col } from 'antd'
import { usePolicyStats } from '../../hooks/usePolicies'
import type { Policy } from '../../types/policy'

interface PolicyStatsModalProps {
  open: boolean
  policy: Policy | null
  onClose: () => void
}

export default function PolicyStatsModal({ open, policy, onClose }: PolicyStatsModalProps) {
  const { data: stats, isLoading, error } = usePolicyStats(policy?.ruleId || 0)

  return (
    <Modal
      title={policy ? `Policy Statistics - Rule ${policy.ruleId}` : 'Policy Statistics'}
      open={open}
      onCancel={onClose}
      footer={null}
      width={700}
    >
      {isLoading && (
        <div style={{ textAlign: 'center', padding: '40px 0' }}>
          <Spin size="large" />
        </div>
      )}

      {error && (
        <Alert
          message="Failed to load statistics"
          description={error instanceof Error ? error.message : 'Unknown error occurred'}
          type="error"
          showIcon
        />
      )}

      {!isLoading && !error && policy && stats && (
        <>
          {/* Policy Details */}
          <Descriptions title="Policy Details" bordered column={2} size="small" style={{ marginBottom: 24 }}>
            <Descriptions.Item label="Rule ID" span={2}>
              {policy.ruleId}
            </Descriptions.Item>
            <Descriptions.Item label="Source IP">{policy.srcIp}</Descriptions.Item>
            <Descriptions.Item label="Destination IP">{policy.dstIp}</Descriptions.Item>
            <Descriptions.Item label="Source Port">
              {policy.srcPort === 0 ? 'Any' : policy.srcPort}
            </Descriptions.Item>
            <Descriptions.Item label="Destination Port">
              {policy.dstPort === 0 ? 'Any' : policy.dstPort}
            </Descriptions.Item>
            <Descriptions.Item label="Protocol">{policy.protocol.toUpperCase()}</Descriptions.Item>
            <Descriptions.Item label="Action">{policy.action.toUpperCase()}</Descriptions.Item>
            <Descriptions.Item label="Priority">{policy.priority}</Descriptions.Item>
            <Descriptions.Item label="Status">
              {policy.enabled !== false ? 'Enabled' : 'Disabled'}
            </Descriptions.Item>
            {policy.description && (
              <Descriptions.Item label="Description" span={2}>
                {policy.description}
              </Descriptions.Item>
            )}
          </Descriptions>

          {/* Statistics */}
          <Row gutter={16}>
            <Col span={12}>
              <Statistic
                title="Total Hits"
                value={stats.hitCount}
                valueStyle={{ color: stats.hitCount > 0 ? '#3f8600' : '#999' }}
              />
            </Col>
            <Col span={12}>
              <Statistic
                title="Last Hit"
                value={stats.lastHit ? new Date(stats.lastHit).toLocaleString() : 'Never'}
                valueStyle={{ fontSize: 16 }}
              />
            </Col>
          </Row>

          {stats.hitCount === 0 && (
            <Alert
              message="No hits recorded"
              description="This policy has not been matched by any traffic yet."
              type="info"
              showIcon
              style={{ marginTop: 16 }}
            />
          )}
        </>
      )}
    </Modal>
  )
}
