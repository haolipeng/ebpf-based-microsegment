import { Space, Segmented, DatePicker, Switch, InputNumber, Button, Select, Card, Tooltip, Input } from 'antd'
import {
  ReloadOutlined,
  DownloadOutlined,
  ClusterOutlined,
  AppstoreOutlined,
  ApartmentOutlined,
  ContainerOutlined,
  ThunderboltOutlined,
  GlobalOutlined,
} from '@ant-design/icons'
import dayjs from 'dayjs'
import type { TopologyViewMode, TopologyFilters } from '../../types/topology'

const { RangePicker } = DatePicker
const { Search } = Input

/**
 * View mode options with icons and descriptions
 */
const VIEW_MODE_OPTIONS = [
  {
    label: (
      <Tooltip title="Group traffic by Kubernetes namespace">
        <Space size={4}>
          <ClusterOutlined />
          <span>Namespace</span>
        </Space>
      </Tooltip>
    ),
    value: 'NAMESPACE',
  },
  {
    label: (
      <Tooltip title="Group traffic by service (app label)">
        <Space size={4}>
          <AppstoreOutlined />
          <span>Service</span>
        </Space>
      </Tooltip>
    ),
    value: 'SERVICE',
  },
  {
    label: (
      <Tooltip title="Show individual pods">
        <Space size={4}>
          <ApartmentOutlined />
          <span>Pod</span>
        </Space>
      </Tooltip>
    ),
    value: 'POD',
  },
  {
    label: (
      <Tooltip title="Show containers within pods">
        <Space size={4}>
          <ContainerOutlined />
          <span>Container</span>
        </Space>
      </Tooltip>
    ),
    value: 'CONTAINER',
  },
  {
    label: (
      <Tooltip title="Show processes within containers">
        <Space size={4}>
          <ThunderboltOutlined />
          <span>Process</span>
        </Space>
      </Tooltip>
    ),
    value: 'PROCESS',
  },
  {
    label: (
      <Tooltip title="Raw IP address view">
        <Space size={4}>
          <GlobalOutlined />
          <span>IP</span>
        </Space>
      </Tooltip>
    ),
    value: 'IP',
  },
]

interface TopologyControlsProps {
  /** Current view mode */
  viewMode: TopologyViewMode
  /** View mode change callback */
  onViewModeChange: (mode: TopologyViewMode) => void
  /** Filter settings */
  filters: TopologyFilters
  /** Filter change callback */
  onFiltersChange: (filters: Partial<TopologyFilters>) => void
  /** Real-time update state */
  realtimeEnabled: boolean
  /** Real-time update toggle callback */
  onRealtimeToggle: (enabled: boolean) => void
  /** Refresh callback */
  onRefresh?: () => void
  /** Available namespaces for filtering */
  namespaces?: string[]
}

/**
 * Topology controls component with K8s/Docker view modes
 */
