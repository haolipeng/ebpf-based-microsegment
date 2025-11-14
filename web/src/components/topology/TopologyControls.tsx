import { Space, Segmented, DatePicker, Switch, InputNumber, Button, Select, Card } from 'antd'
import { ReloadOutlined, DownloadOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import type { TopologyViewMode, TopologyFilters } from '../../types/topology'

const { RangePicker } = DatePicker

interface TopologyControlsProps {
  /** 视图模式 */
  viewMode: TopologyViewMode
  /** 视图模式变更回调 */
  onViewModeChange: (mode: TopologyViewMode) => void
  /** 筛选条件 */
  filters: TopologyFilters
  /** 筛选条件变更回调 */
  onFiltersChange: (filters: Partial<TopologyFilters>) => void
  /** 实时更新状态 */
  realtimeEnabled: boolean
  /** 实时更新切换回调 */
  onRealtimeToggle: (enabled: boolean) => void
  /** 刷新回调 */
  onRefresh?: () => void
}

/**
 * 拓扑图控制栏组件
 * 
 * 提供视图切换、筛选、实时更新等控制功能
 */
export default function TopologyControls({
  viewMode,
  onViewModeChange,
  filters,
  onFiltersChange,
  realtimeEnabled,
  onRealtimeToggle,
  onRefresh,
}: TopologyControlsProps) {
  // 处理时间范围变更
  const handleTimeRangeChange = (dates: any) => {
    if (dates && dates[0] && dates[1]) {
      onFiltersChange({
        startTime: dates[0].toISOString(),
        endTime: dates[1].toISOString(),
      })
    }
  }

  // 重置筛选条件
  const handleReset = () => {
    onFiltersChange({
      protocol: undefined,
      state: undefined,
      policyAction: undefined,
      maxNodes: 100,
      startTime: dayjs().subtract(7, 'days').toISOString(),
      endTime: dayjs().toISOString(),
    })
  }

  return (
    <Card size="small" style={{ marginBottom: 16 }}>
      <Space size="middle" wrap>
        {/* 视图模式切换 */}
        <Space direction="vertical" size={0}>
          <span style={{ fontSize: 12, color: '#666' }}>View Mode</span>
          <Segmented
            value={viewMode}
            onChange={(value) => onViewModeChange(value as TopologyViewMode)}
            options={[
              { label: 'IP View', value: 'IP' },
              { label: 'Service View', value: 'LABEL' },
            ]}
          />
        </Space>

        {/* 时间范围选择 */}
        <Space direction="vertical" size={0}>
          <span style={{ fontSize: 12, color: '#666' }}>Time Range</span>
          <RangePicker
            showTime
            value={
              filters.startTime && filters.endTime
                ? [dayjs(filters.startTime), dayjs(filters.endTime)]
                : [dayjs().subtract(7, 'days'), dayjs()]
            }
            onChange={handleTimeRangeChange}
            format="YYYY-MM-DD HH:mm"
          />
        </Space>

        {/* 协议筛选 */}
        <Space direction="vertical" size={0}>
          <span style={{ fontSize: 12, color: '#666' }}>Protocol</span>
          <Select
            value={filters.protocol}
            onChange={(value) => onFiltersChange({ protocol: value })}
            style={{ width: 120 }}
            allowClear
            placeholder="All Protocols"
          >
            <Select.Option value="TCP">TCP</Select.Option>
            <Select.Option value="UDP">UDP</Select.Option>
            <Select.Option value="ICMP">ICMP</Select.Option>
          </Select>
        </Space>

        {/* 状态筛选 */}
        <Space direction="vertical" size={0}>
          <span style={{ fontSize: 12, color: '#666' }}>State</span>
          <Select
            value={filters.state}
            onChange={(value) => onFiltersChange({ state: value })}
            style={{ width: 120 }}
            allowClear
            placeholder="All States"
          >
            <Select.Option value="ACTIVE">Active</Select.Option>
            <Select.Option value="CLOSED">Closed</Select.Option>
            <Select.Option value="TIMEOUT">Timeout</Select.Option>
          </Select>
        </Space>

        {/* 动作筛选 */}
        <Space direction="vertical" size={0}>
          <span style={{ fontSize: 12, color: '#666' }}>Action</span>
          <Select
            value={filters.policyAction}
            onChange={(value) => onFiltersChange({ policyAction: value })}
            style={{ width: 120 }}
            allowClear
            placeholder="All Actions"
          >
            <Select.Option value="ALLOW">Allow</Select.Option>
            <Select.Option value="DENY">Deny</Select.Option>
            <Select.Option value="LOG">Log</Select.Option>
          </Select>
        </Space>

        {/* 最大节点数 */}
        <Space direction="vertical" size={0}>
          <span style={{ fontSize: 12, color: '#666' }}>Max Nodes</span>
          <InputNumber
            value={filters.maxNodes || 100}
            onChange={(value) => onFiltersChange({ maxNodes: value || 100 })}
            min={10}
            max={200}
            step={10}
            style={{ width: 100 }}
          />
        </Space>

        {/* 实时更新开关 */}
        <Space direction="vertical" size={0}>
          <span style={{ fontSize: 12, color: '#666' }}>Real-time</span>
          <Switch
            checked={realtimeEnabled}
            onChange={onRealtimeToggle}
            checkedChildren="On"
            unCheckedChildren="Off"
          />
        </Space>

        {/* 刷新按钮 */}
        <Space direction="vertical" size={0}>
          <span style={{ fontSize: 12, color: 'transparent' }}>-</span>
          <Button icon={<ReloadOutlined />} onClick={onRefresh}>
            Refresh
          </Button>
        </Space>

        {/* 重置按钮 */}
        <Space direction="vertical" size={0}>
          <span style={{ fontSize: 12, color: 'transparent' }}>-</span>
          <Button onClick={handleReset}>Reset</Button>
        </Space>

        {/* 导出按钮（占位，可选） */}
        <Space direction="vertical" size={0}>
          <span style={{ fontSize: 12, color: 'transparent' }}>-</span>
          <Button icon={<DownloadOutlined />} disabled>
            Export
          </Button>
        </Space>
      </Space>
    </Card>
  )
}

