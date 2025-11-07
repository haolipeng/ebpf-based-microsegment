import { Layout, Typography } from 'antd'

const { Header: AntHeader } = Layout
const { Title } = Typography

export default function Header() {
  return (
    <AntHeader
      style={{
        display: 'flex',
        alignItems: 'center',
        background: '#001529',
        padding: '0 24px',
      }}
    >
      <Title level={3} style={{ color: 'white', margin: 0 }}>
        eBPF Microsegmentation
      </Title>
    </AntHeader>
  )
}
