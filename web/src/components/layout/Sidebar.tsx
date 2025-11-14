import { Layout, Menu } from 'antd'
import {
  DashboardOutlined,
  ClusterOutlined,
  ApartmentOutlined,
  SafetyOutlined,
  ShareAltOutlined,
} from '@ant-design/icons'
import { useNavigate, useLocation } from 'react-router-dom'
import type { MenuProps } from 'antd'

const { Sider } = Layout

type MenuItem = Required<MenuProps>['items'][number]

const menuItems: MenuItem[] = [
  {
    key: '/',
    icon: <DashboardOutlined />,
    label: 'Dashboard',
  },
  {
    key: '/agents',
    icon: <ClusterOutlined />,
    label: 'Agents',
  },
  {
    key: '/flows',
    icon: <ApartmentOutlined />,
    label: 'Flows',
  },
  {
    key: '/topology',
    icon: <ShareAltOutlined />,
    label: 'Topology',
  },
  {
    key: '/policies',
    icon: <SafetyOutlined />,
    label: 'Policies',
  },
]

export default function Sidebar() {
  const navigate = useNavigate()
  const location = useLocation()

  const handleMenuClick: MenuProps['onClick'] = e => {
    navigate(e.key)
  }

  return (
    <Sider width={200} theme="dark">
      <Menu
        mode="inline"
        selectedKeys={[location.pathname]}
        items={menuItems}
        onClick={handleMenuClick}
        style={{ height: '100%', borderRight: 0 }}
      />
    </Sider>
  )
}
