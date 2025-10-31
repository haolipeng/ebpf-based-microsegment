# 变更提案：实施策略执行引擎

**状态**：已归档（回顾性）  
**实施时间**：2025-10-30  
**Change ID**: implement-policy-enforcement

## Why

在策略匹配确定操作（ALLOW/DENY/LOG）后，系统需要通过允许数据包通过、丢弃它们或记录安全事件来执行这些操作。这是网络微隔离的关键执行点。

## What Changes

- 实施 TC 操作执行（TC_ACT_OK 用于允许，TC_ACT_SHOT 用于丢弃）
- 添加 ALLOW 操作：通过数据包
- 添加 DENY 操作：立即丢弃数据包
- 添加 LOG 操作：允许数据包但报告给用户空间
- 在会话中缓存策略操作以实现快速路径执行
- 与统计跟踪集成
- 可选的 bpf_printk 调试（DEBUG_MODE 控制）

## Impact

**受影响的规范：**
- `data-plane/policy-enforcement`（新增）
- `data-plane/packet-processing`（修改）

**受影响的代码：**
- `src/bpf/tc_microsegment.bpf.c` - 主 TC 程序中的执行逻辑
- `src/bpf/headers/common_types.h` - 操作枚举定义

**安全影响：**
- 对 DENY 策略立即丢弃数据包
- 无绕过路径 - 所有数据包都受执行约束
- 调试日志可以在生产中禁用（DEBUG_MODE=0）

**性能影响：**
- 热路径（缓存会话）：< 5μs 执行决策
- ALLOW/DENY 无额外延迟
- LOG 操作触发环形缓冲区事件（~1μs 开销）

