import { Card, Space, Typography } from 'antd'
import type { TopologyViewMode } from '../../types/topology'

const { Text } = Typography

interface TopologyLegendProps {
  /** 视图模式 */
  viewMode: TopologyViewMode
}

/**
 * 拓扑图图例组件
 * 
 * 显示节点和边的视觉编码规则
 */
export default function TopologyLegend({ viewMode }: TopologyLegendProps) {
  return (
    <Card
      title="Legend"
      size="small"
      style={{ width: 280, position: 'absolute', right: 20, top: 80, zIndex: 10 }}
    >
      <Space direction="vertical" size="small" style={{ width: '100%' }}>
        {/* 节点说明 */}
        <div>
          <Text strong>Node Type</Text>
          <div style={{ marginTop: 8 }}>
            <div style={{ display: 'flex', alignItems: 'center', marginBottom: 4 }}>
              <div
                style={{
                  width: 30,
                  height: 30,
                  borderRadius: '50%',
                  backgroundColor: '#5470c6',
                  marginRight: 8,
                  border: '2px solid #fff',
                  boxShadow: '0 2px 4px rgba(0,0,0,0.2)',
                }}
              />
              <Text>{viewMode === 'IP' ? 'IP Address' : 'Service (IP View)'}</Text>
            </div>
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <div
                style={{
                  width: 30,
                  height: 30,
                  borderRadius: '50%',
                  backgroundColor: '#91cc75',
                  marginRight: 8,
                  border: '2px solid #fff',
                  boxShadow: '0 2px 4px rgba(0,0,0,0.2)',
                }}
              />
              <Text>Service (Label View)</Text>
            </div>
          </div>
        </div>

        {/* 节点大小说明 */}
        <div>
          <Text strong>Node Size</Text>
          <div style={{ marginTop: 8 }}>
            <div style={{ display: 'flex', alignItems: 'center', marginBottom: 4 }}>
              <div
                style={{
                  width: 20,
                  height: 20,
                  borderRadius: '50%',
                  backgroundColor: '#999',
                  marginRight: 8,
                }}
              />
              <Text>Low Traffic</Text>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', marginBottom: 4 }}>
              <div
                style={{
                  width: 40,
                  height: 40,
                  borderRadius: '50%',
                  backgroundColor: '#999',
                  marginRight: 8,
                }}
              />
              <Text>Medium Traffic</Text>
            </div>
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <div
                style={{
                  width: 60,
                  height: 60,
                  borderRadius: '50%',
                  backgroundColor: '#999',
                  marginRight: 8,
                }}
              />
              <Text>High Traffic</Text>
            </div>
          </div>
        </div>

        {/* 连接线说明 */}
        <div>
          <Text strong>Protocol</Text>
          <div style={{ marginTop: 8 }}>
            <div style={{ display: 'flex', alignItems: 'center', marginBottom: 4 }}>
              <div
                style={{
                  width: 40,
                  height: 3,
                  backgroundColor: '#5470c6',
                  marginRight: 8,
                }}
              />
              <Text>TCP</Text>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', marginBottom: 4 }}>
              <div
                style={{
                  width: 40,
                  height: 3,
                  backgroundColor: '#91cc75',
                  marginRight: 8,
                }}
              />
              <Text>UDP</Text>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', marginBottom: 4 }}>
              <div
                style={{
                  width: 40,
                  height: 3,
                  backgroundColor: '#fac858',
                  marginRight: 8,
                }}
              />
              <Text>ICMP</Text>
            </div>
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <div
                style={{
                  width: 40,
                  height: 3,
                  backgroundColor: '#999',
                  marginRight: 8,
                }}
              />
              <Text>Other</Text>
            </div>
          </div>
        </div>

        {/* 连接线粗细说明 */}
        <div>
          <Text strong>Connection Traffic</Text>
          <div style={{ marginTop: 8 }}>
            <div style={{ display: 'flex', alignItems: 'center', marginBottom: 4 }}>
              <div
                style={{
                  width: 40,
                  height: 1,
                  backgroundColor: '#666',
                  marginRight: 8,
                }}
              />
              <Text>Small</Text>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', marginBottom: 4 }}>
              <div
                style={{
                  width: 40,
                  height: 3,
                  backgroundColor: '#666',
                  marginRight: 8,
                }}
              />
              <Text>Medium</Text>
            </div>
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <div
                style={{
                  width: 40,
                  height: 6,
                  backgroundColor: '#666',
                  marginRight: 8,
                }}
              />
              <Text>Large</Text>
            </div>
          </div>
        </div>

        {/* 交互提示 */}
        <div style={{ marginTop: 8, padding: 8, backgroundColor: '#f5f5f5', borderRadius: 4 }}>
          <Text type="secondary" style={{ fontSize: 12 }}>
            💡 Tips:
            <br />
            • Hover to view details
            <br />
            • Click node for full info
            <br />
            • Drag nodes to reposition
            <br />
            • Scroll to zoom
            <br />• Double-click to focus
          </Text>
        </div>
      </Space>
    </Card>
  )
}

