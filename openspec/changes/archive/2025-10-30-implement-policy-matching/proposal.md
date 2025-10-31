# 变更提案：实施 5 元组策略匹配引擎

**状态**：已归档（回顾性）  
**实施时间**：2025-10-30  
**Change ID**: implement-policy-matching

## Why

微隔离需要基于数据包 5 元组（源 IP、目标 IP、源端口、目标端口、协议）的精确网络策略执行。系统需要高效的 O(1) 精确匹配来确定 ALLOW/DENY/LOG 操作，而不会降低性能。

## What Changes

- 实施基于 HASH 映射的策略存储（最大 10k 条目）
- 使用 `policy_key` 结构添加精确的 5 元组匹配
- 创建带有 action、priority、rule_id 的 `policy_value` 结构
- 使用直接映射访问的策略查找优化
- 命中计数器跟踪用于策略分析
- 支持通配符匹配（0.0.0.0 表示任何 IP，0 表示任何端口）

## Impact

**受影响的规范：**
- `data-plane/policy-matching`（新增）
- `control-plane/policy-management`（新增）

**受影响的代码：**
- `src/bpf/tc_microsegment.bpf.c` - 策略匹配逻辑
- `src/bpf/headers/common_types.h` - 策略数据结构
- `src/agent/pkg/policy/policy.go` - 策略管理 API

**性能影响：**
- 策略查找：O(1) 哈希映射访问
- 冷路径开销：新会话 ~8-15μs
- 支持多达 10,000 个策略规则

