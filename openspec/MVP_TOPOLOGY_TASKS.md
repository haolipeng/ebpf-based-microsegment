# MVP 任务清单：网络拓扑可视化 (最小可行版本)

> **目标**: 用最少的工作量实现一个基础可用的网络拓扑可视化功能
> **预计工作量**: 2-3 天 (约 16-24 小时)
> **成功标准**: 能在浏览器中看到网络拓扑图,显示 IP 节点和连接关系,支持基本交互

## 📊 实施状态

- **阶段 1 (数据基础层)**: ✅ 已完成 - 发现已有完整实现
- **阶段 2 (可视化核心)**: ✅ 已完成 - 创建了 TopologyGraph 组件和样式
- **阶段 3 (页面集成)**: ✅ 已完成 - 创建页面组件并完成路由和导航集成
- **阶段 4 (最终验证)**: ✅ 代码质量检查完成 | ⏳ 功能测试待手动验证

**实施说明**:
- 大部分基础设施（类型定义、数据聚合、Hook、ECharts 配置）已存在且功能完整
- MVP 创建的新文件：
  - `src/components/topology/TopologyGraph.tsx` (完整版，带事件处理)
  - `src/components/topology/topologyConfig.ts` (ECharts 配置)
  - `src/components/topology/TopologyControls.tsx` (控制面板 - 超出 MVP)
  - `src/components/topology/TopologyLegend.tsx` (图例 - 超出 MVP)
  - `src/components/topology/NodeDetailPanel.tsx` (详情面板 - 超出 MVP)
  - `src/pages/Topology/index.tsx` (页面组件)
  - `src/styles/topology.css` (基础样式)
- 路由和导航菜单集成已完成
- TypeScript 编译通过 (0 errors)
- 所有依赖已安装 (Vite, ECharts, React Query, Ant Design)
- 删除了重复的 web/web 目录
- **下一步**：查看 `TOPOLOGY_MVP_VERIFICATION.md` 进行手动功能验证

---

## 阶段 1: 数据基础层 (4-6 小时)

### 1.1 核心类型定义 ✅
- [x] 新建 `web/src/types/topology.ts`
- [x] 定义最简 `TopologyNode` 接口 (id, label, type, connections)
- [x] 定义最简 `TopologyEdge` 接口 (source, target, protocol)
- [x] 定义最简 `TopologyData` 接口 (nodes, edges)
- [x] **注**: 实际实现包含了更完整的定义 (超出 MVP 要求)

### 1.2 基础数据聚合 ✅
- [x] 新建 `web/src/utils/topologyUtils.ts`
- [x] 实现 `aggregateFlowsToTopology()` 主函数
- [x] 实现 `aggregateByIP()` - **仅支持 IP 视图** (Service 视图留待后期)
- [x] 实现简单的 `getNodeLabel()` (直接显示 IP)
- [x] **注**: 实际实现包含了对数缩放、实时更新等功能 (超出 MVP 要求)

### 1.3 简单的 React Hook ✅
- [x] 新建 `web/src/hooks/useTopology.ts`
- [x] 实现 `useTopology()` Hook
- [x] 与 `useFlows()` 集成获取历史数据
- [x] **注**: 实际实现包含了实时流、防抖等功能 (超出 MVP 要求)

**阶段 1 验证点**:
```bash
npm run typecheck  # TypeScript 编译通过
console.log(aggregateFlowsToTopology(flows))  # 输出正确的 nodes/edges
```

---

## 阶段 2: 可视化核心 (6-8 小时)

### 2.1 ECharts 基础配置 ✅
- [x] 新建 `web/src/components/topology/topologyConfig.ts`
- [x] 实现 `getTopologyChartOption()` 基础版
- [x] 配置力导向 Graph 系列 (使用默认布局参数)
- [x] 配置简单的 tooltip (显示 ID 和连接数)
- [x] **注**: 实际实现包含了高级样式、颜色映射等功能 (超出 MVP 要求)

### 2.2 TopologyGraph 组件 ✅
- [x] 新建 `web/src/components/topology/TopologyGraph.tsx`
- [x] 定义 `TopologyGraphProps` 接口 (data, loading, height)
- [x] 使用 ReactECharts 渲染图表
- [x] 实现基础的加载状态 (Spin)
- [x] 实现基础的空状态 (Empty)
- [x] **注**: MVP 版本暂未实现节点点击事件

### 2.3 最小样式 ✅
- [x] 新建 `web/src/styles/topology.css`
- [x] 编写 `.topology-graph-container` 基础样式
- [x] 包含加载和空状态样式

**阶段 2 验证点**:
```bash
npm run dev
# 浏览器中看到拓扑图渲染成功
# 能拖拽、缩放、悬停查看 tooltip
```

---

## 阶段 3: 页面集成 (4-6 小时)

### 3.1 最简页面组件 ✅
- [x] 新建 `web/src/pages/Topology/index.tsx`
- [x] 调用 `useTopology()` 获取数据
- [x] 渲染页面标题 "Network Topology"
- [x] 渲染 `TopologyGraph` 组件
- [x] 实现基础的错误提示
- [ ] **待完成**: 控制面板、筛选器、详情面板 (留待后期)

**页面结构 (MVP 版本)**:
```tsx
function Topology() {
  const { data, loading } = useTopology();

  return (
    <div style={{ padding: 24 }}>
      <h1>Network Topology</h1>
      <TopologyGraph
        data={data}
        loading={loading}
        height={600}
      />
    </div>
  );
}
```

### 3.2 路由集成 ✅
- [x] 在 `web/src/router.tsx` 添加路由
- [x] 添加 `{ path: '/topology', element: <Topology /> }`

