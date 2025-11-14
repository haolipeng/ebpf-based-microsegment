# 任务清单：Topology Data Foundation

> **实施状态**: ✅ 核心功能已完成 | ⏳ 测试待启动服务器
>
> **完成情况**:
> - ✅ 类型定义 (7/7)
> - ✅ 数据聚合工具 (10/10)
> - ✅ React Hook 集成 (8/8)
> - ⏳ 测试 (0/8) - MVP 暂未实现单元测试
> - ✅ 验证 (2/7) - 代码质量检查完成，功能测试需启动服务器

## 1. 类型定义 ✅
- [x] 1.1 新建 `web/src/types/topology.ts`
- [x] 1.2 定义带指标字段的 `TopologyNode` 接口
- [x] 1.3 定义包含方向的 `TopologyEdge` 接口
- [x] 1.4 定义包含统计字段的 `TopologyData` 接口
- [x] 1.5 定义 `TopologyViewMode` 类型与 `TopologyFilters` 接口
- [x] 1.6 为详情面板定义 `NodeDetail` 接口
- [x] 1.7 确认 TypeScript 编译无错误

## 2. 数据聚合工具 ✅
- [x] 2.1 新建 `web/src/utils/topologyUtils.ts`
- [x] 2.2 实现主函数 `aggregateFlowsToTopology()`
- [x] 2.3 实现 IP 视图聚合 `aggregateByIP()`
- [x] 2.4 实现标签视图聚合 `aggregateByLabel()`
- [x] 2.5 实现标签提取辅助函数 `getServiceLabel()`
- [x] 2.6 实现对数缩放的 `calculateNodeSize()`
- [x] 2.7 实现对数缩放的 `calculateEdgeWidth()`
- [x] 2.8 实现实时更新合并函数 `mergeTopologyUpdate()`
- [x] 2.9 实现节点展示辅助函数 `getNodeLabel()`
- [x] 2.10 为所有导出函数添加 JSDoc 注释

## 3. React Hook 集成 ✅
- [x] 3.1 新建 `web/src/hooks/useTopology.ts`
- [x] 3.2 实现带筛选参数的 `useTopology()` Hook
- [x] 3.3 与 `useFlows()` 集成获取基础数据
- [x] 3.4 与 `useFlowStream()` 集成实时更新
- [x] 3.5 实现 500ms 防抖机制
- [x] 3.6 管理实时 Flow 的本地状态
- [x] 3.7 添加定时器清理逻辑
- [x] 3.8 返回数据、加载、错误以及连接状态

## 4. 测试
- [ ] 4.1 为 `aggregateByIP()` 编写单测
- [ ] 4.2 为 `aggregateByLabel()` 编写单测
- [ ] 4.3 为计算函数编写单测
- [ ] 4.4 为 `mergeTopologyUpdate()` 编写单测
- [ ] 4.5 测试空 Flow 数组场景
- [ ] 4.6 测试缺少标签的 Flow（标签视图）
- [ ] 4.7 测试 maxNodes 限制逻辑
- [ ] 4.8 确认覆盖率 ≥ 80%

## 5. 验证 ✅ (部分完成)
- [x] 5.1 运行 TypeScript 编译并修复所有错误
- [x] 5.2 运行 ESLint 并修复所有告警 (已修复新增代码的告警)
- [ ] 5.3 使用示例 Flow 数据验证聚合结果 (需启动服务器测试)
- [ ] 5.4 验证节点大小计算在 20-80px 之间 (需启动服务器测试)
- [ ] 5.5 验证边宽计算在 1-10px 之间 (需启动服务器测试)
- [ ] 5.6 压测实时更新性能 (需启动服务器测试)
- [ ] 5.7 验证防抖有效降低重渲染 (需启动服务器测试)

