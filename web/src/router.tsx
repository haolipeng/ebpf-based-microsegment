import { createBrowserRouter } from 'react-router-dom'
import MainLayout from './components/layout/MainLayout'
import Dashboard from './pages/Dashboard'
import Agents from './pages/Agents'
import AgentDetail from './pages/Agents/AgentDetail'
import Flows from './pages/Flows'
import Policies from './pages/Policies'
import Topology from './pages/Topology'

const router = createBrowserRouter([
  {
    path: '/',
    element: <MainLayout />,
    children: [
      {
        index: true,
        element: <Dashboard />,
      },
      {
        path: 'agents',
        element: <Agents />,
      },
      {
        path: 'agents/:id',
        element: <AgentDetail />,
      },
      {
        path: 'flows',
        element: <Flows />,
      },
      {
        path: 'policies',
        element: <Policies />,
      },
      {
        path: 'topology',
        element: <Topology />,
      },
    ],
  },
])

export default router