### 3.3 导航菜单 ✅
- [x] 在 `web/src/components/layout/Sidebar.tsx` 添加菜单项
- [x] 引入 `ShareAltOutlined` 图标
- [x] 添加到 menuItems: `{ key: '/topology', icon: <ShareAltOutlined />, label: 'Topology' }`

### 3.4 样式引入 ✅
- [x] 在 `web/src/main.tsx` 添加 `import './styles/topology.css'`

**阶段 3 验证点**:
```bash
# 访问 http://localhost:5173/topology
# 看到拓扑图页面
# 菜单中 Topology 项可点击并高亮
```

---

## 阶段 4: 最终验证 (2-4 小时)

### 4.1 基础测试
- [ ] 测试 10 个节点的渲染 (需启动开发服务器)
- [ ] 测试 50 个节点的渲染 (需启动开发服务器)
- [ ] 测试拖拽、缩放交互 (需启动开发服务器)
- [ ] 测试 tooltip 显示 (需启动开发服务器)

### 4.2 代码质量 ✅
- [x] 运行 TypeScript 编译检查 (通过，无错误)
- [x] 修复 TopologyGraph.tsx 的 ESLint 告警 (已修复 any 类型问题)
- [x] 删除重复的 web/web 目录
- [ ] 确认浏览器 Console 无错误 (需启动开发服务器)
- **注**: 项目中存在一些其他文件的 lint 警告（非 MVP 引入）

### 4.3 跨浏览器测试 (待完成)
- [ ] Chrome 测试通过 (需启动开发服务器)
- [ ] Firefox 测试通过 (可选)

---

## MVP 功能范围

### ✅ MVP 包含的功能
1. **数据层**:
   - 从 Flow 数据聚合生成拓扑
   - 仅支持 IP 视图 (不支持 Service 视图)
   - 静态数据 (不支持实时更新)

2. **可视化层**:
   - 力导向图布局
   - 节点显示 IP 地址
   - 边显示连接关系
   - 基础的 tooltip (ID + 连接数)
   - 支持拖拽、缩放

3. **页面层**:
   - 独立的 `/topology` 路由
   - 简单的标题和图表
   - 导航菜单集成

### ❌ MVP 暂不包含 (留待 v2)
1. **高级数据功能**:
   - Service 视图 (按标签聚合)
   - 实时更新 (WebSocket)
   - 复杂筛选 (协议、状态、动作)
   - 时间范围选择

2. **高级可视化**:
   - 节点颜色/大小映射
   - 边宽度/颜色映射
   - 图例组件
   - 高亮/聚焦动画
   - 节点/边点击事件

3. **高级交互**:
   - 控制面板 (ViewMode、筛选器)
   - 节点详情面板
   - 导出功能
   - 响应式布局

---

## MVP 开发顺序建议

```
Day 1 (8 小时):
  Morning:   阶段 1.1-1.2 (类型定义 + 数据聚合)
  Afternoon: 阶段 1.3 + 阶段 2.1 (Hook + ECharts 配置)

Day 2 (8 小时):
  Morning:   阶段 2.2-2.3 (组件 + 样式)
  Afternoon: 阶段 3 (页面集成)

Day 3 (8 小时):
  Morning:   阶段 4 (测试 + 修复)
  Afternoon: MVP 演示 + 收集反馈
```

---

## 快速启动命令

```bash
# 1. 创建文件结构
mkdir -p web/src/{types,utils,hooks,components/topology,pages/Topology,styles}

# 2. 安装依赖 (如果需要)
cd web && npm install echarts echarts-for-react

# 3. 开发服务器
npm run dev

# 4. 类型检查
npm run typecheck

# 5. 代码检查
npm run lint
```

---

## 成功标准

MVP 完成后,应该能:
1. ✅ 访问 `/topology` 看到拓扑图
2. ✅ 图中显示所有 IP 节点和连接关系
3. ✅ 可以拖拽节点、缩放图表
4. ✅ 悬停节点/边时显示基本信息
5. ✅ 无浏览器错误,TypeScript 编译通过

**不需要**:
- ❌ 完美的视觉效果
- ❌ 复杂的交互
- ❌ 实时更新
- ❌ 详细的筛选

---

## 后续迭代 (v2, v3...) 功能清单

### v2: 增强交互
- [ ] 节点点击显示详情面板
- [ ] 添加控制面板 (ViewMode 切换、刷新按钮)
- [ ] 添加图例组件

### v3: 高级视图
- [ ] Service 视图 (按标签聚合)
- [ ] 节点颜色/大小映射
- [ ] 边宽度/颜色映射

### v4: 实时更新
- [ ] WebSocket 集成
- [ ] 实时数据合并
- [ ] 防抖优化

### v5: 完整筛选
- [ ] 时间范围选择
- [ ] 协议筛选
- [ ] 状态/动作筛选
- [ ] maxNodes 限制

### v6: 优化和完善
- [ ] 响应式布局
- [ ] 性能优化 (100+ 节点)
- [ ] 单元测试
- [ ] 文档和注释

---

## 注意事项

1. **专注 MVP**: 抵制添加"看起来很酷"但不必要的功能的冲动
2. **快速迭代**: 先让基础版本跑起来,再根据反馈优化
3. **技术债控制**: 写清晰的 TODO 注释标记未来要做的功能
4. **用户反馈**: MVP 完成后立即让用户试用,收集真实需求

---

**开始开发前的最后检查**:
- [ ] 确认已有 Flow 数据可用 (通过 `/api/flows` 获取)
- [ ] 确认 ECharts 依赖已安装
- [ ] 确认开发环境正常运行
- [ ] 阅读并理解三个原始提案 (了解完整愿景)

**准备好了就开始吧! 🚀**
