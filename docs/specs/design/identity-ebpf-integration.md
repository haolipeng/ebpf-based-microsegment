# eBPF 身份策略集成实现文档

## 概述

本文档记录了将身份策略（Identity-based Policy）集成到 eBPF 数据平面的实现过程。该集成实现了 Cilium 风格的身份优先策略匹配，在内核层面提供基于安全身份的访问控制。

## 实现日期

2024-12-19

## 架构设计

### 策略匹配流程

```
数据包进入
    ↓
Session Cache 查询 (HOT PATH, >99% 的包)
    ├─ 命中 → 直接返回缓存的 action
    └─ 未命中 ↓
[新增] 身份策略查询 (IDENTITY PATH)
    ├─ IPCache 查询 src_ip → src_identity
    ├─ IPCache 查询 dst_ip → dst_identity
    ├─ 若双方都有有效身份:
    │   └─ identity_policy_map 哈希查询
    │       ├─ 命中 → 创建 session，返回 action
    │       └─ 未命中 → 继续 ↓
    └─ 若任一方无身份 → 继续 ↓
现有 IP 策略查询 (IP PATH)
    ├─ 精确匹配 (policy_map)
    └─ 通配符匹配 (wildcard_policy_map)
    ↓
创建 Session，返回 action
```

### 新增 BPF Maps

| Map 名称 | 类型 | 条目数 | 用途 |
|----------|------|--------|------|
| `ipcache_map` | LPM_TRIE | 65,536 | IP 前缀到身份 ID 的映射 |
| `identity_policy_map` | HASH | 50,000 | 身份策略规则存储 |

## 修改的文件

### 1. src/bpf/tc_microsegment.bpf.c

**位置 1: 特性标志定义（第 45-50 行）**

```c
// Enable identity-based policy matching
// Set to 1 to enable identity-first policy lookup (priority over IP policies)
// Set to 0 to disable and use IP-only policies
#ifndef ENABLE_IDENTITY_POLICY
#define ENABLE_IDENTITY_POLICY 1
#endif
```

**位置 2: 头文件包含（第 158-162 行）**

```c
// Include identity-based policy matching (optional, controlled by ENABLE_IDENTITY_POLICY)
#if ENABLE_IDENTITY_POLICY
#include "headers/ipcache.h"
#include "headers/identity_policy.h"
#endif
```

**位置 3: 身份策略前置查询（第 743-776 行）**

```c
    // === Identity-based policy matching (priority over IP policies) ===
#if ENABLE_IDENTITY_POLICY
    __u32 src_identity = IDENTITY_UNKNOWN;
    __u32 dst_identity = IDENTITY_UNKNOWN;

    // Lookup source and destination identities from IPCache
    src_identity = ipcache_lookup(original_key.src_ip, original_key.ip_version);
    dst_identity = ipcache_lookup(original_key.dst_ip, original_key.ip_version);

    // If both have valid identities, try identity-based policy first
    if (src_identity != IDENTITY_UNKNOWN && dst_identity != IDENTITY_UNKNOWN) {
        int identity_result = match_identity_policy(
            src_identity,
            dst_identity,
            bpf_ntohs(original_key.dst_port),
            original_key.protocol
        );

        if (identity_result > 0) {
            // Identity policy matched
            __u8 identity_action = (identity_result == 1) ? POLICY_ACTION_ALLOW : POLICY_ACTION_DENY;
            update_stats(identity_action == POLICY_ACTION_ALLOW ?
                        STATS_IDENTITY_ALLOWED : STATS_IDENTITY_DENIED);

#if DEBUG_MODE
            bpf_printk("Identity policy matched: src_id=%u dst_id=%u action=%d\n",
                       src_identity, dst_identity, identity_action);
#endif
            // Create session and return early
            create_session(skb, &key, identity_action, now, skb->len, 0, direction, &proc_info);
            return identity_action == POLICY_ACTION_ALLOW ? TC_ACT_OK : TC_ACT_SHOT;
        }
    }
#endif
```

**位置 4: flow_event 身份字段初始化（第 376-379 行）**

```c
    // Identity context (will be populated when identity policy matches)
    // TODO: Pass identity from caller when identity-based policy matches
    event->src_identity = 0;
    event->dst_identity = 0;
```

### 2. src/bpf/headers/common_types.h

**位置 1: 统计计数器（第 194-197 行）**

```c
    // Identity-based policy statistics
    STATS_IDENTITY_ALLOWED,     // Packets allowed by identity policy
    STATS_IDENTITY_DENIED,      // Packets denied by identity policy
    STATS_IDENTITY_LOOKUPS,     // Total IPCache lookups performed
```

**位置 2: flow_event 结构扩展（第 268-270 行）**

