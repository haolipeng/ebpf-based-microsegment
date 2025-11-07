import { Modal, Form, Input, InputNumber, Select, message, Alert } from 'antd'
import { useEffect, useState } from 'react'
import type { Policy } from '../../types/policy'
import { validatePolicy } from '../../utils/policyValidation'

interface PolicyFormProps {
  open: boolean
  mode: 'create' | 'edit'
  policy?: Policy
  existingPolicies?: Policy[]
  onSubmit: (values: Omit<Policy, 'ruleId'> | Partial<Policy>) => Promise<void>
  onCancel: () => void
}

export default function PolicyForm({
  open,
  mode,
  policy,
  existingPolicies = [],
  onSubmit,
  onCancel,
}: PolicyFormProps) {
  const [form] = Form.useForm()
  const [validationResult, setValidationResult] = useState<{
    conflicts: Policy[]
    warnings: string[]
  }>({ conflicts: [], warnings: [] })

  useEffect(() => {
    if (open) {
      if (mode === 'edit' && policy) {
        form.setFieldsValue(policy)
      } else {
        form.resetFields()
      }
      setValidationResult({ conflicts: [], warnings: [] })
    }
  }, [open, mode, policy, form])

  const handleValuesChange = () => {
    // 获取当前表单值
    const values = form.getFieldsValue()

    // 检查所有必填字段是否都有值
    if (
      values.srcIp &&
      values.dstIp &&
      values.srcPort !== undefined &&
      values.dstPort !== undefined &&
      values.protocol &&
      values.action &&
      values.priority !== undefined
    ) {
      // 执行验证
      const result = validatePolicy(
        {
          ...values,
          ruleId: policy?.ruleId,
        } as Policy,
        existingPolicies
      )

      setValidationResult({
        conflicts: result.conflicts,
        warnings: result.warnings,
      })
    }
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      await onSubmit(values)
      form.resetFields()
      message.success(`Policy ${mode === 'create' ? 'created' : 'updated'} successfully`)
    } catch (error) {
      console.error('Form validation failed:', error)
    }
  }

  const validateIP = (_: unknown, value: string) => {
    if (!value) {
      return Promise.reject(new Error('IP address is required'))
    }

    // Support both single IP and CIDR notation
    const ipRegex = /^(\d{1,3}\.){3}\d{1,3}(\/\d{1,2})?$/
    if (!ipRegex.test(value)) {
      return Promise.reject(new Error('Invalid IP address format (e.g., 10.0.1.10 or 10.0.1.0/24)'))
    }

    // Validate IP octets
    const parts = value.split('/')[0].split('.')
    for (const part of parts) {
      const num = parseInt(part, 10)
      if (num < 0 || num > 255) {
        return Promise.reject(new Error('IP octet must be between 0 and 255'))
      }
    }

    // Validate CIDR prefix if present
    if (value.includes('/')) {
      const prefix = parseInt(value.split('/')[1], 10)
      if (prefix < 0 || prefix > 32) {
        return Promise.reject(new Error('CIDR prefix must be between 0 and 32'))
      }
    }

    return Promise.resolve()
  }

  const validatePort = (_: unknown, value: number) => {
    if (value === undefined || value === null) {
      return Promise.reject(new Error('Port is required'))
    }
    if (value < 0 || value > 65535) {
      return Promise.reject(new Error('Port must be between 0 and 65535'))
    }
    return Promise.resolve()
  }

  return (
    <Modal
      title={mode === 'create' ? 'Create Policy' : 'Edit Policy'}
      open={open}
      onOk={handleSubmit}
      onCancel={onCancel}
      okText={mode === 'create' ? 'Create' : 'Update'}
      cancelText="Cancel"
      width={600}
      destroyOnClose
    >
      {/* Validation Alerts */}
      {validationResult.conflicts.length > 0 && (
        <Alert
          message="Policy Conflicts Detected"
          description={
            <div>
              <p>This policy conflicts with the following existing policies:</p>
              <ul>
                {validationResult.conflicts.map(conflict => (
                  <li key={conflict.ruleId}>
                    Rule {conflict.ruleId}: {conflict.srcIp} → {conflict.dstIp} ({conflict.action})
                  </li>
                ))}
              </ul>
              <p>
                <strong>Note:</strong> Conflicts are resolved based on priority. Lower priority values are
                evaluated first.
              </p>
            </div>
          }
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
        />
      )}

      {validationResult.warnings.length > 0 && (
        <Alert
          message="Warnings"
          description={
            <ul style={{ margin: 0, paddingLeft: 20 }}>
              {validationResult.warnings.map((warning, index) => (
                <li key={index}>{warning}</li>
              ))}
            </ul>
          }
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
        />
      )}

      <Form form={form} layout="vertical" preserve={false} onValuesChange={handleValuesChange}>
        <Form.Item
          label="Source IP"
          name="srcIp"
          rules={[{ validator: validateIP }]}
          help="Format: 10.0.1.10 or 10.0.1.0/24"
        >
          <Input placeholder="e.g., 10.0.1.0/24" />
        </Form.Item>

        <Form.Item
          label="Destination IP"
          name="dstIp"
          rules={[{ validator: validateIP }]}
          help="Format: 10.0.1.10 or 10.0.1.0/24"
        >
          <Input placeholder="e.g., 10.0.2.0/24" />
        </Form.Item>

        <Form.Item
          label="Source Port"
          name="srcPort"
          rules={[{ validator: validatePort }]}
          help="0 for any port"
          initialValue={0}
        >
          <InputNumber min={0} max={65535} style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item
          label="Destination Port"
          name="dstPort"
          rules={[{ validator: validatePort }]}
          help="0 for any port"
          initialValue={0}
        >
          <InputNumber min={0} max={65535} style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item
          label="Protocol"
          name="protocol"
          rules={[{ required: true, message: 'Protocol is required' }]}
        >
          <Select placeholder="Select protocol">
            <Select.Option value="tcp">TCP</Select.Option>
            <Select.Option value="udp">UDP</Select.Option>
            <Select.Option value="icmp">ICMP</Select.Option>
            <Select.Option value="any">Any</Select.Option>
          </Select>
        </Form.Item>

        <Form.Item
          label="Action"
          name="action"
          rules={[{ required: true, message: 'Action is required' }]}
        >
          <Select placeholder="Select action">
            <Select.Option value="allow">Allow</Select.Option>
            <Select.Option value="deny">Deny</Select.Option>
            <Select.Option value="log">Log</Select.Option>
          </Select>
        </Form.Item>

        <Form.Item
          label="Priority"
          name="priority"
          rules={[
            { required: true, message: 'Priority is required' },
            { type: 'number', min: 0, max: 1000, message: 'Priority must be between 0 and 1000' },
          ]}
          initialValue={100}
        >
          <InputNumber min={0} max={1000} style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item label="Description" name="description">
          <Input.TextArea rows={3} placeholder="Optional description" />
        </Form.Item>
      </Form>
    </Modal>
  )
}
