import { useParams, useNavigate } from 'react-router-dom'
import { Typography, Button, Space, Alert, Skeleton, Row, Col } from 'antd'
import { ArrowLeftOutlined, ReloadOutlined } from '@ant-design/icons'
import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useAgent } from '../../hooks/useAgents'
import AgentInfoCard from '../../components/agents/AgentInfoCard'
import AgentMetricsCard from '../../components/agents/AgentMetricsCard'

const { Title } = Typography

export default function AgentDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { data: agent, isLoading, error } = useAgent(id!)
  const [isRefreshing, setIsRefreshing] = useState(false)

  const handleRefresh = async () => {
    setIsRefreshing(true)
    await queryClient.invalidateQueries({ queryKey: ['agent', id] })
    setIsRefreshing(false)
  }

  const handleBack = () => {
    navigate('/agents')
  }

  // Error state
  if (error) {
    return (
      <div>
        <Button icon={<ArrowLeftOutlined />} onClick={handleBack} style={{ marginBottom: 16 }}>
          Back to Agents
        </Button>
        <Alert
          message="Failed to load agent details"
          description={error instanceof Error ? error.message : 'Unknown error occurred'}
          type="error"
          showIcon
          action={
            <Button size="small" onClick={handleRefresh}>
              Retry
            </Button>
          }
        />
      </div>
    )
  }

  // Loading state
  if (isLoading) {
    return (
      <div>
        <Button icon={<ArrowLeftOutlined />} onClick={handleBack} style={{ marginBottom: 16 }}>
          Back to Agents
        </Button>
        <Skeleton active paragraph={{ rows: 8 }} />
      </div>
    )
  }

  // Agent not found
  if (!agent) {
    return (
      <div>
        <Button icon={<ArrowLeftOutlined />} onClick={handleBack} style={{ marginBottom: 16 }}>
          Back to Agents
        </Button>
        <Alert
          message="Agent not found"
          description={`Agent with ID "${id}" does not exist`}
          type="warning"
          showIcon
        />
      </div>
    )
  }

  return (
    <div>
      {/* Header */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 16,
        }}
      >
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={handleBack}>
            Back
          </Button>
          <Title level={2} style={{ margin: 0 }}>
            Agent: {agent.hostname}
          </Title>
        </Space>
        <Button
          icon={<ReloadOutlined spin={isRefreshing} />}
          onClick={handleRefresh}
          loading={isRefreshing}
        >
          Refresh
        </Button>
      </div>

      {/* Content Cards */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <AgentInfoCard agent={agent} />
        </Col>
        <Col xs={24} lg={12}>
          <AgentMetricsCard agent={agent} />
        </Col>
      </Row>
    </div>
  )
}
