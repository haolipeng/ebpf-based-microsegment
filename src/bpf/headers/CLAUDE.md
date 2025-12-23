[上级索引](../CLAUDE.md) > **headers**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# headers

## 架构定位

eBPF 共享头文件 | 输入: 无（类型和函数定义） | 输出: 数据类型、Map 定义、内联函数供 .bpf.c 文件使用

## 文件清单

### 核心类型定义

| 文件 | 职责 | 核心定义 |
|------|------|----------|
| common_types.h | 公共数据类型 | `flow_key`, `session_value`, `policy_key`, `flow_event` |
| process_monitor.h | 进程监控类型 | `process_info`, `process_event` |

### 策略匹配

| 文件 | 职责 | 核心函数 |
|------|------|----------|
| policy_match.h | 基础策略匹配 | `matches_wildcard()`, `lookup_policy_action()` |
| indexed_policy_match_v3.h | 索引策略匹配（最新） | `indexed_policy_lookup()` |
| process_policy_match.h | 进程感知策略匹配 | `match_process_policy()` |
| identity_policy.h | 身份策略匹配 | `lookup_identity_policy()` |
| policy_match_identity.h | 身份策略辅助 | `get_identity_from_ip()` |

### 连接跟踪

| 文件 | 职责 | 核心函数 |
|------|------|----------|
| tcp_state_machine.h | TCP 状态机 | `update_tcp_state()`, `is_connection_closed()` |
| flow_processing.h | 流处理逻辑 | `process_flow()`, `emit_flow_event()` |
| nat_support.h | NAT 支持 | `lookup_conntrack()`, `handle_nat()` |

### 特殊处理

| 文件 | 职责 | 核心函数 |
|------|------|----------|
| fragment_handler.h | IP 分片处理 | `handle_fragment()`, `reassemble_fragment()` |
| fragment_tracking.h | 分片跟踪 | `track_fragment()` |
| protocol_detection.h | 协议检测 | `detect_protocol()`, `parse_l7_header()` |
| protocol_detection_simple.h | 简化协议检测 | `simple_detect_protocol()` |

### IP 缓存

| 文件 | 职责 | 核心函数 |
|------|------|----------|
| ipcache.h | IP 缓存管理 | `ipcache_lookup()`, `ipcache_update()` |

### 历史版本（保留兼容）

| 文件 | 说明 |
|------|------|
| indexed_policy_match.h | v1 版本，已弃用 |
| indexed_policy_match_v2.h | v2 版本，已弃用 |

## 包含顺序

```c
#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include "headers/common_types.h"      // 必须首先包含
#include "headers/tcp_state_machine.h" // 依赖 common_types.h
#include "headers/policy_match.h"      // 依赖 common_types.h
#include "headers/flow_processing.h"   // 依赖上述所有
```

## 编码规范

- 所有函数使用 `static __always_inline` 确保内联
- 边界检查必须在访问数据前完成
- 循环必须有明确的上界（验证器要求）

---

**最后更新**: 2025-12-23
