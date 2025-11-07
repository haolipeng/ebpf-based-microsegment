import { Typography, Card, Spin, Alert, Row, Col, Button, Space } from 'antd'
import {
  ClusterOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  CloudServerOutlined,
  SafetyOutlined,
  DatabaseOutlined,
  ReloadOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons'
import { useQueryClient } from '@tanstack/react-query'
import { useState, useEffect } from 'react'
import { useAgents, useHealthCheck } from '../../hooks/useAgents'
import { useFlowSummary } from '../../hooks/useFlows'
import { usePolicies } from '../../hooks/usePolicies'
import MetricCard from '../../components/dashboard/MetricCard'
import ProtocolChart from '../../components/dashboard/ProtocolChart'
import PolicyActionChart from '../../components/dashboard/PolicyActionChart'
import TopTalkersList from '../../components/dashboard/TopTalkersList'
import FlowTrendChart from '../../components/visualization/FlowTrendChart'
import BytesTrendChart from '../../components/visualization/BytesTrendChart'
import ProtocolTrendChart from '../../components/visualization/ProtocolTrendChart'
import TopTalkersChart from '../../components/visualization/TopTalkersChart'
import ProtocolPieChart from '../../components/visualization/ProtocolPieChart'
import ProtocolDonutChart from '../../components/visualization/ProtocolDonutChart'
import PolicyEffectivenessChart from '../../components/visualization/PolicyEffectivenessChart'
import TopPoliciesChart from '../../components/visualization/TopPoliciesChart'
import { formatBytes, formatNumber } from '../../utils/format'

const { Title, Paragraph } = Typography

export default function Dashboard() {
  const queryClient = useQueryClient()
  const [lastUpdated, setLastUpdated] = useState(new Date())
  const [isRefreshing, setIsRefreshing] = useState(false)

  // 时间范围: 默认最近 1 小时
  const [timeRange] = useState(() => {
    const endTime = new Date()
    const startTime = new Date(endTime.getTime() - 60 * 60 * 1000) // 1 小时前
    return {
      startTime: startTime.toISOString(),
      endTime: endTime.toISOString(),
    }
  })

  const { data: agentsData, isLoading: agentsLoading, error: agentsError } = useAgents()
  const { data: healthData, isLoading: healthLoading } = useHealthCheck()
  const { data: flowSummary, isLoading: flowsLoading } = useFlowSummary()
  const { data: policiesData, isLoading: policiesLoading } = usePolicies()

  // Update last updated time when data changes
  useEffect(() => {
    if (!flowsLoading) {
      setLastUpdated(new Date())
    }
  }, [flowSummary, flowsLoading])

  // Manual refresh handler
  const handleRefresh = async () => {
    setIsRefreshing(true)
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['agents'] }),
      queryClient.invalidateQueries({ queryKey: ['health'] }),
      queryClient.invalidateQueries({ queryKey: ['flowSummary'] }),
      queryClient.invalidateQueries({ queryKey: ['policies'] }),
    ])
    setIsRefreshing(false)
    setLastUpdated(new Date())
  }

  // Calculate metrics
  const onlineAgents = agentsData?.agents.filter(a => a.status === 'online').length || 0
  const totalAgents = agentsData?.total || 0
  const totalBytes = flowSummary?.totalBytes || 0
  const totalPolicies = policiesData?.total || 0

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title level={2} style={{ margin: 0 }}>Dashboard</Title>
        <Space>
          <span style={{ fontSize: 14, color: '#8c8c8c' }}>
            <ClockCircleOutlined /> Last updated: {lastUpdated.toLocaleTimeString()}
          </span>
          <Button
            icon={<ReloadOutlined spin={isRefreshing} />}
            onClick={handleRefresh}
            loading={isRefreshing}
          >
            Refresh
          </Button>
        </Space>
      </div>

      <Paragraph>
        Welcome to eBPF Microsegmentation Dashboard. Real-time system monitoring and overview.
      </Paragraph>

      {/* Health Check Status */}
      <Card
        title="System Health"
        style={{ marginBottom: 24 }}
        extra={
          healthLoading ? (
            <Spin size="small" />
          ) : healthData?.healthy ? (
            <CheckCircleOutlined style={{ color: '#52c41a', fontSize: 20 }} />
          ) : (
            <CloseCircleOutlined style={{ color: '#ff4d4f', fontSize: 20 }} />
          )
        }
      >
        {healthLoading ? (
          <Spin tip="Checking system health..." />
        ) : healthData?.healthy ? (
          <Alert message="System is healthy" type="success" showIcon />
        ) : (
          <Alert message="System health check failed" type="error" showIcon />
        )}
      </Card>

      {/* Key Metrics */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <MetricCard
            title="Active Agents"
            value={`${onlineAgents}/${totalAgents}`}
            prefix={<ClusterOutlined />}
            loading={agentsLoading}
            valueStyle={{ color: onlineAgents > 0 ? '#3f8600' : '#cf1322' }}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <MetricCard
            title="Total Traffic"
            value={formatBytes(totalBytes)}
            prefix={<CloudServerOutlined />}
            loading={flowsLoading}
            valueStyle={{ color: '#1890ff' }}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <MetricCard
            title="Active Policies"
            value={totalPolicies}
            prefix={<SafetyOutlined />}
            loading={policiesLoading}
            valueStyle={{ color: '#722ed1' }}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <MetricCard
            title="Total Flows"
            value={formatNumber(flowSummary?.totalFlows || 0)}
            prefix={<DatabaseOutlined />}
            loading={flowsLoading}
            valueStyle={{ color: '#13c2c2' }}
          />
        </Col>
      </Row>

      {/* Flow Statistics */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} lg={12}>
          <Card title="Flow Statistics" loading={flowsLoading}>
            {flowSummary ? (
              <Row gutter={16}>
                <Col span={12}>
                  <div style={{ marginBottom: 16 }}>
                    <div style={{ fontSize: 14, color: '#8c8c8c' }}>Active Flows</div>
                    <div style={{ fontSize: 24, fontWeight: 500 }}>
                      {formatNumber(flowSummary.activeFlows)}
                    </div>
                  </div>
                  <div>
                    <div style={{ fontSize: 14, color: '#8c8c8c' }}>Total Packets</div>
                    <div style={{ fontSize: 24, fontWeight: 500 }}>
                      {formatNumber(flowSummary.totalPackets)}
                    </div>
                  </div>
                </Col>
                <Col span={12}>
                  <div style={{ marginBottom: 16 }}>
                    <div style={{ fontSize: 14, color: '#8c8c8c' }}>Closed Flows</div>
                    <div style={{ fontSize: 24, fontWeight: 500 }}>
                      {formatNumber(flowSummary.closedFlows)}
                    </div>
                  </div>
                  <div>
                    <div style={{ fontSize: 14, color: '#8c8c8c' }}>Total Bytes</div>
                    <div style={{ fontSize: 24, fontWeight: 500 }}>
                      {formatBytes(flowSummary.totalBytes)}
                    </div>
                  </div>
                </Col>
              </Row>
            ) : (
              <Alert message="No flow data available" type="info" />
            )}
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card title="Policy Statistics" loading={flowsLoading}>
            {flowSummary ? (
              <Row gutter={16}>
                <Col span={12}>
                  <div style={{ marginBottom: 16 }}>
                    <div style={{ fontSize: 14, color: '#8c8c8c' }}>Allowed Flows</div>
                    <div style={{ fontSize: 24, fontWeight: 500, color: '#52c41a' }}>
                      {formatNumber(flowSummary.allowedFlows)}
                    </div>
                  </div>
                </Col>
                <Col span={12}>
                  <div style={{ marginBottom: 16 }}>
                    <div style={{ fontSize: 14, color: '#8c8c8c' }}>Denied Flows</div>
                    <div style={{ fontSize: 24, fontWeight: 500, color: '#ff4d4f' }}>
                      {formatNumber(flowSummary.deniedFlows)}
                    </div>
                  </div>
                </Col>
              </Row>
            ) : (
              <Alert message="No policy data available" type="info" />
            )}
          </Card>
        </Col>
      </Row>

      {/* Charts */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} lg={12}>
          <ProtocolChart data={flowSummary?.topProtocols} loading={flowsLoading} />
        </Col>
        <Col xs={24} lg={12}>
          <PolicyActionChart
            allowedFlows={flowSummary?.allowedFlows || 0}
            deniedFlows={flowSummary?.deniedFlows || 0}
            loading={flowsLoading}
          />
        </Col>
      </Row>

      {/* Top Talkers */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24}>
          <TopTalkersList
            sourceIps={flowSummary?.topSourceIps}
            destIps={flowSummary?.topDestIps}
            loading={flowsLoading}
          />
        </Col>
      </Row>

      {/* Trend Charts */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} xl={12}>
          <FlowTrendChart
            startTime={timeRange.startTime}
            endTime={timeRange.endTime}
            height={350}
          />
        </Col>
        <Col xs={24} xl={12}>
          <BytesTrendChart
            startTime={timeRange.startTime}
            endTime={timeRange.endTime}
            height={350}
          />
        </Col>
      </Row>

      {/* Protocol Trend Chart */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24}>
          <ProtocolTrendChart
            startTime={timeRange.startTime}
            endTime={timeRange.endTime}
            height={350}
          />
        </Col>
      </Row>

      {/* Top Talkers & Protocol Distribution */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} xl={12}>
          <TopTalkersChart
            startTime={timeRange.startTime}
            endTime={timeRange.endTime}
            height={400}
            topN={10}
          />
        </Col>
        <Col xs={24} xl={12}>
          <Row gutter={[16, 16]}>
            <Col xs={24}>
              <ProtocolPieChart
                startTime={timeRange.startTime}
                endTime={timeRange.endTime}
                height={180}
              />
            </Col>
            <Col xs={24}>
              <ProtocolDonutChart
                startTime={timeRange.startTime}
                endTime={timeRange.endTime}
                height={180}
              />
            </Col>
          </Row>
        </Col>
      </Row>

      {/* Policy Effectiveness */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} xl={12}>
          <PolicyEffectivenessChart
            startTime={timeRange.startTime}
            endTime={timeRange.endTime}
            height={350}
          />
        </Col>
        <Col xs={24} xl={12}>
          <TopPoliciesChart height={350} topN={10} />
        </Col>
      </Row>

      {/* Error Handling */}
      {agentsError && (
        <Alert
          message="Failed to load agents"
          description={agentsError instanceof Error ? agentsError.message : 'Unknown error'}
          type="error"
          showIcon
          style={{ marginBottom: 24 }}
        />
      )}
    </div>
  )
}