export default function TopologyControls({
  viewMode,
  onViewModeChange,
  filters,
  onFiltersChange,
  realtimeEnabled,
  onRealtimeToggle,
  onRefresh,
  namespaces = [],
}: TopologyControlsProps) {
  // Handle time range change
  const handleTimeRangeChange = (dates: unknown) => {
    const d = dates as [dayjs.Dayjs, dayjs.Dayjs] | null
    if (d && d[0] && d[1]) {
      onFiltersChange({
        startTime: d[0].toISOString(),
        endTime: d[1].toISOString(),
      })
    }
  }

  // Reset filters
  const handleReset = () => {
    onFiltersChange({
      protocol: undefined,
      state: undefined,
      policyAction: undefined,
      namespace: undefined,
      service: undefined,
      podPattern: undefined,
      showExternal: true,
      onlySuspicious: false,
      maxNodes: 100,
      minFlowCount: undefined,
      startTime: dayjs().subtract(1, 'hour').toISOString(),
      endTime: dayjs().toISOString(),
    })
  }

  // Quick time range presets
  const handleQuickTimeRange = (minutes: number) => {
    onFiltersChange({
      startTime: dayjs().subtract(minutes, 'minute').toISOString(),
      endTime: dayjs().toISOString(),
      timeRangeMinutes: minutes,
    })
  }

  return (
    <Card size="small" style={{ marginBottom: 16 }}>
      <Space direction="vertical" size="middle" style={{ width: '100%' }}>
        {/* Row 1: View Mode Selection */}
        <Space size="middle" wrap>
          <Space direction="vertical" size={0}>
            <span style={{ fontSize: 12, color: '#666', fontWeight: 500 }}>View Mode</span>
            <Segmented
              value={viewMode}
              onChange={(value) => onViewModeChange(value as TopologyViewMode)}
              options={VIEW_MODE_OPTIONS}
              size="small"
            />
          </Space>
        </Space>

        {/* Row 2: Filters */}
        <Space size="middle" wrap>
          {/* Time Range */}
          <Space direction="vertical" size={0}>
            <span style={{ fontSize: 12, color: '#666' }}>Time Range</span>
            <Space.Compact>
              <RangePicker
                showTime
                value={
                  filters.startTime && filters.endTime
                    ? [dayjs(filters.startTime), dayjs(filters.endTime)]
                    : [dayjs().subtract(1, 'hour'), dayjs()]
                }
                onChange={handleTimeRangeChange}
                format="MM-DD HH:mm"
                size="small"
                style={{ width: 260 }}
              />
              <Button.Group size="small">
                <Tooltip title="Last 5 minutes">
                  <Button onClick={() => handleQuickTimeRange(5)}>5m</Button>
                </Tooltip>
                <Tooltip title="Last 15 minutes">
                  <Button onClick={() => handleQuickTimeRange(15)}>15m</Button>
                </Tooltip>
                <Tooltip title="Last 1 hour">
                  <Button onClick={() => handleQuickTimeRange(60)}>1h</Button>
                </Tooltip>
                <Tooltip title="Last 24 hours">
                  <Button onClick={() => handleQuickTimeRange(1440)}>24h</Button>
                </Tooltip>
              </Button.Group>
            </Space.Compact>
          </Space>

          {/* Namespace Filter (for K8s views) */}
          {viewMode !== 'IP' && (
            <Space direction="vertical" size={0}>
              <span style={{ fontSize: 12, color: '#666' }}>Namespace</span>
              <Select
                value={filters.namespace}
                onChange={(value) => onFiltersChange({ namespace: value })}
                style={{ width: 150 }}
                allowClear
                placeholder="All Namespaces"
                size="small"
              >
                {namespaces.map(ns => (
                  <Select.Option key={ns} value={ns}>{ns}</Select.Option>
                ))}
              </Select>
            </Space>
          )}

          {/* Service Filter (for Service/Pod/Container views) */}
          {(viewMode === 'SERVICE' || viewMode === 'POD' || viewMode === 'CONTAINER') && (
            <Space direction="vertical" size={0}>
              <span style={{ fontSize: 12, color: '#666' }}>Service</span>
              <Search
                placeholder="Filter by service"
                value={filters.service}
                onChange={(e) => onFiltersChange({ service: e.target.value || undefined })}
                style={{ width: 150 }}
                size="small"
                allowClear
              />
            </Space>
          )}

          {/* Protocol Filter */}
          <Space direction="vertical" size={0}>
            <span style={{ fontSize: 12, color: '#666' }}>Protocol</span>
            <Select
              value={filters.protocol}
              onChange={(value) => onFiltersChange({ protocol: value })}
              style={{ width: 100 }}
              allowClear
              placeholder="All"
              size="small"
            >
              <Select.Option value="TCP">TCP</Select.Option>
              <Select.Option value="UDP">UDP</Select.Option>
              <Select.Option value="ICMP">ICMP</Select.Option>
            </Select>
          </Space>

          {/* Policy Action Filter */}
          <Space direction="vertical" size={0}>
            <span style={{ fontSize: 12, color: '#666' }}>Action</span>
            <Select
              value={filters.policyAction}
              onChange={(value) => onFiltersChange({ policyAction: value })}
              style={{ width: 100 }}
              allowClear
              placeholder="All"
              size="small"
            >
              <Select.Option value="ALLOW">Allow</Select.Option>
              <Select.Option value="DENY">Deny</Select.Option>
              <Select.Option value="LOG">Log</Select.Option>
            </Select>
          </Space>

          {/* Max Nodes */}
          <Space direction="vertical" size={0}>
            <span style={{ fontSize: 12, color: '#666' }}>Max Nodes</span>
            <InputNumber
              value={filters.maxNodes || 100}
              onChange={(value) => onFiltersChange({ maxNodes: value || 100 })}
              min={10}
              max={500}
              step={10}
              style={{ width: 80 }}
              size="small"
            />
          </Space>

          {/* Show External */}
          <Space direction="vertical" size={0}>
            <span style={{ fontSize: 12, color: '#666' }}>External</span>
            <Switch
              checked={filters.showExternal !== false}
              onChange={(checked) => onFiltersChange({ showExternal: checked })}
              checkedChildren="Show"
              unCheckedChildren="Hide"
              size="small"
            />
          </Space>

          {/* Only Suspicious */}
          <Space direction="vertical" size={0}>
            <span style={{ fontSize: 12, color: '#666' }}>Suspicious</span>
            <Switch
              checked={filters.onlySuspicious === true}
              onChange={(checked) => onFiltersChange({ onlySuspicious: checked })}
              checkedChildren="Only"
              unCheckedChildren="All"
              size="small"
            />
          </Space>

          {/* Real-time Toggle */}
          <Space direction="vertical" size={0}>
            <span style={{ fontSize: 12, color: '#666' }}>Real-time</span>
            <Switch
              checked={realtimeEnabled}
              onChange={onRealtimeToggle}
              checkedChildren="On"
              unCheckedChildren="Off"
              size="small"
            />
          </Space>

          {/* Action Buttons */}
          <Space direction="vertical" size={0}>
            <span style={{ fontSize: 12, color: 'transparent' }}>-</span>
            <Space.Compact>
              <Tooltip title="Refresh data">
                <Button icon={<ReloadOutlined />} onClick={onRefresh} size="small">
                  Refresh
                </Button>
              </Tooltip>
              <Button onClick={handleReset} size="small">
                Reset
              </Button>
              <Tooltip title="Export topology (coming soon)">
                <Button icon={<DownloadOutlined />} disabled size="small">
                  Export
                </Button>
              </Tooltip>
            </Space.Compact>
          </Space>
        </Space>
      </Space>
    </Card>
  )
}
