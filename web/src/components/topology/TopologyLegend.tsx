import { Card, Space, Typography } from 'antd'
import type { TopologyViewMode, TopologyNodeType } from '../../types/topology'
import { getNodeTypeColor, getNodeTypeSymbol } from '../../utils/topologyUtils'

const { Text } = Typography

interface TopologyLegendProps {
  /** View mode */
  viewMode: TopologyViewMode
}

/**
 * Node type configurations for legend
 */
const NODE_TYPE_CONFIG: Record<TopologyNodeType, { label: string; description: string }> = {
  NAMESPACE: { label: 'Namespace', description: 'K8s namespace' },
  SERVICE: { label: 'Service', description: 'K8s service/app' },
  POD: { label: 'Pod', description: 'K8s pod' },
  CONTAINER: { label: 'Container', description: 'Docker container' },
  PROCESS: { label: 'Process', description: 'Running process' },
  IP: { label: 'IP', description: 'Internal IP' },
  EXTERNAL: { label: 'External', description: 'External endpoint' },
}

/**
 * Get relevant node types for the current view mode
 */
function getRelevantNodeTypes(viewMode: TopologyViewMode): TopologyNodeType[] {
  switch (viewMode) {
    case 'NAMESPACE':
      return ['NAMESPACE', 'EXTERNAL']
    case 'SERVICE':
      return ['SERVICE', 'IP', 'EXTERNAL']
    case 'POD':
      return ['POD', 'IP', 'EXTERNAL']
    case 'CONTAINER':
      return ['CONTAINER', 'POD', 'EXTERNAL']
    case 'PROCESS':
      return ['PROCESS', 'CONTAINER', 'EXTERNAL']
    case 'IP':
    default:
      return ['IP', 'EXTERNAL']
  }
}

/**
 * Render symbol for node type
 */
function NodeSymbol({ type, size = 24 }: { type: TopologyNodeType; size?: number }) {
  const color = getNodeTypeColor(type)
  const symbol = getNodeTypeSymbol(type)

  const style: React.CSSProperties = {
    width: size,
    height: size,
    backgroundColor: color,
    marginRight: 8,
    border: '2px solid rgba(255,255,255,0.8)',
    boxShadow: '0 2px 4px rgba(0,0,0,0.15)',
  }

  switch (symbol) {
    case 'circle':
      return <div style={{ ...style, borderRadius: '50%' }} />
    case 'diamond':
      return (
        <div
          style={{
            ...style,
            transform: 'rotate(45deg)',
            borderRadius: 3,
          }}
        />
      )
    case 'rect':
      return <div style={{ ...style, borderRadius: 3 }} />
    case 'roundRect':
      return <div style={{ ...style, borderRadius: 6 }} />
    case 'triangle':
      return (
        <div
          style={{
            width: 0,
            height: 0,
            borderLeft: `${size / 2}px solid transparent`,
            borderRight: `${size / 2}px solid transparent`,
            borderBottom: `${size}px solid ${color}`,
            marginRight: 8,
          }}
        />
      )
    case 'pin':
      return (
        <div
          style={{
            ...style,
            borderRadius: '50% 50% 50% 0',
            transform: 'rotate(-45deg)',
          }}
        />
      )
    default:
      return <div style={{ ...style, borderRadius: '50%' }} />
  }
}

/**
 * Topology legend component
 *
 * Shows node types, sizes, and edge meanings based on current view mode
 */
