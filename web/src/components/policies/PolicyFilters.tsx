import { Card, Row, Col, Input, Select, Button } from 'antd'
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons'

interface PolicyFiltersProps {
  filters: {
    srcIp?: string
    dstIp?: string
    protocol?: string
    action?: string
    enabled?: boolean
  }
  onFiltersChange: (filters: PolicyFiltersProps['filters']) => void
  onReset: () => void
}

export default function PolicyFilters({ filters, onFiltersChange, onReset }: PolicyFiltersProps) {
  const handleChange = (key: keyof PolicyFiltersProps['filters'], value: string | boolean | undefined) => {
    onFiltersChange({ ...filters, [key]: value })
  }

  return (
    <Card style={{ marginBottom: 16 }}>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} md={8} lg={6}>
          <Input
            placeholder="Source IP"
            prefix={<SearchOutlined />}
            value={filters.srcIp}
            onChange={e => handleChange('srcIp', e.target.value)}
            allowClear
          />
        </Col>

        <Col xs={24} sm={12} md={8} lg={6}>
          <Input
            placeholder="Destination IP"
            prefix={<SearchOutlined />}
            value={filters.dstIp}
            onChange={e => handleChange('dstIp', e.target.value)}
            allowClear
          />
        </Col>

        <Col xs={24} sm={12} md={8} lg={4}>
          <Select
            placeholder="Protocol"
            value={filters.protocol}
            onChange={value => handleChange('protocol', value)}
            allowClear
            style={{ width: '100%' }}
          >
            <Select.Option value="tcp">TCP</Select.Option>
            <Select.Option value="udp">UDP</Select.Option>
            <Select.Option value="icmp">ICMP</Select.Option>
            <Select.Option value="any">Any</Select.Option>
          </Select>
        </Col>

        <Col xs={24} sm={12} md={8} lg={4}>
          <Select
            placeholder="Action"
            value={filters.action}
            onChange={value => handleChange('action', value)}
            allowClear
            style={{ width: '100%' }}
          >
            <Select.Option value="allow">Allow</Select.Option>
            <Select.Option value="deny">Deny</Select.Option>
            <Select.Option value="log">Log</Select.Option>
          </Select>
        </Col>

        <Col xs={24} sm={12} md={8} lg={4}>
          <Select
            placeholder="Status"
            value={filters.enabled}
            onChange={value => handleChange('enabled', value)}
            allowClear
            style={{ width: '100%' }}
          >
            <Select.Option value={true}>Enabled</Select.Option>
            <Select.Option value={false}>Disabled</Select.Option>
          </Select>
        </Col>

        <Col xs={24} sm={12} md={8} lg={4}>
          <Button icon={<ReloadOutlined />} onClick={onReset} block>
            Reset Filters
          </Button>
        </Col>
      </Row>
    </Card>
  )
}
