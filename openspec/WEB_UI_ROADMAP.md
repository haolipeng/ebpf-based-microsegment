# Web UI 开发路线图

本文档概述微隔离系统前端 Web UI 的开发路线图,将整个开发工作拆分为多个独立的 OpenSpec 变更提案,每个控制在 2-3 天的工作量。

---

## 总体规划

**总工作量**: 12-18 天 (约 2.5-4 周)
**技术栈**: React 18 + TypeScript + Vite + Ant Design
**开发方式**: 规范驱动开发(OpenSpec)

---

## 变更提案列表

### Phase 1: 基础架构 (2-3 天)

**Change ID**: `add-web-ui-foundation`
**优先级**: P0 (必须最先完成)
**状态**: 📝 提案中

**包含内容**:
- 项目初始化(Vite + React + TypeScript)
- 核心依赖安装(antd, react-router, tanstack-query, axios)
- 基础布局组件(Header, Sidebar, Content)
- 路由系统配置
- API 客户端封装
- TypeScript 类型定义
- 开发工具配置(ESLint, Prettier)

**交付产出**:
- 可运行的前端项目框架
- 基础布局和路由正常工作
- API 客户端可连接后端

**文档位置**: [`openspec/changes/add-web-ui-foundation/`](openspec/changes/add-web-ui-foundation/)

---

### Phase 2: Dashboard 仪表板 (2-3 天)

**Change ID**: `add-web-ui-dashboard`
**优先级**: P1 (核心功能)
**状态**: 📝 提案中
**依赖**: add-web-ui-foundation

**包含内容**:
- 关键指标卡片(活跃 Agent、总流量、策略数量)
- 流量趋势图(ECharts 折线图)
- 协议分布图(ECharts 饼图)
- 策略动作分布图(ECharts 柱状图)
- Top Talkers 列表
- 自动刷新机制(30 秒间隔)

**交付产出**:
- 完整的 Dashboard 页面
- 实时数据可视化
- 自动刷新功能

**文档位置**: [`openspec/changes/add-web-ui-dashboard/`](openspec/changes/add-web-ui-dashboard/)

---

### Phase 3: Agent 管理模块 (2-3 天)

**Change ID**: `add-web-ui-agent-management`
**优先级**: P1 (核心功能)
**状态**: 🔜 待创建
**依赖**: add-web-ui-foundation

**包含内容**:
- Agent 列表页面(表格展示)
- 搜索和过滤功能
- Agent 状态指示器(在线/离线)
- Agent 详情页面:
  - 基本信息卡片
  - 性能指标图表(CPU, 内存, 处理包数)
  - 关联的 Flow 列表
  - 应用的 Policy 列表
- 分页和排序

**后端依赖**:
- ✅ `GET /api/v1/agents` - 已实现
- ❌ `GET /api/v1/agents/:id` - 需要后端补充

**预计工作量**: 2-3 天

---

### Phase 4: Flow 分析模块 (3 天)

**Change ID**: `add-web-ui-flow-analytics`
**优先级**: P1 (核心功能)
**状态**: 🔜 待创建
**依赖**: add-web-ui-foundation

**包含内容**:
- Flow 列表页面(高性能表格)
  - 虚拟滚动(支持大数据量)
  - 分页加载
- 高级过滤器面板:
  - 时间范围选择器
  - IP 地址输入
  - 协议选择
  - Agent 选择
  - 标签筛选
- Flow 详情抽屉:
  - 完整元数据展示
  - 源/目标标签
  - 关联策略
- Flow 统计视图:
  - 流量趋势图(时间序列)
  - 协议分布饼图
  - 状态分布柱状图
- 实时 Flow 流(WebSocket):
  - 开关实时更新
  - 可配置订阅过滤器
  - 连接状态指示器

**后端依赖**:
- ✅ `GET /api/v1/flows` - 已实现
- ✅ `GET /api/v1/flows/summary` - 已实现
- ✅ `WebSocket /api/v1/flows/stream` - 已实现

**预计工作量**: 3 天

---

### Phase 5: Policy 管理模块 (2-3 天)

**Change ID**: `add-web-ui-policy-management`
**优先级**: P1 (核心功能)
**状态**: 🔜 待创建
**依赖**: add-web-ui-foundation

**包含内容**:
- Policy 列表页面(表格展示)
- Policy 创建表单(Modal):
  - IP 范围输入(CIDR 格式验证)
  - 标签选择器(多选下拉)
  - 协议和端口配置
  - 动作选择(ALLOW/DENY/LOG)
  - 优先级设置
  - 描述输入
