import { Row, Col, Input, Select, DatePicker, Button, Card } from 'antd'
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons'
import type { FlowQuery } from '../../types/flow'
import type { Dayjs } from 'dayjs'

const { RangePicker } = DatePicker

interface FlowFiltersProps {
  filters: FlowQuery
  onFiltersChange: (filters: FlowQuery) => void
  onReset: () => void
}

export default function FlowFilters({ filters, onFiltersChange, onReset }: FlowFiltersProps) {
  const handleSourceIpChange = (value: string) => {
    onFiltersChange({ ...filters, sourceIp: value || undefined })
  }

  const handleDestIpChange = (value: string) => {
    onFiltersChange({ ...filters, destIp: value || undefined })
  }

  const handleProtocolChange = (value: string) => {
    onFiltersChange({ ...filters, protocol: value === 'ALL' ? undefined : value })
  }

  const handleStateChange = (value: string) => {
    onFiltersChange({ ...filters, state: value === 'ALL' ? undefined : value })
  }

  const handleActionChange = (value: string) => {
    onFiltersChange({ ...filters, policyAction: value === 'ALL' ? undefined : value })
  }

  const handleProcessNameChange = (value: string) => {
    onFiltersChange({ ...filters, processName: value || undefined })
  }

  const handleContainerIdChange = (value: string) => {
    onFiltersChange({ ...filters, containerId: value || undefined })
  }

  const handleTimeRangeChange = (dates: null | [Dayjs | null, Dayjs | null]) => {
    if (dates && dates[0] && dates[1]) {
      onFiltersChange({
        ...filters,
        startTime: dates[0].toISOString(),
        endTime: dates[1].toISOString(),
      })
    } else {
      onFiltersChange({
        ...filters,
        startTime: undefined,
        endTime: undefined,
      })
    }
  }

  return (
    <Card style={{ marginBottom: 16 }}>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} md={8} lg={6}>
          <Input
            placeholder="Source IP"
            prefix={<SearchOutlined />}
            value={filters.sourceIp}
            onChange={e => handleSourceIpChange(e.target.value)}
            allowClear
          />
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <Input
            placeholder="Destination IP"
            prefix={<SearchOutlined />}
            value={filters.destIp}
            onChange={e => handleDestIpChange(e.target.value)}
            allowClear
          />
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <Select
            style={{ width: '100%' }}
            placeholder="Protocol"
            value={filters.protocol || 'ALL'}
            onChange={handleProtocolChange}
          >
            <Select.Option value="ALL">All Protocols</Select.Option>
            <Select.Option value="TCP">TCP</Select.Option>
            <Select.Option value="UDP">UDP</Select.Option>
            <Select.Option value="ICMP">ICMP</Select.Option>
          </Select>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <Select
            style={{ width: '100%' }}
            placeholder="State"
            value={filters.state || 'ALL'}
            onChange={handleStateChange}
          >
            <Select.Option value="ALL">All States</Select.Option>
            <Select.Option value="ACTIVE">Active</Select.Option>
            <Select.Option value="CLOSED">Closed</Select.Option>
            <Select.Option value="TIMEOUT">Timeout</Select.Option>
          </Select>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <Select
            style={{ width: '100%' }}
            placeholder="Action"
            value={filters.policyAction || 'ALL'}
            onChange={handleActionChange}
          >
            <Select.Option value="ALL">All Actions</Select.Option>
            <Select.Option value="ALLOW">Allow</Select.Option>
            <Select.Option value="DENY">Deny</Select.Option>
            <Select.Option value="LOG">Log</Select.Option>
          </Select>
        </Col>
        <Col xs={24} sm={12} md={10} lg={8}>
          <RangePicker
            style={{ width: '100%' }}
            showTime
            format="YYYY-MM-DD HH:mm"
            onChange={handleTimeRangeChange}
          />
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <Input
            placeholder="Process Name"
            prefix={<SearchOutlined />}
            value={filters.processName}
            onChange={e => handleProcessNameChange(e.target.value)}
            allowClear
          />
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <Input
            placeholder="Container ID"
            prefix={<SearchOutlined />}
            value={filters.containerId}
            onChange={e => handleContainerIdChange(e.target.value)}
            allowClear
          />
        </Col>
        <Col xs={24} sm={12} md={6} lg={4}>
          <Button icon={<ReloadOutlined />} onClick={onReset} block>
            Reset Filters
          </Button>
        </Col>
      </Row>
    </Card>
  )
}
