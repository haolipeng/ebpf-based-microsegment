import { Typography, Button, Alert, Row, Col, Switch, Badge } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import { useState, useCallback, useMemo } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useFlows, useFlowSummary } from '../../hooks/useFlows'
import { useFlowStream } from '../../hooks/useFlowStream'
import FlowFilters from '../../components/flows/FlowFilters'
import FlowTable from '../../components/flows/FlowTable'
import FlowSummaryCards from '../../components/flows/FlowSummaryCards'
import ProtocolStats from '../../components/flows/ProtocolStats'
import type { FlowQuery, Flow } from '../../types/flow'

const { Title, Paragraph } = Typography

export default function Flows() {
  const queryClient = useQueryClient()
  const [filters, setFilters] = useState<FlowQuery>({
    limit: 50,
    offset: 0,
    sortBy: 'startTime',
    sortOrder: 'desc',
  })
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [realtimeEnabled, setRealtimeEnabled] = useState(false)
  const [realtimeFlows, setRealtimeFlows] = useState<Flow[]>([])
  const [highlightedFlowIds, setHighlightedFlowIds] = useState<Set<string>>(new Set())

  const { data: flows, isLoading, error } = useFlows(filters)
  const {
    data: summary,
    isLoading: summaryLoading,
    error: summaryError,
  } = useFlowSummary(filters.startTime, filters.endTime)

  const handleFiltersChange = (newFilters: FlowQuery) => {
    setFilters({ ...filters, ...newFilters, offset: 0 })
  }

  const handleResetFilters = () => {
    setFilters({
      limit: 50,
      offset: 0,
      sortBy: 'startTime',
      sortOrder: 'desc',
    })
  }

  const handleRefresh = async () => {
    setIsRefreshing(true)
    await queryClient.invalidateQueries({ queryKey: ['flows'] })
    await queryClient.invalidateQueries({ queryKey: ['flowSummary'] })
    setIsRefreshing(false)
  }

  const handleNewFlow = useCallback((flow: Flow) => {
    setRealtimeFlows(prev => {
      // Add new flow to the top, limit to 100 flows
      const updated = [flow, ...prev].slice(0, 100)
      return updated
    })

    // Highlight the new flow
    setHighlightedFlowIds(prev => new Set(prev).add(flow.id))

    // Remove highlight after 3 seconds
    setTimeout(() => {
      setHighlightedFlowIds(prev => {
        const next = new Set(prev)
        next.delete(flow.id)
        return next
      })
    }, 3000)

    // Invalidate queries to update summary
    queryClient.invalidateQueries({ queryKey: ['flowSummary'] })
  }, [queryClient])

  const { isConnected, error: wsError } = useFlowStream({
    enabled: realtimeEnabled,
    onFlow: handleNewFlow,
  })

  // Merge realtime flows with fetched flows when realtime is enabled
  const displayFlows = useMemo(() => {
    if (!realtimeEnabled || realtimeFlows.length === 0) {
      return flows || []
    }
    // Show realtime flows at the top
    const flowIds = new Set((flows || []).map(f => f.id))
    const uniqueRealtimeFlows = realtimeFlows.filter(f => !flowIds.has(f.id))
    return [...uniqueRealtimeFlows, ...(flows || [])]
  }, [realtimeEnabled, realtimeFlows, flows])

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
          Network Flows
        </Title>
        <Space>
          <div>
            <Switch
              checked={realtimeEnabled}
              onChange={setRealtimeEnabled}
              checkedChildren="Real-time ON"
              unCheckedChildren="Real-time OFF"
            />
            {realtimeEnabled && (
              <Badge
                status={isConnected ? 'success' : wsError ? 'error' : 'processing'}
                text={
                  isConnected ? 'Connected' : wsError ? 'Error' : 'Connecting...'
                }
                style={{ marginLeft: 8 }}
              />
            )}
          </div>
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
        Monitor and analyze network flow data. Filter by IP, protocol, state, and action to find
        specific flows.
      </Paragraph>

      {/* Summary Cards */}
      {summary && !summaryError && (
        <FlowSummaryCards summary={summary} loading={summaryLoading} />
      )}

      {/* Protocol Stats */}
      {summary && summary.topProtocols && summary.topProtocols.length > 0 && (
        <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
          <Col xs={24} lg={12}>
            <ProtocolStats protocols={summary.topProtocols} loading={summaryLoading} />
          </Col>
        </Row>
      )}

      {/* Filters */}
      <FlowFilters filters={filters} onFiltersChange={handleFiltersChange} onReset={handleResetFilters} />

      {/* Error Alert */}
      {error && (
        <Alert
          message="Failed to load flows"
          description={error instanceof Error ? error.message : 'Unknown error occurred'}
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          action={
            <Button size="small" onClick={handleRefresh}>
              Retry
            </Button>
          }
        />
      )}

      {/* Flows Table */}
      {!error && (!flows || flows.length === 0) && !isLoading && (
        <Alert
          message="No flows found"
          description={
            filters.sourceIp || filters.destIp || filters.protocol || filters.state
              ? 'No flows match your filter criteria. Try adjusting or resetting the filters.'
              : 'No flow data has been recorded yet. Flows will appear here once network traffic is detected.'
          }
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />
      )}

      {!error && displayFlows && displayFlows.length > 0 && (
        <>
          <div style={{ marginBottom: 8, color: '#8c8c8c' }}>
            Showing {displayFlows.length} flow{displayFlows.length !== 1 ? 's' : ''}
            {realtimeEnabled && realtimeFlows.length > 0 && (
              <span style={{ marginLeft: 8 }}>
                ({realtimeFlows.length} real-time)
              </span>
            )}
          </div>
          <FlowTable
            flows={displayFlows}
            loading={isLoading}
            highlightedIds={highlightedFlowIds}
          />
        </>
      )}

      {!error && isLoading && !displayFlows && <FlowTable flows={[]} loading={true} />}
    </div>
  )
}