- Policy 编辑功能
- Policy 删除功能(带二次确认)
- Policy 版本显示
- 批量操作(启用/禁用/删除)

**后端依赖**:
- ✅ `GET /api/v1/policies` - 已实现
- ❌ `POST /api/v1/policies` - 需要后端补充
- ❌ `PUT /api/v1/policies/:id` - 需要后端补充
- ❌ `DELETE /api/v1/policies/:id` - 需要后端补充

**预计工作量**: 2-3 天
**注意**: 此模块依赖后端先完成 Policy CRUD API

---

### Phase 6: 依赖关系可视化 (3 天)

**Change ID**: `add-web-ui-visualization`
**优先级**: P2 (重要功能)
**状态**: 🔜 待创建
**依赖**: add-web-ui-foundation

**包含内容**:
- 服务依赖关系图(Cytoscape.js):
  - 节点代表工作负载(按标签聚合)
  - 边代表通信关系(宽度表示流量大小)
  - 颜色编码(按策略动作或协议)
- 交互功能:
  - 拖拽节点调整布局
  - 点击节点查看详情
  - 缩放和平移
- 布局算法选择:
  - 力导向布局
  - 层次布局
  - 圆形布局
- 筛选和搜索:
  - 按协议筛选
  - 按时间范围筛选
  - 节点搜索
- Top Talkers 侧边栏

**后端依赖**:
- ✅ `GET /api/v1/aggregator/dependencies` - 已实现
- ✅ `GET /api/v1/aggregator/top-talkers` - 已实现

**额外依赖**:
- 安装 Cytoscape.js: `npm install cytoscape`
- 安装 React wrapper(可选)

**预计工作量**: 3 天

---

### Phase 7: 实时监控模块 (2 天)

**Change ID**: `add-web-ui-realtime-monitor`
**优先级**: P2 (重要功能)
**状态**: 🔜 待创建
**依赖**: add-web-ui-foundation, add-web-ui-flow-analytics

**包含内容**:
- 实时监控大屏页面
- 实时流量速率图(每秒更新):
  - 滚动时间窗口
  - 实时计算 packets/sec 和 bytes/sec
- 活跃连接计数器
- 策略动作分布(实时饼图)
- 异常流量告警:
  - 自定义阈值配置
  - 告警触发和通知
- WebSocket 状态面板:
  - 连接状态
  - 延迟统计
  - 丢包率
- 全屏模式支持

**后端依赖**:
- ✅ `WebSocket /api/v1/flows/stream` - 已实现
- ✅ `GET /api/v1/flows/stream/stats` - 已实现

**预计工作量**: 2 天

---

### Phase 8: 优化和完善 (2-3 天)

**Change ID**: `add-web-ui-enhancements`
**优先级**: P2 (质量提升)
**状态**: 🔜 待创建
**依赖**: 所有前面的模块

**包含内容**:
- 性能优化:
  - 代码分割和懒加载
  - 组件缓存和 memo
  - 虚拟滚动优化
  - 图表渲染优化
- 用户体验提升:
  - 错误边界(Error Boundary)
  - 友好的错误提示
  - Loading 骨架屏
  - Toast 通知
- 响应式设计完善:
  - 移动端适配
  - 平板适配
  - 暗色模式(可选)
- 国际化支持(可选):
  - i18n 配置
  - 中英文切换
- 单元测试:
  - 关键组件测试
  - Hooks 测试
  - 工具函数测试
- E2E 测试:
  - Playwright 配置
  - 关键流程测试

**预计工作量**: 2-3 天

---

### Phase 9: 部署和文档 (1 天)

**Change ID**: `add-web-ui-deployment`
**优先级**: P1 (生产就绪)
**状态**: 🔜 待创建
**依赖**: 所有前面的模块

**包含内容**:
- Docker 部署:
  - 创建 `web/Dockerfile`(多阶段构建)
  - 使用 Nginx 提供静态文件
  - 配置 Nginx 反向代理(API 和 WebSocket)
- docker-compose 集成:
  - 更新 `deployments/docker-compose.yml`
  - 添加 web 服务
  - 配置网络和依赖
- 环境配置:
  - 生产环境变量配置
  - CORS 配置说明
- 文档:
  - 更新 `web/README.md`
  - 添加开发指南
  - 添加部署指南
  - 添加故障排查指南
- CI/CD(可选):
  - 添加 GitHub Actions 工作流
  - 自动构建和发布 Docker 镜像

**预计工作量**: 1 天