```c
    // Identity context (8 bytes)
    __u32 src_identity;   // Source security identity (0 = unknown)
    __u32 dst_identity;   // Destination security identity (0 = unknown)
```

## 依赖的头文件

以下头文件在之前的开发阶段已创建，本次集成直接使用：

### src/bpf/headers/ipcache.h

```c
// IPCache BPF Map 定义
struct ipcache_key {
    __u32 prefixlen;    // LPM 前缀长度
    __u8  ip[16];       // IPv4-mapped IPv6 格式
} __attribute__((packed));

struct ipcache_value {
    __u32 identity;     // 安全身份 ID
    __u8  pad[4];
} __attribute__((packed));

// 核心查询函数
static __always_inline __u32 ipcache_lookup(__u32 *ip, __u8 ip_version);
```

### src/bpf/headers/identity_policy.h

```c
// 身份策略 BPF Map 定义
struct identity_policy_key {
    __u32 src_identity;
    __u32 dst_identity;
    __u16 dst_port;
    __u8  protocol;
    __u8  pad;
} __attribute__((packed));

struct identity_policy_value {
    __u8  action;       // ALLOW/DENY/LOG
    __u8  log_enabled;
    __u16 priority;
    __u32 rule_id;
    __u64 hit_count;
} __attribute__((packed));

// 核心匹配函数
static __always_inline int match_identity_policy(
    __u32 src_identity, __u32 dst_identity,
    __u16 dst_port, __u8 protocol);
```

## 编译验证

### 编译命令

```bash
cd /home/work/ebpf-based-microsegment
make bpf
```

### 编译结果

✅ **编译成功**

警告信息（不影响功能）：
1. `identity_policy.h` 中的 unaligned pointer 警告 - 在 x86 架构上无影响
2. `indexed_policy_match_v3.h` 中的循环展开警告 - 已有警告，不影响功能

### 生成的 Go 绑定

编译后自动生成 `src/agent/pkg/dataplane/bpf_x86_bpfel.go`，包含：

```go
// 身份策略相关类型
type bpfIdentityPolicyKey struct {
    SrcIdentity uint32
    DstIdentity uint32
    DstPort     uint16
    Protocol    uint8
    Pad         uint8
}

type bpfIdentityPolicyValue struct {
    Action     uint8
    LogEnabled uint8
    Priority   uint16
    RuleId     uint32
    HitCount   uint64
}

type bpfIpcacheKey struct {
    Prefixlen uint32
    Ip        [16]uint8
}

type bpfIpcacheValue struct {
    Pad      uint8
    Identity uint32
}

// Map 引用
IdentityPolicyMap *ebpf.Map `ebpf:"identity_policy_map"`
IpcacheMap        *ebpf.Map `ebpf:"ipcache_map"`
```

## 特性标志

| 标志 | 默认值 | 描述 |
|------|--------|------|
| `ENABLE_IDENTITY_POLICY` | 1 | 启用身份策略匹配 |
| `DEBUG_MODE` | 0 | 启用调试日志（含身份匹配日志） |

### 禁用身份策略

如需禁用身份策略（回退到纯 IP 策略），编译时添加：

```bash
make bpf BPF_CFLAGS="-DENABLE_IDENTITY_POLICY=0"
```

## 统计指标

新增以下统计计数器：

| 指标 | 描述 |
|------|------|
| `STATS_IDENTITY_ALLOWED` | 被身份策略允许的数据包数 |
| `STATS_IDENTITY_DENIED` | 被身份策略拒绝的数据包数 |
| `STATS_IDENTITY_LOOKUPS` | IPCache 查询总次数 |

## 后续工作

### 必须完成

1. **用户空间 Map 填充**
   - Agent 需实现填充 `ipcache_map` 的逻辑
   - Agent 需实现填充 `identity_policy_map` 的逻辑
   - 参考: `src/agent/pkg/ipcache/ipcache.go`

2. **用户空间事件解析**
   - 更新 `src/agent/pkg/api/models/flow.go` 以解析 `src_identity/dst_identity` 字段
   - 更新相关的事件处理代码

### 可选优化

1. **警告修复**
   - 调整 `identity_policy_value` 结构对齐消除 unaligned pointer 警告

2. **事件增强**
   - 修改 `push_flow_event` 函数签名，接受身份参数
   - 在身份策略匹配时传递实际身份值到事件

3. **性能监控**
   - 添加身份查询延迟统计
   - 添加 IPCache 命中率统计

## 相关文档

- [Cilium 身份机制研究](./cilium-identity-mechanism-research.md)
- [身份访问控制架构设计](./identity-access-control-architecture.md)
- [BPF 模块文档](../../src/bpf/CLAUDE.md)

## 变更记录

| 日期 | 变更 | 作者 |
|------|------|------|
| 2024-12-19 | 初始 eBPF 集成实现 | Claude Code |
