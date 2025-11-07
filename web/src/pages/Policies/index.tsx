import { Typography, Button, Alert, message, Popconfirm, Space } from 'antd'
import { PlusOutlined, ReloadOutlined, DeleteOutlined, CheckOutlined, CloseOutlined } from '@ant-design/icons'
import { useState, useMemo, useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import {
  usePolicies,
  useCreatePolicy,
  useUpdatePolicy,
  useDeletePolicy,
  useBatchDeletePolicies,
  useBatchUpdatePolicies,
  useAllPolicyStats,
} from '../../hooks/usePolicies'
import PolicyTable from '../../components/policies/PolicyTable'
import PolicyForm from '../../components/policies/PolicyForm'
import PolicyStatsCards from '../../components/policies/PolicyStatsCards'
import PolicyFilters from '../../components/policies/PolicyFilters'
import PolicyStatsModal from '../../components/policies/PolicyStatsModal'
import type { Policy } from '../../types/policy'

const { Title, Paragraph } = Typography

type Filters = {
  srcIp?: string
  dstIp?: string
  protocol?: string
  action?: string
  enabled?: boolean
}

export default function Policies() {
  const queryClient = useQueryClient()
  const [isFormOpen, setIsFormOpen] = useState(false)
  const [formMode, setFormMode] = useState<'create' | 'edit'>('create')
  const [selectedPolicy, setSelectedPolicy] = useState<Policy | undefined>()
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [filters, setFilters] = useState<Filters>({})
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([])
  const [statsModalOpen, setStatsModalOpen] = useState(false)
  const [selectedPolicyForStats, setSelectedPolicyForStats] = useState<Policy | null>(null)

  const { data: policiesData, isLoading, error } = usePolicies()
  const allPolicies = policiesData?.policies || []
  const ruleIds = allPolicies.map(p => p.ruleId)
  const { data: policyStats } = useAllPolicyStats(ruleIds)
  const createMutation = useCreatePolicy()
  const updateMutation = useUpdatePolicy()
  const deleteMutation = useDeletePolicy()
  const batchDeleteMutation = useBatchDeletePolicies()
  const batchUpdateMutation = useBatchUpdatePolicies()

  // Filter policies based on filters
  const policies = useMemo(() => {
    return allPolicies.filter(policy => {
      if (filters.srcIp && !policy.srcIp.includes(filters.srcIp)) return false
      if (filters.dstIp && !policy.dstIp.includes(filters.dstIp)) return false
      if (filters.protocol && policy.protocol !== filters.protocol) return false
      if (filters.action && policy.action !== filters.action) return false
      if (filters.enabled !== undefined && policy.enabled !== filters.enabled) return false
      return true
    })
  }, [allPolicies, filters])

  const handleResetFilters = () => {
    setFilters({})
  }

  const handleCreate = () => {
    setFormMode('create')
    setSelectedPolicy(undefined)
    setIsFormOpen(true)
  }

  const handleEdit = (policy: Policy) => {
    setFormMode('edit')
    setSelectedPolicy(policy)
    setIsFormOpen(true)
  }

  const handleDelete = async (ruleId: number) => {
    const hideLoading = message.loading('Deleting policy...', 0)
    try {
      await deleteMutation.mutateAsync(ruleId)
      hideLoading()
      message.success('Policy deleted successfully')
    } catch (error) {
      hideLoading()
      const errorMsg = error instanceof Error ? error.message : 'Unknown error'
      console.error('Delete policy error:', error)
      message.error(`Failed to delete policy: ${errorMsg}`)
    }
  }

  const handleToggleEnabled = async (ruleId: number, enabled: boolean) => {
    try {
      await updateMutation.mutateAsync({ ruleId, policy: { enabled } })
      message.success(`Policy ${enabled ? 'enabled' : 'disabled'} successfully`)
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : 'Unknown error'
      console.error('Toggle enabled error:', error)
      message.error(`Failed to update policy: ${errorMsg}`)
    }
  }

  const handleFormSubmit = async (values: Omit<Policy, 'ruleId'> | Partial<Policy>) => {
    if (formMode === 'create') {
      await createMutation.mutateAsync(values as Omit<Policy, 'ruleId'>)
    } else if (selectedPolicy) {
      await updateMutation.mutateAsync({ ruleId: selectedPolicy.ruleId, policy: values })
    }
    setIsFormOpen(false)
  }

  const handleRefresh = async () => {
    setIsRefreshing(true)
    await queryClient.invalidateQueries({ queryKey: ['policies'] })
    setIsRefreshing(false)
    message.success('Policies refreshed')
  }

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('Please select policies to delete')
      return
    }

    const hideLoading = message.loading(`Deleting ${selectedRowKeys.length} policies...`, 0)
    try {
      await batchDeleteMutation.mutateAsync(selectedRowKeys)
      hideLoading()
      message.success(`${selectedRowKeys.length} policies deleted successfully`)
      setSelectedRowKeys([])
    } catch (error) {
      hideLoading()
      const errorMsg = error instanceof Error ? error.message : 'Unknown error'
      console.error('Batch delete error:', error)
      message.error(`Failed to delete policies: ${errorMsg}`)
    }
  }

  const handleBatchEnable = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('Please select policies to enable')
      return
    }

    const hideLoading = message.loading(`Enabling ${selectedRowKeys.length} policies...`, 0)
    try {
      await batchUpdateMutation.mutateAsync({ ruleIds: selectedRowKeys, policy: { enabled: true } })
      hideLoading()
      message.success(`${selectedRowKeys.length} policies enabled successfully`)
      setSelectedRowKeys([])
    } catch (error) {
      hideLoading()
      const errorMsg = error instanceof Error ? error.message : 'Unknown error'
      console.error('Batch enable error:', error)
      message.error(`Failed to enable policies: ${errorMsg}`)
    }
  }

  const handleBatchDisable = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('Please select policies to disable')
      return
    }

    const hideLoading = message.loading(`Disabling ${selectedRowKeys.length} policies...`, 0)
    try {
      await batchUpdateMutation.mutateAsync({ ruleIds: selectedRowKeys, policy: { enabled: false } })
      hideLoading()
      message.success(`${selectedRowKeys.length} policies disabled successfully`)
      setSelectedRowKeys([])
    } catch (error) {
      hideLoading()
      const errorMsg = error instanceof Error ? error.message : 'Unknown error'
      console.error('Batch disable error:', error)
      message.error(`Failed to disable policies: ${errorMsg}`)
    }
  }

  const handleViewStats = (ruleId: number) => {
    const policy = allPolicies.find(p => p.ruleId === ruleId)
    if (policy) {
      setSelectedPolicyForStats(policy)
      setStatsModalOpen(true)
    }
  }

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeyPress = (e: KeyboardEvent) => {
      // Ctrl/Cmd + N: Create new policy
      if ((e.ctrlKey || e.metaKey) && e.key === 'n') {
        e.preventDefault()
        handleCreate()
      }
      // Ctrl/Cmd + R: Refresh
      if ((e.ctrlKey || e.metaKey) && e.key === 'r') {
        e.preventDefault()
        handleRefresh()
      }
    }

    window.addEventListener('keydown', handleKeyPress)
    return () => window.removeEventListener('keydown', handleKeyPress)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title level={2} style={{ margin: 0 }}>
          Security Policies
        </Title>
        <div style={{ display: 'flex', gap: 8 }}>
          <Button icon={<ReloadOutlined spin={isRefreshing} />} onClick={handleRefresh} loading={isRefreshing}>
            Refresh
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            Create Policy
          </Button>
        </div>
      </div>

      <Paragraph>
        Manage network security policies. Create, edit, and delete rules to control traffic flow.
      </Paragraph>

      {/* Statistics Cards */}
      {!error && allPolicies.length > 0 && (
        <PolicyStatsCards policies={allPolicies} loading={isLoading} />
      )}

      {/* Filters */}
      {!error && allPolicies.length > 0 && (
        <PolicyFilters filters={filters} onFiltersChange={setFilters} onReset={handleResetFilters} />
      )}

      {/* Batch Operations */}
      {selectedRowKeys.length > 0 && (
        <Alert
          message={`${selectedRowKeys.length} policies selected`}
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          action={
            <Space>
              <Button size="small" icon={<CheckOutlined />} onClick={handleBatchEnable}>
                Enable
              </Button>
              <Button size="small" icon={<CloseOutlined />} onClick={handleBatchDisable}>
                Disable
              </Button>
              <Popconfirm
                title="Delete Selected Policies"
                description={`Are you sure you want to delete ${selectedRowKeys.length} policies?`}
                onConfirm={handleBatchDelete}
                okText="Yes"
                cancelText="No"
              >
                <Button size="small" danger icon={<DeleteOutlined />}>
                  Delete
                </Button>
              </Popconfirm>
            </Space>
          }
        />
      )}

      {error && (
        <Alert
          message="Failed to load policies"
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

      {!error && policies.length === 0 && !isLoading && allPolicies.length === 0 && (
        <Alert
          message="No policies configured"
          description="No security policies have been created yet. Click the 'Create Policy' button to add your first policy."
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />
      )}

      {!error && policies.length === 0 && !isLoading && allPolicies.length > 0 && (
        <Alert
          message="No matching policies"
          description="No policies match the current filter criteria. Try adjusting your filters or clearing them."
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          action={
            <Button size="small" onClick={handleResetFilters}>
              Clear Filters
            </Button>
          }
        />
      )}

      <PolicyTable
        policies={policies}
        policyStats={policyStats}
        loading={isLoading}
        onEdit={handleEdit}
        onDelete={handleDelete}
        onToggleEnabled={handleToggleEnabled}
        onViewStats={handleViewStats}
        selectedRowKeys={selectedRowKeys}
        onSelectionChange={setSelectedRowKeys}
      />

      <PolicyForm
        open={isFormOpen}
        mode={formMode}
        policy={selectedPolicy}
        existingPolicies={allPolicies}
        onSubmit={handleFormSubmit}
        onCancel={() => setIsFormOpen(false)}
      />

      <PolicyStatsModal
        open={statsModalOpen}
        policy={selectedPolicyForStats}
        onClose={() => {
          setStatsModalOpen(false)
          setSelectedPolicyForStats(null)
        }}
      />
    </div>
  )
}