---

## 开发顺序建议

### 关键路径(必须按顺序)
1. **Phase 1** (基础架构) - 必须最先完成
2. **Phase 2** (Dashboard) - 尽早完成,提供整体视图
3. **Phase 4** (Flow 分析) - 核心功能,高优先级
4. **Phase 3** (Agent 管理) - 可与 Phase 4 并行
5. **Phase 5** (Policy 管理) - 依赖后端 API 完成

### 可并行开发
- Phase 3 (Agent 管理) 和 Phase 4 (Flow 分析) 可以并行
- Phase 6 (可视化) 和 Phase 7 (实时监控) 可以并行

### 建议时间线

**Week 1**:
- Day 1-3: Phase 1 (基础架构)
- Day 4-5: Phase 2 (Dashboard) 开始

**Week 2**:
- Day 1: Phase 2 (Dashboard) 完成
- Day 2-4: Phase 4 (Flow 分析)
- Day 5: Phase 3 (Agent 管理) 开始

**Week 3**:
- Day 1-2: Phase 3 (Agent 管理) 完成
- Day 3-5: Phase 5 (Policy 管理) - 需要等待后端 API

**Week 4**:
- Day 1-3: Phase 6 (可视化)
- Day 4-5: Phase 7 (实时监控)

**Week 5** (可选):
- Day 1-3: Phase 8 (优化和完善)
- Day 4: Phase 9 (部署和文档)

---

## 后端 API 依赖总结

### 已实现(可立即使用)
- ✅ `GET /health` - 健康检查
- ✅ `GET /api/v1/agents` - Agent 列表
- ✅ `GET /api/v1/flows` - Flow 列表(分页、过滤)
- ✅ `GET /api/v1/flows/summary` - Flow 统计
- ✅ `GET /api/v1/policies` - Policy 列表
- ✅ `GET /api/v1/aggregator/dependencies` - 依赖关系
- ✅ `GET /api/v1/aggregator/top-talkers` - Top Talkers
- ✅ `WebSocket /api/v1/flows/stream` - 实时流
- ✅ `GET /api/v1/flows/stream/stats` - WebSocket 统计

### 需要后端补充(阻塞部分功能)
- ❌ `GET /api/v1/agents/:id` - Agent 详情(阻塞 Phase 3)
- ❌ `POST /api/v1/policies` - 创建策略(阻塞 Phase 5)
- ❌ `PUT /api/v1/policies/:id` - 更新策略(阻塞 Phase 5)
- ❌ `DELETE /api/v1/policies/:id` - 删除策略(阻塞 Phase 5)

### 需要后端配置
- ⚠️ **CORS 配置** - 必须允许前端域名访问(开发环境 localhost:5173)

---

## 技术债务和未来改进

### 短期改进(1-2 周内)
- 添加更多单元测试
- 完善错误处理
- 优化移动端体验

### 中期改进(1-2 月内)
- 用户认证和权限管理
- 自定义仪表板布局
- 导出报表功能(PDF, CSV)
- 告警规则配置

### 长期改进(3-6 月)
- 多租户支持
- RBAC 权限控制
- 审计日志
- 高级分析(ML 异常检测)

---

## 成功指标

### 功能完整性
- ✅ 所有 6 大核心模块实现
- ✅ WebSocket 实时推送稳定
- ✅ 响应式设计支持移动端

### 性能指标
- ✅ 首屏加载 < 3 秒
- ✅ 页面切换 < 500ms
- ✅ WebSocket 延迟 < 200ms
- ✅ 大数据量列表流畅滚动(60fps)

### 用户体验
- ✅ 操作直观,学习成本低
- ✅ 错误提示友好
- ✅ 加载状态明确

### 代码质量
- ✅ TypeScript 类型覆盖率 > 90%
- ✅ ESLint 无警告
- ✅ 核心功能测试覆盖率 > 70%

---

## 文档索引

- [Web UI 基础架构提案](openspec/changes/add-web-ui-foundation/proposal.md)
- [Web UI 基础架构任务](openspec/changes/add-web-ui-foundation/tasks.md)
- [Web UI 基础架构规范](openspec/changes/add-web-ui-foundation/specs/web-ui-foundation/spec.md)
- [Dashboard 仪表板提案](openspec/changes/add-web-ui-dashboard/proposal.md)
- [Dashboard 仪表板任务](openspec/changes/add-web-ui-dashboard/tasks.md)

---

**最后更新**: 2025-11-07
**维护者**: OpenSpec 规范驱动开发
