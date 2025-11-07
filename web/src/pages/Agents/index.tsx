import { Typography, Card, Row, Col, Input, Select, Space, Button, Alert } from 'antd'
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons'
import { useState, useMemo } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useAgents } from '../../hooks/useAgents'
import AgentTable from '../../components/agents/AgentTable'
import type { Agent } from '../../types/agent'

const { Title, Paragraph } = Typography

export default function Agents() {
  const queryClient = useQueryClient()
  const { data: agentsData, isLoading, error } = useAgents()
  const [searchText, setSearchText] = useState('')
  const [statusFilter, setStatusFilter] = useState<Agent['status'] | 'all'>('all')
  const [isRefreshing, setIsRefreshing] = useState(false)

  // Filter agents based on search and status
  const filteredAgents = useMemo(() => {
    if (!agentsData?.agents) return []

    let filtered = agentsData.agents

    // Apply search filter
    if (searchText) {
      const lowerSearch = searchText.toLowerCase()
      filtered = filtered.filter(
        agent =>
          agent.hostname.toLowerCase().includes(lowerSearch) ||
          agent.ipAddress.toLowerCase().includes(lowerSearch) ||
          agent.id.toLowerCase().includes(lowerSearch)
      )
    }

    // Apply status filter
    if (statusFilter !== 'all') {
      filtered = filtered.filter(agent => agent.status === statusFilter)
    }

    return filtered
  }, [agentsData?.agents, searchText, statusFilter])

  const handleRefresh = async () => {
    setIsRefreshing(true)
    await queryClient.invalidateQueries({ queryKey: ['agents'] })
    setIsRefreshing(false)
  }

  return (
    <div>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 16,
        }}
      >
        <Title level={2} style={{ margin: 0 }}>
          Agents
        </Title>
        <Button
          icon={<ReloadOutlined spin={isRefreshing} />}
          onClick={handleRefresh}
          loading={isRefreshing}
        >
          Refresh
        </Button>
      </div>

      <Paragraph>
        Manage and monitor all registered agents. View agent status, health information, and
        performance metrics.
      </Paragraph>

      {/* Search and Filter */}
      <Card style={{ marginBottom: 24 }}>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12} lg={8}>
            <Input
              placeholder="Search by hostname, IP, or ID"
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={e => setSearchText(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} md={12} lg={6}>
            <Select
              style={{ width: '100%' }}
              placeholder="Filter by status"
              value={statusFilter}
              onChange={setStatusFilter}
            >
              <Select.Option value="all">All Status</Select.Option>
              <Select.Option value="online">Online</Select.Option>
              <Select.Option value="offline">Offline</Select.Option>
              <Select.Option value="error">Error</Select.Option>
            </Select>
          </Col>
          <Col xs={24} md={24} lg={10}>
            <Space>
              <span style={{ color: '#8c8c8c' }}>
                Showing {filteredAgents.length} of {agentsData?.total || 0} agents
              </span>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* Error Alert */}
      {error && (
        <Alert
          message="Failed to load agents"
          description={error instanceof Error ? error.message : 'Unknown error occurred'}
          type="error"
          showIcon
          style={{ marginBottom: 24 }}
          action={
            <Button size="small" onClick={handleRefresh}>
              Retry
            </Button>
          }
        />
      )}

      {/* Agents Table */}
      {!error && filteredAgents.length === 0 && !isLoading && (
        <Card>
          <div style={{ textAlign: 'center', padding: '40px 0', color: '#999' }}>
            {searchText || statusFilter !== 'all' ? (
              <>
                <p style={{ fontSize: 16 }}>No agents match your search criteria</p>
                <Button
                  onClick={() => {
                    setSearchText('')
                    setStatusFilter('all')
                  }}
                >
                  Clear Filters
                </Button>
              </>
            ) : (
              <p style={{ fontSize: 16 }}>No agents registered yet</p>
            )}
          </div>
        </Card>
      )}

      {!error && (filteredAgents.length > 0 || isLoading) && (
        <AgentTable agents={filteredAgents} loading={isLoading} />
      )}
    </div>
  )
}
