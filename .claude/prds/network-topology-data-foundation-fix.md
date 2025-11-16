# PRD: 网络拓扑数据基础层完善

## 背景

在实现网络拓扑可视化功能时,发现数据基础层的实时更新功能 `mergeTopologyUpdate` 缺少 LABEL 视图的支持,导致在使用标签聚合模式时,实时流量更新无法正确合并到拓扑数据中。

## 目标

完善网络拓扑数据基础层,确保 IP 和 LABEL 两种视图模式都能正确支持实时数据更新。

## 需求

### 功能需求
1. 补充 `mergeTopologyUpdate` 函数的 LABEL 视图逻辑
2. 支持 SERVICE 类型节点的创建和更新
3. 正确处理标签提取(使用 getServiceLabel)
4. 跳过无标签的流
5. 更新节点和边的流量指标

### 技术需求
1. TypeScript 编译通过
2. 与现有 IP 模式逻辑保持一致
3. 边界情况处理完善

## 成功标准

- [x] LABEL 视图的实时更新逻辑完整实现
- [x] TypeScript 编译通过(0 errors)
- [x] 代码逻辑与 IP 模式一致
- [x] 正确处理无标签流的情况

## 实施结果

- 文件修改: `web/src/utils/topologyUtils.ts`
- 代码行数: 94 行新增代码
- 修复位置: 第 474-567 行
- 验证状态: ✅ 通过

## 相关文档

- [TOPOLOGY_DATA_FOUNDATION_ANALYSIS.md](../../TOPOLOGY_DATA_FOUNDATION_ANALYSIS.md)
- [openspec/TOPOLOGY_PROPOSALS_SPLIT.md](../../openspec/TOPOLOGY_PROPOSALS_SPLIT.md)
