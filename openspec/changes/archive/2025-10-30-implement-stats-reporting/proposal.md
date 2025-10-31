# 变更提案：实施流量统计和报告

**状态**：已归档（回顾性）  
**实施时间**：2025-10-30  
**Change ID**: implement-stats-reporting

## Why

运营可见性需要关于数据包处理、策略执行和系统健康状况的实时统计信息。控制平面需要在不影响数据平面性能的情况下高效访问指标。

## What Changes

- 实施用于无锁统计的每 CPU 数组映射
- 添加 8 个关键指标：总数据包数、允许/拒绝数、新建/关闭/活跃会话数、策略命中/未命中数
- 用于向用户空间报告流事件的环形缓冲区
- 流事件结构，包含 5 元组、时间戳、操作和事件类型
- 选择性事件报告（仅 DENY/LOG 以减少开销）
- 跨 CPU 的用户空间统计聚合
- 实时流事件监控

## Impact

**受影响的规范：**
- `data-plane/statistics`（新增）
- `data-plane/event-reporting`（新增）
- `control-plane/monitoring`（新增）

**受影响的代码：**
- `src/bpf/tc_microsegment.bpf.c` - 统计和事件逻辑
- `src/bpf/headers/common_types.h` - 统计和事件结构
- `src/agent/pkg/dataplane/dataplane.go` - 统计读取和事件监控

**性能影响：**
- 每 CPU 统计：无锁争用，每次更新 < 10ns
- 选择性事件：仅 1% 的数据包生成事件（仅 DENY/LOG）
- 环形缓冲区：高效的内核到用户空间通信