export default function TopologyLegend({ viewMode }: TopologyLegendProps) {
  const relevantTypes = getRelevantNodeTypes(viewMode)

  return (
    <Card
      title="Legend"
      size="small"
      style={{
        width: 260,
        position: 'absolute',
        right: 20,
        top: 80,
        zIndex: 10,
        maxHeight: 'calc(100% - 100px)',
        overflow: 'auto',
      }}
    >
      <Space direction="vertical" size="small" style={{ width: '100%' }}>
        {/* Node Types */}
        <div>
          <Text strong>Node Types</Text>
          <div style={{ marginTop: 8 }}>
            {relevantTypes.map(type => (
              <div
                key={type}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  marginBottom: 6,
                }}
              >
                <NodeSymbol type={type} size={22} />
                <div>
                  <Text>{NODE_TYPE_CONFIG[type].label}</Text>
                  <Text type="secondary" style={{ fontSize: 11, marginLeft: 4 }}>
                    ({NODE_TYPE_CONFIG[type].description})
                  </Text>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Node Size */}
        <div>
          <Text strong>Node Size = Traffic Volume</Text>
          <div style={{ marginTop: 8, display: 'flex', alignItems: 'flex-end', gap: 8 }}>
            <div style={{ textAlign: 'center' }}>
              <div
                style={{
                  width: 20,
                  height: 20,
                  borderRadius: '50%',
                  backgroundColor: '#999',
                  margin: '0 auto',
                }}
              />
              <Text type="secondary" style={{ fontSize: 10 }}>Low</Text>
            </div>
            <div style={{ textAlign: 'center' }}>
              <div
                style={{
                  width: 35,
                  height: 35,
                  borderRadius: '50%',
                  backgroundColor: '#999',
                  margin: '0 auto',
                }}
              />
              <Text type="secondary" style={{ fontSize: 10 }}>Medium</Text>
            </div>
            <div style={{ textAlign: 'center' }}>
              <div
                style={{
                  width: 50,
                  height: 50,
                  borderRadius: '50%',
                  backgroundColor: '#999',
                  margin: '0 auto',
                }}
              />
              <Text type="secondary" style={{ fontSize: 10 }}>High</Text>
            </div>
          </div>
        </div>

        {/* Edge Colors */}
        <div>
          <Text strong>Connection Status</Text>
          <div style={{ marginTop: 8 }}>
            <div style={{ display: 'flex', alignItems: 'center', marginBottom: 4 }}>
              <div style={{ width: 30, height: 3, backgroundColor: '#91caff', marginRight: 8 }} />
              <Text>Normal</Text>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', marginBottom: 4 }}>
              <div style={{ width: 30, height: 3, backgroundColor: '#95de64', marginRight: 8 }} />
              <Text>Bidirectional</Text>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', marginBottom: 4 }}>
              <div style={{ width: 30, height: 3, backgroundColor: '#ff7875', marginRight: 8 }} />
              <Text>Denied</Text>
            </div>
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <div style={{ width: 30, height: 3, backgroundColor: '#ffc069', marginRight: 8 }} />
              <Text>Warning</Text>
            </div>
          </div>
        </div>

        {/* Node Health */}
        <div>
          <Text strong>Node Health</Text>
          <div style={{ marginTop: 8 }}>
            <div style={{ display: 'flex', alignItems: 'center', marginBottom: 4 }}>
              <div
                style={{
                  width: 20,
                  height: 20,
                  borderRadius: '50%',
                  backgroundColor: '#8c8c8c',
                  border: '2px solid #8c8c8c',
                  marginRight: 8,
                }}
              />
              <Text>Healthy</Text>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', marginBottom: 4 }}>
              <div
                style={{
                  width: 20,
                  height: 20,
                  borderRadius: '50%',
                  backgroundColor: '#8c8c8c',
                  border: '3px solid #faad14',
                  marginRight: 8,
                }}
              />
              <Text>Warning</Text>
            </div>
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <div
                style={{
                  width: 20,
                  height: 20,
                  borderRadius: '50%',
                  backgroundColor: '#8c8c8c',
                  border: '3px solid #f5222d',
                  marginRight: 8,
                }}
              />
              <Text>Critical</Text>
            </div>
          </div>
        </div>

        {/* Tips */}
        <div style={{ marginTop: 8, padding: 8, backgroundColor: '#f5f5f5', borderRadius: 4 }}>
          <Text type="secondary" style={{ fontSize: 11 }}>
            💡 Tips:
            <br />• Hover for details
            <br />• Click node for info
            <br />• Drag to reposition
            <br />• Scroll to zoom
          </Text>
        </div>
      </Space>
    </Card>
  )
}
