# 网络拓扑 MVP 功能验证指南

> **状态**: 代码开发 100% 完成,等待功能验证
> **更新时间**: 2025-11-15

## ✅ 已完成项

###  1. 代码实现 (100%)
- ✅ 核心组件: TopologyGraph.tsx, topologyConfig.ts
- ✅ 高级组件: TopologyControls, TopologyLegend, NodeDetailPanel (超出 MVP)
- ✅ 页面集成: pages/Topology/index.tsx
- ✅ 数据层: types/topology.ts, utils/topologyUtils.ts, hooks/useTopology.ts
- ✅ 样式文件: styles/topology.css
- ✅ 路由配置: router.tsx (第 36 行)
- ✅ 导航菜单: Sidebar.tsx (第 33 行)

### 2. 依赖检查 (100%)
- ✅ Vite 已安装
- ✅ ECharts 已安装 (echarts + echarts-for-react)
- ✅ React Query 已安装
- ✅ Ant Design 已安装
- ✅ TypeScript 编译通过 (0 errors)

## ⏳ 待验证项 (需手动操作)

### 第 1 步: 启动后端服务

由于拓扑图需要从后端 API 获取流量数据,首先确保后端服务运行:

```bash
# 方式 1: 如果有 systemd 服务
sudo systemctl status microsegment-agent
sudo systemctl status microsegment-server

# 方式 2: 如果是手动启动
cd /home/work/ebpf-based-microsegment/src/agent
./bin/microsegment-server &
./bin/microsegment-agent &

# 方式 3: 使用启动脚本(如果存在)
./start-all.sh
```

### 第 2 步: 启动 Web 开发服务器

```bash
cd /home/work/ebpf-based-microsegment

# 启动开发服务器 (前台运行)
npm run dev

# 或在后台运行
nohup npm run dev > dev-server.log 2>&1 &
```

预期输出:
```
  VITE v5.x.x  ready in xxx ms

  ➜  Local:   http://localhost:5173/
  ➜  Network: use --host to expose
  ➜  press h + enter to show help
```

### 第 3 步: 访问拓扑页面

1. 打开浏览器访问: `http://localhost:5173/topology`
2. 如果看到重定向到登录页,先登录系统
3. 在左侧菜单找到 "Topology" 项并点击

### 第 4 步: 功能验证清单

#### 基础渲染 (MVP 核心)
- [ ] **页面加载**: 拓扑图页面正常加载,无白屏/崩溃
- [ ] **节点渲染**: 能看到 IP 节点以圆形显示
- [ ] **连接渲染**: 能看到节点之间的连线
- [ ] **布局算法**: 节点使用力导向布局自动分布

#### 交互功能 (MVP 核心)
- [ ] **拖拽节点**: 可以拖动单个节点移动位置
- [ ] **缩放画布**: 鼠标滚轮可以缩放整个图表
- [ ] **平移画布**: 拖动空白区域可以平移视图
- [ ] **Tooltip**: 悬停节点/边时显示详细信息

#### 数据正确性
- [ ] **节点数量**: 节点数量与实际 IP 数量一致
- [ ] **连接关系**: 边的连接关系正确 (源→目标)
- [ ] **空状态**: 无数据时显示 "No Data" 提示

#### 高级功能 (超出 MVP,可选)
- [ ] **控制面板**: 页面顶部显示控制面板
- [ ] **图例**: 显示颜色/大小图例
- [ ] **详情面板**: 点击节点显示详情侧边栏
- [ ] **筛选**: 可以按协议/状态筛选

### 第 5 步: 常见问题排查

#### 问题 1: 拓扑图不渲染
**可能原因**:
- 后端 API 未启动或无数据

**解决方案**:
```bash
# 检查后端 API 是否可访问
curl http://localhost:8080/api/v1/flows

# 检查浏览器控制台是否有 API 错误
# 按 F12 打开开发者工具 → Console 标签
```

#### 问题 2: 显示 "No Data"
**可能原因**:
- 系统中没有流量数据
- 时间范围过滤导致无数据

**解决方案**:
```bash
# 生成一些测试流量
# 方式 1: 使用 curl 生成 HTTP 流量
for i in {1..10}; do
  curl http://localhost:8080/api/v1/health &
done

# 方式 2: 使用 ping 生成 ICMP 流量
ping -c 10 8.8.8.8

# 等待 5-10 秒后刷新页面
```

#### 问题 3: TypeScript 错误
**解决方案**:
```bash
# 清理并重新构建
rm -rf node_modules/.vite
npm run dev
```

#### 问题 4: 端口被占用
**解决方案**:
```bash
# 查找占用 5173 端口的进程
lsof -i:5173

# 杀死进程或更换端口
PORT=5174 npm run dev
```

### 第 6 步: 性能测试 (可选)

如果要测试大规模拓扑:

```bash
# 1. 创建测试数据(通过后端 API 或生成脚本)
# 目标: 10-50 个节点

# 2. 观察渲染性能
# - 初始加载时间 < 2 秒
# - 拖拽流畅度 (60 FPS)
# - 缩放响应时间 < 100ms

# 3. 检查内存使用
# 浏览器开发者工具 → Performance Monitor
# 内存 < 200 MB
```

## 📸 预期效果截图参考

### 正常状态
```
┌─────────────────────────────────────────┐
│  Network Topology            [Controls] │
├─────────────────────────────────────────┤
│                                         │
│     ●────────●                         │
│  10.0.1.1  10.0.1.2                   │
│      │                                  │
│      │                                  │
│      ●                                  │
│   10.0.2.1                             │
│                                         │
│  [Legend: ● IP Node  ── Connection]   │
└─────────────────────────────────────────┘
```

### 空状态
```
┌─────────────────────────────────────────┐
│  Network Topology                       │
├─────────────────────────────────────────┤
│                                         │
│           📊 No Data                   │
│       No network topology data         │
│         available to display           │
│                                         │
└─────────────────────────────────────────┘
```

## ✅ 验证完成标准

当以下所有项都打勾时,MVP 验证完成:

- [ ] 页面正常渲染
- [ ] 节点和边显示正确
- [ ] 基本交互(拖拽/缩放/tooltip)正常
- [ ] 浏览器控制台无错误
- [ ] TypeScript 编译无警告

## 🎯 验证后更新

验证完成后,请更新 `openspec/MVP_TOPOLOGY_TASKS.md`:

```bash
# 将阶段 4 的待办项标记为完成
# 更新实施状态为"全部完成"
# 记录任何发现的问题
```

## 📝 问题反馈模板

如果发现问题,请使用以下格式记录:

```markdown
### 问题: [简短描述]

**环境**:
- 浏览器: Chrome/Firefox x.x
- 操作系统: Linux/Mac/Windows
- 后端版本: x.x.x

**重现步骤**:
1. 访问 /topology
2. 点击...
3. 观察...

**预期行为**:
[描述预期看到什么]

**实际行为**:
[描述实际发生了什么]

**错误信息**:
```
[浏览器控制台错误]
```

**截图**:
[如果有的话]
```

## 🚀 下一步 (验证完成后)

1. **标记 MVP 完成**: 更新所有文档状态
2. **收集用户反馈**: 让团队成员试用
3. **规划 v2 功能**: 根据反馈决定下一步增强方向
4. **性能优化**: 如果发现性能瓶颈,进行针对性优化
5. **文档完善**: 补充用户使用文档

---

**祝验证顺利! 🎉**
